package ui

import(
	"encoding/gob"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"

	"github.com/skypies/util/date"
	"github.com/skypies/util/widget"
	"github.com/skypies/geo"

	fdb "github.com/skypies/flightdb2"
	"github.com/skypies/flightdb2/fgae"
)

func init() {
	http.HandleFunc("/fdb/vector", vectorHandler)  // Returns an idpsec as vector lines in JSON

	http.HandleFunc("/api/flight/lookup", flightLookupHandler)
	http.HandleFunc("/api/procedures", ProcedureHandler)
}

// {{{ vectorHandler

// ?idspec=F12123@144001232[,...]
// &json=1

func vectorHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	db := fgae.FlightDB{C:c}
	
	var idspec fdb.IdSpec
	if idspecs,err := FormValueIdSpecs(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}	else if len(idspecs) != 1 {
		http.Error(w, "wanted just one idspec arg", http.StatusBadRequest)
		return
	} else {
		idspec = idspecs[0]
	}

	if r.FormValue("json") == "" {
		http.Error(w, "vectorHandler is json only at the moment", http.StatusBadRequest)
		return
	}

	f,err := db.LookupMostRecent(db.NewQuery().ByIdSpec(idspec))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if f == nil {
		http.Error(w, fmt.Sprintf("idspec %s not found", idspec), http.StatusBadRequest)
		return
	}

	OutputFlightAsVectorJSON(w, r, f)
}

// }}}
// {{{ OutputFlightAsVectorJSON

// This function is called from complaints/app/v2ui.go
func OutputFlightAsVectorJSON(w http.ResponseWriter, r *http.Request, f *fdb.Flight) {
	// This is such a botch job
	trackspecs := widget.FormValueCommaSepStrings(r, "trackspec")
	if len(trackspecs) == 0 {
		trackspecs = []string{"FOIA", "ADSB", "MLAT", "FA", "fr24"}
	}
	trackName,_ := f.PreferredTrack(trackspecs)

	colorscheme := FormValueColorScheme(r)
	complaintTimes := []time.Time{}
	if colorscheme.Strategy == ByComplaints || colorscheme.Strategy == ByTotalComplaints {
		client := urlfetch.Client(appengine.NewContext(r))
		if times,err := getComplaintTimesFor(client, f); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else {
			complaintTimes = times
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	lines := FlightToMapLines(f, trackName, colorscheme, complaintTimes)
	jsonBytes,err := json.Marshal(lines)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(jsonBytes)
}

// }}}
// {{{ MapLineFormat

func MapLineFormat(f *fdb.Flight, trackName string, l geo.LatlongLine, numComplaints int, colorscheme ColorScheme) (string, float64) {
	// Defaults
	color := "#101000"
	opacity := colorscheme.DefaultOpacity
	
	t := f.Tracks[trackName]
	tp := (*t)[l.I]

	switch colorscheme.Strategy {
	case ByAltitude:
		color = ColorByAltitude(tp.Altitude)

	case ByAngleOfInclination:
		color = ColorByAngle(tp.AngleOfInclination)

	case ByComplaints:
		color = ColorByComplaintCount(numComplaints)
		if numComplaints == 0 {
			opacity = 0.1
		}

	case ByTotalComplaints:
		color = ColorByTotalComplaintCount(numComplaints, 4)  // magic scaling factor
		if numComplaints == 0 {
			opacity = 0.1
		}

	case ByData:
		fallthrough
	default:
		color = "#223399" // FOIA
		colorMap := map[string]string{"ADSB":"#dd6610", "fr24":"#08aa08", "FA":"#0808aa"}
		if k,exists := colorMap[trackName]; exists { color = k }
	}
	
	return color,opacity
}

// }}}
// {{{ FlightToMapLines

func FlightToMapLines(f *fdb.Flight, trackName string, colorscheme ColorScheme, times []time.Time) []MapLine{
	lines   := []MapLine{}

	if trackName == "" { trackName = "fr24"}
	
	sampleRate := time.Millisecond * 2500
	_,origTrack := f.PreferredTrack([]string{trackName})

	// There was once a track with a crazy datapoint in ...
	origTrack.PostProcess()
	track := origTrack.AsSanityFilteredTrack()
	flightLines := track.AsLinesSampledEvery(sampleRate)

	complaintCounts := make([]int, len(flightLines))
	if colorscheme.Strategy == ByComplaints {
		// Walk through lines; for each, bucket up the complaints that occur during it
		j := 0
		t := f.Tracks[trackName]
		for i,l := range flightLines {
			s, e := (*t)[l.I].TimestampUTC, (*t)[l.J].TimestampUTC
			for j < len(times) {
				if times[j].After(s) && !times[j].After(e) {
					// This complaint timestamp hits this flightline
					complaintCounts[i]++
				} else if times[j].After(e) {
					// This complaint is for a future line; move to next line
					break
				}
				// The complaint is not for the future, so consume it
				j++
			}
		}
	}
	
	for i,_ := range flightLines {
		color,opacity := MapLineFormat(f, trackName, flightLines[i], complaintCounts[i], colorscheme)

		if colorscheme.Strategy == ByTotalComplaints {
			color,opacity = MapLineFormat(f, trackName, flightLines[i], len(times), colorscheme)
		}

		mapLine := MapLine{
			Start: flightLines[i].From,
			End: flightLines[i].To,
			Color: color,
			Opacity: opacity,
		}
		lines = append(lines, mapLine)
	}

	return lines
}

// }}}

////////////////////////////////////////////////////////////////////////////////////////////

// {{{ GetContext

// GetContext preps the context with ACL / data-access tokens (some day ...)
func GetContext(r *http.Request) context.Context {
	c,_ := context.WithTimeout(appengine.NewContext(r), 10 * time.Minute)
	return c //appengine.NewContext(r)
}

// }}}
// {{{ flightLookupHandler

// http://fdb.serfr1.org/api/flight/lookup?idspec=A3C3E6@1464046200:1464046200

// ?idspec=F12123@144001232:155001232   (note - time range - may return multiple matches)
//   &trackdata=1                       (include trackdata; omitted by default)

func flightLookupHandler(w http.ResponseWriter, r *http.Request) {
	c := GetContext(r)
	db := fgae.FlightDB{C:c}

	_=db
	str := "OK\n"

	idspecs,err := FormValueIdSpecs(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for _,idspec := range idspecs {
		str += fmt.Sprintf("* %s\n", idspec)

		if flights,err := db.LookupAll(db.NewQuery().ByIdSpec(idspec)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if len(flights) == 0 {
			http.Error(w, fmt.Sprintf("idspec %s not found", idspec), http.StatusBadRequest)
			return
		} else {
			for _,f := range flights {
				str += fmt.Sprintf("  %s\n", f)
			}
		}
	}
	
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(str))
}

// }}}

// {{{ WriteEncodedData

func WriteEncodedData(w http.ResponseWriter, r *http.Request, data interface{}) {
	switch r.FormValue("encoding") {
	case "gob":
		w.Header().Set("Content-Type", "application/octet-stream")
		if err := gob.NewEncoder(w).Encode(data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

	default:  // json
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// }}}
	
// {{{ ProcedureHandler

func ProcedureHandler(w http.ResponseWriter, r *http.Request) {
	db := fgae.FlightDB{C:GetContext(r)}

	tags := widget.FormValueCommaSpaceSepStrings(r,"tags")
	s,e := widget.FormValueEpochTime(r,"s"), widget.FormValueEpochTime(r,"e")
	if s.Unix() == 0 {
		s,e = date.WindowForYesterday()
		s = s.Add(-24 * time.Hour)
		e = e.Add(-24 * time.Hour)
	}

	tStart := time.Now()
	cfs,err,str := db.FetchCondensedFlights(s,e,tags)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.FormValue("text") != "" {
		str += "(elapsed: "+time.Since(tStart).String()+")\n"	
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(str))	
		return
	}

	WriteEncodedData(w,r,cfs)
}

// }}}
// {{{ ProcedureHandlerOld

func ProcedureHandlerOld(w http.ResponseWriter, r *http.Request) {
	db := fgae.FlightDB{C:GetContext(r)}

	// s,e := widget.FormValueEpochTime(r,"s"), widget.FormValueEpochTime(r,"e")
	s,e := date.WindowForYesterday()

	str := fmt.Sprintf("* s: %s\n* e: %s\n", s, e)
	tStart := time.Now()
	
	tags := widget.FormValueCommaSpaceSepStrings(r,"tags")
	q := db.QueryForTimeRange(tags, s, e)
	iter := db.NewIterator(q)
	i := 0
	for iter.Iterate() {
		if iter.Err() != nil { break }
		f := iter.Flight()
		if i<1000 {
			str += fmt.Sprintf(" [%3d] %s %v\n", i, f.BestFlightNumber(), f.WaypointList())
		}
		i++
	}
	if iter.Err() != nil {
		http.Error(w, iter.Err().Error(), http.StatusInternalServerError)
		return
	}

	str += fmt.Sprintf("\nAll done ! %d results, took %s\n", i, time.Since(tStart))
	
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(str))	
}

// }}}

/* 

For 2016/05/24, in DBv2:
 4284  []
 2961  [AL]
  903  [NORCAL:]
  923  [:NORCAL]

  564 [SFO:]
  198 [OAK:]
  141 [SJC:]

All done ! 4323 results, took 32.292807401s   []
All done ! 913 results, took 5.516500834s     [NORCAL:]
All done ! 927 results, took 7.74727197s      [:NORCAL]

 */

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
