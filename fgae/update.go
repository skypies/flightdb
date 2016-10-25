package fgae

import(
	"fmt"
	"time"
	
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/datastore"

	"github.com/skypies/geo/sfo"

	fdb "github.com/skypies/flightdb2"
	"github.com/skypies/flightdb2/ref"
)

// Will be nil if we don't have the data we need to specify an ancestor ID
func (db *FlightDB)rootKeyOrNil(f *fdb.Flight) *datastore.Key {
	if f.IcaoId != "" {
		return datastore.NewKey(db.C, kFlightKind, string(f.IcaoId), 0, nil)
	} else if f.Callsign != "" {
		return datastore.NewKey(db.C, kFlightKind, "c:"+f.Callsign, 0, nil)
	}

	return nil
}

func (db *FlightDB)findOrGenerateFlightKey(f *fdb.Flight) (*datastore.Key, error) {
	if f.GetDatastoreKey() != "" {
		return datastore.DecodeKey(f.GetDatastoreKey())
	}
		
	// We use IcaoId/Callsign (if we have either) to build the unique
	// ancestor key. This is so we can use ancestor queries when we're
	// looking up by IcaoId, and get strongly consistent query results
	// (e.g. read-your-writes). (We need this for AddTrackFragment)
	rootKey := db.rootKeyOrNil(f)

	// Avoid incomplete keys if we can ...
	//k := datastore.NewIncompleteKey(db.C, kFlightKind, rootKey)

	// In some circumstances, AppEngine will trigger a URL twice; if
	// this happens for URLs that do batch loading of flight data from
	// GCS, this will cause duplicate flight entries. So, if we have some kind
	// of track data, turn the first timestamp into an integer ID; then if we end
	// up trying to create the exact same flight twice, we will avoid dupes.
	var intKey int64 = 0
	if t := f.AnyTrack(); len(t) >= 0 {
		intKey = t[0].TimestampUTC.Unix()
	}
	k := datastore.NewKey(db.C, kFlightKind, "", intKey, rootKey)
	
	log.Infof(db.C, "creating a new key: %v", k)
	
	return k, nil
}

func (db *FlightDB)PersistFlight(f *fdb.Flight) error {
	key,err := db.findOrGenerateFlightKey(f)
	if err != nil { return err }
	
	if blob,err := f.ToBlob(); err != nil {
		return err
	} else {
		_, err = datastore.Put(db.C, key, blob)
		if err != nil {
			db.Errorf("PersistFlight[%s]: %v", f, err)
		}
		return err
	}
}

func (db FlightDB)AddTrackFragment(frag *fdb.TrackFragment) error {
	db.Debugf("* adding frag %s\n", frag)
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

		trackKey := frag.TrackName() // ADSB, or MLAT
		
		if track,exists := f.Tracks[trackKey]; !exists {
			f.DebugLog += "-- AddFrag "+prefix+": first frag on pre-existing flight\n"
			db.Infof("* %s no pre-existing track; adding right in", prefix)
			f.Tracks[trackKey] = &frag.Track

		} else if plausible,debug := track.PlausibleExtension(&frag.Track); plausible==true {
			f.DebugLog += fmt.Sprintf("-- AddFrag %s: extending (adding %d to %d points)\n",
				prefix, len(frag.Track), len(*track))
			db.Debugf("* %s extending track ... debug:\n%s", prefix, debug)

			// For MLAT data, callsigns can take a while to show up in the stream
			if f.Identity.Callsign == "" && frag.Callsign != "" {
				f.DebugLog += fmt.Sprintf(" - prev callsign was nil; adding it in now\n")
				f.Identity.Callsign = frag.Callsign
			}

			// Determine whether this frag is strictly a suffix to existing track data; this is the
			// common case. If so, keep a pointer to the trackpoint that precedes the frag 
			n := len(*f.Tracks[trackKey])
			if n>0 && (*f.Tracks[trackKey])[n-1].TimestampUTC.Before(frag.Track[0].TimestampUTC) {
				db.Debugf("** new frag is strictly a suffix; prev = %d", n-1)
				prevTP = &((*f.Tracks[trackKey])[n-1])
			}
			
			db.Debugf("** pre : %s", f.Tracks[trackKey])
			f.Tracks[trackKey].Merge(&frag.Track)
			db.Debugf("** post: %s", f.Tracks[trackKey])

		}	else {
			f = fdb.NewFlightFromTrackFragment(frag)
			f.DebugLog += "-- AddFrag "+prefix+": was not plausible, so new flight\n"
			db.Infof("* %s not a plausible addition; starting afresh ... debug\n%s", prefix, debug)
			f.DebugLog += debug+"\n"
		}
	}

	// Consult the airframe cache, and perhaps add some metadata, if not already present
	if f.Airframe.Registration == "" {
		airframes := ref.NewAirframeCache(db.C)
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
		// a = append([]T{x}, a...)
		frag.Track = append([]fdb.Trackpoint{*prevTP}, frag.Track...)
	}

	// Incrementally identify waypoints, frag by frag
	for wp,t := range frag.Track.MatchWaypoints(sfo.KFixes) {
		f.DebugLog += "-- AddFrag "+prefix+": found waypoint "+wp+"\n"
		f.SetWaypoint(wp,t)
	}
	
	return db.PersistFlight(f)
}

// Say we've pulled some identity information from somewhere; if it matches something,
// let's merge it in
//func (db FlightDB)AddPartialIdentity(id *fdb.Identity) error {
//	return nil
//}
