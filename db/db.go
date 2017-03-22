package db

import(
	"fmt"
	"golang.org/x/net/context"
	fdb "github.com/skypies/flightdb"
)

/*  import "github.com/skypies/flightdb/db"

    backend := db.AppengineDSProvider{}  // Or db.CloudDSProvider{"projname"}, if outside appengine
	
    q := db.NewFlightQuery().ByIdSpec(idspec)
    flights,err := db.GetAllByQuery(ctx, backend, q)

    // update the first flight ...

    _,err := db.PersistFlight(ctx, backend, flights[0])
 */

type DatastoreProvider interface {
	Get(ctx context.Context, keyer Keyer, dst interface{}) error
	GetAll(ctx context.Context, q *Query, dst interface{}) ([]Keyer, error)
	Put(ctx context.Context, keyer Keyer, src interface{}) (Keyer, error)
	Delete(ctx context.Context, keyer Keyer) error
	DeleteMulti(ctx context.Context, keyers []Keyer) error
	
	NewIncompleteKey(ctx context.Context, kind string, root Keyer) Keyer
	NewNameKey(ctx context.Context, kind, name string, root Keyer) Keyer
	NewIDKey(ctx context.Context, kind string, id int64, root Keyer) Keyer
	DecodeKey(encoded string) (Keyer, error)

	// Infof, etc ?
}

// Functions in this file work on top of any implementation of the DatastoreProvider interface

// {{{ GetByKey

func GetByKey(ctx context.Context, p DatastoreProvider, keyer Keyer) (*fdb.Flight, error) {
	blob := fdb.IndexedFlightBlob{}

	if err := p.Get(ctx, keyer, &blob); err != nil {
		return nil, fmt.Errorf("GetByKey: %v", err)
	}

	f, err := blob.ToFlight(keyer.Encode())
	if err != nil { err = fmt.Errorf("GetByKey: %v", err) }
	return f,err
}

// }}}
// {{{ GetAllByQuery

func GetAllByQuery(ctx context.Context, p DatastoreProvider, q *Query) ([]*fdb.Flight, error) {
	blobs := []fdb.IndexedFlightBlob{}

	keyers, err := p.GetAll(ctx, q, &blobs)
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
// {{{ GetKeysByQuery

func GetKeysByQuery(ctx context.Context, p DatastoreProvider, q *Query) ([]Keyer, error) {
	keyers, err := p.GetAll(ctx, q.KeysOnly(), nil)
	if err != nil { err = fmt.Errorf("GetKeysByQuery: %v", err) }
	return keyers,err
}

// }}}
// {{{ PersistFlight

func PersistFlight(ctx context.Context, p DatastoreProvider, f *fdb.Flight) error {
	keyer,err := findOrGenerateFlightKey(ctx, p, f)
	if err != nil { return fmt.Errorf("PersistFlight: %v", err) }
	
	if blob,err := f.ToBlob(); err != nil {
		return fmt.Errorf("PersistFlight: %v", err)
	} else {
		_, err = p.Put(ctx, keyer, blob)
		if err != nil {
			return fmt.Errorf("PersistFlight: %v", err)
		}
	}

	return nil
}

// }}}

// {{{ findOrGenerateFlightKey

// Will be nil if we don't have the data we need to specify an ancestor ID
func rootKeyOrNil(ctx context.Context, p DatastoreProvider, f *fdb.Flight) Keyer {
	if f.IcaoId != "" {
		return p.NewNameKey(ctx, kFlightKind, string(f.IcaoId), nil)
	} else if f.Callsign != "" {
		return p.NewNameKey(ctx, kFlightKind, "c:"+f.Callsign, nil)
	}
	return nil
}

func findOrGenerateFlightKey(ctx context.Context, p DatastoreProvider, f *fdb.Flight) (Keyer, error) {
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
