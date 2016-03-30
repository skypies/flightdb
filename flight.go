package flightdb2

import(
	"fmt"
	"sort"
	"time"
	"github.com/skypies/geo/altitude"
	"github.com/skypies/geo/sfo"
	"github.com/skypies/flightdb2/metar"
	"github.com/skypies/util/date"
)

type Flight struct {
	Identity // embedded
	Airframe // embedded
	Tracks map[string]*Track
	Tags map[string]int
	Waypoints map[string]time.Time
	
	// Internal fields
	datastoreKey string
	lastUpdate   time.Time
	DebugLog     string
}

func BlankFlight() Flight {
	return Flight{
		Tracks: map[string]*Track{},
		Tags: map[string]int{},
		Waypoints: map[string]time.Time{},
	}
}

// USE THIS ONE AT ALL TIMES
func (f Flight)IdentityString() string {
	str := f.Identity.FullString()
	str += fmt.Sprintf(" r:%s", f.Airframe.Registration)
	return str
}

func (f Flight)String() string {
	str := f.IdentString() + " "
	str += f.Airframe.String() + " "
	for k,t := range f.Tracks {
		str += fmt.Sprintf(" %s %s", k, t)
	}
	return str
}

func (f Flight)Legend() string {
	s,_ := f.Times()
	l := fmt.Sprintf("<b>%s</b>, %s<br/>Tags=<b>%v</b> Tracks=%v<br/>Route=<b>%v</b>",
		f.IdentityString(), date.InPdt(s).Format("2006/01/02 15:03 MST"),
		f.TagList(), f.ListTracks(), f.WaypointList())
	return l
}


// This happens at the flight level, as we shuffle data between identity & airframe
func (f *Flight)ParseCallsign() CallsignType {
	c := NewCallsign(f.Identity.Callsign)
	// Upate the identity with any useful data
	switch c.CallsignType {
	case Registration:
		f.Airframe.Registration = f.Identity.Callsign
	case IcaoFlightNumber:
		f.Identity.Schedule.ICAO, f.Identity.Schedule.Number = c.IcaoPrefix, c.Number
	case BareFlightNumber:
		f.Identity.Schedule.Number = c.Number
	}

	return c.CallsignType
}

// Normalization: only applies to the ICAO style ones (and then, really just SWA, etc)
// 1. Remove all zero padding on numbers
// 2. Incorporate missing carrier code, if we have it from airframe
func (f *Flight)NormalizedCallsignString() string {
	c := NewCallsign(f.Identity.Callsign)
	if c.CallsignType == BareFlightNumber && f.Airframe.CallsignPrefix != "" {
		c.MaybeAddPrefix(f.Airframe.CallsignPrefix)
	}
	return c.String()
}

func (f *Flight)SetTag(tag string) {
	f.Tags[tag]++
}
func (f *Flight)HasTag(tag string) bool {
	_,exists := f.Tags[tag]
	return exists
}
func (f Flight)TagList() []string {
	ret := []string{}
	for tag,_ := range f.Tags {
		ret = append(ret,tag)
	}
	sort.Strings(ret)
	return ret
}

func (f *Flight)SetWaypoint(wp string, t time.Time) {
	f.Waypoints[wp] = t
}
func (f *Flight)HasWaypoint(wp string) bool {
	_,exists := f.Waypoints[wp]
	return exists
}

// Yay, sorting funtime here again !
type WaypointAndTime struct {
	WP string
	time.Time
}
type WaypointAndTimeList []WaypointAndTime
func (a WaypointAndTimeList) Len() int           { return len(a) }
func (a WaypointAndTimeList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a WaypointAndTimeList) Less(i, j int) bool { return a[i].Before(a[j].Time) }
func (f Flight)WaypointList() []string {
	wptl := []WaypointAndTime{}
	for k,v := range f.Waypoints { wptl = append(wptl, WaypointAndTime{k,v}) }
	sort.Sort(WaypointAndTimeList(wptl))
	wp := []string{}
	for _,wpt := range wptl { wp = append(wp, wpt.WP) }
	return wp
}


func (f Flight)HasTrack(name string) bool {
	_,exists := f.Tracks[name]
	return exists
}
func (f Flight)ListTracks() []string {
	ret := []string{}
	for k := range f.Tracks { ret = append(ret, k) }
	sort.Strings(ret)
	return ret
}
func (f Flight)AnyTrack() Track {
	for _,name := range []string{"ADSB", "MLAT", "FA:TA", "FA:TZ", "fr24", "FOIA"} {
		if f.HasTrack(name) { return *f.Tracks[name] }
	}

	return Track{}
}
func (f Flight)PreferredTrack(pref []string) (string, Track) {
	for _,name := range pref {
		if f.HasTrack(name) { return name, *f.Tracks[name] }
	}
	return "", Track{}
}


func (f Flight)Times() (s,e time.Time) {
	if len(f.Tracks) == 0 { return }
	s,_ = time.Parse("2006.01.02", "2199.01.01")
	e,_ = time.Parse("2006.01.02", "1972.01.01")
	for _,t := range f.Tracks {
		ts,te := t.Times()
		if ts.Before(s) { s = ts }
		if te.After(e)  { e = te }
	}
	return
}

func (f Flight)MidTime() time.Time {
	s,e := f.Times()
	dur := e.Sub(s) / 2.0
	return s.Add(dur)
}

func (f Flight)IdSpecString() string {
	return f.IdSpec().String()
	//	times := f.Timeslots(time.Minute * 30)  // ARGH
//	return fmt.Sprintf("%s@%d", f.Identity.IcaoId, times[0].Unix())
}


// This is to help save RAM when compiling lists of flights
func (f *Flight)PruneTrackContents() {
	for k,track := range f.Tracks {
		if len(*track)>2 {
			t := Track{(*track)[0], (*track)[len(*track)-1]}
			f.Tracks[k] = &t
		}
	}
}

// This is all so horrid.
func (f *Flight)AnalyseWaypoints() {
	for _,trackName := range f.ListTracks() {
		for wp,t := range f.Tracks[trackName].MatchWaypoints(sfo.KFixes) {
			f.SetWaypoint(wp,t)
		}
	}
}

func (f *Flight)Analyse() (error, string) {
	// proc,str,err := MatchProcedure(f.AnyTrack())

	f.DebugLog += "-- Analyse\n"
	
	pc := NewCallsign(f.Callsign)
	f.DebugLog += fmt.Sprintf("callsign: [%s] (%d)\n", f.Callsign, pc.CallsignType)
	switch pc.CallsignType {
	case BareFlightNumber:
		fallthrough
	case IcaoFlightNumber:
		f.SetTag("AL")

	default:
		f.SetTag("GA")
	}

	if f.HasTrack("MLAT") { f.SetTag("MLAT") }
	
	if f.Origin != ""           { f.SetTag(fmt.Sprintf("%s:", f.Origin)) }
	if f.Destination != ""      { f.SetTag(fmt.Sprintf(":%s", f.Destination)) }
	//if f.Destination == "SFO" { f.SetTag("SFO") }

	f.AnalyseWaypoints()
	
	return nil, ""
}

func NewFlightFromTrackFragment(frag *TrackFragment) *Flight {
	f := BlankFlight()
	f.Identity = Identity{
		IcaoId: string(frag.IcaoId),
		Callsign: frag.Callsign,
	}

	trackKey := frag.TrackName()
	f.Tracks[trackKey] = &frag.Track
	f.DebugLog += "-- NewFlightFromTrackFragment ("+trackKey+")\n"
	
	f.Analyse() // Initial analysis of flight ID (AL vs GA etc)
	
	return &f
}

// ComputeIndicatedAltitudes
// Would be nice to find a better home for this, to not pollute metar with this file
func (f *Flight)ComputeIndicatedAltitudes(metars *metar.Archive) {
	if ! f.HasTrack("ADSB") { return }

	track := *f.Tracks["ADSB"]
	for i,tp := range track {
		lookup := metars.Lookup(tp.TimestampUTC)
		track[i].AnalysisAnnotation += fmt.Sprintf("* inHg: %v\n", lookup)
		if lookup == nil || lookup.Raw == "" {
			track[i].AnalysisAnnotation += fmt.Sprintf("* No metar, skipping\n")
			continue
		}
				
		track[i].IndicatedAltitude = altitude.PressureAltitudeToIndicatedAltitude(
			tp.Altitude, lookup.AltimeterSettingInHg)
		track[i].AnalysisAnnotation += fmt.Sprintf("* PressureAlt: %.0f, IndicatedAlt: %.0f\n",
			tp.Altitude, track[i].IndicatedAltitude)
	}
}


// Functions to support indexing & retrieval in the DB
func (f *Flight)GetDatastoreKey() string { return f.datastoreKey }
func (f *Flight)SetDatastoreKey(k string) { f.datastoreKey = k }
func (f *Flight)GetLastUpdate() time.Time { return f.lastUpdate }
func (f *Flight)SetLastUpdate(t time.Time) { f.lastUpdate = t }
func (f *Flight)Timeslots(d time.Duration) []time.Time {
	// ARGH
	// This is a mess; depending on which tracks are available, we end up with very
	// smaller or larger timespans. So we pick the likely smallest in all cases, ADSB.	
	if f.HasTrack("ADSB") {
		s,e := f.Tracks["ADSB"].Times()
		return date.Timeslots(s,e,d)
	}
	s,e := f.Times()
	return date.Timeslots(s,e,d)
}
