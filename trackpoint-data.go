package flightdb

/* Cleanups ...

// The creation flow
consolidator.go:  flushTrackToDatastore

trackfragment.go: MessagesToTrackTragment
trackfragment.go: TrackName
trackpoint.go:    TrackpointFromADSB

fgae/update.go:   AddTrackFragment
flight.go:        NewFlightFromTrackFragmenet

// The rendering flow
mapshapes.go:     TrackToMapPoints
trackpoint.go     LongSource


 */

/*

1. Identify all callsites of tp.DataSource; move to GetData{System|Provider} or stringifiers
2. Some are to do with restricting operations to just locally received data, or to ADSB
3. Prob need a more formal approach to 'trackspec', PreferredTrack et al.
4. What to do with track keynames ? Make them irrelevant - *always* pluck a track via a
    trackspec ??

Intermediate goal:
* no piece of code direactly accesses tp.DataSource, except this file (and CTORs)

5. Add the Data{System|Provider} fields to fdb.Trackpoint
6. Update CTORs to specify these fields, and not tp.DataSource
7. Leave tp.DataSource as legacy / deprecated.

End state
* Everything uses these semantically clean methods, even on old crappy data

*/

type DataProvider string
const(
	DPUnknown DataProvider = "?"
	DPSkypi   DataProvider = "SkyPi"  // AKA 'ADSB'
	DPFA      DataProvider = "FA"
	DPFR24    DataProvider = "fr24"
	DPFAAFOIA DataProvider = "FOIA"
)

type DataSystem string
const(
	DSUnknown         DataSystem = "?"
	DSADSB            DataSystem = "A"   // AKA TA
	DSMLAT            DataSystem = "M"
	DSRadar           DataSystem = "Z"   // AKA TZ
	DSCorrectedRadar  DataSystem = "F"
)

// These two functions serve as a backwards-compatibility layer
func (tp Trackpoint)GetDataSystem() DataSystem {
//	if tp.DataSystem != "" {
//		return tp.DataSystem
//	}
	switch tp.DataSource {
	case "ADSB":  return DSADSB
	case "fr24":  return DSUnknown
	case "FA:TZ": return DSRadar
	case "FA:TA": return DSADSB
	case "FOIA":  return DSCorrectedRadar
	default:      return DSUnknown
	}
}

func (tp Trackpoint)GetDataProvider() DataProvider {
//	if tp.DataProvider != "" {
//		return tp.DataProvider
//	}
	switch tp.DataSource {
	case "ADSB":  return DPSkypi
	case "fr24":  return DPFR24
	case "FA:TZ": return DPFA
	case "FA:TA": return DPFA
	case "FOIA":  return DPFAAFOIA
	default:      return DPUnknown
	}
}

//func (tp Trackpoint)TrackKey() string {
//	name := tp.GetDataProvider().String()
//	return name
//}


func (dp DataProvider)LongString() string {
	switch dp {
	case DPSkypi:   return "Private ADS-B receiver"
	case DPFA:      return "FlightAware"
	case DPFR24:    return "Flightradar24"
	case DPFAAFOIA: return "FAA FOIA"
	case DPUnknown: return "(unknown)"
	default:        return "(unknown2)"
	}
}
func (ds DataSystem)LongString() string {
	switch ds {
	case DSADSB:           return "ADS-B, Mode-ES"
	case DSMLAT:           return "Multilateration"
	case DSRadar:          return "Radar"
	case DSCorrectedRadar: return "Corrected Radar"
	case DSUnknown:        return "(unknown)"
	default:               return "(unknown2)"
	}
}
