package fgae

import(
	"fmt"
	
	"google.golang.org/appengine/datastore"
	fdb "github.com/skypies/flightdb2"
	"github.com/skypies/flightdb2/ref"
)

// Will be nil if we don't have the data we need to specify an ancestor ID
func (db *FlightDB)rootKeyOrNil(f *fdb.Flight) *datastore.Key {
	if f.IcaoId == "" {
		return nil
	}

	return datastore.NewKey(db.C, kFlightKind, string(f.IcaoId), 0, nil)
}

func (db *FlightDB)findOrGenerateFlightKey(f *fdb.Flight) (*datastore.Key, error) {
	if f.GetDatastoreKey() != "" {
		return datastore.DecodeKey(f.GetDatastoreKey())
	}
		
	// We use IcaoId (if we have one) to build the unique ancestor key.
	// This is so we can use ancestor queries when we're looking up by
	// IcaoId, and get strongly consistent query results (e.g.
	// read-your-writes). (We need this for AddADSBTrackFragment)
	rootKey := db.rootKeyOrNil(f)
	return datastore.NewIncompleteKey(db.C, kFlightKind, rootKey), nil
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

func (db FlightDB)AddADSBTrackFragment(frag *fdb.ADSBTrackFragment) error {
	db.Debugf("* adding frag %s\n", frag)
	f,err := db.LookupMostRecent(db.NewQuery().ByIcaoId(frag.IcaoId))
	if err != nil { return err }

	prefix := fmt.Sprintf("[%s/%s]", frag.IcaoId, frag.Callsign)
	
	if f == nil {
		f = fdb.NewFlightFromADSBTrackFragment(frag)
		db.Debugf("* %s brand new IcaoID: %s", prefix, f)
		
	} else {
		db.Debugf("* %s found %s", prefix, f)

		if adsbTrack,exists := f.Tracks["ADSB"]; !exists {
			db.Infof("* %s no pre-existing ADSB track; adding right in", prefix)
			f.Tracks["ADSB"] = &frag.Track

		} else if plausible,debug := adsbTrack.PlausibleExtension(&frag.Track); plausible==true {
			db.Infof("* %s extending ADSB track ... debug:\n%s", prefix, debug)
			db.Debugf("** pre : %s", f.Tracks["ADSB"])
			f.Tracks["ADSB"].Merge(&frag.Track)
			db.Debugf("** post: %s", f.Tracks["ADSB"])

		}	else {
			f = fdb.NewFlightFromADSBTrackFragment(frag)
			db.Infof("* %s not a plausible addition; starting afresh ... debug\n%s", prefix, debug)
			f.DebugLog = debug
		}
	}

	// Consult the airframe cache, and perhaps add some metadata, if not already present
	// This doesn't appear to be working :(
	if f.Airframe.Registration == "" {
		airframes := ref.NewAirframeCache(db.C)
		if af := airframes.Get(f.IcaoId); af != nil {
			f.Airframe = *af
		}
	}
	
	return db.PersistFlight(f)
	//return nil
}

// Say we've pulled some identity information from somewhere; if it matches something,
// let's merge it in
func (db FlightDB)AddPartialIdentity(id *fdb.Identity) error {
	return nil
}
