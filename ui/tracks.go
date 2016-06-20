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
	http.HandleFunc("/fdb/tracks", trackHandler)
	http.HandleFunc("/fdb/trackset", tracksetHandler)
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
		"Waypoints": WaypointMapVar(sfo.KFixes),
		"Zoom": 9,
	}

	if err := templates.ExecuteTemplate(w, "fdb2-tracks", params); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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

// {{{ WaypointMapVar

func WaypointMapVar(in map[string]geo.Latlong) template.JS {
	str := "{\n"
	for name,pos := range in {
		if len(name)>2 && name[0] == 'X' && name[1] == '_' { continue }
		str += fmt.Sprintf("    %q: {pos:{lat:%.6f,lng:%.6f}},\n", name, pos.Lat, pos.Long)
	}
	return template.JS(str + "  }\n")		
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
		"Waypoints": WaypointMapVar(sfo.KFixes),
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
		"Waypoints": WaypointMapVar(sfo.KFixes),
		"Zoom": 8,
	}
	if err := templates.ExecuteTemplate(w, "fdb2-tracks", params); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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
	
	opacity	:= widget.FormValueFloat64EatErrs(r, "maplineopacity")
	if opacity == 0.0 { opacity = 0.6 }

	trackspec := ""
	legend := fmt.Sprintf("%d flights", len(idspecs))
	if rep,err := getReport(r); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if rep != nil {
		if r.FormValue("nofurniture") == "" {
			ms.Add(renderReportFurniture(rep))
		}
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
		"Waypoints": WaypointMapVar(sfo.KFixes),

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

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
