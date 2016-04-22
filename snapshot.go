package flightdb2

import(
	"fmt"
	"github.com/skypies/geo"
)

type FlightSnapshot struct {
	Flight
	Trackpoint
	
	PrevPos Trackpoint         // Where the aircraft is at this point in time (contains a timestamp)
	NextPos Trackpoint     // For historic results, the trackpoint that follows the time

	Reference geo.Latlong       // If we have a reference point, report where this flight was in
	DistToReferenceKM float64   // relation to it.
	BearingToReference float64
}

func (fs FlightSnapshot)String() string {
	return fmt.Sprintf("%s:%-21.21s @ %s", fs.IcaoId, fs.FullString(), fs.Trackpoint) //date.InPdt(fs.Pos.TimestampUTC), fs.Pos.Latlong)
}
