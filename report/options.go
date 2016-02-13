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
  TrackDataSource    string

	Procedures       []string	
	Waypoints        []string
	
	Center             geo.Latlong
	RadiusKM           float64  // For defining areas of interest around waypoints
	SideKM             float64  // For defining areas of interest around waypoints

	ReferencePoint     geo.Latlong // Some reports do things in relation to a fixed point
	AltitudeTolerance  float64  // Some reports care about this

	// Window. Represent this a bit better.
	WindowTo, WindowFrom geo.Latlong	
	WindowMin, WindowMax float64
}

// If fields absent or blank, returns {0.0, 0.0}
// Expects two string fields that start with stem, with 'lat' or 'long' appended.
func FormValueLatlong(r *http.Request, stem string) geo.Latlong {
	lat  := widget.FormValueFloat64EatErrs(r, stem+"lat")
	long := widget.FormValueFloat64EatErrs(r, stem+"long")
	return geo.Latlong{lat,long}
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

		Center: FormValueLatlong(r, "center"),

		ReferencePoint: FormValueLatlong(r, "refpt"),

		// Hmm.
		WindowTo: FormValueLatlong(r, "winto"),
		WindowFrom: FormValueLatlong(r, "winfrom"),
		WindowMin: widget.FormValueFloat64EatErrs(r, "winmin"),
		WindowMax: widget.FormValueFloat64EatErrs(r, "winmax"),
	}
	
	// This is somewhat ignored for now
	switch r.FormValue("datasource") {
	case "adsb": opt.TrackDataSource = "ADSB"
	case "fa":   opt.TrackDataSource = "FA"
	default:     opt.TrackDataSource = "fr24"
	}
	
	// The form only sends one value (via dropdown), but keep it as a list just in case
	for _, waypoint := range widget.FormValueCommaSepStrings(r,"waypoint") {
		waypoint = strings.ToUpper(waypoint)
		if _,exists := sfo.KFixes[waypoint]; !exists {
			return Options{}, fmt.Errorf("Waypoint '%s' not known", waypoint)
		}
		opt.Waypoints = append(opt.Waypoints, waypoint)
	}

	if r.FormValue("refpt_waypoint") != "" {
		waypoint := strings.ToUpper(r.FormValue("refpt_waypoint"))
		if _,exists := sfo.KFixes[waypoint]; !exists {
			return Options{}, fmt.Errorf("Waypoint '%s' not known (ref pt)", waypoint)
		}
		opt.ReferencePoint = sfo.KFixes[waypoint]
	}
	
	if opt.Center.Lat>0.1 && opt.RadiusKM<=0.1 && opt.SideKM<=0.1 {
		return Options{}, fmt.Errorf("Must define a radius or boxside for the region")
	}
	
	return opt, nil
}
 
func (o Options)ListGeoRestrictors() []geo.GeoRestrictor {
	ret := []geo.GeoRestrictor{}
	if !o.WindowFrom.IsNil() && !o.WindowTo.IsNil() {
		window := geo.Window{
			LatlongLine: o.WindowFrom.BuildLine(o.WindowTo),
			MinAltitude: o.WindowMin,
			MaxAltitude: o.WindowMax,
		}
		ret = append(ret, window)
	}

	addCenteredRestriction := func(pos geo.Latlong) {
		if o.RadiusKM > 0 {
			ret = append(ret, pos.Circle(o.RadiusKM))
		} else if o.SideKM > 0 {
			ret = append(ret, pos.Box(o.SideKM,o.SideKM))
		}
	}

	if math.Abs(o.Center.Lat)>0.1 { addCenteredRestriction(o.Center) }
	for _,waypoint := range o.Waypoints { addCenteredRestriction(sfo.KFixes[waypoint]) }	

	return ret
}

func (o Options)ListMapRenderers() []geo.MapRenderer {
	ret := []geo.MapRenderer{}

	for _,gr  := range o.ListGeoRestrictors() {
		ret = append(ret, gr)
	}

	if !o.ReferencePoint.IsNil() {
		ret = append(ret, o.ReferencePoint.Box(2,2))
	}
	
	return ret
}

func (o Options)String() string {
	str := fmt.Sprintf("%#v\n--\n", o)
	str += "GeoRestrictors:-\n"
	for _,gr := range o.ListGeoRestrictors() {
		str += fmt.Sprintf("  %s\n", gr)
	}
	str += "--\n"
	
	return str
}

func LatlongToCGIArgs(stem string, pos geo.Latlong) string {
	return fmt.Sprintf("%slat=%.5f&%slong=%.5f", stem, pos.Lat, stem, pos.Long)
}

// A bare minimum of args, to embed in track links, so tracks can render with report tooltips
// and maps can see the geometry used
func (r *Report)ToCGIArgs() string {
	str := fmt.Sprintf("rep=%s&%s", r.Options.Name, widget.DateRangeToCGIArgs(r.Start, r.End))

	if len(r.Waypoints) > 0 { str += "&waypoint=" + strings.Join(r.Options.Waypoints,",") }

	if !r.Center.IsNil() { str += "&" + LatlongToCGIArgs("center", r.Center) }
	if !r.ReferencePoint.IsNil() { str += "&" + LatlongToCGIArgs("refpt", r.ReferencePoint) }

	if r.RadiusKM > 0.01 { str += fmt.Sprintf("&radiuskm=%.2f", r.RadiusKM) }
	if r.SideKM > 0.01 { str += fmt.Sprintf("&sidekm=%.2f", r.SideKM) }

	if !r.WindowFrom.IsNil() {
		str += "&" + LatlongToCGIArgs("winfrom", r.WindowFrom)
		str += "&" + LatlongToCGIArgs("winto", r.WindowTo)
		if r.WindowMin > 0.01 { str += fmt.Sprintf("&winmin=%.0f", r.WindowMin) }
		if r.WindowMax > 0.01 { str += fmt.Sprintf("&winmax=%.0f", r.WindowMax) }
	}
	
	return str
}
