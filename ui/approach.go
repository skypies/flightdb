package ui

import(
	"fmt"
	"net/http"
	"strings"
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

// {{{ approachHandler

// ?idspec=XX,YY,...    (or ?idspec=XX&idspec=YYY&...)
//  &colorby=delta      (delta groundspeed, instead of groundspeed)

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

// ?idspec=XX,YY,...    (or ?idspec=XX&idspec=YYY&...)
//  &sample=N           (sample the track every N seconds)
//  &averagingwindow=2m (duration to average over)
//  &alt=30000          (max altitude for graph)
//  &length=80          (max distance from origin; in nautical miles)
//  &dist=from          (for distance axis, use dist from airport; by default, uses dist along path)
//  &colorby=delta      (delta groundspeed, instead of groundspeed)

func descentHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	db := fgae.FlightDB{C:c}
	
	// This whole Airframe cache thing should be automatic, and upstream from here.
	airframes := ref.NewAirframeCache(c)

	idspecs,err := FormValueIdSpecs(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dp := DescentPDFInit(w, r, len(idspecs))

	if len(idspecs) > 10 {
		dp.LineThickness = 0.1
		dp.LineOpacity = 0.25
	}
	
	for _,idspec := range idspecs {
		f,err := db.LookupMostRecent(db.NewQuery().ByIdSpec(idspec))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if f == nil {
			http.Error(w, fmt.Sprintf("idspec %s not found", idspec), http.StatusBadRequest)
			return
		}
		if af := airframes.Get(f.IcaoId); af != nil { f.Airframe = *af }

		if err := DescentPDFAddFlight(r, dp, f); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	DescentPDFFinalize(w,r,dp)
}

// }}}

// {{{ OutputApproachesAsPDF

// This is the old handler, useful for Class B
func OutputApproachesAsPDF(w http.ResponseWriter, r *http.Request, flights []*fdb.Flight) {
	colorscheme := FormValuePDFColorScheme(r)

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

		trackType,track := f.PreferredTrack([]string{"FOIA", "ADSB"})
		if track == nil || trackType == "" {
			continue
		}

		track.PostProcess() // Calculate groundspeed data for FOIA data
		
		if trackType == "ADSB" {
			track.AdjustAltitudes(metars)
		} else {
			for i,_ := range track {
				track[i].IndicatedAltitude = track[i].Altitude
			}
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

// {{{ DescentPDFInit

func DescentPDFInit(w http.ResponseWriter, r *http.Request, numFlights int) *fpdf.DescentPdf {
	colorscheme := FormValuePDFColorScheme(r)
	colorscheme = fpdf.ByPlotKind

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
		AltitudeMax: float64(altitudeMax),
		LengthNM:    float64(lengthNM),
		ToShow:      map[string]bool{"altitude":true, "groundspeed":true, "verticalspeed":true},
		ShowDebug:  (r.FormValue("debug") != ""),
		AveragingWindow: widget.FormValueDuration(r, "averagingwindow"),
	}

	if originPos,err := FormValueAirportLocation(r, "airportdest"); err == nil {
		dp.OriginPoint = originPos
		dp.OriginLabel = r.FormValue("airportdest")
	}
	
	if widget.FormValueCheckbox(r, "showaccelerations") {
		dp.ToShow["groundacceleration"],dp.ToShow["verticalacceleration"] = true,true
	}
	if widget.FormValueCheckbox(r, "showangleofinclination") {
		dp.ToShow["angleofinclination"] = true
	}

	dp.Init()

	if r.FormValue("asdepartures") != "" {
		dp.ReconfigureForDepartures()
	}

	dp.DrawFrames()

	if rep,err := getReport(r); err==nil && rep!=nil {
		dp.Caption += fmt.Sprintf("%d flights, %s\n", numFlights, rep.DescriptionText())
	}
	
	return &dp
}

// }}}
// {{{ DescentPDFAddFlight

func DescentPDFAddFlight(r *http.Request, dp *fpdf.DescentPdf, f *fdb.Flight) error {
	if t,err := flightToDescentTrack(r, f); err != nil {
		return err
	} else {
		if r.FormValue("dist") == "from" {
			dp.DrawTrackAsDistanceFromOrigin(t)
		} else if r.FormValue("asdepartures") != "" {
			dp.DrawTrackAsDistanceTravelledAlongPath(t)
		} else {
			dp.DrawTrackAsDistanceRemainingAlongPath(t)
		}

		if strings.Count(dp.Caption, "\n") < 4 {
			dp.Caption += fmt.Sprintf("%s %s\n", f.IdentString(), t.MediumString())
		}
	}

	return nil
}

// }}}
// {{{ DescentPDFFinalize

func DescentPDFFinalize(w http.ResponseWriter, r *http.Request, dp *fpdf.DescentPdf) {
	dp.DrawCaption()

	w.Header().Set("Content-Type", "application/pdf")
	if err := dp.Output(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// }}}

// {{{ flightToDescentTrack

// Resamples the track; does full post-processing; attempts altitude correction
// Extracts a bunch of args from the request (sample, DateRange widget)

func flightToDescentTrack(r *http.Request, f *fdb.Flight) (fdb.Track, error) {
	trackKeyName,track := f.PreferredTrack([]string{"ADSB", "MLAT", "FOIA", "FA", "fr24"})
	if track == nil {
		return nil, fmt.Errorf("no track found (saw %q)", f.ListTracks())
	}

	sampleRate := widget.FormValueDuration(r, "sample")
	if sampleRate == 0 { sampleRate = 15 * time.Second }
	track = track.SampleEvery(sampleRate, false)
	track.PostProcess()

	if trackKeyName == "FOIA" {
		track.AdjustAltitudes(nil) // FOIA track altitudes are already pressure-corrected

	} else {
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
			return nil, err
		}
		track.AdjustAltitudes(m)
	}

	return track, nil
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
