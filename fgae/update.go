package fgae

import(
	"fmt"
	
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
	
	if blob,err := f.ToBlob(kTimeslotDuration); err != nil {
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

	prefix := fmt.Sprintf("[%s/%s]%s", frag.IcaoId, frag.Callsign, frag.DataSystem())
	
	if f == nil {
		f = fdb.NewFlightFromTrackFragment(frag)
		f.DebugLog += "-- AddFrag "+prefix+": new IcaoID\n"
		db.Debugf("* %s brand new IcaoID: %s", prefix, f)
		
	} else {
		db.Debugf("* %s found %s", prefix, f)

		trackType := frag.DataSystem() // MLAT or ADSB
		
		if track,exists := f.Tracks[trackType]; !exists {
			f.DebugLog += "-- AddFrag "+prefix+": first frag on pre-existing flight\n"
			db.Infof("* %s no pre-existing track; adding right in", prefix)
			f.Tracks[trackType] = &frag.Track

		} else if plausible,debug := track.PlausibleExtension(&frag.Track); plausible==true {
			f.DebugLog += fmt.Sprintf("-- AddFrag "+prefix+": extending (adding %d to %d points)\n",
				len(frag.Track), len(*track))
			db.Debugf("* %s extending track ... debug:\n%s", prefix, debug)
			db.Debugf("** pre : %s", f.Tracks[trackType])
			f.Tracks[trackType].Merge(&frag.Track)
			db.Debugf("** post: %s", f.Tracks[trackType])

		}	else {
			f = fdb.NewFlightFromTrackFragment(frag)
			f.DebugLog += "-- AddFrag "+prefix+": was not plausible, so new flight\n"
			db.Infof("* %s not a plausible addition; starting afresh ... debug\n%s", prefix, debug)
			f.DebugLog = debug
		}
	}

	// Consult the airframe cache, and perhaps add some metadata, if not already present
	// This doesn't appear to be working :(
	if f.Airframe.Registration == "" {
		airframes := ref.NewAirframeCache(db.C)
		if af := airframes.Get(f.IcaoId); af != nil {
			f.DebugLog += "-- AddFrag "+prefix+": found airframe\n"
			f.Airframe = *af
		}
	}

	// Incrementally identify waypoints, frag by frag
	for wp,t := range frag.Track.MatchWaypoints(sfo.KFixes) {
		f.DebugLog += "-- AddFrag "+prefix+": found waypoint "+wp+"\n"
		f.SetWaypoint(wp,t)
	}
	
	return db.PersistFlight(f)
	//return nil
}

// Say we've pulled some identity information from somewhere; if it matches something,
// let's merge it in
func (db FlightDB)AddPartialIdentity(id *fdb.Identity) error {
	return nil
}
