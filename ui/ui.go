package ui

import(
	"fmt"
	"math/rand"
	"net/http"
	
	"github.com/skypies/geo"
	"github.com/skypies/geo/sfo"
	"github.com/skypies/util/widget"

	fdb "github.com/skypies/flightdb2"
	"github.com/skypies/flightdb2/fpdf"
	_ "github.com/skypies/flightdb2/analysis" // Populate the report registry
)

func FormValuePDFColorScheme(r *http.Request) fpdf.ColorScheme {
	switch r.FormValue("colorby") {
	case "delta": return fpdf.ByDeltaGroundspeed
	case "plot": return fpdf.ByPlotKind
	default: return fpdf.ByGroundspeed
	}
}

// Presumes a form field 'idspec', as per identity.go, and also maxflights (as a cap)
func FormValueIdSpecStrings(r *http.Request) ([]string) {
	idspecs := widget.FormValueCommaSepStrings(r, "idspec")

	// If asked for a random subset, go get 'em
	maxFlights := widget.FormValueInt64(r, "maxflights")	
	if maxFlights > 0 && len(idspecs) > int(maxFlights) {
		randomSubset := map[string]int{}

		for i:=0; i<int(maxFlights * 10); i++ {
			if len(randomSubset) >= int(maxFlights) { break }
			randomSubset[idspecs[rand.Intn(len(idspecs))]]++
		}
		
		idspecs = []string{}
		for id,_ := range randomSubset {
			idspecs = append(idspecs, id)
		}
	}

	return idspecs
}

// Presumes a form field 'idspec', as per identity.go, and also maxflights (as a cap)
func FormValueIdSpecs(r *http.Request) ([]fdb.IdSpec, error) {
	ret := []fdb.IdSpec{}
	for _,str := range FormValueIdSpecStrings(r) {
		id,err := fdb.NewIdSpec(str)
		if err != nil { continue } // FIXME - why does this happen ? e.g. ACA564@1389250800
		//if err != nil { return nil, err }
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
