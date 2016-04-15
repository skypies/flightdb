package fgae

import(
	"time"
	"google.golang.org/appengine/datastore"
	"github.com/skypies/adsb"
	"github.com/skypies/util/date"

	fdb "github.com/skypies/flightdb2"
)

// Wrap the default query, so we can provide a higher level fluent query API
type Query struct {
	*datastore.Query
	DB FlightDB
}

func (db FlightDB)NewQuery() *Query {
	q := Query{
		Query: datastore.NewQuery(kFlightKind),
		DB: db,
	}
	return &q
}

func (q *Query)ByTime(t time.Time) *Query {
	// Round the time off to the nearest timeslot; and then assert flights posess it
	slots := date.Timeslots(t,t,kTimeslotDuration)
	q.Query = q.Query.Filter("Timeslots = ", slots[0])
	return q
}

// Note that using this will prevent OrderBy stuff, unless it orders by timeslots ...
func (q *Query)ByTimeRange(s,e time.Time) *Query {
	// https://cloud.google.com/appengine/docs/go/datastore/queries#Go_Restrictions_on_queries
	// This pair of filters assert that a match must have at least one timeslot that matches
	// both inequalities - i.e. that it has a timeslot within the range.
	slots := date.Timeslots(s,e,kTimeslotDuration)
	q.Query = q.Query.
		Filter("Timeslots >= ", slots[0]).
		Filter("Timeslots <= ", slots[len(slots)-1])

	// This is the wrong way to do it; it asserts that the match has a matching timeslot for
	// every interval in the range - i.e. that the flight lasts the entire time range.
	//for _,slot := range 
	//	q.Query = q.Query.Filter("Timeslots = ", slot) //.Format("1/2/06, 3:30 PM PST"))
	//}
	return q
}

func (q *Query)ByIcaoId(id adsb.IcaoId) *Query {
	q.Query = q.Query.Filter("Icao24 = ", string(id))
	return q
}
func (q *Query)ByCallsign(callsign string) *Query {
	q.Query = q.Query.Filter("Ident = ", callsign)
	return q
}
func (q *Query)ByTags(tags []string) *Query {
	for _,tag := range tags {
		q.Query = q.Query.Filter("Tags = ", tag)
	}
	return q
}
// Collapse into tag searching
func (q *Query)ByWaypoints(waypoints []string) *Query {
	for _,wp := range waypoints {
		q.Query = q.Query.Filter("Tags = ", fdb.KWaypointTagPrefix + wp)
	}
	return q
}

func (q *Query)ByIdSpec(idspec fdb.IdSpec) *Query {
	q = q.ByTime(idspec.Time)

	if idspec.IcaoId != "" {
		q = q.ByIcaoId(adsb.IcaoId(idspec.IcaoId))
	} else if idspec.Callsign != "" {
		q = q.ByCallsign(idspec.Callsign)
	} else if idspec.Registration != "" {
		q = q.ByCallsign(idspec.Registration) // Hmm
	}

	return q
}

// Some canned queries
func (db FlightDB)QueryForRecentWaypoint(tags []string, waypoints []string, n int) *Query {
	q := db.NewQuery().ByTags(tags).ByWaypoints(waypoints)
	q.Query = q.Query.Order("-Timeslots").Limit(n)
	return q
}
func (db FlightDB)QueryForRecent(tags []string, n int) *Query {
	q := db.NewQuery().ByTags(tags)
	q.Query = q.Query.Order("-Timeslots").Limit(n)
	return q
}
func (db FlightDB)QueryForRecentIcaoId(icaoid string, n int) *Query {
	q := db.NewQuery().ByIcaoId(adsb.IcaoId(icaoid))
	q.Query = q.Query.Order("-LastUpdate").Limit(n)
	return q
}

// This one is broke
func (db FlightDB)QueryForTimeRange(tags []string, s,e time.Time) *Query {
	q := db.NewQuery().ByTags(tags).ByTimeRange(s,e)
	//q.Query = q.Query.Order("-LastUpdate")  // No index
	return q
}


func (db FlightDB)QueryForTimeRangeWaypoint(tags []string, waypoints []string, s,e time.Time) *Query {
	q := db.NewQuery().ByTags(tags).ByWaypoints(waypoints).ByTimeRange(s,e)
	//q.Query = q.Query.Order("-LastUpdate")  // No index
	return q
}
