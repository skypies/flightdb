package flightdb2

import (
	"fmt"
	"time"

	"github.com/skypies/adsb"
	"github.com/skypies/geo"
)

// Trackpoint is a data point that locates an aircraft in space and time, etc
type Trackpoint struct {
	DataSource   string    // What kind of trackpoint is this; flightaware radar, local ADSB, etc
	ReceiverName string    // For local ADSB
	TimestampUTC time.Time // Always in UTC, to make life SIMPLE

	geo.Latlong            // Embedded type, so we can call all the geo stuff directly on trackpoints

	Altitude     float64   // This is pressure altitude (i.e. altitude in a standard atmosphere)
	GroundSpeed  float64   // In knots
	Heading      float64   // [0.0, 360.0) degrees. Direction plane is pointing in. Mag or real north?
	VerticalRate float64   // In feet per minute (multiples of 64)
	Squawk       string    // Generally, a string of four digits.

	// These two are transient fields, populated during analysis, and displayed on the map view
	AnalysisAnnotation string `datastore:"-" json:"-"`
	AnalysisMapIcon    string `datastore:"-" json:"-"`

	// These fields are derived
	IndicatedAltitude float64 `datastore:"-" json:"-"` // Corrected for local air pressure
}

type InterpolatedTrackpoint struct {
	Trackpoint   // Embedded struct; only the interpolatable bits will be populated

	Pre, Post *Trackpoint // The points we were interpolated from
	Ratio      float64    // How far we were inbetween them

	Ref        geo.Latlong      // The point we were in reference to
	Line       geo.LatlongLine  // The line that connects the ref point to the line {pre->post}
	Perp       geo.LatlongLine
}

func (tp Trackpoint)String() string {
	return fmt.Sprintf("[%s] %s %.0fft, %.0fkts, %.0fdeg", tp.TimestampUTC, tp.Latlong,
		tp.Altitude, tp.GroundSpeed, tp.Heading)
}

func (tp Trackpoint)ToJSString() string {
	return fmt.Sprintf("source:%q, receiver:%q, pos:{lat:%.6f,lng:%.6f}, "+
		"alt:%.0f, speed:%.0f, track:%.0f, vert:%.0f, t:\"%s\"",
		tp.DataSource, tp.ReceiverName, tp.Lat, tp.Long,
		tp.Altitude, tp.GroundSpeed, tp.Heading, tp.VerticalRate, tp.TimestampUTC)
}

func (tp Trackpoint)LongSource() string {
	switch tp.DataSource {
	case "":      return "(none specified)"
	case "FA:TZ": return "FlightAware, Radar (TZ)"
	case "FA:TA": return "FlightAware, ADS-B Mode-ES (TA)"
	case "ADSB":  return "Private receiver, ADS-B Mode-ES ("+tp.ReceiverName+")"
	}
	return tp.DataSource
}

func TrackpointFromADSB(m *adsb.CompositeMsg) Trackpoint {
	return Trackpoint{
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
		},
	}
	return itp
}

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
