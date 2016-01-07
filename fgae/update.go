package fgae

import(
	"google.golang.org/appengine/datastore"
	fdb "github.com/skypies/flightdb2"
)

func (db *FlightDB)findOrGenerateFlightKey(f *fdb.Flight) (*datastore.Key, error) {
	if f.GetDatastoreKey() != "" {
		return datastore.DecodeKey(f.GetDatastoreKey())
	}
	//rootKey := datastore.NewKey(db.C, kFlightKind, "foo", 0, nil)
	return datastore.NewIncompleteKey(db.C, kFlightKind, nil), nil
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
	
	if f == nil {
		f = fdb.NewFlightFromADSBTrackFragment(frag)
		db.Debugf("* brand new IcaoID: %s", f)
		
	} else {
		db.Debugf("* found %s", f)

		if adsbTrack,exists := f.Tracks["ADSB"]; !exists {
			db.Debugf("* no pre-existing ADSB track %v", 1)
			f.Tracks["ADSB"] = &frag.Track

		} else if plausible,debug := adsbTrack.PlausibleExtension(&frag.Track); plausible==true {
			db.Debugf("* extending ADSB track ... debug:\n%s", debug)
			db.Debugf("** pre : %s", f.Tracks["ADSB"])
			f.Tracks["ADSB"].Merge(&frag.Track)
			db.Debugf("** post: %s", f.Tracks["ADSB"])

		}	else {
			f = fdb.NewFlightFromADSBTrackFragment(frag)
			db.Infof("* not a plausible addition for [%s]; starting afresh ...", frag.Callsign)
			db.Infof("* debug:\n%s", debug)
			f.DebugLog = debug
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
