package flightdb2

import(
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// For scheduled flights, get what data we can about it
type Schedule struct {
	Number  int64
	IATA    string // 2 chars
	ICAO    string // 3 chars

	PlannedDepartureUTC time.Time
	PlannedArrivalUTC time.Time
	//ActualDepartureUTC time.Time
	//ActualArrivalUTC time.Time
	ArrivalLocationName string   // For extra credit ;)
	DepartureLocationName string // For extra credit ;)

	Origin string
	Destination string
}
func (s Schedule)IcaoFlight() string {
	if s.ICAO != "" { return fmt.Sprintf("%s%d", s.ICAO, s.Number) } else { return "" }
}
func (s Schedule)IataFlight() string {
	if s.IATA != "" { return fmt.Sprintf("%s%d", s.IATA, s.Number) } else { return "" }
}

type Identity struct {
	IcaoId          string   // hex string (cf. adsb.IcaoId)

	// For callsigns, consider f.NormalizedCallsignString() instead, or the stuff in callsign.go
	Callsign        string   // This is raw, as found in ADS-B transmission.

	Schedule // embedded; not always populated

	ForeignKeys     map[string]string // fr24, fa, fdbV1(?), etc
}

func (id Identity)FullString() string {
	return fmt.Sprintf("{%s|%s}%d[%s-%s] [%s]c:%s",
		id.Schedule.IATA, id.Schedule.ICAO, id.Schedule.Number, id.Origin, id.Destination,
		id.IcaoId, id.Callsign)
}

func (id Identity)IdentString() string {
	str := id.IcaoFlight()
	if str == "" {
		str = id.IataFlight()
	}
	if str == "" {
		str = id.Callsign
	}
	
	str += fmt.Sprintf(" [%s]", id.IcaoId)
	return str
}

func (f Flight)IdentString() string { return f.OldIdentifier() }
func (f Flight)OldIdentifier() string {
	str := f.IcaoFlight()
	if str == "" {
		str = f.IataFlight()
	}
	if str == "" {
		str = f.Callsign
	}

	str += "["
	if !f.Schedule.PlannedDepartureUTC.IsZero() {
		str += f.Schedule.PlannedDepartureUTC.Format("Jan02:")
	} else if len(f.Tracks) > 0 {
		s,_ := f.Times()
		str += s.Format("Jan02:")
	}
	if f.Origin != "" {
		str += fmt.Sprintf("%s-%s", f.Origin, f.Destination)
	}
	str += "]"
	
	return str
}

// Also: faUrl := fmt.Sprintf("http://flightaware.com/live/flight/%s", m.Callsign)
func (f Flight)TrackUrl() string {
	u := fmt.Sprintf("/fdb/tracks?idspec=%s", f.IdSpec())
	return u
}


func (id *Identity)ParseIata(s string) error {
	iata := regexp.MustCompile("^([A-Z][0-9A-Z])([0-9]{1,4})$").FindStringSubmatch(s)
	if iata != nil && len(iata)==3 {
		id.Schedule.Number,_ = strconv.ParseInt(iata[2], 10, 64) // no errors here :)
		id.Schedule.IATA = iata[1]
		return nil
	}
	return fmt.Errorf("ParseIata: could not parse '%s'", s)
}


// An identifier specifier - something we receive that should uniquely identify a flight
// Can be {airframe+time}, or maybe later we'll support flight designators.
type IdSpec struct {
	IcaoId       string
	Registration string
	Callsign     string
	time.Time    // embed
}
func (idspec IdSpec)String() string {
	return fmt.Sprintf("%s@%s", idspec.IcaoId,
		idspec.Time.Format("2006.01.02-15:04:05-0700"))
}

// Input formats:
//  A23A23@14123123123123  (IcaoId at an epoch time)
func NewIdSpec(idspec string) (IdSpec,error) {
	bits := strings.Split(idspec, "@")
	if len(bits) != 2 {
		return IdSpec{}, fmt.Errorf("IdSpec '%s' did not match <airframe>@<epoch>", idspec)
	}

	epochInt,err := strconv.ParseInt(bits[1], 10, 64)
	if err != nil {
		return IdSpec{}, fmt.Errorf("IdSpec '%s' did not have parseable int after @: %v", idspec, err)
	}

	// Should check for IcaoId vs. Registration or callsign; for now, just assume IcaoId.
	return IdSpec{
		IcaoId: bits[0],
		Time: time.Unix(epochInt, 0),
	}, nil
}

func (id Identity)IsScheduled() bool {
	return (id.Number > 0 && (id.IATA != "" || id.ICAO != ""))
}


/* Some notes on identifiers for aircraft

# ADSB Mode-[E]S Identifiers (Icao24)

These are six digit hex codes, handed out to aircraft. Most aircraft
using ADS-B have this is a (semi?) permanent 'airframe' ID, but some
aircraft spoof it. And some have two transponders or something.

# Aircraft registration, e.g. N12312, HP-1846CMP.

Codes assigned by governments, to physical aircraft. Many private
aircraft will use this as their callsign. FlightAware uses this as
their primary search ID.

# Callsigns : see callsigns.go

# Foreign identifiers

## Flightaware

The FA API uses an 'ident' for initial lookup, which can be one of three things:
 * ICAO flightnumber (3+4)
 * Registration / tailnumber
 * their own 'faFlightId'

 */
