package flightdb2

import(
	"time"
	"github.com/skypies/geo"
	"github.com/skypies/geo/sfo"
)

var(
	KWaypointSnapKM = 1.0
)

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

// Find a better home for this config
var (
	OceanicAirports = map[string]int{
		"LIH":1, "OGG":1, "HNL":1, "KOA":1, "NRT":1, "HND":1, "KIX":1, "PVG":1, "PEK":1, "CAN":1,
		"CTU":1, "WUH":1, "HKG":1, "TPE":1, "ICN":1, "MNL":1, "NHL":1, "SYD":1, "VRD":1, "AKL":1,
	}
	SouthwestAirports = map[string]int{
		"PHX":1, "TUS":1, "SBP":1, "LAX":1, "LGB":1, "BUR":1, "ONT":1, "SNA":1, "DCA":1,
		"SBA":1, "PSP":1, "SAN":1,
	}
	NorCalAirports = map[string]int{
		"SFO":1, "SJC":1, "OAK":1,
	}
)


// SFO_S_A for southern arrivals:  :SFO && 30 km box around ANJEE, WWAVE, or their midpoint)
// SFO_S_D for southern departures:  (SFO: ||OAK:) && 30 km (TBR) box around PPEGS
func (f *Flight)TagCoarseFlightpathForSFO() {
	box := geo.LatlongBox{}
	tag := ""
	
	if f.Destination == "SFO" {
		box = sfo.KFixes["WWAVS"].Box(30,30)
		tag = ":SFO_S"
	} else if f.Origin == "SFO" || f.Origin == "OAK" {
		box = sfo.KFixes["PPEGS"].Box(30,30)
		tag = "SFO_S:"
	} else {
		return
	}
	
	for _,trackName := range f.ListTracks() {
		t := f.Tracks[trackName]
		lines := t.AsLinesSampledEvery(time.Second*1)
	
		for _,line := range lines {
			if box.IntersectsLine(line) {
				f.SetTag(tag)
				return
			}
		}
	}
}

