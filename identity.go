package flightdb2

import(
	"fmt"
	"regexp"
	"strconv"
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
	Callsign        string   // up to 8 chars; can be many things, including blank
	Registration    string

	Schedule // embedded; not always populated

	ForeignKeys     map[string]string // fr24, fa, fdbV1(?), etc
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
	u := fmt.Sprintf("/fdb/tracks?icaoid=%s", f.IcaoId)
	times := f.Timeslots(time.Minute * 30)  // ARGH
	if len(times)>0 {
		u += fmt.Sprintf("&t=%d", times[0].Unix())
	}
	return u
}

// https://en.wikipedia.org/wiki/Airline_codes#Call_signs_.28flight_identification_or_flight_ID.29
type CallsignType int
const(
	Undefined         CallsignType = iota
	JunkCallsign
	Registration      // Callsign Type A
                    // Callsign Type B - we never see it, and it's useless anyway
	IcaoFlightNumber  // Callsign Type C
	BareFlightNumber  // Some airlines omit the Icao carrier code, grr
	// EquipType      // We sometime see this, but it's useless
)
func (id *Identity)ParseCallsign() CallsignType {
	// Registration (e.g. N23ST). Wikipedia:
	// An N-number may only consist of one to five characters, must
	// start with a digit other than zero, and cannot end in a run of
	// more than two letters. In addition, N-numbers may not contain the
	// letters I or O
	reg := regexp.MustCompile("^(N[1-9][0-9A-HJ-NP-Z]{0,4})$").FindStringSubmatch(id.Callsign)
	if reg != nil && len(reg)==2 {
		id.Registration = id.Callsign
		return Registration
	}
	
	icao := regexp.MustCompile("^([A-Z]{3})([0-9]{1,4})$").FindStringSubmatch(id.Callsign)
	if icao != nil && len(icao)==3 {
		id.Schedule.Number,_ = strconv.ParseInt(icao[2], 10, 64) // no errors here :)
		id.Schedule.ICAO = icao[1]
		return IcaoFlightNumber
	}

	bare := regexp.MustCompile("^([0-9]{2,4})$").FindStringSubmatch(id.Callsign)
	if bare != nil && len(bare)==2 {
		id.Schedule.Number,_ = strconv.ParseInt(bare[1], 10, 64) // no errors here :)
		return BareFlightNumber
	}
	
	return JunkCallsign
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

/* Some notes on identifiers for aircraft

# ADSB Mode-[E]S Identifiers (Icao24)

These are six digit hex codes, handed out to aircraft. Most aircraft
using ADS-B have this is a (semi?) permanent 'airframe' ID, but some
aircraft spoof it. And some have two transponders or something.

# Aircraft registration, e.g. N12312

# ADSB Callsigns

1. Many airlines use the ICAO flight number: SWA3848
2. Many private aircraft use their registration: N839AL
3. Some (private ?) aircraft use just their equipment type:
4. Annoyingly, some airlines use a bare flight number:
		1106  - was FFT 1106 (F9 Frontier)
		4517  - was SWA 4517 (WN Southwest)
		 948  - was SWA 948  (WN Southwest)
     210  - was AAY 210  (G4 Allegiant Air)

5. Various kinds of null idenitifiers:
 00000000   - was A69C72 (a Cessna Citation)
 ????????   - was A85D50 (a Virgin America A320, flying as VX938 !!)
 ''         - sometimes a MSG,1 will contain an empty string for the callsign

# Foreign identifiers

## Flightaware

The FA API uses an 'ident' for initial lookup, which can be one of three things:
 * ICAO flightnumber (3+4)
 * Registration / tailnumber
 * their own 'faFlightId'

 */
