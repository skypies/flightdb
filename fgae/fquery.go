package fgae

// This package contains flight query builders that sit on top of db/query.go

import(
	"time"
	"github.com/skypies/adsb"
	"github.com/skypies/util/date"

	ds "github.com/skypies/util/dsprovider"
	fdb "github.com/skypies/flightdb"
)

const kFlightKind = "flight" // where should this *really* live ?

type FQuery ds.Query // Create our own type, so we can hang a fluent API off it

func NewFlightQuery() *FQuery { return (*FQuery)(ds.NewQuery(kFlightKind)) }

func (fq *FQuery)Order(str string) *FQuery { return (*FQuery)((*ds.Query)(fq).Order(str)) }
func (fq *FQuery)Limit(val int) *FQuery { return (*FQuery)((*ds.Query)(fq).Limit(val)) }
func (fq *FQuery)Filter(str string, val interface{}) *FQuery {
	return (*FQuery)((*ds.Query)(fq).Filter(str,val))
}


func (q *FQuery)ByTime(t time.Time) *FQuery {
	// Round the time off to the nearest timeslot; and then assert flights posess it
	slots := date.Timeslots(t,t,fdb.TimeslotDuration)
	return q.Filter("Timeslots = ", slots[0])
}

// Note that using this will prevent OrderBy stuff, unless it orders by timeslots ...
func (q *FQuery)ByTimeRange(s,e time.Time) *FQuery {
	// https://cloud.google.com/appengine/docs/go/datastore/queries#Go_Restrictions_on_queries
	// This pair of filters assert that a match must have at least one timeslot that matches
	// both inequalities - i.e. that it has a timeslot within the range.
	slots := date.Timeslots(s,e,fdb.TimeslotDuration)
	return q.
		Filter("Timeslots >= ", slots[0]).
		Filter("Timeslots <= ", slots[len(slots)-1])

	// This would be the wrong way to do it; it asserts that the match has a matching timeslot for
	// every interval in the range - i.e. that the flight lasts the entire time range.
	//for _,slot := range 
	//	q.Query = q.Query.Filter("Timeslots = ", slot) //.Format("1/2/06, 3:30 PM PST"))
	//}
}

func (q *FQuery)ByIcaoId(id adsb.IcaoId) *FQuery {
	return q.Filter("Icao24 = ", string(id))
}
func (q *FQuery)ByCallsign(callsign string) *FQuery {
	return q.Filter("Ident = ", callsign)
}
func (q *FQuery)ByTags(tags []string) *FQuery {
	for _,tag := range tags {
		q.Filter("Tags = ", tag)
	}
	return q
}
// Collapse into tag searching
func (q *FQuery)ByWaypoints(waypoints []string) *FQuery {
	for _,wp := range waypoints {
		q.Filter("Tags = ", fdb.KWaypointTagPrefix + wp)
	}
	return q
}

func (q *FQuery)ByIdSpec(idspec fdb.IdSpec) *FQuery {
	if idspec.Duration != 0 {
		q.ByTimeRange(idspec.Time, idspec.Time.Add(idspec.Duration))
	} else {
		q.ByTime(idspec.Time)
	}

	if idspec.IcaoId != "" {
		q.ByIcaoId(adsb.IcaoId(idspec.IcaoId))
	} else if idspec.Callsign != "" {
		q.ByCallsign(idspec.Callsign)
	} else if idspec.Registration != "" {
		q.ByCallsign(idspec.Registration) // Hmm
	}

	return q
}

// Some canned queries
func QueryForRecentWaypoint(tags []string, waypoints []string, n int) *FQuery {
	return NewFlightQuery().
		ByTags(tags).
		ByWaypoints(waypoints).
		Order("-Timeslots").
		Limit(n)
}
func QueryForRecent(tags []string, n int) *FQuery {
	return NewFlightQuery().
		ByTags(tags).
		Order("-Timeslots").
		Limit(n)
}
func QueryForRecentIcaoId(icaoid string, n int) *FQuery {
	return NewFlightQuery().
		ByIcaoId(adsb.IcaoId(icaoid)).
		Order("-LastUpdate").
		Limit(n)
}

func QueryForTimeRange(tags []string, s,e time.Time) *FQuery {
	return NewFlightQuery().
		ByTags(tags).
		ByTimeRange(s,e)
	//.Order("-LastUpdate")  // No index
}

func QueryForTimeRangeWaypoint(tags []string, waypoints []string, s,e time.Time) *FQuery {
	return NewFlightQuery().
		ByTags(tags).
		ByWaypoints(waypoints).
		ByTimeRange(s,e)
	//.Order("-LastUpdate")  // No index
}
