package flightdb

import (
	"fmt"
	"time"

	"github.com/skypies/adsb"
	"github.com/skypies/geo"
)

// Trackpoint is a data point that locates an aircraft in space and time, etc
type Trackpoint struct {
	DataSource   string    // What kind of trackpoint is this; flightaware radar, local ADSB, etc

	//DataSystem   // embedded string
	//DataProvider // embedded string

	ReceiverName string    // For local ADSB

	TimestampUTC time.Time // Always in UTC, to make life SIMPLE

	geo.Latlong            // Embedded type, so we can call all the geo stuff directly on trackpoints

	Altitude     float64   // This is pressure altitude (i.e. altitude in a standard atmosphere)
	GroundSpeed  float64   // In knots
	Heading      float64   // [0.0, 360.0) degrees. Direction plane is pointing in. Mag or real north?
	VerticalRate float64   // In feet per minute (multiples of 64)
	Squawk       string    // Generally, a string of four digits.

	//// None of the fields below are stored in the database
	
	// These two are transient fields, populated during analysis, and displayed on the map view
	AnalysisAnnotation string `datastore:"-" json:"-"`
	AnalysisDisplay    AnalysisDisplayEnum `datastore:"-" json:"-"`

	// These fields are derived
	IndicatedAltitude         float64 `datastore:"-" json:"-"` // Corrected for local air pressure
	DistanceTravelledKM       float64 `datastore:"-" json:"-"` // Totted up from point to point
	GroundAccelerationKPS     float64 `datastore:"-" json:"-"` // In knots per second
	VerticalSpeedFPM          float64 `datastore:"-" json:"-"` // Feet per minute (~== VerticalRate)
	VerticalAccelerationFPMPS float64 `datastore:"-" json:"-"` // In (feet per minute) per second
	AngleOfInclination        float64 `datastore:"-" json:"-"` // In degrees. +ve means climbing
	
	// Populated just in first trackpoint, to hold transient notes for the whole track.
	Notes                     string  `datastore:"-" json:"-"`
}

type InterpolatedTrackpoint struct {
	Trackpoint   // Embedded struct; only the interpolatable bits will be populated

	Pre, Post *Trackpoint // The points we were interpolated from
	Ratio      float64    // How far we were inbetween them

	Ref        geo.Latlong      // The point we were in reference to
	Line       geo.LatlongLine  // The line that connects the ref point to the line {pre->post}
	Perp       geo.LatlongLine
}

type AnalysisDisplayEnum int
const(
	AnalysisDisplayDefault AnalysisDisplayEnum = iota
	AnalysisDisplayOmit
	AnalysisDisplayHighlight  // "red-large"
)

// {{{ tp.ShortString

func (tp Trackpoint)ShortString() string {
	str := fmt.Sprintf("[%s] %s %.0fft (%.0f), %.0fkts, %.0fdeg", tp.TimestampUTC, tp.Latlong,
		tp.Altitude, tp.IndicatedAltitude, tp.GroundSpeed, tp.Heading)
	if tp.DistanceTravelledKM > 0.0 {
		str += fmt.Sprintf(" [path:%.3fKM]", tp.DistanceTravelledKM)
	}
	return str
}

// }}}
// {{{ tp.String

func (tp Trackpoint)String() string {
	str := fmt.Sprintf("[%s] %s %.0fft, %.0fkts, %.0fdeg", tp.TimestampUTC, tp.Latlong,
		tp.Altitude, tp.GroundSpeed, tp.Heading)

	if tp.DistanceTravelledKM > 0.0 {
		str += fmt.Sprintf("\n* Travelled Dist: %.3f KM\n"+
			"* Vertical rates: computed: %.0f feet/min; received: %.0f feet/min\n"+
			"* Acceleration: horiz: %.2f knots/sec, vert %.0f feetpermin/sec",
			tp.DistanceTravelledKM,
			tp.VerticalSpeedFPM, tp.VerticalRate,
			tp.GroundAccelerationKPS, tp.VerticalAccelerationFPMPS)
	}
	
	return str
}

// }}}
// {{{ tp.ToJSString

func (tp Trackpoint)ToJSString() string {
	return fmt.Sprintf("source:%q, receiver:%q, pos:{lat:%.6f,lng:%.6f}, "+
		"alt:%.0f, speed:%.0f, track:%.0f, vert:%.0f, t:\"%s\"",
		tp.DataSource, tp.ReceiverName, tp.Lat, tp.Long,
		tp.Altitude, tp.GroundSpeed, tp.Heading, tp.VerticalRate, tp.TimestampUTC)
}

// }}}
// {{{ tp.LongSource

func (tp Trackpoint)LongSource() string {
	switch tp.DataSource {
	case "":      return "(none specified)"
	case "FA:TZ": return "FlightAware, Radar (TZ)"
	case "FA:TA": return "FlightAware, ADS-B Mode-ES (TA)"
	case "ADSB":  return "Private receiver, ADS-B Mode-ES ("+tp.ReceiverName+")"
	case "MLAT":  return "MLAT ("+tp.ReceiverName+")"
	}
	return tp.DataSource
}

// }}}

// {{{ TrackpointFromADSB

func TrackpointFromADSB(m *adsb.CompositeMsg) Trackpoint {
	tp := Trackpoint{
		//DataSystem: "ADSB",
		DataSource: "ADSB",
		ReceiverName: m.ReceiverName,
		TimestampUTC: m.GeneratedTimestampUTC,
		Latlong: m.Position,
		Altitude: float64(m.Altitude),
		GroundSpeed: float64(m.GroundSpeed),
		Heading: float64(m.Track),
		VerticalRate: float64(m.VerticalRate),
		Squawk: m.Squawk,
	}

	// Need to really clean all this up
	if m.IsMLAT() {
		tp.DataSource = "MLAT"
	}

	return tp
}

// }}}
// {{{ TrackpointFromAverage

func TrackpointFromAverage(in []Trackpoint) Trackpoint {
	if len(in) == 0 { return Trackpoint{} }

	out := in[0] // Initialize, to get all the non-numeric stuff (and timestamp of in[0])

	for _,tp := range in {
		out.Altitude     += tp.Altitude
		out.GroundSpeed  += tp.GroundSpeed
		out.Heading      += tp.Heading
		out.VerticalRate += tp.VerticalRate

		out.IndicatedAltitude         += tp.IndicatedAltitude
		out.DistanceTravelledKM       += tp.DistanceTravelledKM
		out.GroundAccelerationKPS     += tp.GroundAccelerationKPS
		out.VerticalSpeedFPM          += tp.VerticalSpeedFPM
		out.VerticalAccelerationFPMPS += tp.VerticalAccelerationFPMPS
	}

	out.Altitude     /= float64(len(in))
	out.GroundSpeed  /= float64(len(in))
	out.Heading      /= float64(len(in))
	out.VerticalRate /= float64(len(in))

	out.IndicatedAltitude         /= float64(len(in))
	out.DistanceTravelledKM       /= float64(len(in))
	out.GroundAccelerationKPS     /= float64(len(in))
	out.VerticalSpeedFPM          /= float64(len(in))
	out.VerticalAccelerationFPMPS /= float64(len(in))

	out.Notes += fmt.Sprintf("(avg, from %d points)", len(in))
	
	return out
}

// }}}

// {{{ tp.InterpolateTo

func interpolateFloat64(from, to, ratio float64) float64 {
	return from + (to-from)*ratio
}

func interpolateTime(from, to time.Time, ratio float64) time.Time {
	d1 := to.Sub(from)
	nanosToAdd := ratio * float64(d1.Nanoseconds())
	d2 := time.Nanosecond * time.Duration(nanosToAdd)
	d3 := time.Second * time.Duration(d2.Seconds()) // Round down to second precision
	return from.Add(d3)
}

func (from Trackpoint)InterpolateTo(to Trackpoint, ratio float64) InterpolatedTrackpoint {
	itp := InterpolatedTrackpoint{
		Pre: &from,
		Post: &to,
		Ratio: ratio,
		Trackpoint: Trackpoint{
			GroundSpeed: interpolateFloat64(from.GroundSpeed, to.GroundSpeed, ratio),
			VerticalRate: interpolateFloat64(from.VerticalRate, to.VerticalRate, ratio),
			Altitude: interpolateFloat64(from.Altitude, to.Altitude, ratio),
			Heading: geo.InterpolateHeading(from.Heading, to.Heading, ratio),
			Latlong: from.Latlong.InterpolateTo(to.Latlong, ratio),
			TimestampUTC: interpolateTime(from.TimestampUTC, to.TimestampUTC, ratio),

			// Also interpolate the synthetic fields
			DistanceTravelledKM: interpolateFloat64(from.DistanceTravelledKM, to.DistanceTravelledKM, ratio),
			GroundAccelerationKPS: interpolateFloat64(from.GroundAccelerationKPS, to.GroundAccelerationKPS, ratio),
			VerticalSpeedFPM: interpolateFloat64(from.VerticalSpeedFPM, to.VerticalSpeedFPM, ratio),
			VerticalAccelerationFPMPS: interpolateFloat64(from.VerticalAccelerationFPMPS, to.VerticalAccelerationFPMPS, ratio),
		},
	}
	return itp
}

// }}}
// {{{ tp.RepositionByTime

// RepositionByTime returns a trackpoint that has been repositioned, assuming it was
// travelling at constant velocity. The duration passed in determines how far to move in either
// direction.
func (s Trackpoint)RepositionByTime(d time.Duration) Trackpoint {
	e := s

	hDistMeters := geo.NMph2mps(s.GroundSpeed) * d.Seconds() // 1 knot == 1 NM/hour

	e.Latlong = s.Latlong.MoveKM(s.Heading, hDistMeters/1000.0)
	e.Altitude += (float64(s.VerticalRate) / 60.0) * d.Seconds() // vert rat in feet per minute
	e.TimestampUTC = s.TimestampUTC.Add(d)
	
	return e
}

// }}}


// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
