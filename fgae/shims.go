package fgae

import(
	fdb "github.com/skypies/flightdb"
	"github.com/skypies/flightdb/db"
)

// A few shims, porting old fgae.FlightDB stuff over the top of new db.DatastoreProvider stuff

func (flightdb  FlightDB)NewQuery() *db.FQuery {
	return db.NewFlightQuery()
}

func (flightdb *FlightDB)PersistFlight(f *fdb.Flight) error {
	return db.PersistFlight(flightdb.Ctx(), flightdb.Backend, f)
}

func (flightdb FlightDB)LookupAll(q *db.FQuery) ([]*fdb.Flight, error) {
	// Results are not ordered ... for timerange idspecs, would need to sort on Timeslots
	return db.GetAllByQuery(flightdb.Ctx(), flightdb.Backend, q)
}

func (flightdb FlightDB)LookupMostRecent(q *db.FQuery) (*fdb.Flight, error) {
	// Adding the ordering will break some queries, due to lack of indices
	return db.GetFirstByQuery(flightdb.Ctx(), flightdb.Backend, q.Order("-LastUpdate"))
}
