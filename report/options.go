package report

// All reports share this same options struct. Some options apply to all reports, some
// are interpreted creatively by others, and some only apply to one kind of report.
// They are all parsed of the http.request, including the report name.

import(
	"html/template"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
	
	"golang.org/x/net/context"

	"github.com/skypies/geo"
	"github.com/skypies/geo/sfo"
	"github.com/skypies/util/date"
	"github.com/skypies/util/widget"

	fdb "github.com/skypies/flightdb2"
	"github.com/skypies/flightdb2/fgae"
)

type Options struct {
	Name               string
	Start, End         time.Time
	Tags             []string
	Waypoints        []string

	NotTags          []string  // Tags that are blacklisted from results; not efficient
	NotWaypoints     []string  // Tags that are blacklisted from results; not efficient

	TimeOfDay          date.TimeOfDayRange  // If initialized, only find flights that 'match' it

	// {{{ DIE

	///// DIE
	// Geo restriction 1: Box.
	BoxCenter          geo.NamedLatlong
	BoxRadiusKM        float64  // [BROKEN FOR NOW] For defining areas of interest around waypoints
	BoxSideKM          float64  // For defining areas of interest around waypoints
	BoxExcludes        bool     // By default[false], tracks must intersect the box. NOT IMPLEMENTED
	// Min/max altitudes. 0 == don't care
	AltitudeMin        int64
	AltitudeMax        int64
	//
	// Geo-restriction 2: Window. Represent this a bit better.	
	WindowStart        geo.NamedLatlong
	WindowEnd          geo.NamedLatlong
	WindowMin, WindowMax float64
	//
	////

	// }}}

	GRS                fdb.GeoRestrictorSet
	
	// Data specification
	CanSeeFOIA         bool    // This is locked down to a few users. Upgrade to full ACL model?
	TrackDataSource    string  // TODO: replace with a cleverer track-specification thing
	
	// Options applicable to various reports
	TextString         string  // An arbitrary string
	ReferencePoint     geo.NamedLatlong // Some reports do things in relation to a fixed point
	ReferencePoint2    geo.NamedLatlong // Some reports do things in relation to a fixed point
	RefDistanceKM      float64     // ... and maybe within {dist} of {refpoint}
	AltitudeTolerance  float64  // Some reports care about this
	time.Duration      // embedded; a time tolerance
	
	// Formatting / output options
	ResultsFormat      string // csv, html

	ReportLogLevel // For debugging
}

// {{{ FormValueReportOptions

func FormValueReportOptions(ctx context.Context, r *http.Request) (Options, error) {
	if r.FormValue("rep") == "" {
		return Options{}, fmt.Errorf("url arg 'rep' missing (no report specified")
	}

	s,e,err := widget.FormValueDateRange(r)	
	if err != nil { return Options{}, err }
	
	opt := Options{
		Name: r.FormValue("rep"),
		Start: s,
		End: e,
		Tags: widget.FormValueCommaSpaceSepStrings(r,"tags"),
		NotTags: widget.FormValueCommaSpaceSepStrings(r,"nottags"),
		Waypoints: []string{},
		NotWaypoints: []string{},
		
		// {{{ DIE

		//// DIE
		//
		BoxCenter: sfo.FormValueNamedLatlong(r, "boxcenter"),
		BoxRadiusKM: widget.FormValueFloat64EatErrs(r, "boxradiuskm"),
		BoxSideKM: widget.FormValueFloat64EatErrs(r, "boxsidekm"),
		BoxExcludes: widget.FormValueCheckbox(r, "boxexcludes"),
		AltitudeMin: widget.FormValueInt64(r, "altitudemin"),
		AltitudeMax: widget.FormValueInt64(r, "altitudemax"),
		//
		WindowStart: sfo.FormValueNamedLatlong(r, "winstart"),
		WindowEnd:   sfo.FormValueNamedLatlong(r, "winend"),
		WindowMin: widget.FormValueFloat64EatErrs(r, "winmin"),
		WindowMax: widget.FormValueFloat64EatErrs(r, "winmax"),
		//
		////

		// }}}

		TextString: r.FormValue("textstring"),
		AltitudeTolerance: widget.FormValueFloat64EatErrs(r, "altitudetolerance"),
		Duration: widget.FormValueDuration(r, "duration"),
		ReferencePoint: sfo.FormValueNamedLatlong(r, "refpt"),
		ReferencePoint2: sfo.FormValueNamedLatlong(r, "refpt2"),
		RefDistanceKM: widget.FormValueFloat64EatErrs(r, "refdistancekm"),

		ResultsFormat: r.FormValue("resultformat"),
	}

	//return opt, fmt.Errorf("WTF")
	
	if grs,err := FormValueGeoRestrictorSetLoadOrAdHoc(ctx, r); err != nil {
		return opt,err
	} else if ! grs.IsNil(){
		opt.GRS = grs
	}

	if tod,err := date.FormValueTimeOfDayRange(r, "tod"); err == nil {
		opt.TimeOfDay = tod
	}
	
	// Dates sadly hardcoded for now.
	if widget.FormValueCheckbox(r, "preferfoia") {
		foiaEnd,_ := time.Parse("2006.01.02", "2016.06.24")
		if e.Before(foiaEnd) {
			opt.Tags = append(opt.Tags, "FOIA")
		}
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

	//if !opt.BoxCenter.IsNil() && opt.BoxRadiusKM<=0.1 && opt.BoxSideKM<=0.1 {
	//	return Options{}, fmt.Errorf("Must define a radius or boxside for the region")
	//}
	
	return opt, nil
}

// }}}
// {{{ FormValueGeoRestrictorSetLoadOrAdHoc

// If there is a dskey, load up the corresponding RestrictorSet. Else, look for a single
// ad-hoc restrictor in the CGI args, and generate a GRS to wrap it.
func FormValueGeoRestrictorSetLoadOrAdHoc(ctx context.Context, r *http.Request) (fdb.GeoRestrictorSet, error) {
	grs := fdb.GeoRestrictorSet{}
	
	if err := maybeLoadGRSDSKey(ctx, r, &grs); err != nil {
		return grs,err
	}

	if grs.IsNil() && r.FormValue("gr_type") != "" {
		if gr,err := fdb.FormValueGeoRestrictor(r); err != nil {
			return grs,err
		} else if ! gr.IsNil() {
			grs.R = append(grs.R, gr)
			grs.Name = "ad-hoc"
		}
	}

	return grs,nil
}

func maybeLoadGRSDSKey(ctx context.Context, r *http.Request, grs *fdb.GeoRestrictorSet) (error) {
	db := fgae.FlightDB{C:ctx}

	// TODO: move to grs_dskey or something
	if dskey := r.FormValue("grs_dskey"); dskey == "" {
		return nil
	} else	if grsIn,err := db.LoadRestrictorSet(dskey); err != nil {
		return err
	} else {
		*grs = grsIn
		return nil
	}
}

// }}}

// {{{ o.ListGeoRestrictors

func (o Options)ListGeoRestrictors() []geo.NewRestrictor {
/*
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
*/

	return o.GRS.R
}

// }}}
// {{{ o.GetRefpointRestrictor

func (o Options)GetRefpointRestrictor() geo.Restrictor {
	return geo.LatlongBoxRestrictor{LatlongBox:o.ReferencePoint.Box(o.RefDistanceKM,o.RefDistanceKM)}
}

// }}}
// {{{ o.ListMapRenderers

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

// }}}
// {{{ o.String

func (o Options)String() string {
	str := fmt.Sprintf("%#v\n--\n", o)
	str += "GeoRestrictors:-\n"
	for _,gr := range o.ListGeoRestrictors() {
		str += fmt.Sprintf("  %s\n", gr)
	}
	str += "--\n"
	
	return str
}

// }}}
// {{{ o.DescriptionText

func (o Options)DescriptionText() string {
	str := ""
	if o.Name != ".list" { str += o.Name+": " }

	s, e := o.Start.Format("Mon 2006/01/02"), o.End.Format("Mon 2006/01/02")
	str += s
	if s != e { str += "-"+e }

	if o.TimeOfDay.IsInitialized() {
		str += fmt.Sprintf(", TimeOfDay=%s", o.TimeOfDay)
	}

	if len(o.Tags)>0 { str += fmt.Sprintf(", tags=%v", o.Tags) }
	if len(o.NotTags)>0 { str += fmt.Sprintf(", not-tags=%v", o.NotTags) }
	if len(o.Waypoints)>0 { str += fmt.Sprintf(", waypoints=%v", o.Waypoints) }
	if len(o.NotWaypoints)>0 { str += fmt.Sprintf(", not-waypoints=%v", o.NotWaypoints) }

	if !o.GRS.IsNil() { str += fmt.Sprintf(", %s", o.GRS.OnelineString()) }

	// TODO: implement altitude in the Restrictions.
	if o.AltitudeMin > 0 || o.AltitudeMax > 0 {
		str += fmt.Sprintf("@[%d,%d]ft", o.AltitudeMin, o.AltitudeMax)
	}

	if o.TextString != "" {
		str += fmt.Sprintf(", str='%s'", o.TextString)
	}
	
	return str
}

// }}}

func (r Report)ToCGIArgs() string { return r.Options.URLValues().Encode() }
// These two are for html/template, so we can build URLs that aren't over-escaped
func (r Report)CGIArgs() template.HTML { return template.HTML(r.ToCGIArgs()) }
func (r Report)QuotedCGIArgs() template.JS { return template.JS("\""+r.ToCGIArgs()+"\"") }

// {{{ o.URLValues

// A bare minimum of args, to embed in track links, so tracks can render with report tooltips
// and maps can see the geometry used
func (o Options)URLValues() url.Values {
	v := url.Values{}

	v.Set("rep", o.Name)
	widget.AddValues(v, widget.DateRangeToValues(o.Start, o.End))
	if len(o.Tags) > 0 { v.Set("tags", strings.Join(o.Tags,",")) }
	if len(o.NotTags) > 0 { v.Set("nottags", strings.Join(o.NotTags,",")) }
	for i,wp := range o.Waypoints {
		v.Set(fmt.Sprintf("waypoint%s", i+1), wp)
	}
	for i,wp := range o.NotWaypoints {
		v.Set(fmt.Sprintf("notwaypoint%s", i+1), wp)
	}

	if o.GRS.IsAdhoc() && len(o.GRS.R)==1 {
		widget.AddValues(v, fdb.GeoRestrictorAsValues(o.GRS.R[0]))
	} else if o.GRS.DSKey != "" {
		v.Set("grs_dskey", o.GRS.DSKey)
	}
	
	if o.TextString != "" { v.Set("textstring", o.TextString) }
	if o.AltitudeTolerance > 0.0 {
		v.Set("altitudetolerance", fmt.Sprintf("%.2f", o.AltitudeTolerance))
	}
	if o.Duration != 0 { v.Set("duration", o.Duration.String()) }
	if !o.ReferencePoint.IsNil() { widget.AddPrefixedValues(v, o.ReferencePoint.Values(), "refpt") }
	if !o.ReferencePoint2.IsNil() {widget.AddPrefixedValues(v, o.ReferencePoint2.Values(), "refpt2") }

	if o.RefDistanceKM > 0.0 { v.Set("refdistancekm", fmt.Sprintf("%.2f", o.RefDistanceKM)) }

	if o.TimeOfDay.IsInitialized() { widget.AddPrefixedValues(v, o.TimeOfDay.Values(), "tod") }
	
	if o.TrackDataSource != "" { v.Set("datasource", o.TrackDataSource) }

	return v
}

// }}}

/*
	// {{{ r.ToCGIArgs

// A bare minimum of args, to embed in track links, so tracks can render with report tooltips
// and maps can see the geometry used
func (r *Report)ToCGIArgs() string {
	str := fmt.Sprintf("rep=%s&%s", r.Options.Name, widget.DateRangeToCGIArgs(r.Start, r.End))

	if len(r.Tags) > 0 { str += fmt.Sprintf("&tags=%s", strings.Join(r.Tags,",")) }
	if len(r.NotTags) > 0 { str += fmt.Sprintf("&nottags=%s", strings.Join(r.NotTags,",")) }
	for i,wp := range r.Waypoints {
		str += fmt.Sprintf("&waypoint%d=%s", i+1, wp)
	}
	for i,wp := range r.NotWaypoints {
		str += fmt.Sprintf("&notwaypoint%d=%s", i+1, wp)
	}

	// TODO: inline dskey or ad-hoc GR
	//// DIE
	//
	if !r.BoxCenter.IsNil() {
		str += "&" + r.BoxCenter.ToCGIArgs("boxcenter")
		if r.BoxRadiusKM > 0.0 { str += fmt.Sprintf("&boxradiuskm=%.2f", r.BoxRadiusKM) }
		if r.BoxSideKM > 0.0 { str += fmt.Sprintf("&boxsidekm=%.2f", r.BoxSideKM) }
		if r.BoxExcludes { str += "&boxexcludes=checked" }
	}
	if r.AltitudeMin > 0 { str += fmt.Sprintf("&altitudemin=%d", r.AltitudeMin) }
	if r.AltitudeMax > 0 { str += fmt.Sprintf("&altitudemax=%d", r.AltitudeMax) }
	
	if !r.WindowStart.IsNil() {
		str += "&" + r.WindowStart.ToCGIArgs("winstart")
		str += "&" + r.WindowEnd.ToCGIArgs("winend")
		if r.WindowMin > 0.0 { str += fmt.Sprintf("&winmin=%.0f", r.WindowMin) }
		if r.WindowMax > 0.0 { str += fmt.Sprintf("&winmax=%.0f", r.WindowMax) }
	}
	//
	/////

	if r.TextString != "" { str += fmt.Sprintf("&textstring=%s", r.TextString) } // CGI Encoding ?
	if r.AltitudeTolerance > 0.0 { str += fmt.Sprintf("&altitudetolerance=%.2f", r.AltitudeTolerance)}
	if r.Duration != 0 { str += fmt.Sprintf("&duration=%s", r.Duration) }
	if !r.ReferencePoint.IsNil() { str += "&" + r.ReferencePoint.ToCGIArgs("refpt") }
	if !r.ReferencePoint2.IsNil() { str += "&" + r.ReferencePoint2.ToCGIArgs("refpt2") }
	if r.RefDistanceKM > 0.0 { str += fmt.Sprintf("&refdistancekm=%.2f", r.RefDistanceKM) }

	if r.TimeOfDay.IsInitialized() { str += "&"+r.TimeOfDay.ToCGIArgs("tod") }
	
	if r.TrackDataSource != "" { str += "&datasource="+r.TrackDataSource }

	return str
}

// }}}
*/

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
