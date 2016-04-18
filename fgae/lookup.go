package fgae

import(
	"google.golang.org/appengine/datastore"
	fdb "github.com/skypies/flightdb2"
)

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

func (db FlightDB)LookupAllKeys(q *Query) ([]*datastore.Key, error) {
	return q.Query.KeysOnly().GetAll(db.C, nil)
}

func (db FlightDB)LookupAll(q *Query) ([]*fdb.Flight, error) {
	// TODO: I'm not sure we want to secretly add this Order() everywhere ...
	return db.getallByQuery(q.Query.Order("-LastUpdate"))
}

func (db FlightDB)LookupMostRecent(q *Query) (*fdb.Flight, error) {
	q.Query = q.Query.Order("-LastUpdate").Limit(1)	
	if flights,err := db.getallByQuery(q.Query); err != nil {
		return nil,err
	} else if len(flights)==0 {
		return nil,nil
	} else {
		return flights[0], nil
	}
}
