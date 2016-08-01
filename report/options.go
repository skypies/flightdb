package report

// All reports share this same options struct. Some options apply to all reports, some
// are interpreted creatively by others, and some only apply to one kind of report.
// They are all parsed of the http.request, including the report name.

import(
	"html/template"
	"fmt"
	"net/http"
	"strings"
	"time"
	
	"github.com/skypies/geo"
	"github.com/skypies/geo/sfo"
	"github.com/skypies/util/widget"

	fdb "github.com/skypies/flightdb2"
)

type Options struct {
	Name               string
	Start, End         time.Time
	Tags             []string
	Waypoints        []string

	NotTags          []string  // Tags that are blacklisted from results; not efficient
	NotWaypoints     []string  // Tags that are blacklisted from results; not efficient
	
	// Geo restriction 1: Box.
	BoxCenter          geo.NamedLatlong
	BoxRadiusKM        float64  // [BROKEN FOR NOW] For defining areas of interest around waypoints
	BoxSideKM          float64  // For defining areas of interest around waypoints
	BoxExcludes        bool     // By default[false], tracks must intersect the box. NOT IMPLEMENTED
	// Min/max altitudes. 0 == don't care
	AltitudeMin        int64
	AltitudeMax        int64

	// Geo-restriction 2: Window. Represent this a bit better.	
	WindowStart        geo.NamedLatlong
	WindowEnd          geo.NamedLatlong
	WindowMin, WindowMax float64

	// Data specification
	CanSeeFOIA         bool    // This is locked down to a few users. Upgrade to full ACL model?
	TrackDataSource    string  // TODO: replace with a cleverer track-specification thing
	
	// Options applicable to various reports
	TextString         string  // An arbitrary string
	ReferencePoint     geo.NamedLatlong // Some reports do things in relation to a fixed point
	RefDistanceKM      float64     // ... and maybe within {dist} of {refpoint}
	AltitudeTolerance  float64  // Some reports care about this
	time.Duration      // embedded; a time tolerance
	
	// Formatting / output options
	ResultsFormat      string // csv, html

	ReportLogLevel // For debugging
}

func FormValueNamedLatlong(r *http.Request, stem string) geo.NamedLatlong {
	if wp := strings.ToUpper(r.FormValue(stem+"_name")); wp != "" {
		if _,exists := sfo.KFixes[wp]; !exists {
			return geo.NamedLatlong{Name:"[UNKNOWN]"} //, fmt.Errorf("Waypoint '%s' not known", wp)
		}
		return geo.NamedLatlong{wp, sfo.KFixes[wp]}
	}

	return geo.NamedLatlong{"", geo.FormValueLatlong(r,stem)}
}

func FormValueReportOptions(r *http.Request) (Options, error) {
	if r.FormValue("rep") == "" {
		return Options{}, fmt.Errorf("url arg 'rep' missing (no report specified")
	}

	s,e,err := widget.FormValueDateRange(r)	
	if err != nil { return Options{}, err }

//	cutoff,_ := time.Parse("2006.01.02", "2015.08.01")
//	if s.Before(cutoff) || e.Before(cutoff) {
//		return Options{}, fmt.Errorf("earliest date for reports is '%s'", cutoff)
//	}
	
	opt := Options{
		Name: r.FormValue("rep"),
		Start: s,
		End: e,
		Tags: widget.FormValueCommaSpaceSepStrings(r,"tags"),
		NotTags: widget.FormValueCommaSpaceSepStrings(r,"nottags"),
		Waypoints: []string{},
		NotWaypoints: []string{},
		
		BoxCenter: FormValueNamedLatlong(r, "boxcenter"),
		BoxRadiusKM: widget.FormValueFloat64EatErrs(r, "boxradiuskm"),
		BoxSideKM: widget.FormValueFloat64EatErrs(r, "boxsidekm"),
		BoxExcludes: widget.FormValueCheckbox(r, "boxexcludes"),
		AltitudeMin: widget.FormValueInt64(r, "altitudemin"),
		AltitudeMax: widget.FormValueInt64(r, "altitudemax"),

		WindowStart: FormValueNamedLatlong(r, "winstart"),
		WindowEnd:   FormValueNamedLatlong(r, "winend"),
		WindowMin: widget.FormValueFloat64EatErrs(r, "winmin"),
		WindowMax: widget.FormValueFloat64EatErrs(r, "winmax"),

		TextString: r.FormValue("textstring"),
		AltitudeTolerance: widget.FormValueFloat64EatErrs(r, "altitudetolerance"),
		Duration: widget.FormValueDuration(r, "duration"),
		ReferencePoint: FormValueNamedLatlong(r, "refpt"),
		RefDistanceKM: widget.FormValueFloat64EatErrs(r, "refdistancekm"),

		ResultsFormat: r.FormValue("resultformat"),
	}
	
	switch r.FormValue("datasource") {
	case "ADSB": opt.TrackDataSource = "ADSB"
	case "fr24": opt.TrackDataSource = "fr24"
		// default means let the report pick
	}

	switch r.FormValue("log") {
	case "debug": opt.ReportLogLevel = DEBUG
	default:      opt.ReportLogLevel = INFO
	}

	// for _,name := range []string{"waypoint1", "waypoint2", "waypoint3"} {
	// for _,name := range []string{"notwaypoint1", "notwaypoint2", "notwaypoint3"} {
	for i:=1; i<=9; i++ {
		name := fmt.Sprintf("waypoint%d", i)
		if r.FormValue(name) != "" {
			waypoint := strings.ToUpper(r.FormValue(name))
			opt.Waypoints = append(opt.Waypoints, waypoint)
		}

		name = fmt.Sprintf("notwaypoint%d", i)
		if r.FormValue(name) != "" {
			waypoint := strings.ToUpper(r.FormValue(name))
			opt.NotWaypoints = append(opt.NotWaypoints, waypoint)
		}
	}
	
	if !opt.BoxCenter.IsNil() && opt.BoxRadiusKM<=0.1 && opt.BoxSideKM<=0.1 {
		return Options{}, fmt.Errorf("Must define a radius or boxside for the region")
	}
	
	return opt, nil
}
 
func (o Options)ListGeoRestrictors() []geo.Restrictor {
	ret := []geo.Restrictor{}

	if !o.WindowStart.IsNil() && !o.WindowEnd.IsNil() {
		window := geo.WindowRestrictor{
			Window: geo.Window{
				LatlongLine: o.WindowStart.BuildLine(o.WindowEnd.Latlong),
				MinAltitude: o.WindowMin,
				MaxAltitude: o.WindowMax,
			},
		}
		ret = append(ret, window)
	}

	addCenteredRestriction := func(pos geo.Latlong) {
		if o.BoxSideKM > 0 {
			restric := geo.LatlongBoxRestrictor{LatlongBox: pos.Box(o.BoxSideKM,o.BoxSideKM) }
			restric.Floor, restric.Ceil = o.AltitudeMin, o.AltitudeMax
			restric.MustNotIntersectVal = o.BoxExcludes
			ret = append(ret, restric)
			// BROKEN } else if o.BoxRadiusKM > 0 {
			// ret = append(ret, pos.Circle(o.BoxRadiusKM))
		}
	}

	if !o.BoxCenter.IsNil() { addCenteredRestriction(o.BoxCenter.Latlong) }

	return ret
}

func (o Options)GetRefpointRestrictor() geo.Restrictor {
	return geo.LatlongBoxRestrictor{LatlongBox:o.ReferencePoint.Box(o.RefDistanceKM,o.RefDistanceKM)}
}

func (o Options)ListMapRenderers() []geo.MapRenderer {
	ret := []geo.MapRenderer{}

	for _,gr  := range o.ListGeoRestrictors() {
		ret = append(ret, gr)
	}

	if !o.ReferencePoint.IsNil() {
		dist := o.RefDistanceKM
		if dist == 0.0 { dist = 2 }
		ret = append(ret, o.ReferencePoint.Box(dist,dist))
	}

	for _,wp := range o.Waypoints {
		ret = append(ret, sfo.KFixes[wp].Box(fdb.KWaypointSnapKM,fdb.KWaypointSnapKM))
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

func NamedLatlongToCGIArgs(stem string, nl geo.NamedLatlong) string {
	return fmt.Sprintf("%s_name=%s&%s", stem, nl.Name, nl.Latlong.ToCGIArgs(stem))
}

// for html/template, which chokes 
func (r Report)CGIArgs() template.HTML { return template.HTML(r.ToCGIArgs()) }

// A bare minimum of args, to embed in track links, so tracks can render with report tooltips
// and maps can see the geometry used
func (r *Report)ToCGIArgs() string {
	str := fmt.Sprintf("rep=%s&%s", r.Options.Name, widget.DateRangeToCGIArgs(r.Start, r.End))
	if r.TrackDataSource != "" { str += "&datasource="+r.TrackDataSource }

	if !r.BoxCenter.IsNil() {
		str += "&" + NamedLatlongToCGIArgs("boxcenter", r.BoxCenter)
		if r.BoxRadiusKM > 0.0 { str += fmt.Sprintf("&boxradiuskm=%.2f", r.BoxRadiusKM) }
		if r.BoxSideKM > 0.0 { str += fmt.Sprintf("&boxsidekm=%.2f", r.BoxSideKM) }
		if r.BoxExcludes { str += "&boxexcludes=checked" }
	}

	if r.AltitudeMin > 0 { str += fmt.Sprintf("&altitudemin=%d", r.AltitudeMin) }
	if r.AltitudeMax > 0 { str += fmt.Sprintf("&altitudemax=%d", r.AltitudeMax) }

	if !r.ReferencePoint.IsNil() { str += "&" + NamedLatlongToCGIArgs("refpt", r.ReferencePoint) }

	if r.RefDistanceKM > 0.0 { str += fmt.Sprintf("&refdistancekm=%.2f", r.RefDistanceKM) }

	if !r.WindowStart.IsNil() {
		str += "&" + NamedLatlongToCGIArgs("winstart", r.WindowStart)
		str += "&" + NamedLatlongToCGIArgs("winend", r.WindowEnd)
		if r.WindowMin > 0.0 { str += fmt.Sprintf("&winmin=%.0f", r.WindowMin) }
		if r.WindowMax > 0.0 { str += fmt.Sprintf("&winmax=%.0f", r.WindowMax) }
	}
	
	for i,wp := range r.Waypoints {
		str += fmt.Sprintf("&waypoint%d=%s", i+1, wp)
	}
	for i,wp := range r.NotWaypoints {
		str += fmt.Sprintf("&notwaypoint%d=%s", i+1, wp)
	}

	if len(r.Tags) > 0 { str += fmt.Sprintf("&tags=%s", strings.Join(r.Tags,",")) }
	if len(r.NotTags) > 0 { str += fmt.Sprintf("&nottags=%s", strings.Join(r.NotTags,",")) }

	if r.TextString != "" { str += fmt.Sprintf("&textstring=%s", r.TextString) }
	
	return str
}

func (o Options)DescriptionText() string {
	str := ""
	if o.Name != ".list" { str += o.Name+": " }

	s, e := o.Start.Format("Mon 2006/01/02"), o.End.Format("Mon 2006/01/02")
	str += s
	if s != e { str += "-"+e }

	if len(o.Tags)>0 { str += fmt.Sprintf(", tags=%v", o.Tags) }
	if len(o.NotTags)>0 { str += fmt.Sprintf(", not-tags=%v", o.NotTags) }
	if len(o.Waypoints)>0 { str += fmt.Sprintf(", waypoints=%v", o.Waypoints) }
	if len(o.NotWaypoints)>0 { str += fmt.Sprintf(", not-waypoints=%v", o.NotWaypoints) }

	altStr := ""
	if o.AltitudeMin > 0 || o.AltitudeMax > 0 {
		altStr = fmt.Sprintf("@[%d,%d]ft", o.AltitudeMin, o.AltitudeMax)
	}

	if !o.BoxCenter.IsNil() {
		boxname := "box"
		if o.BoxExcludes { boxname = "exclude-box" }
		str += fmt.Sprintf(", %s %s@%.1fKM%s applied", boxname, o.BoxCenter.ShortString(),
			o.BoxSideKM, altStr)
	}
	if !o.WindowStart.IsNil() {
		str += fmt.Sprintf(", geo-window[%s-%s] applied", o.WindowStart.ShortString(),
			o.WindowEnd.ShortString())
	}
	if o.TextString != "" {
		str += fmt.Sprintf(", str='%s'", o.TextString)
	}
	
	return str
}
