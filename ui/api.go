package ui

import(
	"encoding/gob"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"
	
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"

	"github.com/skypies/util/date"
	"github.com/skypies/util/widget"
	"github.com/skypies/geo"

	fdb "github.com/skypies/flightdb"
	"github.com/skypies/flightdb/fgae"
)

// {{{ VectorHandler

// ?idspec=F12123@144001232[,...]
// &json=1

func VectorHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	opt,_ := GetUIOptions(ctx)
	db := fgae.NewDB(ctx)

	idspecs,err := opt.IdSpecs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if len(idspecs) != 1 {
		http.Error(w, "wanted just one idspec arg", http.StatusBadRequest)
		return
	}
	idspec := idspecs[0]

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

	OutputFlightAsVectorJSON(ctx, w, r, f)
}

// }}}
// {{{ OutputFlightAsVectorJSON

func OutputFlightAsVectorJSON(ctx context.Context, w http.ResponseWriter, r *http.Request, f *fdb.Flight) {
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

	// If we have CGI args for a report, process the flight, to get display hints.
	opt,_ := GetUIOptions(ctx)
	if opt.Report != nil {
		if _,err := opt.Report.Process(f); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
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

func MapLineFormat(f *fdb.Flight, t fdb.Track, trackName string, l geo.LatlongLine, numComplaints int, colorscheme ColorScheme) (string, float64) {
	// Defaults
	color := "#101000"
	opacity := colorscheme.DefaultOpacity

	//t := f.Tracks[trackName]
	//tp := (*t)[l.I]
	tp := t[l.I]
	
	// TODO: find a more generic API into the colorscheme, and retire this switch
	switch colorscheme.Strategy {
	case ByAltitude:
		color = colorscheme.ColorByAltitude(tp.Altitude)

	case ByAngleOfInclination:
		color = colorscheme.ColorByAngle(tp.AngleOfInclination)

	case ByComplaints:
		color = colorscheme.ColorByComplaintCount(numComplaints)
		if numComplaints == 0 {
			opacity = 0.1
		}

	case ByTotalComplaints:
		color = colorscheme.ColorByTotalComplaintCount(numComplaints, 4)  // magic scaling factor
		if numComplaints == 0 {
			opacity = 0.1
		}

	case ByExplicitColor:
		color = "#" + colorscheme.ExplicitColor
		
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

	origTrack.PostProcess()
	track := origTrack.AsSanityFilteredTrack()

	// If a report said some data points were uninteresting, we remove them here.
	toRemove := []int{}
	for i,tp := range track {
		if tp.AnalysisDisplay == fdb.AnalysisDisplayOmit {
			toRemove = append(toRemove, i)
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(toRemove))) // Need to remove biggest index first ...
	for _,index := range toRemove {
		track = append(track[:index], track[index+1:]...)
	}
	
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
		color,opacity := MapLineFormat(f, track, trackName, flightLines[i], complaintCounts[i], colorscheme)

		if colorscheme.Strategy == ByTotalComplaints {
			color,opacity = MapLineFormat(f, track, trackName, flightLines[i], len(times), colorscheme)
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

// {{{ FlightLookupHandler

// http://fdb.serfr1.org/api/flight/lookup?idspec=A3C3E6@1464046200:1464046200

// ?idspec=F12123@144001232:155001232   (note - time range - may return multiple matches)
//   &trackdata=1                       (include trackdata; omitted by default)

func FlightLookupHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	db := fgae.NewDB(ctx)
	opt,_ := GetUIOptions(ctx)
	str := "OK\n"

	idspecs,err := opt.IdSpecs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

func ProcedureHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	db := fgae.NewDB(ctx)

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

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
