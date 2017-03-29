package fgae

import(
	"time"

	"github.com/skypies/adsb"
	"github.com/skypies/pi/airspace"
	"github.com/skypies/geo"

	fdb "github.com/skypies/flightdb"
)

// {{{ snapshot2AirspaceAircraft

func snapshot2AircraftData(fs fdb.FlightSnapshot, id adsb.IcaoId) airspace.AircraftData {	
	msg := adsb.CompositeMsg{
		Msg: adsb.Msg{
			Type: "MSG", // ADSB
			Icao24: id,
			GeneratedTimestampUTC: fs.Trackpoint.TimestampUTC,
			Callsign: fs.Flight.NormalizedCallsignString(),
			Altitude: int64(fs.Trackpoint.Altitude),
			GroundSpeed: int64(fs.Trackpoint.GroundSpeed),
			Track: int64(fs.Trackpoint.Heading),
			Position: fs.Trackpoint.Latlong,
		},
		ReceiverName: fs.Trackpoint.ReceiverName,
	}

	af := fs.Flight.Airframe
	af.Icao24 = string(id)

	if fs.Trackpoint.DataSource == "MLAT" { msg.Type = "MLAT" }
	
	return airspace.AircraftData{
 		Msg: &msg,
		Airframe: af,
		NumMessagesSeen: 1,
		Source: "SkyPi",
	}
}

// }}}

// {{{ db.LookupHistoricalAirspace

func (flightdb FlightDB)LookupHistoricalAirspace(t time.Time, pos geo.Latlong, max int) (airspace.Airspace, error) {
	as := airspace.NewAirspace()
	
	flights,err := flightdb.LookupAll(NewFlightQuery().ByTime(t.UTC()))
	if err != nil { return as, err }
	//db.Infof("LookupHistorical for %s: found %d", t, len(flights))
	for _,f := range flights {
		if fs := f.TakeSnapshotAt(t); fs != nil {
			if !pos.IsNil() {
				fs.LocalizeTo(pos, 0.0)
			}
			icaoId := adsb.IcaoId(fs.IcaoId)
			as.Aircraft[icaoId] = snapshot2AircraftData(*fs, icaoId)
		}
	}
	
	//db.Infof("LookupHistorical output:-\n%s", as)

	return as,nil
}

// }}}


// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
