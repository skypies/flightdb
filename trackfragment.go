package flightdb

import(
	"fmt"
	"sort"

	"github.com/skypies/adsb"
)

// A TrackFragment is part of a track, built from ADSB (or MLAT)
// messages. A series of these are typically glued together to form a
// complete Track, as they are received in batches.

type TrackFragment struct {
	IcaoId     adsb.IcaoId
	Callsign   string  // Might not yet be populated
	Track      // embedded Track
	DataSystem // embedded; e.g. MLAT vs. ADSB
}

func MessagesToTrackFragment(msgs []*adsb.CompositeMsg) *TrackFragment {
	if len(msgs)==0 { return nil }

	dataSystem := DSADSB
	if msgs[0].IsMLAT() {
		dataSystem = DSMLAT
	}
	
	frag := TrackFragment{
		IcaoId: msgs[0].Icao24,
		Callsign: msgs[0].Callsign,
		DataSystem: dataSystem,
	}

	sort.Sort(adsb.CompositeMsgPtrByTimeAsc(msgs))
	
	for _,m := range msgs {
		frag.Track = append(frag.Track, TrackpointFromADSB(m))
	}

	return &frag
}

func (frag TrackFragment)String() string {
	n := len(frag.Track)
	str := fmt.Sprintf("[%s/%s]%s %s +%s (%d points)", frag.Callsign, frag.IcaoId,
		frag.DataSystem, frag.Track[0].TimestampUTC.Format("15:04:05 MST"),
		frag.Track[n-1].TimestampUTC.Sub(frag.Track[0].TimestampUTC), n)
	return str
}

// Remains backward compatible with existing tracks.
// More generally, this should be in trackpoint-data, as a function on the trackpoint
func (frag TrackFragment)TrackName() string {
	switch frag.DataSystem {
	case DSADSB: return "ADSB"
	case DSMLAT: return "MLAT"
	default: return fmt.Sprintf("t_%s", frag.DataSystem) // This would be an error
	}
}
