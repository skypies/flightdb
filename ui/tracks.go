package ui

import(
	"fmt"
	"net/http"
	"time"
	
	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
	"golang.org/x/net/context"

	// "github.com/skypies/adsb"
	"github.com/skypies/geo/sfo"
	// "github.com/skypies/util/widget"
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
	c := appengine.NewContext(r)
	db := fgae.FlightDB{C:c}
	
	// This whole Airframe cache thing should be automatic, and upstream from here.
	airframes := ref.NewAirframeCache(c)

	//colorscheme := FormValueColorScheme(r)
	idspecs,err := FormValueIdSpecs(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
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
		flights = append(flights,f)
	}

	OutputTracksAsLinesOnAMap(w,r,flights)
}

// }}}
// {{{ MapHandler

func MapHandler(w http.ResponseWriter, r *http.Request) {
	points  := []MapPoint{}
	lines   := []MapLine{}
	circles := []MapCircle{}
	
	var params = map[string]interface{}{
		"Legend": "hello",
		"Points": MapPointsToJSVar(points),
		"Lines": MapLinesToJSVar(lines),
		"Circles": MapCirclesToJSVar(circles),
		"MapsAPIKey": "",//kGoogleMapsAPIKey,
		"Center": sfo.KFixes["EPICK"], //sfo.KLatlongSFO,
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
			x := line
			ms.AddLine(MapLine{Line:&x, Color:"#080808"})
		}
		for _,circle := range mr.ToCircles() {
			x := circle
			ms.AddCircle(MapCircle{C:&x, Color:"#080808"})
		}
	}

	return ms
}

// }}}

// {{{ OutputTrackOnAMap

// ?idspec=F12123@144001232[,...]
//  colorby=rcvr
//  fr24=1
//  debug=1
//  boxes=1 boxes=fr24  (see boxes for just that track)
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

	} else {
		f := flights[0]
		// Pick most recent instance, and colorize all visible tracks.
		for _,trackType := range []string{"ADSB", "fr24", "FA:TA", "FA:TZ"} {
			if len(r.FormValue("track")) > 1 && r.FormValue("track") != trackType { continue }
			if _,exists := f.Tracks[trackType]; !exists { continue }
			ms.Points = append(ms.Points, TrackToMapPoints(f.Tracks[trackType], "", bannerText, coloring)...)
		}

		// &boxes=1
		if r.FormValue("boxes") != "" {
			for name,color := range map[string]string{
				"ADSB":"#888811","fr24":"#11aa11","FA:TA":"#1111aa","FA:TZ":"#1111aa",
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
	
	legend := fmt.Sprintf("%s %v", flights[0].IdentString(), flights[0].TagList())
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
// {{{ OutputTracksAsLinesOnAMap

// ?idspec==XX,YY,...
// &colorby=procedure   (what we tagged them as)

func OutputTracksAsLinesOnAMap(w http.ResponseWriter, r *http.Request, flights []*fdb.Flight) {
	lines   := []MapLine{}

	for _,f := range flights {
		flightLines := FlightToMapLines(f)
		lines = append(lines, flightLines...)
	}

	OutputMapLinesOnAMap(w,r,lines)
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
		"MapsAPIKey": "",//kGoogleMapsAPIKey,
		"Center": sfo.KFixes["EPICK"], //sfo.KLatlongSFO,
		"Zoom": 8,
	}
	if err := templates.ExecuteTemplate(w, "fdb2-tracks", params); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// }}}

// {{{ FlightToMapLines

func FlightToMapLines(f *fdb.Flight) []MapLine{
	lines   := []MapLine{}

	sampleRate := time.Second * 5
	flightLines := f.AnyTrack().AsLinesSampledEvery(sampleRate)
	
	for i,_ := range flightLines {
		color := "#ff8822"
		mapLine := MapLine{
			Line: &flightLines[i],
			Color: color,
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
