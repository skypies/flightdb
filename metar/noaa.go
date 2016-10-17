// Package Metar provides tools for retrieving, storing and parsing Metar weather records.
package metar

import(
	"bufio"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"regexp"
	"time"
)

// {{{ parseNOAA

func parseNOAA(s string) ([]Report, error) {
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

		// If this line has no altim_hg value, let's skip it
		if vals[headers["altim_in_hg"]] == "" {
			continue
		}

		// OK, so at last we're just parsing a regular line
		r := Report{
			Raw: vals[headers["raw_text"]],
			Source: "NOAA",
			IcaoAirport: vals[headers["station_id"]],
		}
		if val,err := strconv.ParseFloat(vals[headers["altim_in_hg"]], 64); err != nil {
			return out, fmt.Errorf("parse error %v, '%v', data[%d]:%#vals", err,
				vals[headers["altim_in_hg"]], headers["altim_in_hg"], vals)
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
// {{{ assembleNOAA

func assembleNOAA(reports []Report) (*Archive) {
	a := NewArchive()
	for _,r := range reports {	
		a.Add(r)
	}
	
	return a
}

// }}}
// {{{ fetchFromNOAA

func fetchFromNOAA(c *http.Client, station string, s,e time.Time) ([]Report,error) {
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

	reports,err := parseNOAA(string(body))
	if err != nil { return nil, err }

	//return nil, fmt.Errorf("url:"+url+"\nOho: "+string(body))
	
	return reports,nil
}

// }}}

// {{{ fetchArchiveFromNOAA

func fetchArchiveFromNOAA(c *http.Client, station string, s,e time.Time) (*Archive,error) {
	reports,err := fetchFromNOAA(c,station,s,e)
	if err != nil { return nil, err }

	a := assembleNOAA(reports)	
	return a,nil
}

// }}}
// {{{ fetchReportsFromNOAA

// We use the archive approach to get just the latest report per hour

func fetchReportsFromNOAA(c *http.Client, station string, s,e time.Time) ([]Report,error) {
	ar,err := fetchArchiveFromNOAA(c,station,s,e)
	if err != nil { return []Report{}, err }

	return ar.allReports(), nil
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
