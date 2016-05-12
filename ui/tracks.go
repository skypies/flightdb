package ui

import(
	"encoding/json"
	"html/template"
	"fmt"
	"net/http"
	"strings"
	"time"
	
	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
	"golang.org/x/net/context"

	"github.com/skypies/geo"
	"github.com/skypies/geo/sfo"
	"github.com/skypies/util/widget"
	fdb "github.com/skypies/flightdb2"
	"github.com/skypies/flightdb2/fgae"
	"github.com/skypies/flightdb2/fr24"
	"github.com/skypies/flightdb2/report"
	"github.com/skypies/flightdb2/ref"
)

func init() {
	http.HandleFunc("/fdb/map", MapHandler)
	http.HandleFunc("/fdb/debug", debugHandler)

	http.HandleFunc("/fdb/tracks", trackHandler)
	http.HandleFunc("/fdb/trackset", tracksetHandler)
	http.HandleFunc("/fdb/vector", vectorHandler)  // Returns an idpsec as vector lines in JSON
}

// {{{ maybeAddFr24Track

func MaybeAddFr24Track(c context.Context, f *fdb.Flight) string {
	fr,_ := fr24.NewFr24(urlfetch.Client(c))
	fr.Prefix = "fr.worrall.cc/"
	fr24Id,debug,err := fr.GetFr24Id(f)
	str := fmt.Sprintf("** fr24 ID lookup: %s, %v\n* debug:-\n%s***\n", fr24Id, err, debug)

	if fr24Id == "" { return str }
	
	var tF *fdb.Track
	if fr24Flight,err := fr.LookupPlaybackTrack(fr24Id); err != nil {
		str += fmt.Sprintf("* fr24 tracklookup: err: %v\n", err)
		return str
	} else {
		// TODO: sanity check this found flight is anything sensible at all
		str += fmt.Sprintf("* fr24 tracklookup found: %s\n", fr24Flight.IdentityString())
		tF = fr24Flight.Tracks["fr24"]
	}

	str += fmt.Sprintf("* [r2] %-6.6s : %s\n", "fr24", tF)

	for name,t := range f.Tracks {
		str += fmt.Sprintf("* [r1] %-6.6s : %s\n", name, t)
		overlaps, conf, debug := t.OverlapsWith(*tF)
		str += fmt.Sprintf("* --> %v, %f\n%s\n***\n", overlaps, conf, debug)
	}

	f.Tracks["fr24"] = tF
	
	return str
}

// }}}

// {{{ debugHandler

func debugHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	str := ""
	
	idspecs,err := FormValueIdSpecs(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//str += fmt.Sprintf("** Idspecs:-\n%#v\n\n", idspecs)
	
	db := fgae.FlightDB{C:c}
	for _,idspec := range idspecs {
		str += fmt.Sprintf("*** %s\n", idspec)
		f,err := db.LookupMostRecent(db.NewQuery().ByIdSpec(idspec))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if f == nil {
			http.Error(w, fmt.Sprintf("idspec %s not found", idspec), http.StatusInternalServerError)
			return
		}
		str += fmt.Sprintf("    %s\n", f.IdSpec())
		str += fmt.Sprintf("    %s\n", f.FullString())
		str += fmt.Sprintf("    %s\n\n", f)
		str += fmt.Sprintf("    index tags: %v\n", f.IndexTagList())
		str += fmt.Sprintf("    /fdb/batch/instance?k=%s\n", f.GetDatastoreKey())
		
		t := f.AnyTrack()
		str += fmt.Sprintf("---- Anytrack: %s\n", t)

		for k,v := range f.Tracks {
			str += fmt.Sprintf("  -- [%-7.7s] %s\n", k, v)
			if r.FormValue("v") != "" {
				for n,tp := range *v {
					str += fmt.Sprintf("    - [%3d] %s\n", n, tp)
				}
			}
		}

		str += fmt.Sprintf("\n--- DebugLog:-\n%s\n", f.DebugLog)
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(fmt.Sprintf("OK\n\n%s", str)))
}

// }}}
// {{{ trackHandler

func trackHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	// This whole Airframe cache thing should be automatic, and upstream from here.
	airframes := ref.NewAirframeCache(c)

	idspecs,err := FormValueIdSpecs(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	db := fgae.FlightDB{C:c}
	flights := []*fdb.Flight{}
	for _,idspec := range idspecs {
		f,err := db.LookupMostRecent(db.NewQuery().ByIdSpec(idspec))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if f == nil {
			http.Error(w, fmt.Sprintf("idspec %s not found", idspec), http.StatusInternalServerError)
			return
		}
		if af := airframes.Get(f.IcaoId); af != nil { f.Airframe = *af }
		flights = append(flights, f)
	}
	
	OutputTracksOnAMap(w, r, flights)
}

// }}}
// {{{ tracksetHandler

// ?idspec=F12123@144001232[,...]
//  &colorby=procedure   (what we tagged them as)

func tracksetHandler(w http.ResponseWriter, r *http.Request) {
	//colorscheme := FormValueColorScheme(r)
	idspecs,err := FormValueIdSpecs(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	idstrings := []string{}
	for _,idspec := range idspecs {
		idstrings = append(idstrings, idspec.String())
	}
	
	OutputMapLinesOnAStreamingMap(w, r, idstrings, "/fdb/vector")
}

// }}}
// {{{ MapHandler

func MapHandler(w http.ResponseWriter, r *http.Request) {
	points  := []MapPoint{}
	lines   := []MapLine{}
	circles := []MapCircle{}
	
	var params = map[string]interface{}{
		"Legend": "purple={SERFR2,BRIXX1,WWAVS1}; cyan={BIGSUR2}",
		"Points": MapPointsToJSVar(points),
		"Lines": MapLinesToJSVar(lines),
		"Circles": MapCirclesToJSVar(circles),
		"MapsAPIKey": "",//kGoogleMapsAPIKey,
		"Center": sfo.KFixes["EDDYY"], //sfo.KLatlongSFO,
		"Zoom": 9,
	}

	if err := templates.ExecuteTemplate(w, "fdb2-tracks", params); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// }}}
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

// {{{ getReport

func getReport(r *http.Request) (*report.Report, error) {
	if r.FormValue("rep") == "" {
		// No report to run
		return nil,nil
	}
	rep,err := report.SetupReport(r)
	return &rep,err
}

// }}}
// {{{ renderReportFurniture

func renderReportFurniture(rep *report.Report) *MapShapes {
	ms := NewMapShapes()

	for _,mr := range rep.Options.ListMapRenderers() {
		for _,line := range mr.ToLines() {
			ms.AddLine(MapLine{Start:line.From, End:line.To, Color:"#080808"})
		}
		for _,circle := range mr.ToCircles() {
			x := circle
			ms.AddCircle(MapCircle{C:&x, Color:"#080808"})
		}
	}

	return ms
}

// }}}

// {{{ OutputTracksOnAMap

// ?idspec=F12123@144001232[,...]
//  colorby=rcvr
//  fr24=1
//  debug=1
//  boxes=1 boxes=fr24  (see boxes for just that track)
//  sample=4  (sample the ADSB track every 4 seconds)
//  track=fr24  (see dots for just that track)

func OutputTracksOnAMap(w http.ResponseWriter, r *http.Request, flights []*fdb.Flight) {
	c := appengine.NewContext(r)
	
	bannerText := ""
	for i,_ := range flights {
		bannerText += fmt.Sprintf("*** %s (%d) %s %s\n", flights[i].IdentityString(),
			len(flights[i].AnyTrack()), "", "")
		//flights[i].AnyTrack().Start(),
		//	flights[i].GetLastUpdate())
	}

	ms := NewMapShapes()
	
//	points  := []MapPoint{}
//	lines   := []MapLine{}
//	circles := []MapCircle{}
	
	rep,err := getReport(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if rep != nil {
		ms.Add(renderReportFurniture(rep))
	}
	
	// This whole Airframe cache thing should be automatic, and upstream from here.
	airframes := ref.NewAirframeCache(c)

	// Preprocess; get airframe data, and run reports (to annotate tracks)
	for i,_ := range flights {
		if af := airframes.Get(flights[i].IcaoId); af != nil {
			flights[i].Airframe = *af
		}
		if rep != nil {
			if _,err := rep.Process(flights[i]); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
	
	coloring := ByADSBReceiver
	switch r.FormValue("colorby") {
	case "src":   coloring = ByDataSource
	case "rcvr":  coloring = ByADSBReceiver
	case "candy": coloring = ByCandyStripe
	}

	// Live fetch, and overlay, a track from fr24.
	if r.FormValue("fr24") != "" {
		coloring = ByDataSource
		//bannerText += MaybeAddFr24Track(c, flights[0])
		MaybeAddFr24Track(c, flights[0])
	}
		
	if r.FormValue("debug") != "" {
		w.Header().Set("Content-Type", "text/plain")
		if rep != nil {
			for reg,_ := range rep.ListGeoRestrictors() {
				bannerText += fmt.Sprintf(" * GeoRestriction: %s\n", reg)
			}
			bannerText += "\n--- report.Log:-\n" + rep.Log
		}
		w.Write([]byte(fmt.Sprintf("OK\n\n%s", bannerText)))
		return
	}

	if len(flights) > 1 {
		// For each flight, translate a track into JS points, add to a JSPointSet
		color := "blue"
		for _,f := range flights {
			text := bannerText + fmt.Sprintf("* %s", f.IdentString())
			ms.Points = append(ms.Points, TrackToMapPoints(f.Tracks["ADSB"], color, text, coloring)...)
			if color == "blue" { color = "yellow" } else { color = "blue" }
		}

	} else if len(flights) == 1 {
		f := flights[0]
		// Pick most recent instance, and colorize all visible tracks.
		for _,trackType := range []string{"ADSB", "MLAT", "fr24", "FA:TA", "FA:TZ", "FOIA"} {
			if len(r.FormValue("track")) > 1 && r.FormValue("track") != trackType { continue }
			if _,exists := f.Tracks[trackType]; !exists { continue }

			if trackType == "ADSB" {
				if secs := widget.FormValueInt64(r, "sample"); secs > 0 {
					newTrack := f.Tracks[trackType].SampleEvery(time.Second * time.Duration(secs), false)
					f.Tracks[trackType] = &newTrack
				}
			}
			f.Tracks[trackType].PostProcess()  // Move upstream
			ms.Points = append(ms.Points, TrackToMapPoints(f.Tracks[trackType], "", bannerText, coloring)...)
		}

		// &boxes=1
		if r.FormValue("boxes") != "" {
			for name,color := range map[string]string{
				"ADSB":"#888811","MLAT":"#8888ff",
				"fr24":"#11aa11","FA:TA":"#1111aa","FA:TZ":"#1111aa","FOIA":"#664433",
			} {
				if len(r.FormValue("boxes")) > 1 && r.FormValue("boxes") != name { continue }
				if t,exists := f.Tracks[name]; exists==true {
					for _,box := range t.AsContiguousBoxes() {
						ms.Lines = append(ms.Lines, LatlongTimeBoxToMapLines(box, color)...)
					}
				}
			}
		}
	}

	legend := flights[0].Legend()
	if len(flights)>1 { legend += fmt.Sprintf(" (%d instances)", len(flights)) }
	
	var params = map[string]interface{}{
		"Legend": legend,
		"Points": MapPointsToJSVar(ms.Points),
		"Lines": MapLinesToJSVar(ms.Lines),
		"Circles": MapCirclesToJSVar(ms.Circles),
		"MapsAPIKey": "",//kGoogleMapsAPIKey,
		"Center": sfo.KFixes["BOLDR"], //sfo.KLatlongSFO,
		"Zoom": 9,
	}

	if err := templates.ExecuteTemplate(w, "fdb2-tracks", params); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// }}}
// {{{ OutputMapLinesOnAMap

// ?idspec==XX,YY,...
// &colorby=procedure   (what we tagged them as)

func OutputMapLinesOnAMap(w http.ResponseWriter, r *http.Request, inputLines []MapLine) {

	ms := NewMapShapes()
	ms.Lines = append(ms.Lines, inputLines...)
	
	if rep,err := getReport(r); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if rep != nil {
		ms.Add(renderReportFurniture(rep))
	}

	legend := fmt.Sprintf("")
	var params = map[string]interface{}{
		"Legend": legend,
		"Points": MapPointsToJSVar(ms.Points),
		"Lines": MapLinesToJSVar(ms.Lines),
		"Circles": MapCirclesToJSVar(ms.Circles),
		"WhiteOverlay": true,
		"MapsAPIKey": "",//kGoogleMapsAPIKey,
		"Center": sfo.KFixes["EPICK"], //sfo.KLatlongSFO,
		"Zoom": 8,
	}
	if err := templates.ExecuteTemplate(w, "fdb2-tracks", params); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// }}}
// {{{ OutputFlightAsVectorJSON

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

// {{{ IdSpecsToJSVar

// Should be a simple list, really
func IdSpecsToJSVar(idspecs []string) template.JS {
	str := "{\n"
	for i,id := range idspecs {
		str += fmt.Sprintf("    %d: {idspec:%q},\n", i, id)
	}
	return template.JS(str + "  }\n")		
}

// }}}
// {{{ OutputMapLinesOnAStreamingMap

// ?idspec==XX,YY,...
// &colorby=procedure   (what we tagged them as - not implemented ?)
// &nofurniture=1       (to suppress furniture)

func OutputMapLinesOnAStreamingMap(w http.ResponseWriter, r *http.Request, idspecs []string, vectorURLPath string) {
	ms := NewMapShapes()
	
	opacity := 0.6
	trackspec := ""
	legend := fmt.Sprintf("%d flights", len(idspecs))
	if rep,err := getReport(r); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if rep != nil {
		if r.FormValue("nofurniture") == "" {
			ms.Add(renderReportFurniture(rep))
		}
		opacity = rep.MapLineOpacity
		trackspec = strings.Join(rep.ListPreferredDataSources(), ",")
		legend += ", "+rep.DescriptionText()
	}

	var params = map[string]interface{}{
		"Legend": legend,
		"Points": MapPointsToJSVar(ms.Points),
		"Lines": MapLinesToJSVar(ms.Lines),
		"Circles": MapCirclesToJSVar(ms.Circles),
		"IdSpecs": IdSpecsToJSVar(idspecs),
		"VectorURLPath": vectorURLPath,  // retire this when DBv1/v2ui.go and friends are gone
		"TrackSpec": trackspec,
		"ColorScheme": FormValueColorScheme(r).String(),
		
		// Would be nice to do something better for rendering hints, before they grow without bound
		"MapLineOpacity": opacity,
		"WhiteOverlay": true,

		"MapsAPIKey": "",//kGoogleMapsAPIKey,
		"Center": sfo.KFixes["EDDYY"], //sfo.KLatlongSFO,
		"Zoom": 10,
	}
	if err := templates.ExecuteTemplate(w, "fdb3-tracks", params); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// }}}

// {{{ getComplaintTimesFor

func getComplaintTimesFor(client *http.Client, f *fdb.Flight) ([]time.Time, error) {
	if f.IataFlight() == "" { return []time.Time{},nil }
	s,e := f.Times()

	times := []time.Time{}
	
	url := fmt.Sprintf("https://stop.jetnoise.net/complaints-for?flight=%s&start=%d&end=%d",
		f.IataFlight(), s.Unix(), e.Unix())

	if resp,err := client.Get(url); err != nil {
		return times, err
	} else {
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			// We don't have a track
			return times, nil
		} else if resp.StatusCode != http.StatusOK {
			return times, fmt.Errorf("Bad status for %s: %v", url, resp.Status)
		} else if err := json.NewDecoder(resp.Body).Decode(&times); err != nil {
			return times, err
		}
	}

	return times, nil
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

	case ByComplaints:
		color = ColorByComplaintCount(numComplaints)
		if numComplaints == 0 {
			opacity = 0.1
		} else {
			opacity = 0.8
		}

	case ByTotalComplaints:
		color = ColorByTotalComplaintCount(numComplaints)
		if numComplaints == 0 {
			opacity = 0.1
		} else {
			opacity = 0.8
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
	_,track := f.PreferredTrack([]string{trackName})
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
