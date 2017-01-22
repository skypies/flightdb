package ui

import(
	"html/template"
	"fmt"
	"net/http"

	"github.com/skypies/geo"
	"github.com/skypies/geo/sfo"
	"github.com/skypies/util/widget"
)

func init() {
	http.HandleFunc("/fdb/map", MapHandler)
}

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

// {{{ MapHandler

// ?boxes=b1[,b2]&...{latlongbox.ToCGIArgs("b1")}, etc - render some arbitrary boxes

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
	
	if r.FormValue("boxes") != "" {
		ms := NewMapShapes()	
		for _,stem := range widget.FormValueCommaSepStrings(r,"boxes") {
			box := geo.FormValueLatlongBox(r, stem)
			for _,line := range box.ToLines() {
				ms.AddLine(MapLine{Start:line.From, End:line.To, Color:"#ff0000"})
			}
		}
		params["Lines"] = MapLinesToJSVar(ms.Lines)
		params["Points"] = MapPointsToJSVar(ms.Points) // Just to trigger map-shapes mode
	}
	
	if err := templates.ExecuteTemplate(w, "map", params); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
