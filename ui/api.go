package ui

import(
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	
	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
	// "golang.org/x/net/context"

	"github.com/skypies/util/widget"
	"github.com/skypies/geo"

	fdb "github.com/skypies/flightdb2"
	"github.com/skypies/flightdb2/fgae"
)

func init() {
	// API handlers - return JSON stuff
	http.HandleFunc("/fdb/vector", vectorHandler)  // Legacy binding; fixup someday
	http.HandleFunc("/api/vector", vectorHandler)  // Returns an idpsec as vector lines in JSON
	http.HandleFunc("/api/flight/lookup", flightLookupHandler)
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
// {{{ flightLookupHandler

// http://fdb.serfr1.org/api/flight/lookup?idspec=A3C3E6@1464046200:1464046200

// ?idspec=F12123@144001232:155001232   (note - time range - may return multiple matches)
//   &trackdata=1                       (include trackdata; omitted by default)

func flightLookupHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
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
	if colorscheme == ByComplaints || colorscheme == ByTotalComplaints {
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
	opacity := 0.6
	
	t := f.Tracks[trackName]
	tp := (*t)[l.I]

	switch colorscheme {
	case ByAltitude:
		color = ColorByAltitude(tp.Altitude)

	case ByAngleOfInclination:
		color = ColorByAngle(tp.AngleOfInclination)

	case ByComplaints:
		color = ColorByComplaintCount(numComplaints)
		if numComplaints == 0 {
			opacity = 0.1
		} else {
			opacity = 0.8
		}

	case ByTotalComplaints:
		color = ColorByTotalComplaintCount(numComplaints, 4)  // magic scaling factor
		if numComplaints == 0 {
			opacity = 0.1
		} else {
			opacity = 0.6
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
	
	sampleRate := time.Second * 5
	_,origTrack := f.PreferredTrack([]string{trackName})

	// There was once a track with a crazy datapoint in ...
	origTrack.PostProcess()
	track := origTrack.AsSanityFilteredTrack()
	flightLines := track.AsLinesSampledEvery(sampleRate)

	complaintCounts := make([]int, len(flightLines))
	if colorscheme == ByComplaints {
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

		if colorscheme == ByTotalComplaints {
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

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
