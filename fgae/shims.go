package fgae

import(
	fdb "github.com/skypies/flightdb"
	"github.com/skypies/flightdb/db"
)

// A few shims, porting old fgae.FlightDB stuff over the top of new db.DatastoreProvider stuff

func (flightdb  FlightDB)NewQuery() *db.Query {
	return db.NewFlightQuery()
}

func (flightdb *FlightDB)PersistFlight(f *fdb.Flight) error {
	return db.PersistFlight(flightdb.Ctx(), flightdb.Backend, f)
}

func (flightdb FlightDB)LookupAll(q *db.Query) ([]*fdb.Flight, error) {
	// Results are not ordered ... for timerange idspecs, would need to sort on Timeslots
	return db.GetAllByQuery(flightdb.Ctx(), flightdb.Backend, q)
}

func (flightdb FlightDB)LookupMostRecent(q *db.Query) (*fdb.Flight, error) {
	q.Order("-LastUpdate").Limit(1)	
	if flights,err := db.GetAllByQuery(flightdb.Ctx(), flightdb.Backend, q); err != nil {
		return nil,err
	} else if len(flights)==0 {
		return nil,nil
	} else {
		return flights[0], nil
	}
}
