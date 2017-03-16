package ui

import(
	"fmt"
	"net/http"
	"strings"
	
	"golang.org/x/net/context"

	"github.com/skypies/geo"
	"github.com/skypies/geo/sfo"
	"github.com/skypies/util/widget"

	fdb "github.com/skypies/flightdb"
	"github.com/skypies/flightdb/fgae"
	"github.com/skypies/flightdb/fpdf"
	"github.com/skypies/flightdb/metar"
	"github.com/skypies/flightdb/ref"
)

func init() {
	http.HandleFunc("/fdb/descent",  UIOptionsHandler(descentHandler))
}

// {{{ descentHandler

// ?idspec=XX,YY,...    (or ?idspec=XX&idspec=YYY&...)
//  &sample=N           (sample the track every N seconds)
//  &averagingwindow=2m (duration to average over)
//  &alt=30000          (max altitude for graph)
//  &length=80          (max distance from origin; in nautical miles)
//  &dist=from          (for distance axis, use dist from airport; by default, uses dist along path)
//  &colorby=delta      (delta groundspeed, instead of groundspeed)

//  &classb=1           (maybe render the SFO class B airpsace)
//  &refpt_lat=36&refpt_long=-122&refpt_label=FOO  (render a reference point onto the graph)

//  &anchor_name=EDDYY  (a geo.NamedLatlong with stem "anchor")
//  &anchor_alt_{min,max} (altitude range; i.e. BRIXX (5000,50000)==first pass, (0,5000) second)

func descentHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	opt,_ := GetUIOptions(ctx)
	db := fgae.FlightDB{C:ctx}
	
	idspecs,err := opt.IdSpecs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// This whole Airframe cache thing should be automatic, and upstream from here.
	airframes := ref.NewAirframeCache(ctx)

	// The UI options should have figured out a good timespan for metars
	metars,_ := metar.LookupArchive(ctx, "KSFO", opt.Start, opt.End)
	
	dp := DescentPDFInit(opt, w, r, len(idspecs))

	if len(idspecs) > 10 {
		dp.LineThickness = 0.1
		dp.LineOpacity = 0.25
	}

	n := 0
	for _,idspec := range idspecs {
		f,err := db.LookupMostRecent(db.NewQuery().ByIdSpec(idspec))
		if err != nil {
			http.Error(w, fmt.Sprintf("[looked up %d so far] %v",n, err.Error()),
				http.StatusInternalServerError)
			return
		} else if f == nil {
			http.Error(w, fmt.Sprintf("idspec %s not found", idspec), http.StatusBadRequest)
			return
		}
		if af := airframes.Get(f.IcaoId); af != nil { f.Airframe = *af }

		if err := DescentPDFAddFlight(opt, r, dp, metars, f); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		n++
	}
	
	DescentPDFFinalize(opt, w,r,dp)
}

// }}}
// {{{ DescentPDFInit

func DescentPDFInit(opt UIOptions, w http.ResponseWriter, r *http.Request, numFlights int) *fpdf.DescentPdf {
	colorscheme := opt.PDFColorScheme
	colorscheme = fpdf.ByPlotKind
	
	dp := fpdf.DescentPdf{
		ColorScheme: colorscheme,
		Anchor:      geo.NamedLatlong{Name:"KSFO", Latlong:sfo.KLatlongSFO},
		AnchorAltitudeMin: 0, //float64(widget.FormValueIntWithDefault(r, "anchor_alt_min", 0)),
		AnchorAltitudeMax: 25000, //float64(widget.FormValueIntWithDefault(r, "anchor_alt_max", 5000)),
		AltitudeMax: float64(widget.FormValueIntWithDefault(r, "alt", 30000)),
		LengthNM:    float64(widget.FormValueIntWithDefault(r, "length", 80)),
		ToShow:      map[string]bool{"altitude":true, "groundspeed":true, "verticalspeed":true},
		ShowDebug:  (r.FormValue("debug") != ""),
		AveragingWindow: widget.FormValueDuration(r, "averagingwindow"),
		Permalink:   opt.Permalink,
	}

	// Hardcode anchor; new sideviewHandler looks at that.
	//anchor := sfo.FormValueNamedLatlong(r, "anchor")  // &anchor_name={KSFO,EDDYY}
	//if anchor.Name != "" {
	//	dp.Anchor = anchor
	//}
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

	if r.FormValue("classb") != "" {
		dp.MaybeDrawSFOClassB()
	}

	if pos := geo.FormValueLatlong(r, "refpt"); !pos.IsNil() {
		dp.DrawReferencePoint(pos, r.FormValue("refpt_label"))
	}
	
	if opt.Report != nil {
		dp.Caption += fmt.Sprintf("%d flights, %s\n", numFlights, opt.Report.DescriptionText())
	}
	
	return &dp
}

// }}}
// {{{ DescentPDFAddFlight

func DescentPDFAddFlight(opt UIOptions, r *http.Request, dp *fpdf.DescentPdf, metars *metar.Archive, f *fdb.Flight) error {
	if t,err := flightToAltitudeTrack(opt, r, metars, f); err != nil {
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

func DescentPDFFinalize(opt UIOptions, w http.ResponseWriter, r *http.Request, dp *fpdf.DescentPdf) {
	dp.DrawCaption()

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
