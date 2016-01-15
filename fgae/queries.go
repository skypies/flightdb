package fgae

import(
	"time"
	"google.golang.org/appengine/datastore"
	"github.com/skypies/adsb"
	"github.com/skypies/util/date"
)

// Wrap the default query, so we can provide a higher level fluent query API
type Query struct {
	*datastore.Query
}

func (db FlightDB)NewQuery() *Query {
	q := Query{
		Query: datastore.NewQuery(kFlightKind),
	}
	return &q
}

func (q *Query)ByTime(t time.Time) *Query {
	return q.ByTimeRange(t,t)
}
func (q *Query)ByTimeRange(s,e time.Time) *Query {
	for _,slot := range date.Timeslots(s,e,kTimeslotDuration) {
		q.Query = q.Query.Filter("Timeslots = ", slot)
	}
	return q
}

func (q *Query)ByIcaoId(id adsb.IcaoId) *Query {
	q.Query = q.Query.Filter("Icao24 = ", string(id))
	return q
}
func (q *Query)ByCallsign(callsign string) *Query {
	q.Query = q.Query.Filter("IcaoFlightNumber = ", callsign)
	return q
}
func (q *Query)ByTags(tags []string) *Query {
	for _,tag := range tags {
		q.Query = q.Query.Filter("Tags = ", tag)
	}
	return q
}

// Some canned queries
func (db FlightDB)QueryForRecent(tags []string, n int) *Query {
	q := db.NewQuery().ByTags(tags)
	q.Query = q.Query.Order("-Timeslots").Limit(n)
	return q
}
