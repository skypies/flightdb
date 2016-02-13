// Package Metar provides tools for retrieving, storing and parsing Metar weather records.
package metar

import(
	"bufio"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"regexp"
	"time"
)

// Metar: https://en.wikipedia.org/wiki/METAR, http://meteocentre.com/doc/metar.html

// We use Ogimet, as it prefixes with real timestamps, making historical fetching easier

// {{{ Report{}

type Report struct {
	Raw                  string // METAR KTTN 051853Z 04011KT 1/2SM A3006 RMK AO2 P0002 T10171017=

	IcaoAirport          string
	time.Time            // embed
	AltimeterSettingInHg float64    // In inches of mercury, e.g. 30.06 (only ever 4 sig figs)
}
func (mr Report)String() string {
	return fmt.Sprintf("%s@%s: %.4f inHg", mr.IcaoAirport,
		mr.Format("2006.01.02-15:04:05-0700"), mr.AltimeterSettingInHg)
}

// }}}
// {{{ Archive{}

type ByTimeAsc []time.Time
func (a ByTimeAsc) Len() int           { return len(a) }
func (a ByTimeAsc) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByTimeAsc) Less(i, j int) bool { return a[i].Before(a[j]) }

type Archive struct {
	Reports      map[time.Time]*[24]Report  // key=UTC midnight; value = [24]Report
	IcaoAirport  string
}

func NewArchive() *Archive {
	a := Archive{ Reports: map[time.Time]*[24]Report{} }
	return &a
}

func (a Archive)String() string {
	str := fmt.Sprintf("METAR for %s:-\n", a.IcaoAirport)

	keys := []time.Time{}
	for k,_ := range a.Reports { keys = append(keys, k) }
	sort.Sort(ByTimeAsc(keys))

	for _,k := range keys {
		str += fmt.Sprintf("%s:-\n", k.Format("2006.01.02"))
		for i,r := range a.Reports[k] {
			if r.Raw == "" { continue }
			str += fmt.Sprintf (" %02d: %s\n", i, r)
		}
	}
	return str
}

// }}}

// {{{ a.AtUTCMidnight

func (a Archive)AtUTCMidnight(in time.Time) time.Time {
	in = in.UTC()
	return time.Date(in.Year(), in.Month(), in.Day(), 0, 0, 0, 0, in.Location())
}

// }}}

// {{{ a.Lookup

// The report for this 'hour' might be ahead of t (they're generally
// at 56m past); so rewind an hour if so.
func (a Archive)Lookup(t time.Time) *Report {
	r := a.DirectLookup(t)
	if r == nil || r.After(t) {
		return a.DirectLookup(t.Add(time.Hour * -1))
	}
	return r
}

// }}}
// {{{ a.DirectLookup

// Return the report we have stored for this 'hour'
func (a Archive)DirectLookup(t time.Time) *Report {
	t = t.UTC()  // Just in case
	tMidnight := a.AtUTCMidnight(t)

	dayReports,exists := a.Reports[tMidnight]
	if !exists { return nil }

	if len(dayReports) < t.Hour() { return nil }
	r := &dayReports[t.Hour()]

	if len(r.Raw) == 0 { return nil }  // This means it was never populated

	return r
}

// }}}
// {{{ a.Add

func (a Archive)Add(r Report) {
	// Get the UTC day (00:00:00) for this report
	tMidnight := time.Date(r.Year(), r.Month(), r.Day(), 0, 0, 0, 0, r.Location())

	if _,exists := a.Reports[tMidnight]; !exists {
		reps := [24]Report{}
		a.Reports[tMidnight] = &reps
	}

	// The 'normal' ones are at 56m past the hour, but results appear in
	// descending time; don't overwrite results with a later timestamp.
	if orig := a.Reports[tMidnight][r.Hour()]; orig.Raw != "" {
		if orig.After(r.Time) { return }
	}

	a.Reports[tMidnight][r.Hour()] = r
}

// }}}

// {{{ ParseNOAA

func ParseNOAA(s string) ([]Report, error) {
	out := []Report{}
	headers := map[string]int{}
	
	lastPremableR := regexp.MustCompile("^[0-9]+ results$")
	preambling := true
	
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if preambling {
			if lastPremableR.MatchString(line) { preambling = false }
			continue
		}

		rdr := csv.NewReader(strings.NewReader(line))
		vals,err := rdr.Read()
		if err != nil { return out,err }

		// First line; store the header
		if len(headers) == 0 {
			for i,k := range vals {
				headers[k] = i
			}
			continue
		}

		// OK, so at last we're just parsing a regular line
		r := Report{
			Raw: vals[headers["raw_text"]],
			IcaoAirport: vals[headers["station_id"]],
		}
		if val,err := strconv.ParseFloat(vals[headers["altim_in_hg"]], 64); err != nil {
			return out, err
		} else {
			r.AltimeterSettingInHg = val
		}

		if tObs,err := time.Parse("2006-01-02T15:04:05Z", vals[headers["observation_time"]]); err !=nil{
			return out,err
		} else {
			r.Time = tObs
		}
		// if val,err := strconv.ParseInt(], 10, 64); err != nil {

		out = append(out, r)
	}

	return out,nil
}

/*

https://aviationweather.gov/adds/dataserver/metars/MetarExamples.php

https://aviationweather.gov/adds/dataserver_current/httpparam?dataSource=metars&requestType=retrieve&format=csv&stationString=KSFO&startTime=2016-01-31T03:15:32Z&endTime=2016-01-31T05:15:32Z

----
No errors
No warnings
6 ms
data source=metars
2 results
raw_text,station_id,observation_time,latitude,longitude,temp_c,dewpoint_c,wind_dir_degrees,wind_speed_kt,wind_gust_kt,visibility_statute_mi,altim_in_hg,sea_level_pressure_mb,corrected,auto,auto_station,maintenance_indicator_on,no_signal,lightning_sensor_off,freezing_rain_sensor_off,present_weather_sensor_off,wx_string,sky_cover,cloud_base_ft_agl,sky_cover,cloud_base_ft_agl,sky_cover,cloud_base_ft_agl,sky_cover,cloud_base_ft_agl,flight_category,three_hr_pressure_tendency_mb,maxT_c,minT_c,maxT24hr_c,minT24hr_c,precip_in,pcp3hr_in,pcp6hr_in,pcp24hr_in,snow_in,vert_vis_ft,metar_type,elevation_m
KSFO 310456Z 28011KT 10SM SCT110 BKN180 11/06 A3001 RMK AO2 SLP161 T01110056 $,KSFO,2016-01-31T04:56:00Z,37.62,-122.37,11.1,5.6,280,11,,10.0,30.008858,1016.1,,,TRUE,TRUE,,,,,,SCT,11000,BKN,18000,,,,,VFR,,,,,,,,,,,,METAR,3.0
KSFO 310356Z 30006KT 10SM BKN110 OVC180 11/05 A3001 RMK AO2 PRESFR SLP163 T01110050 $,KSFO,2016-01-31T03:56:00Z,37.62,-122.37,11.1,5.0,300,6,,10.0,30.008858,1016.3,,,TRUE,TRUE,,,,,,BKN,11000,OVC,18000,,,,,VFR,,,,,,,,,,,,METAR,3.0
----

*/

// }}}
// {{{ AssembleNOAA

func AssembleNOAA(reports []Report) (*Archive) {
	a := NewArchive()
	for _,r := range reports {	
		a.Add(r)
	}
	
	return a
}

// }}}
// {{{ FetchReportsFromNOAA

func FetchReportsFromNOAA(c *http.Client, station string, s,e time.Time) ([]Report,error) {
	url := "https://aviationweather.gov/adds/dataserver_current/httpparam?dataSource=metars&requestType=retrieve&format=csv"

	url += "&stationString="+station
	url += s.Format("&startTime=2006-01-02T15:04:05Z")
	url += e.Format  ("&endTime=2006-01-02T15:04:05Z")

	if c == nil {
		client := http.Client{}
		c = &client
	}
	
	resp,err := c.Get(url)
	if err != nil { return nil, err }

	defer resp.Body.Close()
	body,err := ioutil.ReadAll(resp.Body)
	if err != nil { return nil, err }

	reports,err := ParseNOAA(string(body))
	if err != nil { return nil, err }

	return reports,nil
}

// }}}
// {{{ FetchFromNOAA

func FetchFromNOAA(c *http.Client, station string, s,e time.Time) (*Archive,error) {
	reports,err := FetchReportsFromNOAA(c,station,s,e)
	if err != nil { return nil, err }

	a := AssembleNOAA(reports)	
	return a,nil
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
