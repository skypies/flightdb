package ui

import(
	"html/template"
	"fmt"
	"net/http"

	"golang.org/x/net/context"

	"github.com/skypies/geo"
	"github.com/skypies/geo/sfo"
	"github.com/skypies/util/widget"
)

// {{{ getGoogleMapsParams

//  &whiteveil=1         (bleach out the map, to make vecctor lines more prominent
//  &map_zoom=10
//  &map_center_lat=37&map_center_long=-122 (alternate center point)
//     &map_center_name=ZORSA           (or use known entities (KSFO, waypoints))
//  &maptype=terrain  (roadmap, satellite, hybrid)
//  &noclassb=1                     (hide the class B overlay)

func getGoogleMapsParams(r *http.Request, params map[string]interface{}) {
	classBOverlay := ! widget.FormValueCheckbox(r, "noclassb")
	whiteVeil := widget.FormValueCheckbox(r, "whiteveil")

	zoom := widget.FormValueInt64(r, "map_zoom")
	if zoom == 0 { zoom = 10 }	


	center := sfo.KFixes["EDDYY"]
	if nll := sfo.FormValueNamedLatlong(r, "map_center"); !nll.Latlong.IsNil() {
		center = nll.Latlong
	}

	mapType := r.FormValue("maptype")
	if mapType == "" { mapType = "Silver" }
	
	params["ClassBOverlay"] = classBOverlay
	params["WhiteOverlay"] = whiteVeil
	params["Center"] = center
	params["Zoom"] = zoom
	params["MapType"] = mapType
	params["MapsAPIKey"] = "AIzaSyDZd-t_YjSNGKmtmh6eR4Bt6eRR_w72b18"
	//params["MapsAPIKey"] = ""//kGoogleMapsAPIKey,
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

// {{{ MapHandler

// ?boxes=b1[,b2]&...{latlongbox.ToCGIArgs("b1")}, etc - render some arbitrary boxes
// ?heatmap=2h  - heatmap of complaint locations over past [duration]
// ?usermap=7d  - heatmap of users who were active within [duration]
// ?usermap=all - heatmap of all user profiles

func MapHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {	
	var params = map[string]interface{}{
		"Legend": "purple={SERFR2,BRIXX1,WWAVS1}; cyan={BIGSUR3}",
	}

	if r.FormValue("heatmap") != "" {
		params["Heatmap"] = r.FormValue("heatmap")

	} else if r.FormValue("usermap") != "" {
		params["Usermap"] = r.FormValue("usermap")

	}
	
	ms := NewMapShapes()	
	if r.FormValue("boxes") != "" {
		for _,stem := range widget.FormValueCommaSepStrings(r,"boxes") {
			box := geo.FormValueLatlongBox(r, stem)
			for _,line := range box.ToLines() {
				ms.AddLine(MapLine{Start:line.From, End:line.To, Color:"#ff0000"})
			}
		}
	}

	MapHandlerWithShapesParams(ctx, w, r, ms, params)
}

// }}}
// {{{ MapHandlerWithShapesParams

func MapHandlerWithShapesParams(ctx context.Context, w http.ResponseWriter, r *http.Request, ms *MapShapes, params map[string]interface{}) {	
	tmpl,_ := GetTemplates(ctx)
	getGoogleMapsParams(r, params)

	params["Zoom"] = 9
	params["Shapes"] = ms
	params["Waypoints"] = WaypointMapVar(sfo.KFixes)
	
	if err := tmpl.ExecuteTemplate(w, "map", params); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
