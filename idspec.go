package flightdb2

import(
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// An identifier specifier - something we receive (or generate) that
// uniquely identifies a flight. Can be {airframe+time}, or
// {callsign+time}; maybe later we'll support flight designators.
type IdSpec struct {
	IcaoId       string
	Registration string
	Callsign     string
	time.Time    // embed
}

// The string serialization is used as a basic ID in many places (e.g. the idspec CGI arg)
func (idspec IdSpec)String() string {
 //Format("2006.01.02-15:04:05-0700")) ??
	if idspec.IcaoId != "" {
		return fmt.Sprintf("%s@%d", idspec.IcaoId, idspec.Time.Unix())
	} else if idspec.Callsign != "" {
		return fmt.Sprintf("%s@%d", idspec.Callsign, idspec.Time.Unix())
	} else if idspec.Registration != "" {
		return fmt.Sprintf("%s@%d", idspec.Registration, idspec.Time.Unix())
	}
	return "BadIdSpec@Provided"
}

// Parse a string into a new spec
//     A23A23@14123123123123  (IcaoId at an epoch time)
//     UAL123@14123123123123  (IATACallsign at an epoch time)
//     N1234S@14123123123123  (Registration Callsign at an epoch time)
func NewIdSpec(idspec string) (IdSpec,error) {
	bits := strings.Split(idspec, "@")
	if len(bits) != 2 {
		return IdSpec{}, fmt.Errorf("IdSpec '%s' did not match <airframe>@<epoch>", idspec)
	}
	
	epochInt,err := strconv.ParseInt(bits[1], 10, 64)
	if err != nil {
		return IdSpec{}, fmt.Errorf("IdSpec '%s' did not have parseable int after @: %v", idspec, err)
	}

	// Looks like a 6 digit hex string ? Presume IcaoID
	icaoid := regexp.MustCompile("^[A-F0-9]{6}$").FindStringSubmatch(bits[0])
	if icaoid != nil && len(icaoid)==1 {
		return IdSpec{
			IcaoId: bits[0],
			Time: time.Unix(epochInt, 0),
		}, nil
	}

	// The code below might be over-thinking it; callsign/registration
	// all ends up in the 'ident' field anyhow, so maybe treat
	// everything as callsign.
	
	// Looks like an IATA callsign, or a registration ?
	parsedCallsign := NewCallsign(bits[0])
	switch parsedCallsign.CallsignType {
	case IcaoFlightNumber:
		return IdSpec{ Callsign: bits[0], Time: time.Unix(epochInt, 0), }, nil
	case Registration:
		return IdSpec{ Registration: bits[0], Time: time.Unix(epochInt, 0), }, nil
	}

	// Presume what is left is some other kind of registration (e.g. LN431GW)
	return IdSpec{ Registration: bits[0], Time: time.Unix(epochInt, 0), }, nil
	//return IdSpec{}, fmt.Errorf("IdSpec '%s' unparseable before @", idspec)
}

func (f Flight)IdSpec() IdSpec {
	times := f.Timeslots(time.Minute * 30)  // FIXME: where does slot duration live
	midIndex := len(times) / 2

	return IdSpec{
		IcaoId:        f.IcaoId,
		Registration:  f.Registration,
		Callsign:      f.Callsign, // Need to match whatever ends up in the datastore index
		Time:          times[midIndex],
	}
}
