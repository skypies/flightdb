package ui

import(
	"net/http"
	
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
