package fgae

import(
	"google.golang.org/appengine/datastore"
	fdb "github.com/skypies/flightdb"
	"github.com/skypies/flightdb/db"
)

// LongIterator is a drop-in replacement, to allow the consumer to spend more than 60s
// iterating over the result set.
type LongIterator struct {
	DB     *FlightDB

	Keys []*datastore.Key // The full result set

	i       int
	val    *fdb.Flight
	err     error
}

// Snarf down all the keys from the get go.
func (flightdb *FlightDB)NewLongIterator(q *db.Query) *LongIterator {
	keyers,err := flightdb.Backend.GetAll(flightdb.Ctx(), q, nil)
	//flightdb.LookupAllKeys(q) //q.KeysOnly().GetAll(db.C, nil)
	keys := []*datastore.Key{}
	for _, keyer := range keyers {
		keys = append(keys, keyer.(*datastore.Key))
	}
	i := LongIterator{
		DB: flightdb,
		Keys: keys,
		err: err,
	}

	return &i
}

func (iter *LongIterator)Iterate() bool {
	iter.val,iter.err = iter.NextWithErr()
	return iter.val != nil
}
func (iter *LongIterator)Flight() *fdb.Flight { return iter.val }
func (iter *LongIterator)Err() error { return iter.err }


func (iter *LongIterator)NextWithErr() (*fdb.Flight, error) {
	if iter.err != nil { return nil, iter.err }

	if iter.i >= len(iter.Keys) {
		return nil,nil // We're all done !
	}
	
	key := iter.Keys[iter.i]
	iter.i++
	
	blob := fdb.IndexedFlightBlob{}
	if err := datastore.Get(iter.DB.Ctx(), key, &blob); err != nil {
		iter.DB.Infof("LongNextWithErr/Next fail: %v", err)
		iter.err = err
		return nil, err
	}

	flight,err := blob.ToFlight(key.Encode())
	if err != nil {
		iter.DB.Infof("LongNextWithErr/blob.ToFlight fail: %v", err)
		return nil, err
	}
	
	return flight, nil
}

func (iter *LongIterator)Next() *fdb.Flight {
	f,_ := iter.NextWithErr()
	return f
}
