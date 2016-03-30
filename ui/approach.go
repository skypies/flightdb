package ui

import(
	"fmt"
	"net/http"
	"time"
	
	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
	//"golang.org/x/net/context"

	"github.com/skypies/geo/sfo"
	"github.com/skypies/util/widget"

	fdb "github.com/skypies/flightdb2"
	"github.com/skypies/flightdb2/fgae"
	"github.com/skypies/flightdb2/fpdf"
	"github.com/skypies/flightdb2/metar"
	"github.com/skypies/flightdb2/ref"
)

func init() {
	http.HandleFunc("/fdb/approach", approachHandler)
	http.HandleFunc("/fdb/descent",  descentHandler)
}

// {{{ FormValueColorScheme

func FormValueColorScheme(r *http.Request) fpdf.ColorScheme {
	switch r.FormValue("colorby") {
	case "delta": return fpdf.ByDeltaGroundspeed
	case "plot": return fpdf.ByPlotKind
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
// {{{ descentHandler

// ?idspec=XX,YY,...  (or ?idspec=XX&idspec=YYY&...)
//  &sample=N        (sample the track every N seconds)
//  &alt=30000       (max altitude for graph)
//  &length=80       (max distance from origin; in nautical miles)
//  &dist=from       (for distance axis, use dist from airport; by default, uses dist along path)
//  &colorby=delta   (delta groundspeed, instead of groundspeed)

func descentHandler(w http.ResponseWriter, r *http.Request) {
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

	OutputDescentAsPDF(w,r,*(flights[0]))
}

// }}}

// {{{ OutputApproachesAsPDF

func OutputApproachesAsPDF(w http.ResponseWriter, r *http.Request, flights []*fdb.Flight) {
	colorscheme := FormValueColorScheme(r)

	s,e,_ := widget.FormValueDateRange(r)

	// Default to the time range of the flights
	if time.Since(e) > time.Hour*24*365 {
		// assume undef
		s = time.Now().Add(30*24*time.Hour) // Implausibly in the future
		for _,f := range flights {
			fs,fe := f.Times()
			if fs.Before(s) { s = fs }
			if fe.After(e) { e = fe }
		}
		s = s.Add(-24*time.Hour)
		e = s.Add(24*time.Hour)
	}
	
	c := appengine.NewContext(r)
	metars,err := metar.FetchFromNOAA(urlfetch.Client(c), "KSFO",s.AddDate(0,0,-1), e.AddDate(0,0,1))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	pdf := fpdf.NewApproachPdf(colorscheme)
	fStrs := []string{}
//outerLoop:
	for _,f := range flights {
		fStrs = append(fStrs, f.String())		

		trackType,track := f.PreferredTrack([]string{"ADSB", "FOIA"})
		if track == nil { continue }

		if trackType == "ADSB" {
			track.AdjustAltitudes(metars)
		}
		
		fpdf.DrawTrack(pdf, track, colorscheme)
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
// {{{ OutputDescentAsPDF

func OutputDescentAsPDF(w http.ResponseWriter, r *http.Request, f fdb.Flight) {
	colorscheme := FormValueColorScheme(r)
	colorscheme = fpdf.ByPlotKind

	trackType,track := f.PreferredTrack([]string{"ADSB", "FOIA", "fr24"})
	if track == nil {
			http.Error(w, "no acceptable track found", http.StatusInternalServerError)
			return
	}
	if secs := widget.FormValueInt64(r, "sample"); secs > 0 {
		track = track.SampleEvery(time.Second * time.Duration(secs), false)
	}
	track.PostProcess()

	if trackType != "FOIA" { // FOIA track altitudes are already pressure-corrected
		c := appengine.NewContext(r)
	
		// Default to the time range of the flights
		s,e,_ := widget.FormValueDateRange(r)
		if time.Since(e) > time.Hour*24*365 {
			s,e = f.Times()
			s = s.Add(-24*time.Hour)
			e = s.Add(24*time.Hour)
		}

		m,err := metar.FetchFromNOAA(urlfetch.Client(c), "KSFO",s.AddDate(0,0,-1), e.AddDate(0,0,1))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		track.AdjustAltitudes(m)
	}

	lengthNM := 80
	if length := widget.FormValueInt64(r, "length"); length > 0 {
		lengthNM = int(length)
	}
	altitudeMax := 30000
	if alt := widget.FormValueInt64(r, "alt"); alt > 0 {
		altitudeMax = int(alt)
	}
	
	dp := fpdf.DescentPdf{
		ColorScheme: colorscheme,
		OriginPoint: sfo.KLatlongSFO,
		OriginLabel: trackType,
		AltitudeMax: float64(altitudeMax),
		LengthNM:    float64(lengthNM),
	}
	dp.Init()
	dp.DrawFrames()
	dp.DrawCaption(fmt.Sprintf("Flight: %s\nTrack: %s\n", f.FullString(), track.String()))
	//dp.DrawColorSchemeKey()

	if r.FormValue("dist") == "from" {
		dp.DrawTrackAsDistanceFromOrigin(track)
	} else {
		dp.DrawTrackAsDistanceAlongPath(track)
	}

	w.Header().Set("Content-Type", "application/pdf")
	if err := dp.Output(w); err != nil {
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
