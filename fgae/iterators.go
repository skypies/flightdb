package fgae

import(
	"google.golang.org/appengine/datastore"
	fdb "github.com/skypies/flightdb2"
)

type Iterator struct {
	DB *FlightDB
	I *datastore.Iterator

	val *fdb.Flight
	err error
}

func (db *FlightDB)NewIterator(q *Query) *Iterator {
	i := Iterator{
		DB: db,
		I: q.Run(db.C),
	}
	return &i
}

/*

 iter := db.NewIterator(q)
 for iter.Iterate() {
   if iter.Err() != nil { break }
   f := iter.Flight()
   ...
 }

*/

func (iter *Iterator)Iterate() bool {
	iter.val,iter.err = iter.NextWithErr()
	return iter.val != nil
}
func (iter *Iterator)Flight() *fdb.Flight { return iter.val }
func (iter *Iterator)Err() error { return iter.err }

func (iter *Iterator)NextWithErr() (*fdb.Flight, error) {
	iter.DB.Debugf(" #--- iterating on datastore ... %v", *iter)

	blob := fdb.IndexedFlightBlob{}
	key,err := iter.I.Next(&blob)
	if err == datastore.Done {
		return nil,nil // We're all done
	} else if err != nil {
		iter.DB.Infof("NextWithErr/Next fail: %v", err)
		return nil, err
	}

	flight,err := blob.ToFlight(key.Encode())
	if err != nil {
		iter.DB.Infof("NextWithErr/blob.ToFlight fail: %v", err)
		return nil, err
	}
	
	return flight, nil
}

func (iter *Iterator)Next() *fdb.Flight {
	f,_ := iter.NextWithErr()
	return f
}


//// Some canned iterators
