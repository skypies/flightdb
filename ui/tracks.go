package ui

import(
	"encoding/json"
	"html/template"
	"fmt"
	"net/http"
	"strings"
	"time"
	
	"golang.org/x/net/context"
	"google.golang.org/appengine/urlfetch"

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
	http.HandleFunc("/fdb/tracks", UIOptionsHandler(trackHandler))
	http.HandleFunc("/fdb/trackset", UIOptionsHandler(tracksetHandler))
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

//  &all=1  - show all instances of the IdSpec

func trackHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// This whole Airframe cache thing should be automatic, and upstream from here.
	airframes := ref.NewAirframeCache(ctx)
	opt,_ := GetUIOptions(ctx)

	idspecs,err := opt.IdSpecs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	db := fgae.FlightDB{C:ctx}
	flights := []*fdb.Flight{}
	for _,idspec := range idspecs {
		if r.FormValue("all") != "" {
			results,err := db.LookupAll(db.NewQuery().ByIdSpec(idspec))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			for _,f := range results {
				if af := airframes.Get(f.IcaoId); af != nil { f.OverlayAirframe(*af) }
				flights = append(flights, f)
			}

		} else {
			f,err := db.LookupMostRecent(db.NewQuery().ByIdSpec(idspec))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			} else if f == nil {
				http.Error(w, fmt.Sprintf("idspec %s not found", idspec), http.StatusInternalServerError)
				return
			}
			if af := airframes.Get(f.IcaoId); af != nil { f.OverlayAirframe(*af) }
			flights = append(flights, f)
		}			
	}
	
	OutputTrackpointsOnAMap(ctx, w, r, flights)
}

// }}}
// {{{ tracksetHandler

func tracksetHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	opt,_ := GetUIOptions(ctx)

	// Check we can parse them up-front, so we can return an ascii error
	if _,err := opt.IdSpecs(); err != nil {
		http.Error(w, fmt.Sprintf("idspec parsing: %v", err.Error()), http.StatusInternalServerError)
		return
	}

	OutputMapLinesOnAStreamingMap(ctx, w, r, "/fdb/vector")
}

// }}}
// {{{ MapHandler

func MapHandler(w http.ResponseWriter, r *http.Request) {	
	var params = map[string]interface{}{
		"Legend": "purple={SERFR2,BRIXX1,WWAVS1}; cyan={BIGSUR3}",
		"Waypoints": WaypointMapVar(sfo.KFixes),
	}
	getGoogleMapsParams(r, params)
	params["Zoom"] = 9

	if r.FormValue("heatmap") != "" {
		params["Heatmap"] = r.FormValue("heatmap")
	}
	
	if err := templates.ExecuteTemplate(w, "map", params); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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
// {{{ getGoogleMapsParams

//  &whiteveil=1         (bleach out the map, to make vecctor lines more prominent
//  &zoom=10
//  &center_lat=37&center_long=-122 (alternate center point)
//  &maptype=terrain  (roadmap, satellite, hybrid)
//  &noclassb=1                     (hide the class B overlay)

func getGoogleMapsParams(r *http.Request, params map[string]interface{}) {
	classBOverlay := ! widget.FormValueCheckbox(r, "noclassb")
	whiteVeil := widget.FormValueCheckbox(r, "whiteveil")

	zoom := widget.FormValueInt64(r, "zoom")
	if zoom == 0 { zoom = 10 }	

	center := geo.FormValueLatlong(r, "center")
	if center.IsNil() { center = sfo.KFixes["EDDYY"] }

	mapType := r.FormValue("maptype")
	if mapType == "" { mapType = "Silver" }
	
	params["ClassBOverlay"] = classBOverlay
	params["WhiteOverlay"] = whiteVeil
	params["Center"] = center
	params["Zoom"] = zoom
	params["MapType"] = mapType
	params["MapsAPIKey"] = ""//kGoogleMapsAPIKey,
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

// {{{ OutputTrackpointsOnAMap

// ?idspec=F12123@144001232[,...]
//  colorby=rcvr
//  fr24=1
//  debug=1
//  boxes=1 boxes=fr24       (see boxes for just that track)
//  sample=4                 (sample the ADSB track every 4 seconds)
//  track=fr24               (see dots for just that track)
//  clip1=EPICK&clip2=EDDYY  (clip to points between those waypoints)

func OutputTrackpointsOnAMap(ctx context.Context, w http.ResponseWriter, r *http.Request, flights []*fdb.Flight) {
	opt,_ := GetUIOptions(ctx)

	bannerText := ""
	for i,_ := range flights {
		bannerText += fmt.Sprintf("*** %s (%d) %s %s\n", flights[i].IdentityString(),
			len(flights[i].AnyTrack()), "", "")
	}

	ms := NewMapShapes()	

	if opt.Report != nil {
		ms.Add(renderReportFurniture(opt.Report))
	}
	
	// This whole Airframe cache thing should be automatic, and upstream from here.
	airframes := ref.NewAirframeCache(ctx)

	// Preprocess; get airframe data, and run reports (to annotate tracks)
	for i,_ := range flights {
		if af := airframes.Get(flights[i].IcaoId); af != nil {
			flights[i].OverlayAirframe(*af)
		}
		if opt.Report != nil {
			if _,err := opt.Report.Process(flights[i]); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
	
	coloring := ByADSBReceiver
	switch r.FormValue("colorby") {
	case "src":   coloring = ByData
	case "rcvr":  coloring = ByADSBReceiver
	case "candy": coloring = ByCandyStripe
	}

	// Live fetch, and overlay, a track from fr24.
	if r.FormValue("fr24") != "" {
		coloring = ByData
		//bannerText += MaybeAddFr24Track(c, flights[0])
		MaybeAddFr24Track(ctx, flights[0])
	}
		
	if r.FormValue("debug") != "" {
		w.Header().Set("Content-Type", "text/plain")
		if opt.Report != nil {
			for reg,_ := range opt.Report.ListGeoRestrictors() {
				bannerText += fmt.Sprintf(" * GeoRestriction: %s\n", reg)
			}
			bannerText += "\n--- report.Log:-\n" + opt.Report.Log
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
			
			track := f.Tracks[trackType]
			track.PostProcess()  // Move upstream ?

			// &clip1=EPICK&clip2=EDDYY
			if r.FormValue("clip1") != "" {
				if !f.HasWaypoint(r.FormValue("clip1")) || !f.HasWaypoint(r.FormValue("clip2")) {
					http.Error(w,
						fmt.Sprintf("{%s,%s} not found", r.FormValue("clip1"), r.FormValue("clip2")),
						http.StatusInternalServerError)
					return
				}
				tps := track.ClipTo(f.Waypoints[r.FormValue("clip1")], f.Waypoints[r.FormValue("clip2")])
				t2 := fdb.Track(tps).SampleEveryDist(3.0, false)
				track = &t2				
			}

			ms.Points = append(ms.Points, TrackToMapPoints(track, "", bannerText, coloring)...)
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

	legend := ""
	if len(flights)>0 { legend += flights[0].Legend() }
	if len(flights)>1 { legend += fmt.Sprintf(" (%d instances)", len(flights)) }
	
	var params = map[string]interface{}{
		"Legend": legend,
		"Points": MapPointsToJSVar(ms.Points),
		"Lines": MapLinesToJSVar(ms.Lines),
		"Circles": MapCirclesToJSVar(ms.Circles),
		"Waypoints": WaypointMapVar(sfo.KFixes),
	}

	getGoogleMapsParams(r, params)

	if err := templates.ExecuteTemplate(w, "map", params); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// }}}
// {{{ OutputMapLinesOnAStreamingMap

// ?idspec==XX,YY,...
//  &colorby=procedure   (what we tagged them as - not implemented ?)
//  &nofurniture=1       (to suppress furniture)

func OutputMapLinesOnAStreamingMap(ctx context.Context, w http.ResponseWriter, r *http.Request, vectorURLPath string) {
	opt,_ := GetUIOptions(ctx)
	ms := NewMapShapes()
	legend := ""

	// Add a permalink, if we can
	if opt.ResultsetID != "" {
		// Rejigger all the POST and GET data into a single GET URL, then add our new field.
		vals := widget.ExtractAllCGIArgs(r)
		vals.Del("idspec")
		vals.Set("resultset", opt.ResultsetID)

		urlstr := widget.URLStringReplacingGETArgs(r,&vals)
		legend += fmt.Sprintf("[<a target=\"_blank\" href=\"%s\">permalink</a>] ", urlstr)
	}

	legend += fmt.Sprintf("%d flights", len(opt.IdSpecStrings))

	trackspec := ""
	if opt.Report != nil {
		if r.FormValue("nofurniture") == "" {
			ms.Add(renderReportFurniture(opt.Report))
		}
		trackspec = strings.Join(opt.Report.ListPreferredDataSources(), ",")
		legend += ", "+opt.Report.DescriptionText()
	}
	
	var params = map[string]interface{}{
		"Legend": legend,
		"Points": MapPointsToJSVar(ms.Points),
		"Lines": MapLinesToJSVar(ms.Lines),
		"Circles": MapCirclesToJSVar(ms.Circles),
		"IdSpecs": IdSpecsToJSVar(opt.IdSpecStrings),
		"VectorURLPath": vectorURLPath,  // retire this when DBv1/v2ui.go and friends are gone
		"TrackSpec": trackspec,
		"ColorScheme": opt.ColorScheme,
		
		"Waypoints": WaypointMapVar(sfo.KFixes),
	}
	getGoogleMapsParams(r, params)

	if err := templates.ExecuteTemplate(w, "map", params); err != nil {
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
