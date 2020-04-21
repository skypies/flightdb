package swim

import(
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/skypies/adsb"
	"github.com/skypies/geo"
)

// See swim-types.go for details on the data itself. This file is just helpers for parsing.

// Expects to be passed a JSON object string.
// Beware ! These strings can be very long, and bufio.Scanner's
// default buffersize will cause it to silently fail.
func Json2Flights(jsonStr string) []Flight {
	sMsg := Ns5MessageCollectionSingleMessage{}
	mMsg := Ns5MessageCollectionMultiMessage{}

	// We need to try both single and multi, as they are incompatible in what type they
	// assign to the `message` JSON field. Do multi first, as they're most common.
	if err := json.Unmarshal([]byte(jsonStr), &mMsg); err == nil {
		flights := []Flight{}
		for _,m := range mMsg.Ns5MessageCollection.Message {
			flights = append(flights, m.Flight)
		}
		return flights

	} else if err2 := json.Unmarshal([]byte(jsonStr), &sMsg); err2 == nil {
		return []Flight{sMsg.Ns5MessageCollection.Message.Flight}

	} else {
		// If you start to care about the per-field error handling,
		// https://www.alexedwards.net/blog/how-to-properly-parse-a-json-request-body
		// https://golang.org/src/encoding/json/decode.go
		// fmt.Errorf("ParseFlightMessage: not multi or single: <%v>, <%v>", err, err2)
		return []Flight{}
	}
}

// Convert to an ADSB composite msg
func (f Flight)AsAdsb() adsb.CompositeMsg {
	// Not yet sure how to get this, or if it is even possible.
	icaoId := fmt.Sprintf("SWM%.0f", f.FlightIdentification.ComputerId)

	t,_ := time.Parse(time.RFC3339, f.Timestamp)	

	vals := strings.Split(f.EnRoute.Position.Position.Location.Pos, " ")
	lat,_ := strconv.ParseFloat(vals[0], 64)
	long,_ := strconv.ParseFloat(vals[1], 64)

	// Should really assert that the units are KNOTS and FEET
	alt          := int64(f.EnRoute.Position.Altitude.Content)
	groundspeed  := int64(f.EnRoute.Position.ActualSpeed.Surveillance.Content)
	trackX       := f.EnRoute.Position.TrackVelocity.X.Content
	trackY       := f.EnRoute.Position.TrackVelocity.Y.Content

	trackRadians := math.Atan2(trackY, trackX)
	trackDegrees := int64(trackRadians * (180.0 / math.Pi))
	// Atan2(y,x) has 0deg as positive X axis, and goes anti-clockwise.
	// We want an angle with 0deg as positive Y axis, going clockwise.
	trackDegrees = ((90 - trackDegrees) + 360) % 360
	
	//fmt.Printf("%v\nOho: alt=%d, track=%d, speed=%d\n", f, alt, trackDegrees, groundspeed)
	
	msg := adsb.CompositeMsg{
		ReceiverName: "SWIM",

		Msg: adsb.Msg{
			Type: "SWIM",
			Icao24: adsb.IcaoId(icaoId),
			Callsign: f.FlightIdentification.AircraftIdentification,
			Position: geo.Latlong{Lat:lat, Long:long},
			Altitude: alt,
			GroundSpeed: groundspeed,
			Track: trackDegrees,
			GeneratedTimestampUTC: t,
			LoggedTimestampUTC: time.Now().UTC(),
		},

	}

	msg.SetHasGroundSpeed()
	msg.SetHasTrack()
	msg.SetHasPosition()
	
	return msg
}
