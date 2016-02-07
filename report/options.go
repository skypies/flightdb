package report

// All reports share this same options struct. Some options apply to all reports, some
// are interpreted creatively by others, and some only apply to one kind of report.
// They are all parsed of the http.request, including the report name.

import(
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"
	
	"github.com/skypies/geo"
	"github.com/skypies/geo/sfo"
	"github.com/skypies/util/widget"
)

type Options struct {
	Name               string
	Start, End         time.Time
	Tags             []string

	Procedures       []string	
	Waypoints        []string
	
	Center             geo.Latlong
	RadiusKM           float64  // For defining areas of interest around waypoints
	SideKM             float64  // For defining areas of interest around waypoints

	AltitudeTolerance  float64  // Some reports care about this

  TrackDataSource    string
}

func FormValueReportOptions(r *http.Request) (Options, error) {
	if r.FormValue("rep") == "" {
		return Options{}, fmt.Errorf("url arg 'rep' missing (no report specified")
	}

	s,e,err := widget.FormValueDateRange(r)	
	if err != nil { return Options{}, err }
	
	opt := Options{
		Name: r.FormValue("rep"),
		Start: s,
		End: e,
		Tags: widget.FormValueCommaSepStrings(r,"tags"),
		Procedures: widget.FormValueCommaSepStrings(r,"procedures"),
		Waypoints: []string{},

		RadiusKM: widget.FormValueFloat64EatErrs(r, "radiuskm"),
		SideKM: widget.FormValueFloat64EatErrs(r, "sidekm"),
		AltitudeTolerance: widget.FormValueFloat64EatErrs(r, "altitudetolerance"),
	}
	
	switch r.FormValue("datasource") {
	case "adsb": opt.TrackDataSource = "ADSB"
	case "fa":   opt.TrackDataSource = "FA"
	default:     opt.TrackDataSource = "fr24"
	}

	if r.FormValue("centerlat") != "" && r.FormValue("centerlong") != "" {
		lat  := widget.FormValueFloat64EatErrs(r, "centerlat")
		long := widget.FormValueFloat64EatErrs(r, "centerlong")
		opt.Center = geo.Latlong{lat,long}
	}

	// The form only sends one value (via dropdown), but keep it as a list just in case
	for _, waypoint := range widget.FormValueCommaSepStrings(r,"waypoint") {
		waypoint = strings.ToUpper(waypoint)
		if _,exists := sfo.KFixes[waypoint]; !exists {
			return Options{}, fmt.Errorf("Waypoint '%s' not known", waypoint)
		}
		opt.Waypoints = append(opt.Waypoints, waypoint)
	}
	
	if opt.Center.Lat>0.1 && opt.RadiusKM<=0.1 && opt.SideKM<=0.1 {
		return Options{}, fmt.Errorf("Must define a radius or boxside for the region")
	}
	
	return opt, nil
}

// A bare minimum of args, to embed in track links, so tracks can render with report tooltips
func (r *Report)ToCGIArgs() string {
	str := fmt.Sprintf("rep=%s&%s", r.Options.Name, widget.DateRangeToCGIArgs(r.Start, r.End))
	str += fmt.Sprintf("&waypoint=%s&centerlat=%.5f&centerlong=%.5f&radiuskm=%.2f&sidekm=%.2f",
		strings.Join(r.Options.Waypoints,","),
		r.Options.Center.Lat, r.Options.Center.Long,
		r.Options.RadiusKM, r.Options.SideKM)

	return str
}
 
func (o Options)ListRegions() []geo.Region {
	regions := []geo.Region{}

	addCenteredRegion := func(pos geo.Latlong) {
		if o.RadiusKM > 0 {
			regions = append(regions, pos.Circle(o.RadiusKM))
		} else if o.SideKM > 0 {
			regions = append(regions, pos.Box(o.SideKM,o.SideKM))
		}
	}

	if math.Abs(o.Center.Lat)>0.1 {
		addCenteredRegion(o.Center)
	}

	// Risks accidentally duplicating a region in some circumstances :/
	for _,waypoint := range o.Waypoints {
		addCenteredRegion(sfo.KFixes[waypoint])
	}

	return regions
}
