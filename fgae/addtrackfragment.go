package fgae

import(
	"fmt"
	"time"

	"github.com/skypies/geo/sfo"

	fdb "github.com/skypies/flightdb"
	"github.com/skypies/flightdb/ref"
)

// {{{ currentAccumulationTrack

func currentAccumulationTrack(f *fdb.Flight) *fdb.Track {
	if !f.HasTrack("ADSB") && !f.HasTrack("MLAT") { return nil }
	if !f.HasTrack("ADSB") { return f.Tracks["MLAT"] }
	if !f.HasTrack("MLAT") { return f.Tracks["ADSB"] }

	mlat,adsb := f.Tracks["MLAT"],f.Tracks["ADSB"]

	if len(*mlat) == 0 { return adsb }
	if len(*adsb) == 0 { return mlat }

	// Both tracks exist and are not empty ! Return most recent
	if (*mlat).End().After( (*adsb).End() ) {
		return mlat
	} else {
		return adsb
	}
}

// }}}
// {{{ db.AddTrackFragment

func (db FlightDB)AddTrackFragment(frag *fdb.TrackFragment) error {
	db.Debugf("* adding frag %d\n", len(frag.Track))
	f,err := db.LookupMostRecent(db.NewQuery().ByIcaoId(frag.IcaoId))
	if err != nil { return err }

	prefix := fmt.Sprintf("[%s/%s]%s %s", frag.IcaoId, frag.Callsign, frag.DataSystem, time.Now())

	// If the fragment is strictly a suffix, this will hold the preceding point
	var prevTP *fdb.Trackpoint

	if f == nil {
		f = fdb.NewFlightFromTrackFragment(frag)
		f.DebugLog += "-- AddFrag "+prefix+": new IcaoID\n"
		db.Debugf("* %s brand new IcaoID: %s", prefix, f)
		
	} else {
		db.Debugf("* %s found %s", prefix, f)

		trackKey := frag.TrackName() // ADSB, or MLAT; this is the track we will accumulate into

		// This is the most recent track we've accumulated into (could be ADSB, or MLAT); nil if none.
		// Note that this doesn't have to be the same as trackKey; we might be adding ADSB, but already
		// have some MLAT for the flight.
		accTrack := currentAccumulationTrack(f)
		
		if accTrack == nil {
			f.DebugLog += "-- AddFrag "+prefix+": first frag on pre-existing flight\n"
			db.Debugf("* %s no pre-existing track; adding right in", prefix)
			f.Tracks[trackKey] = &frag.Track

		} else if plausible,debug := accTrack.PlausibleContribution(&frag.Track); plausible==true {
			f.DebugLog += fmt.Sprintf("-- AddFrag %s: extending (adding %d to %d points)\n",
				prefix, len(frag.Track), len(*accTrack))
			db.Debugf("* %s extending track ... debug:\n%s", prefix, debug)

			// For MLAT data, callsigns can take a while to show up in the stream
			if f.Identity.Callsign == "" && frag.Callsign != "" {
				f.DebugLog += fmt.Sprintf(" - prev callsign was nil; adding it in now\n")
				f.Identity.Callsign = frag.Callsign
			}

			if !f.HasTrack(trackKey) {
				// If the accTrack was a different type (MLAT vs. ADSB), then we'll need to init
				f.Tracks[trackKey] = &fdb.Track{}

			} else {
				// Determine whether this frag is strictly a suffix to existing track data; this is the
				// common case. If so, keep a pointer to the trackpoint that precedes the frag 
				n := len(*f.Tracks[trackKey])
				if n>0 && (*f.Tracks[trackKey])[n-1].TimestampUTC.Before(frag.Track[0].TimestampUTC) {
					db.Debugf("** new frag is strictly a suffix; prev = %d", n-1)
					prevTP = &((*f.Tracks[trackKey])[n-1])
				}
			}

			db.Debugf("* %s adding %d points to %d\n", prefix, len(frag.Track), len(*f.Tracks[trackKey]))

			db.Debugf("** pre : %s", f.Tracks[trackKey])
			f.Tracks[trackKey].Merge(&frag.Track)
			db.Debugf("** post: %s", f.Tracks[trackKey])

		}	else {
			f = fdb.NewFlightFromTrackFragment(frag)
			f.DebugLog += "-- AddFrag "+prefix+": was not plausible, so new flight\n"
			db.Debugf("* %s not a plausible addition; starting afresh ... debug\n%s", prefix, debug)
			f.DebugLog += debug+"\n"
		}
	}

	// Consult the airframe cache, and perhaps add some metadata, if not already present
	if f.Airframe.Registration == "" {
		airframes := ref.NewAirframeCache(db.Ctx())
		if af := airframes.Get(f.IcaoId); af != nil {
			f.DebugLog += "-- AddFrag "+prefix+": found airframe\n"
			f.Airframe = *af
		}
	}

	// There could be a big gap between the previous track and this frag.
	// If that's the case, grab the preceding trackpoint and prefix this frag with it; then
	// the waypoint detection code (which builds lines between points) will look at the gap
	// between the frags, and maybe find extra waypoints.
	if prevTP != nil {
		// shift(x,a) : a = append([]T{x}, a...)
		frag.Track = append([]fdb.Trackpoint{*prevTP}, frag.Track...)
	}

	// Incrementally identify waypoints, frag by frag
	for wp,t := range frag.Track.MatchWaypoints(sfo.KFixes) {
		f.DebugLog += "-- AddFrag "+prefix+": found waypoint "+wp+"\n"
		f.SetWaypoint(wp,t)
	}
	
	return db.PersistFlight(f)
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
