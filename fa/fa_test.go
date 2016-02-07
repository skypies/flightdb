package fa

// go test -v github.com/skypies/flightdb/flightaware

import (
	// Consider https://golang.org/pkg/encoding/json/#Indent
	"fmt"
	//"strings"
	//"time"
	"testing"
	//u "net/http/httputil"

	"github.com/skypies/geo/sfo"
)

var (
	TestAPIKey = "..." // Fill in your own
	TestAPIUsername = "abworrall"
)

/*
// Call this just once per flightaware account, to get stable result lists
func TestCallSetMaximumResultSize(t *testing.T) {
	fa := Flightaware{APIKey:TestAPIKey, APIUsername:TestAPIUsername}
	fa.Init()
	fa.CallSetMaximumResultSize(45)
}
*/

func TestCallSearch(t *testing.T) {
	fa := Flightaware{APIKey:TestAPIKey, APIUsername:TestAPIUsername}
	fa.Init()

	box := sfo.KLatlongSFO.Box(120,120)

	q := "" // "-filter airline -inAir 1 -aboveAltitude 8"
	
	ret,err := fa.CallSearch(q, box)
	if err != nil { t.Errorf("init call: %v", err) }
	_=ret
	for i,f := range ret {
		fmt.Printf(" * [%03d] %s\n", i, f)
	}
}


/*
func TestUrlToResp(t * testing.T) {
	fa := Flightaware{APIKey:TestAPIKey, APIUsername:TestAPIUsername}
	fa.Init()
	args := map[string]string{"airport": "KSFO", "startTime": "0", "howMany": "1", "offset": "0"}
	if resp,err := fa.UrlToResp("MetarEx", args); err != nil {
		t.Errorf("UrlToResp failed: %v", err)
	} else {
		_ = resp
		bytes,_ := u.DumpResponse(resp, true)
		fmt.Printf("<<<< resp\n%s<<<<\n", string(bytes))		
	}
}
*/
/*

func TestUrlToJsonMap(t * testing.T) {
	fa := Flightaware{APIKey:TestAPIKey, APIUsername:TestAPIUsername}
	fa.Init()
	verb := "MetarEx"
	args := map[string]string{"airport": "KSFO", "startTime": "0", "howMany": "1", "offset": "0"}
	
	if jsonMap,err := fa.UrlToJsonMap(verb, args); err != nil {
		t.Errorf("UrlToJsonMap failed: %v", err)
	} else {
		fields := jsonMap[verb+"Result"].(map[string]interface{})
		results := fields["metar"].([]interface{})

		result1 := results[0].(map[string]interface{})
		
		if v,exists := result1["visibility"]; exists == false {
			for _,result := range results {	
				for k,_ := range result.(map[string]interface{}) {
					fmt.Printf(" * %s\n", k)
				}
			}
			t.Errorf("UrlToJsonMap had missing visibility data")
		} else {
			fmt.Printf("\n%#v\n", v)
		}
	}
}
*/

/*
func TestCallFlightInfoEx(t * testing.T) {
	fa := Flightaware{APIKey:TestAPIKey, APIUsername:TestAPIUsername}
	fa.Init()

	//ident := "VRD932"
	//ident = "N629VA"
	ident := "UA6432"
	
	if results,err := fa.CallFlightInfoEx(ident); err != nil {
		t.Errorf("CallFlightInfoEx failed: %v", err)
	} else {
			for _,v := range results {
				fmt.Printf("[%s-%s]{%s} %s, %s %s\n", v.Origin, v.Destination, v.Faflightid, v.Ident,
					time.Unix(int64(v.Fileddeparturetime), 0),
					v.Route)
			}
		if len(results) != 15 {
			t.Errorf("Not enough results ? List:\n")
			for _,v := range results {
				fmt.Printf("[%s-%s]{%s} %s, %s\n", v.Origin, v.Destination, v.Faflightid, v.Ident, v.Route)
			}
		}
	}
}
*/

/*
[KSFO-KLAX]{VRD932-1442701800-schedule-0000} VRD932, SSTIK3 EBAYE AVE SADDE6
[KSAN-KSFO]{VRD961-1442690700-schedule-0000} VRD961, PEBLE6 SXC VTU RZS STOKD SERFR SERFR1
[KSFO-KSAN]{VRD956-1442682600-schedule-0001} VRD956, OFFSH9 MCKEY LAX BAYVU4
[KDAL-KSFO]{VRD713-1442667000-schedule-0001} VRD713, KKITY2 HULZE TCC RSK ILC J80 OAL INYOE DYAMD2
[KLAX-KDAL]{VRD882-1442620800-schedule-0001} VRD882, HOLTZ9 TRM PKE CHEAR TURKI JFRYE2
[KJFK-KLAX]{VRD411-1442595600-schedule-0003} VRD411, COATE Q436 RAAKK Q438 BERYS FNT KG78M VIKNG KP75E SAYGE BDROC GLACO J64 HEC RIIVR2
[KLAX-KJFK]{VRD406-1442511000-schedule-0000} VRD406, HOLTZ9 TRM PKE J96 KEYKE FENON SOSPE SAAGS J80 SPI VHP ROD DJB JHW J70 LVZ LENDY6
[KSEA-KLAX]{VRD1780-1442498400-schedule-0000} VRD1780, SUMMA7 SUMMA JINMO Q7 AVE SADDE6
[KSFO-KSEA]{VRD752-1442461800-schedule-0000} VRD752, TRUKN2 DEDHD BTG HAWKZ4
[KPSP-KSFO]{VRD593-1442454000-schedule-0000} VRD593, PSP V386 PMD MAKRS SERFR SERFR1
[KSFO-KPSP]{VRD592-1442446200-schedule-0001} VRD592, SSTIK3 LOSHN PMD V137 PSP
[KJFK-KSFO]{VRD25-1442418900-schedule-0000} VRD25, GAYEL Q818 WOZEE Q935 DERLO Q822 KITOK DUTYS GOOLD ONL NLSEN GAROT OAL INYOE DYAMD2
[KLAX-KJFK]{VRD412-1442349300-schedule-0001} VRD412, HOLTZ9 TRM PKE J96 KEYKE KD39S KD42U ZAROS KK51A JUDGE KK54E SAAGS J80 TWAIN J80 SPI VHP ROD DJB JHW J70 LVZ LENDY6
[KSFO-KLAX]{VRD924-1442340300-schedule-0000} VRD924, SSTIK3 EBAYE AVE SADDE6
[KLAX-KSFO]{VRD1935-1442333100-schedule-0000} VRD1935, VTU5 RZS STOKD SERFR SERFR1

*/

/*
func TestCallGetHistoricalTrack(t * testing.T) {
	fa := Flightaware{APIKey:TestAPIKey, APIUsername:TestAPIUsername}
	fa.Init()

	faFlightId := "SKW6432-1446963757-airline-0008"
	
	if results,err := fa.CallGetHistoricalTrack(faFlightId); err != nil {
		t.Errorf("GetHistoricalTrack failed: %v", err)
	} else {
		for _,v := range results {
			fmt.Printf("%s (%.4f,%.4f) %d knots, %d feet : %s\n",
				time.Unix(int64(v.Timestamp),0),
				v.Latitude, v.Longitude, v.Groundspeed, v.Altitude, v.Updatetype)
		}
	}
}
*/
/*
func TestLookupLastTrackByFlightnumber(t * testing.T) {
	fa := Flightaware{APIKey:TestAPIKey, APIUsername:TestAPIUsername}
	fa.Init()
	fnumber := "UA6432" // "UA1446" was TZ during flight
	
	if track,err := fa.LookupLastTrackByFlightnumber(fnumber); err != nil {
		t.Errorf("CallFlightInfoEx failed: %v", err)
	} else {
		for _,v := range track {
			fmt.Printf("%s (%.4f,%.4f) %d knots, %d feet, src=%s",
				time.Unix(int64(v.Timestamp),0),
				v.Latitude, v.Longitude, v.Groundspeed, v.Altitude*100, v.Updatetype)
		}
	}
}
*/

/*
func TestCallArrived(t *testing.T) {
	fa := Flightaware{APIKey:TestAPIKey, APIUsername:TestAPIUsername}
	fa.Init()

	ret,err := fa.CallArrived("KSJC")
	if err != nil { t.Errorf("init call: %v", err) }
	for i,f := range ret {
		fmt.Printf(" * [%03d] %s (%s-%s)\n", i, f.Ident, f.Origin, f.Destination)
	}
}
*/


/*
// This is VERY EXPENSIVE, and dumps a routing table for everything into SFO.
func TestHackthing(t * testing.T) {
	fa := Flightaware{APIKey:TestAPIKey, APIUsername:TestAPIUsername}
	fa.Init()

	ret,err := fa.CallArrived("KSFO")
	if err != nil { t.Errorf("init call: %v", err) }
	for _,f := range ret {
		results,err := fa.CallFlightInfoEx(f.Ident)
		if err != nil { t.Errorf("CallFlightInfoEx failed: %v", err) }

		for _,v := range results {
			t := time.Unix(int64(v.Fileddeparturetime), 0)

      // Use v.actualdeparturetime to compare against arrived.actualdeparturetime

			if t.After(time.Now()) { continue }

			atoms := strings.Split(v.Route, " ")

			
			fmt.Printf("%s, %s-%s, %s, %s, %s\n", v.Ident, v.Origin, v.Destination, t,
				atoms[len(atoms)-1], v.Route)
			break
		}
	}
}
*/
