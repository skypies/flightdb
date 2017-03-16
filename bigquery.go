package flightdb

import (
	"fmt"
	"sort"
	"time"

	"github.com/skypies/util/date"
)

// FlightForBigQuery is a represenation of a Flight that is slightly denormalized, with
// a track summary instead of a track. It is designed for import into BigQuery, for analysis.
// It has a *lot* in common with CondensedFlight ... perhaps they should be combined ?
type FlightForBigQuery struct {
	FdbId          string // ID back into the flight database

	ModeS          string
	Registration   string
	Equip          string // e.g. B744, A320 etc

	Start,End      time.Time // start and end of track data points
	DatePST        string // Bleargh. This is somewhat approximate.
	TrackSources []string
	Tags         []string

	Waypoint     []WaypointForBigQuery  // Not 'Waypoints', so that the SQL reads more naturally
	Procedure    []FlownProcedure

	// These fields only defined if we have schedule data for the flight
	FlightNumber   string // IATA scheduled flight number
	FlightKey      string // A {flightnumber+date} value; can be used to join against complaints
	Airline        string // IATA airline code, if known
	Callsign       string
	Orig,Dest      string // airport codes
}

type WaypointForBigQuery struct {
	Name string
	Time time.Time
}

func (fbq FlightForBigQuery)String() string {
	proc := ""
	if len(fbq.Procedure) > 0 { proc = fmt.Sprintf("%v", fbq.Procedure[0]) }
	str := fmt.Sprintf("%s %s {%s} %v %v",
		fbq.FlightNumber,
		date.InPdt(fbq.End).Format("2006/01/02"),
		proc,
		fbq.Waypoint,
		fbq.Tags)
	return str
}

func (f *Flight)ForBigQuery() *FlightForBigQuery {
	s,e := f.Times()

	// We need to pick a 'date' for this flight; but we don't have schedule data.
	// Pick the midpoint of the time range we knew about this flight.
	mid := s.Add(e.Sub(s) / 2)
	
	fbq := FlightForBigQuery{
		FdbId: f.IdSpec().String(),
		ModeS: f.IcaoId,
		Registration: f.Registration,
		Equip: f.EquipmentType,

		Start: s,
		End: e,
		DatePST: date.InPdt(mid).Format("2006-01-02"), // Use the same format as BQ's DATE() function
		TrackSources: f.ListTracks(),
		Tags: f.TagList(),

		Waypoint: []WaypointForBigQuery{},
		Procedure: f.DetermineFlownProcedures(),
		
		FlightNumber: f.IataFlight(),
		FlightKey: fmt.Sprintf("%s-%s", f.IataFlight(), date.InPdt(mid).Format("20060102")),
		Airline: f.Schedule.IATA,
		Callsign: f.Callsign,
		Orig: f.Schedule.Origin,
		Dest: f.Schedule.Destination,
	}
	
	wptl := []WaypointAndTime{}
	for k,v := range f.Waypoints { wptl = append(wptl, WaypointAndTime{k,v}) }
	sort.Sort(WaypointAndTimeList(wptl))
	for _,wpt := range wptl {
		fbq.Waypoint = append(fbq.Waypoint, WaypointForBigQuery{wpt.WP, wpt.Time})
	}
	
	return &fbq
}
