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

	fdb "github.com/skypies/flightdb"
	"github.com/skypies/flightdb/fgae"
)

type Options struct {
	Name               string
	Start, End         time.Time
	Tags             []string
	Waypoints        []string

	NotTags          []string  // Tags that are blacklisted from results; not efficient
	NotWaypoints     []string  // Tags that are blacklisted from results; not efficient

	TimeOfDay          date.TimeOfDayRange  // If initialized, only find flights that 'match' it
	
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

		TextString: r.FormValue("textstring"),
		AltitudeTolerance: widget.FormValueFloat64EatErrs(r, "altitudetolerance"),
		Duration: widget.FormValueDuration(r, "duration"),
		ReferencePoint: sfo.FormValueNamedLatlong(r, "refpt"),
		ReferencePoint2: sfo.FormValueNamedLatlong(r, "refpt2"),
		RefDistanceKM: widget.FormValueFloat64EatErrs(r, "refdistancekm"),

		ResultsFormat: r.FormValue("resultformat"),
	}
	
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
	db := fgae.NewDB(ctx)

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

// {{{ o.ListMapRenderers

func (o Options)ListMapRenderers() []geo.MapRenderer {
	ret := []geo.MapRenderer{}

	for _,gr  := range o.GRS.R {
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
	str := fmt.Sprintf("%#v\n--\n%s", o, o.GRS)
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

	if o.TimeOfDay.IsInitialized() { str += fmt.Sprintf(", TimeOfDay=%s", o.TimeOfDay) }
	if len(o.Tags)>0 { str += fmt.Sprintf(", tags=%v", o.Tags) }
	if len(o.NotTags)>0 { str += fmt.Sprintf(", not-tags=%v", o.NotTags) }
	if len(o.Waypoints)>0 { str += fmt.Sprintf(", waypoints=%v", o.Waypoints) }
	if len(o.NotWaypoints)>0 { str += fmt.Sprintf(", not-waypoints=%v", o.NotWaypoints) }
	if !o.GRS.IsNil() { str += fmt.Sprintf(", %s", o.GRS.OnelineString()) }
	// if o.TextString != "" { str += fmt.Sprintf(", str='%s'", o.TextString) }
	
	return str
}

// }}}
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

func (r Report)ToCGIArgs() string { return r.Options.URLValues().Encode() }
// This is for use from a html/template, so we can build URLs
func (r Report)QuotedCGIArgs() template.JS { return template.JS("\""+r.ToCGIArgs()+"\"") }

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
