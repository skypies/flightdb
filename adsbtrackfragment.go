package flightdb2

import(
	"fmt"
	"sort"

	"github.com/skypies/adsb"
)

// An ADSBTrackFragment is part of a track, built from ADSB messages. A series of these are
// typically glued together to form a complete Track, as they are received in batches.

type ADSBTrackFragment struct {
	IcaoId     adsb.IcaoId
	Callsign   string
	Track      // embedded Track
}

func MessagesToADSBTrackFragment(msgs []*adsb.CompositeMsg) *ADSBTrackFragment {
	if len(msgs)==0 { return nil }

	sort.Sort(adsb.CompositeMsgPtrByTimeAsc(msgs))

	frag := ADSBTrackFragment{
		IcaoId: msgs[0].Icao24,
		Callsign: msgs[0].Callsign,
	}
	
	for _,m := range msgs {
		frag.Track = append(frag.Track, TrackpointFromADSB(m))
	}

	return &frag
}

func (frag ADSBTrackFragment)String() string {
	n := len(frag.Track)
	str := fmt.Sprintf("[%s/%s] %s +%s (%d points)", frag.Callsign, frag.IcaoId,
		frag.Track[0].TimestampUTC.Format("15:04:05 MST"),
		frag.Track[n-1].TimestampUTC.Sub(frag.Track[0].TimestampUTC), n)
	return str
}
