package flightdb

import "fmt"

// An Airframe is a thing that flies. We use Icao24 (ADS-B Mode-S) identifiers to identify
// them. Their registration and equipment type values should be constant over many flights.
// If we encounter an airframe using a Type-C callsign (e.g. "AAL353"), we snag the airline
// prefix (e.g. "AAL"), assuming that the aircraft is part of an airline's fleet.
type Airframe struct {
	Icao24         string
	Registration   string
	EquipmentType  string
	CallsignPrefix string
}

func (af Airframe)String() string {
	return fmt.Sprintf("[%s] %10.10s %3.3s %s",
		af.Icao24, af.Registration,	af.CallsignPrefix, af.EquipmentType)
}

func (f *Flight)OverlayAirframe(af Airframe) {
	if f.Airframe.Registration == ""   { f.Airframe.Registration = af.Registration }
	if f.Airframe.EquipmentType == ""  { f.Airframe.EquipmentType = af.EquipmentType }
	if f.Airframe.CallsignPrefix == "" { f.Airframe.CallsignPrefix = af.CallsignPrefix }
}
