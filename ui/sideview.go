package ui

import(
	"fmt"
	"net/http"
	"strings"
	"time"
	
	"golang.org/x/net/context"

	"github.com/skypies/geo"
	"github.com/skypies/geo/sfo"
	"github.com/skypies/util/widget"

	fdb "github.com/skypies/flightdb2"
	"github.com/skypies/flightdb2/fgae"
	"github.com/skypies/flightdb2/fpdf"
	"github.com/skypies/flightdb2/metar"
	"github.com/skypies/flightdb2/ref"
)

func init() {
	http.HandleFunc("/fdb/sideview",  UIOptionsHandler(sideviewHandler))
}

// {{{ sideviewHandler

// ?idspec=XX,YY,...    (or ?idspec=XX&idspec=YYY&...)
//  &sample=N           (sample the track every N seconds)
//  &averagingwindow=2m (duration to average over)
//  &alt=30000          (max altitude for graph)
//  &dist=crowflies     (for distance axis, use dist from airport; by default, uses dist along path)
//  &colorby=delta      (delta groundspeed, instead of groundspeed)

//  &classb=1           (maybe render the SFO class B airpsace)
//  &refpt_lat=36&refpt_long=-122&refpt_label=FOO  (render a reference point onto the graph)

//  &anchor_name=EDDYY  (a geo.NamedLatlong with stem "anchor")
//  &anchor_alt_{min,max} (altitude range; i.e. BRIXX (5000,50000)==first pass, (0,5000) second)
//  &anchor_dist_{min,max} (dist range; [-80,0] for arrivals, [-40,40] for waypoints, [0,80] deps)

//  &showaccelerations=1
//  &showangleofinclination=1

// http://localhost:8080/fdb/sideview?idspec=ASH5408@1425022200&debug=true&anchor_alt_max=40000&anchor_dist_min=-80&anchor_dist_max=10&anchor_name=KSFO&&showaccelerations=1&classb=1&refpt_lat=37.060312&refpt_long=-121.990814&refpt_label=YOU&dist=crowflies

func sideviewHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
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
	
	svp := SideviewPDFInit(opt, w, r, len(idspecs))

	if len(idspecs) > 10 {
		svp.LineThickness = 0.1
		svp.LineOpacity = 0.25
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
		
		if err := SideviewPDFAddFlight(opt, r, svp, metars, f, (n==0)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		n++
	}

	SideviewPDFFinalize(opt,w,r,svp)
}

// }}}
// {{{ SideviewPDFInit

func SideviewPDFInit(opt UIOptions, w http.ResponseWriter, r *http.Request, numFlights int) *fpdf.SideviewPdf {
	colorscheme := opt.PDFColorScheme
	colorscheme = fpdf.ByPlotKind

	svp := fpdf.SideviewPdf{
		ColorScheme:     colorscheme,

		AltitudeMin:     0,
		AltitudeMax:     float64(widget.FormValueIntWithDefault(r, "alt", 30000)),
		AnchorDistMinNM: float64(widget.FormValueIntWithDefault(r, "anchor_dist_min", -80)),
		AnchorDistMaxNM: float64(widget.FormValueIntWithDefault(r, "anchor_dist_max",   0)),

		AnchorPoint: fpdf.AnchorPoint{
			NamedLatlong: sfo.FormValueNamedLatlong(r, "anchor"),  // &anchor_name={KSFO,EDDYY}
			AltitudeMin:  float64(widget.FormValueIntWithDefault(r, "anchor_alt_min", 0)),
			AltitudeMax:  float64(widget.FormValueIntWithDefault(r, "anchor_alt_max", 8000)),
			DistMaxKM:    float64(widget.FormValueIntWithDefault(r, "anchor_max_dist", 80)),
		},

		ToShow:          map[string]bool{"altitude":true, "groundspeed":true, "verticalspeed":true},
		ShowDebug:      (r.FormValue("debug") != ""),
		AveragingWindow: widget.FormValueDuration(r, "averagingwindow"),
		Permalink:       opt.Permalink,
	}

	if svp.AnchorPoint.Name == "" {
		svp.AnchorPoint.NamedLatlong = geo.NamedLatlong{Name:"KSFO", Latlong:sfo.KAirports["KSFO"]}
	}

	if widget.FormValueCheckbox(r, "showaccelerations") {
		svp.ToShow["groundacceleration"],svp.ToShow["verticalacceleration"] = true,true
	}
	if widget.FormValueCheckbox(r, "showangleofinclination") {
		svp.ToShow["angleofinclination"] = true
	}

	if r.FormValue("dist") == "crowflies" {
		svp.TrackProjector = &fpdf.ProjectAsCrowFlies{}
	} else {
		svp.TrackProjector = &fpdf.ProjectAlongPath{}
	}
	
	svp.Init()
	svp.DrawFrames()

	if r.FormValue("classb") != "" {
		svp.MaybeDrawSFOClassB()
	}
	
	if opt.Report != nil {
		svp.Caption += fmt.Sprintf("%d flights, %s\n", numFlights, opt.Report.DescriptionText())
	}
	
	return &svp
}

// }}}
// {{{ SideviewPDFAddFlight

func SideviewPDFAddFlight(opt UIOptions, r *http.Request, svp *fpdf.SideviewPdf, metars *metar.Archive, f *fdb.Flight, first bool) error {
	t,err := flightToDescentTrack(opt, r, metars, f)
	if err != nil { return err }

	svp.DrawProjectedTrack(t, svp.ColorScheme)

	if first {
		if pos := geo.FormValueLatlong(r, "refpt"); !pos.IsNil() {
			svp.DrawPointProjectedIntoTrack(t, pos, r.FormValue("refpt_label"))
		}
	}
	
	if strings.Count(svp.Caption, "\n") < 3 {
		svp.Caption += fmt.Sprintf("%s %s\n", f.IdentString(), t.MediumString())
	}

	return nil
}

// }}}
// {{{ SideviewPDFFinalize

func SideviewPDFFinalize(opt UIOptions, w http.ResponseWriter, r *http.Request, svp *fpdf.SideviewPdf) {
	svp.DrawCaption()
	
	w.Header().Set("Content-Type", "application/pdf")
	if err := svp.Output(w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// }}}

// {{{ flightToDescentTrack

// Resamples the track; does full post-processing; attempts altitude correction
// Extracts a bunch of args from the request (sample, DateRange widget)

func flightToDescentTrack(opt UIOptions, r *http.Request, metars *metar.Archive, f *fdb.Flight) (fdb.Track, error) {
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
		if metars != nil {
			track.AdjustAltitudes(metars)
		}
	}

	return track, nil
}

// }}}


// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
