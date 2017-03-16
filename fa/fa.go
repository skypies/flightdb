package fa   // http://flightxml.flightaware.com/soap/FlightXML2/doc

import (
	// "bytes"
	"encoding/json"
	"fmt"
	"net/http"
	u "net/http/httputil"
	"net/url"
	"time"

	//"github.com/skypies/adsb"
	"github.com/skypies/geo"
	//"github.com/skypies/pi/airspace"

	//fdb "github.com/skypies/flightdb"
)

type Flightaware struct {
	Client *http.Client
	APIKey, APIUsername string
}

func (fa *Flightaware)Init() {
	if fa.Client == nil {
		fa.Client = &http.Client{}
	}
}

// {{{ UrlToResp

// http://flightxml.flightaware.com/json/FlightXML2/METHODNAME
// All requests made must supply the username and API Key as a "basic" Authorization HTTP header

func (fa Flightaware)UrlToResp(verb string, args map[string]string) (*http.Response, error) {
	var debug = false

	urlToCall := "http://flightxml.flightaware.com/json/FlightXML2/" + verb

	postArgs := url.Values{}
	for k,v := range args { postArgs.Set(k,v) }
	urlToCall += "?"+postArgs.Encode()

	if req, err := http.NewRequest("GET", urlToCall, nil); err != nil {
		return nil,err
	} else {
		req.SetBasicAuth(fa.APIUsername, fa.APIKey)
		if (debug) {
			bytes,_ := u.DumpRequest(req, true)
			fmt.Printf(">>>> req\n%s>>>>\n", string(bytes))
		}

		if resp, err := fa.Client.Do(req); err != nil {
			return nil,err
		} else {
			if (debug) {
				bytes,_ := u.DumpResponse(resp, true)
				fmt.Printf("<<<< resp\n%s<<<<\n", string(bytes))
			}

			return resp, nil
		}
	}
}

// }}}
// {{{ RespToJsonMap

func (fa Flightaware)RespToJsonMap(resp *http.Response) (jsonMap map[string]interface{}, err error) {
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&jsonMap)
	return
}

// }}}
// {{{ UrlToJsonMap

func (fa Flightaware)UrlToJsonMap(verb string, args map[string]string) (map[string]interface{}, error) {
	if resp,err := fa.UrlToResp(verb, args); err != nil {
		return nil,err
	} else if jsonMap,err := fa.RespToJsonMap(resp); err != nil {
		return nil,err
	} else {
		return jsonMap,nil
	}
}

// }}}

// {{{ CallSetMaximumResultSize

// This means you get more than 15 results back in a single page. But you still pay
// one API call for every 15 results.

func (fa Flightaware)CallSetMaximumResultSize(n int) {
	args := map[string]string{
		"max_size": fmt.Sprintf("%d", n),
	}
	
	if resp,err := fa.UrlToResp("SetMaximumResultSize", args); err != nil {
		return

	} else {
		defer resp.Body.Close()

		bytes,_ := u.DumpResponse(resp, true)
		fmt.Printf(">>>> %d >>>>\n<<<< resp\n%s<<<<\n", n, string(bytes))		
	}
}

// }}}
// {{{ CallGetHistoricalTrack

// GetHistoricalTrack(FaFlightID) -> ArrayOfTrackStruct
// Most recent datapoint in the track will br ~6m old if flight is in air (TZ/radar/FAA ?)

func (fa Flightaware)CallGetHistoricalTrack(faFlightId string) ([]TrackStruct, error) {
	args := map[string]string{
		"faFlightID": faFlightId,
	}	
	if resp,err := fa.UrlToResp("GetHistoricalTrack", args); err != nil {
		return nil, err

	} else {
		//bytes,_ := u.DumpResponse(resp, true)
		//fmt.Printf("<<<< resp\n%s<<<<\n", string(bytes))
		defer resp.Body.Close()
		r := GetHistoricalTrackResponse{}
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil { return nil, err }
		return r.GetHistoricalTrackResult.Track,nil

		return nil,nil
	}
}

// }}}
// {{{ CallFlightInfoEx

// {{{ Notes

/* https://flightaware.com/commercial/flightxml/explorer/#op_FlightInfoEx
 *
 * The lookup key can be one of:
 *  * specific tail number (e.g., N12345), or
 *  * an ident (typically an ICAO airline with flight number, e.g., SWA2558), or
 *  * [a fscking flightnumber, e,g, "WN2558" !!!]
 *  * a FlightAware-assigned unique flight identifier (e.g. faFlightID)
 *
 * We only call this as it's the only way to find out an faFlightId, which we need for the track.
 *
 * If you call it with a TailNumber, you'll get results for the
 * sequence of flights that specific aircraft has flown, e.g.:

[KLAX-KORD]{..} VRD236, 2015-09-21 17:50:00 -0700 PDT LOOP7 DAG J100 LAS PGA KD51W KK57C IRK TRIDE2
[KSFO-KLAX]{..} VRD932, 2015-09-21 15:30:00 -0700 PDT SSTIK3 EBAYE AVE SADDE6
[KSAN-KSFO]{..} VRD961, 2015-09-21 12:25:00 -0700 PDT PEBLE6 SXC VTU RZS STOKD SERFR SERFR1
[KSFO-KSAN]{..} VRD956, 2015-09-21 10:10:00 -0700 PDT OFFSH9 MCKEY LAX BAYVU4
[KDAL-KSFO]{..} VRD713, 2015-09-21 05:50:00 -0700 PDT KKITY2 HULZE TCC RSK ILC J80 OAL INYOE DYAMD2
[KLAX-KDAL]{..} VRD882, 2015-09-20 17:00:00 -0700 PDT HOLTZ9 TRM PKE CHEAR TURKI JFRYE2

 * If you call it with a callsign ("ident"), you'll get results for
 * each instance of that scheduled flightnumber, e.g.:

[KSFO-KLAX]{..} VRD932, 2015-09-23 15:30:00 -0700 PDT 
[KSFO-KLAX]{..} VRD932, 2015-09-22 15:30:00 -0700 PDT SSTIK3 EBAYE AVE SADDE6
[KSFO-KLAX]{..} VRD932, 2015-09-21 15:30:00 -0700 PDT SSTIK3 EBAYE AVE SADDE6
[KSFO-KLAX]{..} VRD932, 2015-09-20 15:30:00 -0700 PDT SSTIK3 EBAYE AVE SADDE6
[KSFO-KLAX]{..} VRD932, 2015-09-19 15:30:00 -0700 PDT SSTIK3 EBAYE AVE SADDE6
[KSFO-KLAX]{..} VRD932, 2015-09-18 15:30:00 -0700 PDT SSTIK3 EBAYE AVE SADDE6
[KSFO-KLAX]{..} VRD932, 2015-09-17 15:30:00 -0700 PDT SSTIK3 EBAYE AVE SADDE6
[KSFO-KLAX]{..} VRD932, 2015-09-16 15:30:00 -0700 PDT WESLA3 EBAYE AVE SADDE6
[KSFO-KLAX]{..} VRD932, 2015-09-15 15:30:00 -0700 PDT WESLA3 EBAYE AVE SADDE6

 * Note that this query was submitted on 2015-09-21; so this list
 * contains flights scheduled for a few days in advance.
 */

// }}}

func (fa Flightaware)CallFlightInfoEx(key string) ([]FlightExStruct, error) {
	args := map[string]string{
		"ident": key,
		"howMany": "15",
		"offset": "0",
	}
	
	if resp,err := fa.UrlToResp("FlightInfoEx", args); err != nil {
		return nil, err

	} else {
		defer resp.Body.Close()
		r := FlightInfoExResponse{}
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil { return nil, err }
		return r.FlightInfoExResult.Flights,nil
	}
}

// }}}
// {{{ CallArrived

// {{{ Notes

/* https://flightaware.com/commercial/flightxml/explorer/#op_Arrived
 *
 * howMany capped at 15; need to loop.

"ArrivedResult":{
  "next_offset":15,
  "arrivals":[
    {"ident":"VRD751",
     "aircrafttype":"A320",
     "actualdeparturetime":1450278267,
     "actualarrivaltime":1450283493,
     "origin":"KSEA",
     "destination":"KSFO",
     "originName":"Seattle-Tacoma Intl",
     "originCity":"Seattle, WA",
     "destinationName":"San Francisco Intl",
     "destinationCity":"San Francisco, CA"
    },
    {"ident":"VRD925",
     ....}
  ]
}

 */

// }}}

func (fa Flightaware)CallArrived(airportIcao4 string) ([]ArrivalFlightStruct, error) {
	args := map[string]string{
		"howMany": "15",
		"offset": "0",
		"airport": airportIcao4,
		"filter": "airline",
	}

	ret := []ArrivalFlightStruct{}

	for {
		resp,err := fa.UrlToResp("Arrived", args)
		if err != nil { return nil, err }

		defer resp.Body.Close()
		//bytes,_ := u.DumpResponse(resp, true)
		//fmt.Printf(">>>> %s >>>>\n<<<< resp\n%s<<<<\n", airportIcao4, string(bytes))		
		r := ArrivedResponse{}
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil { return nil, err }
		ret = append(ret, r.ArrivedResult.Arrivals...)

		if r.ArrivedResult.Nextoffset <= 0 { break }
		args["offset"] = fmt.Sprintf("%d", r.ArrivedResult.Nextoffset)
	}			

	return ret,nil
}

// }}}
// {{{ CallSearch

// {{{ Notes

/* https://flightaware.com/commercial/flightxml/explorer/#op_Search
 *
 * howMany capped at 15; need to loop.

"ArrivedResult":{
  "next_offset":15,
  "arrivals":[
    {"ident":"VRD751",
     "aircrafttype":"A320",
     "actualdeparturetime":1450278267,

"SearchResult":{
  "next_offset":-1,
  "aircraft":[
    {"faFlightID":"AAL209-1452816900-schedule-0000",
     "ident":"AAL209",
     "prefix":"",
     "type":"B738",
     "suffix":"L",
     "origin":"KLAX",
     "destination":"KSFO",
     "timeout":"ok",
     "timestamp":1452993062,
     "departureTime":1452990000,
     "firstPositionTime":1452990179,
     "arrivalTime":0,
     "longitude":-122.11666999999999916,
     "latitude":37.516669999999997742,
     "lowLongitude":-122.11666999999999916,
     "lowLatitude":33.876390000000000668,
     "highLongitude":-118.43332999999999799,
     "highLatitude":37.516669999999997742,
     "groundspeed":177,
     "altitude":44,
     "heading":332,
     "altitudeStatus":"C",
     "updateType":"TZ",
     "altitudeChange":"D",
     "waypoints":"33.933 -118.4 34 -118.62 34.017 -118.7 34.05 -118.85"},

 */

// }}}

func (f InFlightStruct)String() string {
	return fmt.Sprintf("%-7.7s [%s-%s] %s (%.5f,%.5f), %5df %3dk %3ddeg (age: %s)",
		f.Ident, f.Origin, f.Destination, time.Unix(int64(f.DepartureTime),0).Format("Jan02 15:04 MST"),
		f.Latitude, f.Longitude, f.Altitude*100.0, f.Groundspeed, f.Heading,
		time.Since(time.Unix(int64(f.Timestamp),0)))
}

func (fa Flightaware)CallSearch(query string, box geo.LatlongBox) ([]InFlightStruct, error) {
	// -latlong "MINLAT MINLON MAXLAT MAXLON"
	// -filter {ga|airline}

	query += fmt.Sprintf(" -latlong \"%.5f %.5f %.5f %.5f\"",
		box.SW.Lat, box.SW.Long, box.NE.Lat, box.NE.Long)

	// NOTE - howMany should be enough to capture everything. The ordering/selection between
	// different result pages is inconsistent (dupes, dropped things). For this to be honored
	// though, you need to CallSetMaximumResultSize just once per flightaware account (see the
	// test file for a routine for this.)
	args := map[string]string{
		"howMany": "45",
		"offset": "0",
		"query": query,
	}

	ret := []InFlightStruct{}	
	
	for {
		resp,err := fa.UrlToResp("Search", args)
		if err != nil { return nil, err }

		r := SearchResponse{}
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil { return nil, err }
		ret = append(ret, r.SearchResult.Aircraft...)

		if r.SearchResult.Nextoffset <= 0 { break }
		args["offset"] = fmt.Sprintf("%d", r.SearchResult.Nextoffset)
	}			

	return ret,nil
}

// }}}

// {{{ LookupLastTrackByFlightnumber

// Given a flightnumber, pull up the track that looks the most recent.
// I don't know how this pans out when the same flightnumber is used for multileg flights :/
func (fa Flightaware)LookupLastTrackByFlightnumber(flightnumber string) ([]TrackStruct,error) {
	if results,err := fa.CallFlightInfoEx(flightnumber); err != nil {
		return nil, err
	} else if len(results) == 0 {
		return nil, fmt.Errorf("CallFlightInfoEx('%s') returned no matches", flightnumber)
	} else {
		// Returns a list of the N most recent trips the aircraft 'ident' has made.
		for _,v := range results {
			now := time.Now()
			schedDep := time.Unix(int64(v.Fileddeparturetime), 0)

			if schedDep.After(now) { continue } // Skip past flights not scheduled to take off yet

			// We are now at the most recent flight whose scheduled departure is in the past. Grab it!
			if track, err := fa.CallGetHistoricalTrack(v.Faflightid); err != nil {
				return nil,fmt.Errorf("GetTrack('%s') returned err=%v", flightnumber, err)
			} else {
				return track, nil
			}
		}
	}
	return nil,fmt.Errorf("Found tracks for '%s' but all in future ?", flightnumber)
}

// }}}

/*
// {{{ faFlight2AircraftData

func faFlight2AircraftData(in InFlightStruct, id adsb.IcaoId) airspace.AircraftData {
	msg := adsb.CompositeMsg{
		Msg: adsb.Msg{
			Icao24: id,
			GeneratedTimestampUTC: time.Unix(int64(in.Timestamp),0).UTC(),
			Callsign: in.Ident,
			Altitude: int64(in.Altitude)*100,
			GroundSpeed: int64(in.Groundspeed),
			Track: int64(in.Heading),
			Position: geo.Latlong{in.Latitude, in.Longitude},
		},
		ReceiverName: "FlightAware",
	}

	return airspace.AircraftData{
 		Msg: &msg,

		Airframe: fdb.Airframe{
			Icao24: string(id),
			EquipmentType: in.EquipType,
		},

		NumMessagesSeen: 1,
		Source: "fa",
	}
}

// }}}
// {{{ fa.FetchAirspace

// Overlays the flightaware 'Search' results into the airspace
func  (fa Flightaware)FetchAirspace(box geo.LatlongBox) (*airspace.Airspace, error) {

	// http://flightaware.com/commercial/flightxml/explorer/#op_Search
	//q := "-filter airline -inAir 1 -aboveAltitude 2"	
	q := "-inAir 1"	
	ret,err := fa.CallSearch(q, box)
	if err != nil {
		return nil,err
	}
	
	as := airspace.NewAirspace()
	for i,f := range ret {
		id := adsb.IcaoId(fmt.Sprintf("FF%04x", i))  // They might actually have an IcaoID ?
		as.Aircraft[id] = faFlight2AircraftData(f, id)
	}
	return &as, nil
}

// }}}
*/

/* Costs (https://flightaware.com/commercial/flightxml/pricing_class.rvt)

 * FlightInfoEx       : class 3: $0.0020 / call
 * Arrived            : class 2: $0.0079 / call (also departed)
 * Search             : class 2: $0.0079 / call
 * GetHistoricalTrack : class 2: $0.0079 / call

 * Class 1 == A, Class 2 == B, etc/

To scan & get routes for 1200 flights per day:
 - (40+40.B) + 1200.C
 = $3.03 per day ($100 / month)



To get a scan of ~550 airline flights arriving at SFO:
 - Arrived * 40 (15 results per call)    - 40.B
 - Presume departing flights is the same - 40.B

To get fFlightId & route for those 1200 flights
 - 1200.C

To get tracks for the interesting 33% of them
 - 400.B (

So a full week at SFO is 7.(1200.C + 480.B)
 = 8400.C + 3360.B
 = $16.80 + $26.54
 = $43.34

 */


// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
