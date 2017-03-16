package flightdb

import(
	"fmt"
	"regexp"
	"strconv"
)

/* Callsigns, as used in ADS-B broadcasts

1. Many airlines use the ICAO flight number: SWA3848
2. Many private aircraft use their registration: N839AL
3. Some (private ?) aircraft use just their equipment type:
4. Annoyingly, some airlines use a bare flight number:
		1106  - was FFT 1106 (F9 Frontier)
		4517  - was SWA 4517 (WN Southwest)
		 948  - was SWA 948  (WN Southwest)
     210  - was AAY 210  (G4 Allegiant Air)
   We can use the airframe cache to fix most of these up

5. Various kinds of null idenitifiers:
 00000000   - was A69C72 (a Cessna Citation)
 ????????   - was A85D50 (a Virgin America A320, flying as VX938 !!)
 ''         - sometimes a MSG,1 will contain an empty string for the callsign

6. Data from TRACON frequently has a suffix latter attached to the callsign

*/


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
type Callsign struct {
	Raw           string

	CallsignType
	Registration  string
	IcaoPrefix    string
	ATCSuffix     string // should be one char, really
	Number        int64
}

func (c Callsign)String() string {
	switch c.CallsignType {
	case IcaoFlightNumber:
		return fmt.Sprintf("%s%d", c.IcaoPrefix, c.Number) // Strips leading zeroes and ATC suffix
	default:
		return c.Raw
	}
}

func (c *Callsign)MaybeAddPrefix(prefix string) {
	if c.CallsignType == BareFlightNumber {
		c.IcaoPrefix = prefix
		c.CallsignType = IcaoFlightNumber
	}
}

func (c1 Callsign)Equal(c2 Callsign) bool {
	return c1.String() == c2.String()
}

func CallsignStringsEqual(c1,c2 string) bool {
	return NewCallsign(c1).Equal(NewCallsign(c2))
}

func NewCallsign(callsign string) (ret Callsign) {	
	ret.Raw = callsign
	
	// Registration (e.g. N23ST). Wikipedia:
	// An N-number may only consist of one to five characters, must
	// start with a digit other than zero, and cannot end in a run of
	// more than two letters. In addition, N-numbers may not contain the
	// letters I or O
	reg := regexp.MustCompile("^(N[1-9][0-9A-HJ-NP-Z]{0,4})$").FindStringSubmatch(callsign)
	if reg != nil && len(reg)==2 {
		ret.Registration = callsign
		ret.CallsignType = Registration
		return
	}
	
	icao := regexp.MustCompile("^([A-Z]{3})([0-9]{1,4})([A-Z]?)$").FindStringSubmatch(callsign)
	if icao != nil && len(icao)==4 {
		ret.Number,_ = strconv.ParseInt(icao[2], 10, 64) // no errors here :)
		ret.IcaoPrefix = icao[1]
		ret.ATCSuffix = icao[3]
		ret.CallsignType = IcaoFlightNumber
		return
	}

	bare := regexp.MustCompile("^([0-9]{2,4})$").FindStringSubmatch(callsign)
	if bare != nil && len(bare)==2 {
		ret.Number,_ = strconv.ParseInt(bare[1], 10, 64) // no errors here :)
		ret.CallsignType = BareFlightNumber
		return
	}

	ret.CallsignType = JunkCallsign
	return
}
