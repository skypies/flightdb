package fgae

import(
	"fmt"
	"golang.org/x/net/context"
	ds "github.com/skypies/util/dsprovider"
	fdb "github.com/skypies/flightdb"
)

// {{{ db.PersistFlight

func (db *FlightDB)PersistFlight(f *fdb.Flight) error {
	keyer,err := findOrGenerateFlightKey(db.Ctx(), db.Backend, f)
	if err != nil { return fmt.Errorf("PersistFlight: %v", err) }
	
	if blob,err := f.ToBlob(); err != nil {
		return fmt.Errorf("PersistFlight: %v", err)
	} else {
		_, err = db.Backend.Put(db.Ctx(), keyer, blob)
		if err != nil {
			return fmt.Errorf("PersistFlight: %v", err)
		}
	}

	return nil
}

// }}}

// {{{ db.LookupKey

func (db *FlightDB)LookupKey(keyer ds.Keyer) (*fdb.Flight, error) {
	blob := fdb.IndexedFlightBlob{}

	if err := db.Backend.Get(db.Ctx(), keyer, &blob); err != nil {
		return nil, fmt.Errorf("GetByKey: %v", err)
	}

	f, err := blob.ToFlight(keyer.Encode())
	if err != nil { err = fmt.Errorf("GetByKey: %v", err) }
	return f,err
}

// }}}
// {{{ db.LookupAll

func (db *FlightDB)LookupAll(fq *FQuery) ([]*fdb.Flight, error) {
	// Results are not ordered ... for timerange idspecs, would need to sort on Timeslots
	blobs := []fdb.IndexedFlightBlob{}

	keyers, err := db.Backend.GetAll(db.Ctx(), (*ds.Query)(fq), &blobs)
	if err != nil {
		return nil, fmt.Errorf("GetAllByQuery: %v", err)
	}

	flights := []*fdb.Flight{}
	for i,blob := range blobs {
		if flight,err := blob.ToFlight(keyers[i].Encode()); err != nil {
			return nil, fmt.Errorf("GetAllByQuery: %v", err)
		} else {
			flights = append(flights, flight)
		}
	}

	return flights, nil
}

// }}}
// {{{ db.LookupFirst

func (db *FlightDB)LookupFirst(fq *FQuery) (*fdb.Flight, error) {
	if flights,err := db.LookupAll(fq.Limit(1)); err != nil {
		return nil,fmt.Errorf("GetFirstByQuery: %v", err)
	} else if len(flights) == 0 {
		return nil,nil
	} else {
		return flights[0],nil
	}
}

// }}}
// {{{ db.LookupMostRecent

func (db *FlightDB)LookupMostRecent(fq *FQuery) (*fdb.Flight, error) {
	// Adding the ordering will break some queries, due to lack of indices
	return db.LookupFirst(fq.Order("-LastUpdate"))
}

// }}}
// {{{ db.LookupAllKeys

func (db *FlightDB)LookupAllKeys(fq *FQuery) ([]ds.Keyer, error) {
	q := (*ds.Query)(fq)
	return db.Backend.GetAll(db.Ctx(), q.KeysOnly(), nil)
}

// }}}

// {{{ db.DeleteByKey

func (db *FlightDB)DeleteByKey(keyer ds.Keyer) error {
	return db.Backend.Delete(db.Ctx(), keyer)
}

// }}}
// {{{ db.DeleteAllKeys

func (db *FlightDB)DeleteAllKeys(keyers []ds.Keyer) error {
	return db.Backend.DeleteMulti(db.Ctx(), keyers)
}

// }}}

// {{{ findOrGenerateFlightKey

// Will be nil if we don't have the data we need to specify an ancestor ID
func rootKeyOrNil(ctx context.Context, p ds.DatastoreProvider, f *fdb.Flight) ds.Keyer {
	if f.IcaoId != "" {
		return p.NewNameKey(ctx, kFlightKind, string(f.IcaoId), nil)
	} else if f.Callsign != "" {
		return p.NewNameKey(ctx, kFlightKind, "c:"+f.Callsign, nil)
	}
	return nil
}

func findOrGenerateFlightKey(ctx context.Context, p ds.DatastoreProvider, f *fdb.Flight) (ds.Keyer, error) {
	if f.GetDatastoreKey() != "" {
		return p.DecodeKey(f.GetDatastoreKey())
	}
		
	// We use IcaoId/Callsign (if we have either) to build the unique
	// ancestor key. This is so we can use ancestor queries when we're
	// looking up by IcaoId, and get strongly consistent query results
	// (e.g. read-your-writes). (We need this for AddTrackFragment)
	rootKey := rootKeyOrNil(ctx, p, f)

	// Avoid incomplete keys if we can ...
	//    k := datastore.NewIncompleteKey(db.C, kFlightKind, rootKey)
	// ... as in some circumstances, AppEngine will trigger a URL twice;
	// if this happens for URLs that do batch loading of flight data
	// from GCS, this will cause duplicate flight entries. So, if we
	// have some kind of track data, turn the first timestamp into an
	// integer ID; then if we end up trying to create the exact same
	// flight twice, we will avoid dupes.
	var intKey int64 = 0
	if t := f.AnyTrack(); len(t) >= 0 {
		intKey = t[0].TimestampUTC.Unix()
	}
	keyer := p.NewIDKey(ctx, kFlightKind, intKey, rootKey)

	return keyer, nil
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
