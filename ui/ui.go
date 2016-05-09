package ui

import(
	"fmt"
	"net/http"
	
	"github.com/skypies/geo"
	"github.com/skypies/geo/sfo"
	"github.com/skypies/util/widget"

	fdb "github.com/skypies/flightdb2"
	"github.com/skypies/flightdb2/fpdf"
	_ "github.com/skypies/flightdb2/analysis" // Populate the report registry
)


func FormValueColorScheme(r *http.Request) fpdf.ColorScheme {
	switch r.FormValue("colorby") {
	case "delta": return fpdf.ByDeltaGroundspeed
	case "plot": return fpdf.ByPlotKind
	default: return fpdf.ByGroundspeed
	}
}

// Presumes a form field 'idspec', as per identity.go
func FormValueIdSpecs(r *http.Request) ([]fdb.IdSpec, error) {
	ret := []fdb.IdSpec{}

	for _,str := range widget.FormValueCommaSepStrings(r, "idspec") {
		id,err := fdb.NewIdSpec(str)
		if err != nil { return nil, err }
		ret = append(ret, id)
	}

	return ret, nil
}

func FormValueAirportLocation(r *http.Request, name string) (geo.Latlong, error) {
	if pos,exists := sfo.KAirports[r.FormValue(name)]; exists {
		return pos, nil
	}
	return geo.Latlong{}, fmt.Errorf("airport '%s' not known; try KSFO,KOAK etc", r.FormValue(name))
}
