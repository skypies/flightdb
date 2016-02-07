package ui

import(
	"fmt"
	"net/http"
	
	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
	//"golang.org/x/net/context"

	"github.com/skypies/geo/altitude"
	"github.com/skypies/util/widget"

	fdb "github.com/skypies/flightdb2"
	"github.com/skypies/flightdb2/fgae"
	"github.com/skypies/flightdb2/fpdf"
	"github.com/skypies/flightdb2/metar"
	"github.com/skypies/flightdb2/ref"
)

func init() {
	http.HandleFunc("/fdb/approach", approachHandler)
	http.HandleFunc("/fdb/approachset", approachHandler) // deprecate this URL
}

// {{{ FormValueColorScheme

func FormValueColorScheme(r *http.Request) fpdf.ColorScheme {
	switch r.FormValue("colorby") {
	case "delta": return fpdf.ByDeltaGroundspeed
	default: return fpdf.ByGroundspeed
	}
}

// }}}
// {{{ FormValueIdSpecs

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

// }}}

// {{{ approachHandler

// ?idspec=XX,YY,...  (or ?idspec=XX&idspec=YYY&...)
// &colorby=delta   (delta groundspeed, instead of groundspeed)

func approachHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	db := fgae.FlightDB{C:c}
	
	// This whole Airframe cache thing should be automatic, and upstream from here.
	airframes := ref.NewAirframeCache(c)

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
		flights = append(flights, f)
	}

	OutputApproachesAsPDF(w,r,flights)
}

// }}}

// {{{ OutputApproachesAsPDF

func OutputApproachesAsPDF(w http.ResponseWriter, r *http.Request, flights []*fdb.Flight) {
	colorscheme := FormValueColorScheme(r)

	s,e,_ := widget.FormValueDateRange(r)

	c := appengine.NewContext(r)
	metar,err := metar.FetchFromNOAA(urlfetch.Client(c), "KSFO",s.AddDate(0,0,-1), e.AddDate(0,0,1))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	pdf := fpdf.NewApproachPdf(colorscheme)
	fStrs := []string{}
outerLoop:
	for _,f := range flights {
		fStrs = append(fStrs, f.String())		

		// Correct the altitudes
		if f.HasTrack("ADSB") {
			track := *f.Tracks["ADSB"]
			for i,tp := range track {
				lookup := metar.Lookup(tp.TimestampUTC)
				track[i].AnalysisAnnotation += fmt.Sprintf("* inHg: %v\n", lookup)
				if lookup.Raw == "" {
					track[i].AnalysisAnnotation += fmt.Sprintf("* BAD Metar, aborting\n")
					continue outerLoop
				}
				
				track[i].IndicatedAltitude = altitude.PressureAltitudeToIndicatedAltitude(
					tp.Altitude, lookup.AltimeterSettingInHg)
				track[i].AnalysisAnnotation += fmt.Sprintf("* pAlt: %.0f, iAlt: %.0f\n",
					tp.Altitude, track[i].IndicatedAltitude)
			}
		}

		fpdf.DrawTrack(pdf, f.AnyTrack(), colorscheme)
	}

	if len(flights) == 1 {
		fpdf.DrawTitle(pdf, fmt.Sprintf("%s", fStrs[0]))
	}
	
	w.Header().Set("Content-Type", "application/pdf")
	if err := pdf.Output(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
