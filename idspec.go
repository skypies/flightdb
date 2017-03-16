package flightdb

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
	IcaoId        string
	Registration  string
	Callsign      string
	time.Time     // embed
	time.Duration // embed; optional; for when we're given a time range.
}

// The string serialization is used as a basic ID in many places (e.g. the idspec CGI arg)
func (idspec IdSpec)String() string {
	tStr := fmt.Sprintf("%d", idspec.Time.Unix())
	if idspec.Duration != 0 {
		tStr += fmt.Sprintf(":%d", idspec.Time.Add(idspec.Duration).Unix())
	}

	if idspec.IcaoId != "" {
		return fmt.Sprintf("%s@%s", idspec.IcaoId, tStr)
	} else if idspec.Callsign != "" {
		return fmt.Sprintf("%s@%s", idspec.Callsign, tStr)
	} else if idspec.Registration != "" {
		return fmt.Sprintf("%s@%s", idspec.Registration, tStr)
	}
	return "BadIdSpec@Provided"
}

func StringsToInt64s(in []string) ([]int64, error) {
	out := []int64{}
	for _,str := range in {
		if i,err := strconv.ParseInt(str, 10, 64); err != nil {
			return []int64{}, fmt.Errorf("'%s' not parseable", str)
		} else {
			out = append(out, i)
		}
	}
	return out, nil
}

// Parse a string into a new spec
//     A23A23@14123123123123  (IcaoId at an epoch time)
//     A23A23@2006-01-02T15:04:05Z07:00  (IcaoId at an RFC3339 time)
//     A23A23@14111111111111:14222222222222  (IcaoId within time range; could be multiple matches)
//     UAL123@14123123123123  (IATACallsign instead of IcaoId)
//     N1234S@14123123123123  (Registration Callsign instead of IcaoId)
func NewIdSpec(idspecString string) (IdSpec,error) {
	bits := strings.Split(idspecString, "@")
	if len(bits) != 2 {
		return IdSpec{}, fmt.Errorf("IdSpec '%s' did not match <airframe>@<epoch>", idspecString)
	}
	id, timespec := bits[0], bits[1]

	idspec := IdSpec{}

	if t,err := time.Parse(time.RFC3339, timespec); err == nil {
		idspec.Time = t

	} else if timeInts,err := StringsToInt64s(strings.Split(timespec, ":")); err != nil {
		return IdSpec{}, fmt.Errorf("IdSpec '%s' timespec problem: %v", idspecString, err)

	} else {
		idspec.Time = time.Unix(timeInts[0], 0)
		if len(timeInts) == 2 {
			idspec.Duration = time.Unix(timeInts[1], 0).Sub(idspec.Time)
		}
	}

	// PROBLEM: some sets of IcaoIDs look like callsigns, e.g. ADF06D, ADA526
	// So if our callsign looks like that, pretend it isn't a callsign. This likely breaks a lot
	// of callsign lookups, which is a shame.

	// Looks like a 6 digit hex string ? Presume IcaoID
	icaoid := regexp.MustCompile("^[A-F0-9]{6}$").FindStringSubmatch(id)
	if icaoid != nil && len(icaoid)==1 {
		idspec.IcaoId = id
		return idspec, nil
	}

	// Let's see if it could be an ICAO callsign
	parsedCallsign := NewCallsign(id)
	if parsedCallsign.CallsignType == IcaoFlightNumber {
		idspec.Callsign = id
		return idspec, nil
	}
	
	// Else, if it looked like a registration ...
	if parsedCallsign.CallsignType == Registration {
		idspec.Registration = id
		return idspec, nil
	}

	// Presume what is left is some other kind of registration (e.g. LN431GW)
	idspec.Registration = id
	return idspec, nil
	//return IdSpec{}, fmt.Errorf("IdSpec '%s' unparseable before @", idspec)
}

func (f Flight)IdSpec() IdSpec {
	times := f.Timeslots()
	midIndex := len(times) / 2

	return IdSpec{
		IcaoId:        f.IcaoId,
		Registration:  f.Registration,
		Callsign:      f.Callsign, // Need to match whatever ends up in the datastore index
		Time:          times[midIndex],
	}
}
