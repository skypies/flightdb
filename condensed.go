package flightdb

import(
	"fmt"
	"sort"
	"time"

	"github.com/skypies/util/date"
)

// A CondensedFlight is a very small standalone object that represents
// a flight. The intent is to be able to easily handle large volumes
// of them - in particular, to store a whole day's worth of NorCal in
// <1MB, and so be able to cache them in a single DataStore object.
type CondensedFlight struct {
	IdSpec           string    `json:"id,omitempty"`
	BestFlightNumber string    `json:"f,omitempty"` // IATA, unless it is ICAO
	IcaoId           string    `json:"icao,omitempty"`
	Start            time.Time `json:"s,omitempty"` // f.Times()
	End              time.Time `json:"e,omitempty"`// f.Times()
	
	Tags           []string
	Waypoints        map[string]time.Time `json:"wp,omitempty"`
	Procedure        FlownProcedure       `json:"proc,omitempty"`
}

func (cf CondensedFlight)String() string {
	str := fmt.Sprintf("%s %s {%s} %v %v",
		cf.BestFlightNumber,
		date.InPdt(cf.End).Format("2006/01/02"),
		cf.Procedure,
		cf.WaypointList(),
		cf.Tags)
	return str
}

// Use the waypoint sorting junk from flight.go
func (cf CondensedFlight)WaypointList() []string {
	wptl := []WaypointAndTime{}
	for k,v := range cf.Waypoints { wptl = append(wptl, WaypointAndTime{k,v}) }
	sort.Sort(WaypointAndTimeList(wptl))
	wp := []string{}
	for _,wpt := range wptl { wp = append(wp, wpt.WP) }
	return wp
}


func (f *Flight)Condense() *CondensedFlight {
	s,e := f.Times()

	cf := CondensedFlight{
		IdSpec: f.IdSpec().String(),
		BestFlightNumber: f.BestFlightNumber(),
		IcaoId: f.IcaoId,
		Start: s,
		End: e,
		Tags: f.TagList(),
		Waypoints: map[string]time.Time{},
		Procedure: f.DetermineFlownProcedure(),
	}
	
	for wp,t := range f.Waypoints {
		cf.Waypoints[wp] = t
	}

	return &cf
}

