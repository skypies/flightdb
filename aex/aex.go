package aex

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/skypies/adsb"
	"github.com/skypies/geo"
	"github.com/skypies/pi/airspace"

	fdb "github.com/skypies/flightdb"
)


type AdsbExchange struct {
	Client *http.Client
}

// {{{ args2url

func (aex AdsbExchange)args2url(args map[string]string) string {
	urlStem := "https://public-api.adsbexchange.com/VirtualRadar/AircraftList.json"

	postArgs := url.Values{}
	for k,v := range args { postArgs.Set(k,v) }

	return urlStem+"?"+postArgs.Encode()
}

// }}}
// {{{ UrlToResp

func (aex AdsbExchange)UrlToResp(url string) (*http.Response, error) {
	if req, err := http.NewRequest("GET", url, nil); err != nil {
		return nil,err
	} else if resp, err := aex.Client.Do(req); err != nil {
		return nil,err
	} else {
		return resp, nil
	}
}

// }}}

// {{{ LiveQuery

// https://public-api.adsbexchange.com/VirtualRadar/AircraftList.json?fDstL=0&fDstU=1770&lat=37.618817&lng=-122.375428

func (aex AdsbExchange)LiveQuery(box geo.LatlongBox) ([]AExAircraft, error) {
	args := map[string]string{
		"lat": fmt.Sprintf("%.6f", box.Center().Lat),
		"lng": fmt.Sprintf("%.6f", box.Center().Long),
		"fDstL": "0",
		"fDstU": fmt.Sprintf("%.0f", box.NE.DistKM(box.SW) / 2.0), // search radius; half the box diagonal.
	}

	url := aex.args2url(args)
	
	lqResponse := LiveQueryResponse{}
	
	resp,err := aex.UrlToResp(url)
	if err != nil {
		return nil, fmt.Errorf("AEx/LQ error:%v", err)
	}
	if err := json.NewDecoder(resp.Body).Decode(&lqResponse); err != nil {
		return nil, fmt.Errorf("AEx/Decode error:%v", err)
	}

	return lqResponse.Aircraft, nil
}

// }}}

// {{{ ToAircraftData

// "KSFO San Francisco, United States" --> "SFO"
func toIcaoAirport(in string) string {
	subs := regexp.MustCompile("^K([A-Z]{3})\\s").FindStringSubmatch(in)
	if len(subs) == 2 {
		return subs[1]
	}
	return ""
}

func ToAircraftData (in AExAircraft) airspace.AircraftData {
	// Timestamp is a float Unix epoch millis
	tSecs   := int64(in.PosTime / 1000.0)
	//tMillis := int64(math.Mod(in.PosTime, 1000.0))

	callsign := fdb.NewCallsign(in.Call)
	
	msg := adsb.CompositeMsg{
		Msg: adsb.Msg{
			Type: "MSG", // Default; ADSB.
			Icao24: adsb.IcaoId(in.Icao),
			GeneratedTimestampUTC: time.Unix(tSecs, 0).UTC(),
			Callsign: callsign.String(),
			Altitude: int64(in.Alt), // NOTE - GAlt !!
			GroundSpeed: int64(in.Spd),
			VerticalRate: int64(in.Vsi),
			Track: int64(in.Trak),
			Position: geo.Latlong{
				Lat:in.Lat,
				Long:in.Long,
			},
		},
		ReceiverName: fmt.Sprintf("%.0f", in.Rcvr),
	}

	if in.Mlat {
		msg.Type = "MLAT"
	}

	airframe := fdb.Airframe{
		Icao24: in.Icao,
		Registration: in.Reg,
		EquipmentType: in.Type,
	}

	// This is actually busted; it's for entire segments. Intermediate airports (such as, SFO)
	// are stashed in []in.Stops. It's not clear how to decide which leg is underway. Luckily,
	// we just overwrite all this from the schedule cache.
	schedule := fdb.Schedule{
		Origin: toIcaoAirport(in.From),
		Destination: toIcaoAirport(in.To),
	}

	if callsign.CallsignType == fdb.IcaoFlightNumber {
		airframe.CallsignPrefix = callsign.IcaoPrefix
		schedule.ICAO = callsign.IcaoPrefix // also in.OpIcao
		schedule.Number = callsign.Number
	}

	return airspace.AircraftData{
 		Msg: &msg,
		Airframe: airframe,
		Schedule: schedule,
		NumMessagesSeen: 1,
		Source: "AdsbExchange",
	}
}

// }}}
// {{{ db.FetchAirspace, FetchAirspace

func FetchAirspace(client *http.Client, box geo.LatlongBox) (*airspace.Airspace, error) {
	aex := AdsbExchange{client}
	return aex.FetchAirspace(box)
}

func (aex AdsbExchange)FetchAirspace(box geo.LatlongBox) (*airspace.Airspace, error) {
	as := airspace.NewAirspace()

	ac,err := aex.LiveQuery(box)
	if err != nil {
		return nil,err
	}

	counts := map[string]int{}
	for _,a := range ac {
		// Fake up keys if needed, and prefix them.
		key := a.Icao
		if key == "" {
			// E.g. BC7[HWD:FUK]. Comes via fr24/T-F5M, so no IcaoID. Callsign is often equip type :/
			counts[a.Call]++
			key = fmt.Sprintf("X%s%02d", a.Call, counts[a.Call])
		}

		key = "QQ"+key // Prefix, to avoid collisions

		as.Aircraft[adsb.IcaoId(key)] = ToAircraftData(a)
	}
	
	return &as,nil
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
