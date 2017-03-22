package db

import(
	"golang.org/x/net/context"
	fdb "github.com/skypies/flightdb"
)

/*

  import "github.com/skypies/flightdb/db"

	backend := db.AppengineDSProvider{}  // Or db.CloudDSProvider{"projname"}, if outside appengine
	
	q := db.NewFlightQuery().ByIdSpec(idspec)
	flights,err := db.GetAllByQuery(ctx, backend, q)

  // update the first flight ...

  _,err := db.PersistFlight(ctx, backend, flights[0])

 */

type FlightDBProvider interface {
	GetAll(ctx context.Context, q *Query, dst interface{}) ([]Keyer, error)
	Put(ctx context.Context, keyer Keyer, src interface{}) (Keyer, error)
	
	NewNameKey(ctx context.Context, kind, name string, root Keyer) Keyer
	NewIDKey(ctx context.Context, kind string, id int64, root Keyer) Keyer
	DecodeKey(encoded string) (Keyer, error)
}

// Functions in this file work on top of any implementation of the FlightDBProvider interface

// {{{ GetAllByQuery

func GetAllByQuery(ctx context.Context, backend FlightDBProvider, q *Query) ([]*fdb.Flight, error) {
	blobs := []fdb.IndexedFlightBlob{}

	keyers, err := backend.GetAll(ctx, q, &blobs)
	if err != nil {
		return nil, err
	}

	flights := []*fdb.Flight{}
	for i,blob := range blobs {
		if flight,err := blob.ToFlight(keyers[i].Encode()); err != nil {
			return nil, err
		} else {
			flights = append(flights, flight)
		}
	}

	return flights, nil
}

// }}}
// {{{ GetKeysByQuery

func GetKeysByQuery(ctx context.Context, backend FlightDBProvider, q *Query) ([]Keyer, error) {
	return backend.GetAll(ctx, q.KeysOnly(), nil)
}

// }}}
// {{{ PersistFlight

func PersistFlight(ctx context.Context, backend FlightDBProvider, f *fdb.Flight) error {
	keyer,err := findOrGenerateFlightKey(ctx, backend, f)
	if err != nil { return err }
	
	if blob,err := f.ToBlob(); err != nil {
		return err
	} else {
		_, err = backend.Put(ctx, keyer, blob)
		if err != nil {
			//db.Errorf("PersistFlight[%s]: %v", f, err)
		}
		return err
	}
}

// }}}

// {{{ findOrGenerateFlightKey

// Will be nil if we don't have the data we need to specify an ancestor ID
func rootKeyOrNil(ctx context.Context, db FlightDBProvider, f *fdb.Flight) Keyer {
	if f.IcaoId != "" {
		return db.NewNameKey(ctx, kFlightKind, string(f.IcaoId), nil)
	} else if f.Callsign != "" {
		return db.NewNameKey(ctx, kFlightKind, "c:"+f.Callsign, nil)
	}
	return nil
}

func findOrGenerateFlightKey(ctx context.Context, db FlightDBProvider, f *fdb.Flight) (Keyer, error) {
	if f.GetDatastoreKey() != "" {
		return db.DecodeKey(f.GetDatastoreKey())
	}
		
	// We use IcaoId/Callsign (if we have either) to build the unique
	// ancestor key. This is so we can use ancestor queries when we're
	// looking up by IcaoId, and get strongly consistent query results
	// (e.g. read-your-writes). (We need this for AddTrackFragment)
	rootKey := rootKeyOrNil(ctx, db, f)

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
	keyer := db.NewIDKey(ctx, kFlightKind, intKey, rootKey)

	return keyer, nil
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
