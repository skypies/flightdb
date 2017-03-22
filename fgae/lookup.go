package fgae

import(
	//"google.golang.org/appengine/datastore"

	fdb "github.com/skypies/flightdb"
	"github.com/skypies/flightdb/db"
)

/*

func (db FlightDB)getallByQuery(q *datastore.Query) ([]*fdb.Flight, error) {
	blobs := []fdb.IndexedFlightBlob{}
	db.Debugf(" #--- going to datastore ... %v", *q)
	keys, err := q.GetAll(db.C, &blobs)
	if err != nil {
		return nil, err
	}

	flights := []*fdb.Flight{}
	for i,blob := range blobs {
		if flight,err := blob.ToFlight(keys[i].Encode()); err != nil {
			return nil, err
		} else {
			flights = append(flights, flight)
		}
	}

	db.Debugf(" #--- ... found %d", len(flights))
	return flights,nil
}
*/

func (flightdb FlightDB)getallByQuery(q *db.Query) ([]*fdb.Flight, error) {
	return db.GetAllByQuery(flightdb.Ctx(), flightdb.Backend, q)
}

func (flightdb FlightDB)LookupAllKeys(q *db.Query) ([]db.Keyer, error) {
	backend := flightdb.Backend
	if backend == nil {
		backend = db.AppengineDSProvider{}
	}
	return backend.GetAll(flightdb.Ctx(), q.KeysOnly(), nil)
	//return q.Query.KeysOnly().GetAll(db.C, nil)
}

func (flightdb FlightDB)LookupAll(q *db.Query) ([]*fdb.Flight, error) {
	// Results are not ordered ... for timerange idspecs, would need to sort on Timeslots
	return flightdb.getallByQuery(q) //.Order("-LastUpdate"))
}

func (flightdb FlightDB)LookupMostRecent(q *db.Query) (*fdb.Flight, error) {
	q.Order("-LastUpdate").Limit(1)	
	if flights,err := flightdb.getallByQuery(q); err != nil {
		return nil,err
	} else if len(flights)==0 {
		return nil,nil
	} else {
		return flights[0], nil
	}
}
