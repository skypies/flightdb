package flightdb

import(
	"time"
	"github.com/skypies/geo"
	"github.com/skypies/geo/sfo"
)

var(
	KWaypointSnapKM = 1.0
)

// This is all so horrid.
func (f *Flight)AnalyseWaypoints() {
	// We do a full reset of the waypoints, as we're about to do a full recompute.
	f.Waypoints = map[string]time.Time{}

	for _,trackName := range f.ListTracks() {
		for wp,t := range f.Tracks[trackName].MatchWaypoints(sfo.KFixes) {
			f.SetWaypoint(wp,t)
		}
	}
}

// This is pretty dumb
func (t Track)MatchWaypoints(waypoints map[string]geo.Latlong) (map[string]time.Time) {
	ret := map[string]time.Time{}

	lines := t.AsLinesSampledEvery(time.Second*1)
	
	for name,pos := range waypoints {
		box := pos.Box(KWaypointSnapKM,KWaypointSnapKM)

		for _,line := range lines {
			if box.IntersectsLine(line) {
				ret[name] = t[line.J].TimestampUTC
				break // We've ID'ed this box
			}
		}
	}

	return ret
}


// Find the point in a track at which we intersected waypoint.
// Empty string means no match
func (f Flight)AtWaypoint(wpName string) (string, int) {
	timeWaypoint,exists := f.Waypoints[wpName]
	if !exists { return "", -1 }

	// Really need a better approach to track selection, to avoid MLAT taking priority over ADSB, etc
	for trackName,track := range f.Tracks {
		if i := track.IndexAtTime(timeWaypoint); i >= 0 {
			return trackName, i
		}
	}

	return "", -1
}


func (f *Flight)HasOriginMatch(origins map[string]int) bool {
	_,exists := origins[f.Origin]
	return exists
}

func (f *Flight)HasDestinationMatch(dests map[string]int) bool {
	_,exists := dests[f.Destination]
	return exists
}

// If the aircraft flies to/from any of the airports in the list, set two tags:
//  to airport:    :STEM   :STEM:
//  from airport:  STEM:   :STEM:
func (f *Flight)SetAirportComboTagsFor(airports map[string]int, stem string) {
	if f.HasOriginMatch     (airports)   { f.SetTag(stem+":"); f.SetTag(":"+stem+":") }
	if f.HasDestinationMatch(airports)   { f.SetTag(":"+stem); f.SetTag(":"+stem+":") }
}

// Find a better home for this config
var (
	OceanicAirports = map[string]int{
		"LIH":1, "OGG":1, "HNL":1, "KOA":1, "NRT":1, "HND":1, "KIX":1, "PVG":1, "PEK":1, "CAN":1,
		"CTU":1, "WUH":1, "HKG":1, "TPE":1, "ICN":1, "MNL":1, "NHL":1, "SYD":1, "VRD":1, "AKL":1,
		// The same using ICAO airport codes; FOIA data uses ICAO codes for non-US airports
		"NZAA":1, "ZGGG":1, "ZUUU":1, "VHHH":1, "RJTT":1, "PHNL":1, "RKSI":1, "RJBB":1, "PHKO":1,
		"PHLI":1, "RPLL":1, "RJAA":1, "ZBAA":1, "PHOG":1, "ZSPD":1, "YSSY":1, "RCTP":1, "ZHHH":1,
	}
	SouthwestAirports = map[string]int{
		"PHX":1, "TUS":1, "SBP":1, "LAX":1, "LGB":1, "BUR":1, "ONT":1, "SNA":1, "DCA":1,
		"SBA":1, "PSP":1, "SAN":1,
	}
	NorCalAirports = map[string]int{
		"SFO":1, "SJC":1, "OAK":1,
	}
)

type BoxMatcher struct {
	Tag string
	geo.LatlongBox
}

// :SFO_W   for western arrivals (oceanic) to SFO.
// :SFO_E   for eastern arrivals to SFO.
// :SFO_N   for northen arrivals to SFO.
// :SFO_NE  are SFO_N that loop over FINSH
// :SFO_NW  are SFO_N that pass over BRIXX(KSFO) at >5000'
// :SFO_S   for southern arrivals:  :SFO && 30 km box around ANJEE, WWAVS, or their midpoint)
// SFO_S:   for southern departures:  (SFO: ||OAK:) && 30 km (TBR) box around PPEGS
// :SJC_N   arrivals into SJC that pass through BRIXX (i.e. over KSFO)
func (f *Flight)TagCoarseFlightpathForSFO() {
	matchers := []BoxMatcher{}

	if f.Destination == "SFO" {
		// Various kinds of SFO arrivals
		matchers = append(matchers, BoxMatcher{":SFO_S", sfo.KFixes["WWAVS"].Box(30,30)})
		matchers = append(matchers, BoxMatcher{":SFO_E", sfo.KFixes["ALWYS"].Box(64,64)})
		matchers = append(matchers, BoxMatcher{":SFO_N", sfo.KFixes["LOZIT"].Box(25,25)})
		matchers = append(matchers, BoxMatcher{":SFO_W", sfo.KFixes["PIRAT"].Box(50,50)})

		// This is a provisional matcher; we might remove the tag (see below)
		matchers = append(matchers, BoxMatcher{":SFO_NE", sfo.KFixes["FINSH"].Box(6,6)})

	} else if f.Destination == "SJC" {
		// Various kinds of SJC arrivals
		matchers = append(matchers, BoxMatcher{":SJC_N", sfo.KFixes["BRIXX"].Box(5,5)})

	} else if f.Origin == "SFO" || f.Origin == "OAK" {
		// Departures
		matchers = append(matchers, BoxMatcher{"SFO_S:", sfo.KFixes["PPEGS"].Box(30,30)})
	}

	if len(matchers) == 0 {
		return
	}
	
	for _,trackName := range f.ListTracks() {
		t := f.Tracks[trackName]
		lines := t.AsLinesSampledEvery(time.Second*1)
	
		for _,line := range lines {
			for _,matcher := range matchers {
				if matcher.IntersectsLine(line) {
					f.SetTag(matcher.Tag)
				}
			}
		}
	}

	// Chained matchers
	// SFO_NW is SFO_N && flies over BRIXX (at alt >5000')
	if f.HasTag(":SFO_N") && f.HasWaypoint("BRIXX") {
		altAtBrixx := 0.0
		for _,trackName := range f.ListTracks() {
			t := f.Tracks[trackName]
			if brixxIndex := (*t).IndexAtTime(f.Waypoints["BRIXX"]); brixxIndex >= 0 {
				altAtBrixx = (*t)[brixxIndex].Altitude
			}
		}
		if altAtBrixx > 5000 {
			f.SetTag(":SFO_NW")
		}
	}

	// SFO_NE is SFO_N && flies within 3 km of FINSH (so boxside=6)
	if f.HasTag(":SFO_NE") && !f.HasTag(":SFO_N") { f.DropTag(":SFO_NE") }
}

type Procedure struct {
	Name         string            // E.g. SERFR2
	Waypoints  []string            // The sequence of waypoints that makes it up
	Required     map[string]int   // Which of the waypoints can't be omitted
}

// Did the flight fly the 'Required' waypoints of the procedure ? The string is the name
// of the final waypoint of the procedure that was flown - i.e. the flight was vectored
// off-procedure after that waypoint.
func (f *Flight)FlewProcedure(p Procedure) (bool,string) {
	for i,wp := range p.Waypoints {
		if !f.HasWaypoint(wp) {
			if _,exists := p.Required[wp]; exists { return false, "" }
			if i == 0 { return false, wp } // "This should never happen"
			return true, p.Waypoints[i-1]
		}
	}

	return true,""
}

var NorCalProcedures = []Procedure{
	{
		Name:      "BIGSUR2",
		Waypoints: []string{"ANJEE", "SKUNK", "BOLDR", "MENLO"}, // Ignore CARME
		Required:  map[string]int{"ANJEE":1, "SKUNK":1},
	},
	{
		Name:      "SERFR2",
		Waypoints: []string{"WWAVS", "EPICK", "EDDYY", "SWELS", "MENLO"}, // Ignore SERFR
		Required:  map[string]int{"WWAVS":1, "EPICK":1},
	},
	{
		Name:      "WWAVS1",
		Waypoints: []string{"WWAVS", "WPOUT", "THEEZ", "WESLA", "MVRKK"}, // Ignore SERFR
		Required:  map[string]int{"WWAVS":1, "WPOUT":1},
	},
}

type FlownProcedure struct {
	Name          string  `json:"name,omitempty"` // Name of the proecdure itself
	VectoredAfter string  `json:"vectoredafter,omitempty"` // Name of the final on-procedure waypoint
}
func (fp FlownProcedure)String() string {
	str := fp.Name
	if fp.VectoredAfter != "" {
		str += "/" + fp.VectoredAfter
	}
	return str
}

func (f *Flight)DetermineFlownProcedure() FlownProcedure {
	for _,proc := range NorCalProcedures {
		if flew,vector := f.FlewProcedure(proc); flew {
			return FlownProcedure{Name: proc.Name, VectoredAfter:vector}
		}
	}
	return FlownProcedure{}
}

func (f *Flight)DetermineFlownProcedures() []FlownProcedure {
	ret := []FlownProcedure{}
	for _,proc := range NorCalProcedures {
		if flew,vector := f.FlewProcedure(proc); flew {
			ret = append(ret, FlownProcedure{Name: proc.Name, VectoredAfter:vector})
		}
	}
	return ret
}
