package fr24

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	fdb "github.com/skypies/flightdb2"
	"github.com/skypies/geo"
)

const(
	kURLBalancers      = "www.flightradar24.com/balance.json"
	kURLQuery          = "www.flightradar24.com/v1/search/web/find"
	kURLPlaybackTrack  = "mobile.api.fr24.com/common/v1/flight-playback.json"
)

// {{{ Fr24{}

type Fr24 struct {
	Client *http.Client  // Or just a context ??
	host    string
	Prefix  string
}

func (db *Fr24)Init() error {
	//if err := db.EnsureHostname(); err != nil {	return err }
	return nil
}

func NewFr24(c *http.Client) (*Fr24, error) {
	if c == nil {
		c = &http.Client{}
	}
	db := Fr24{Client: c, host:"krk.data.fr24.com"}
	err := db.Init()
	return &db, err
}

// }}}
// {{{ Get*Url

// id is a fr24 internal hex ID
func (fr *Fr24) GetPlaybackTrackUrl(id string) string {
	return fmt.Sprintf("%s%s?flightId=%s", fr.Prefix, kURLPlaybackTrack, id)
}
func (fr *Fr24) GetCurrentListUrl(bounds string) string {
	return fmt.Sprintf("%s/zones/fcgi/feed.json?array=1&bounds=%s", fr.host, bounds)
}
func (fr *Fr24) GetCurrentDetailsUrl(id string) string {
	return fmt.Sprintf("%s/_external/planedata_json.1.3.php?f=%s", fr.host, id)
}
func (fr *Fr24) GetQueryUrl(query string) string {
	return fmt.Sprintf("%s?query=%s&limit=8", kURLQuery, query)
}

// }}}

// {{{ fr24.url2{resp,body,jsonMap}

func (fr *Fr24) url2resp(url string) (resp *http.Response, err error) {
	if resp,err = fr.Client.Get("http://" + url); err != nil {
		return
	}
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf ("Bad status: %v", resp.Status)
	}
	return
}

func (fr *Fr24) url2body(url string) (body []byte, err error) {
	if resp,err := fr.url2resp(url); err != nil {
		return nil, err
	} else {
		defer resp.Body.Close()
		return ioutil.ReadAll(resp.Body)
	}
}

func (fr *Fr24) url2jsonMap(url string) (jsonMap map[string]interface{}, err error) {
	resp,err2 := fr.url2resp(url)
	if err2 != nil { err = err2; return }
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&jsonMap)
	return
}

// }}}
// {{{ fr24.{Ensure,get}Hostname

// Ask the load balancer which host to use
// http://blog.cykey.ca/post/88174516880/analyzing-flightradar24s-internal-api-structure
func (fr *Fr24) getHostname() error {
	jsonMap,err := fr.url2jsonMap(kURLBalancers)
	if err != nil { return err }

	min := 99999.0
	fr.host = ""
	for k,v := range jsonMap {
		score := v.(float64)
		if (score < min) {
			fr.host,min = k,score
		}
	}

	return nil
}

// {"krk.data.fr24.com":250,"bma.data.fr24.com":250,"arn.data.fr24.com":250,"lhr.data.fr24.com":250}
func (fr *Fr24) EnsureHostname() error {
	if fr.host == "" {
		fr.host = "krk.data.fr24.com"
		return nil
		/*
		if err := fr.getHostname(); err != nil {
			return err
		}
*/
	}
	return nil
}

// }}}

// {{{ playbackTrack2FlightIdentity

func playbackTrack2FlightIdentity(r FlightPlaybackResponse, i *fdb.Identity) error {
	flight := r.Result.Response.Data.Flight
	id := flight.Identification

	if id.Number.Default != "" {
		if err := i.ParseIata(id.Number.Default); err != nil { return err }
	}
	
	i.Origin,i.Destination = flight.Airport.Origin.Code.Iata, flight.Airport.Destination.Code.Iata

	// It would be preferable to find the scheduled date of departure for this; else we
	// misidentify those flights which get delayed past midnight.
	/*
	if flight.Track == nil || len(flight.Track) == 0 {
		return fmt.Errorf("playback2FightIdentifier: no track data, so no departure date!")
	}
	epoch := flight.Track[0].Timestamp
	f.DepartureDate = date.InPdt(time.Unix(int64(epoch), 0))
*/
	
	if i.ForeignKeys == nil { i.ForeignKeys = map[string]string{} }
	i.ForeignKeys["fr24"] = id.Hex

	i.Registration = flight.Aircraft.Identification.Registration
	i.IcaoId       = flight.Aircraft.Identification.ModeS
	i.Callsign     = id.Callsign

	i.ParseCallsign()

	return nil
}

// }}}
// {{{ currentListEntry2FlightIdentity

func currentListEntry2FlightIdentity(v []interface{}, id *fdb.Identity) error {
	id.Origin,id.Destination = v[12].(string), v[13].(string)

	if id.ForeignKeys == nil { id.ForeignKeys = map[string]string{} }
	id.ForeignKeys["fr24"] = v[0].(string)

	id.Registration = v[10].(string)
  id.IcaoId = v[1].(string)
	id.Callsign = v[17].(string)

	id.Schedule.PlannedDepartureUTC = time.Unix(int64(v[11].(float64)), 0)

	id.ParseCallsign()

	if v[14].(string) != "" {
		if err := id.ParseIata(v[14].(string)); err != nil {
			//return err
		}
	}
	
	return nil
}

/* We see three different flavors of record:

1. Normal scheduled flights
["7624382","AC7BF6",37.7370,-122.4019,195,6775,269,"3253","T-KSFO1","CRJ2","N903SW",1441900518,"SFO","BFL","UA5613",0,2176,"",0]
["76319bb","A6E88B",37.6254,-122.3963,276,74,9,    "1414","T-MLAT2","B752","N544UA",1441940807,"OGG","SFO","UA738", 1,0,   "UAL738",0]

2. Unscheduled flights, but with ModeS and registration
["7638091","A8A763",37.6081,-122.3855,197,74,7,    "6337","T-MLAT2","B762","N657GT",1441940842,"","","",            1,0,   "",0]
["76375b8","A1B8B8",37.6351,-122.3929,332,100,10,  "4262","T-MLAT7","B190","N21RZ", 1441940793,"","","",            1,0,   "",0]

3. Anonymous flights, with nothing but a crappy callsign (private jets / general aviation ?)
["7624195","",      37.6762,-122.5215,275,4143,142,"3347","T-MLAT2","GLF4","",      1441900519,"","","",            0,2048,"GLF4",0]
["76226db","",      37.6278,-122.3826,163,0,0,     "3226","F-KSFO1","E55P","",      1441900520,"","","",            1,0,   "E55P",0]
*/
/*
		a := Aircraft{
			Id: v.([]interface{})[0].(string),
			Id2: v.([]interface{})[1].(string),            // ModeS
			Lat: v.([]interface{})[2].(float64),
			Long: v.([]interface{})[3].(float64),
			Track: v.([]interface{})[4].(float64),
			Altitude: v.([]interface{})[5].(float64),
			Speed: v.([]interface{})[6].(float64),
			Squawk: v.([]interface{})[7].(string),
			Radar: v.([]interface{})[8].(string),
			EquipType: v.([]interface{})[9].(string),
			Registration: v.([]interface{})[10].(string),
			Epoch: v.([]interface{})[11].(float64),
			Origin: v.([]interface{})[12].(string),
			Destination: v.([]interface{})[13].(string),
			FlightNumber: v.([]interface{})[14].(string),
			Unknown: v.([]interface{})[15].(float64),     // looks like a binary flag
			VerticalSpeed: v.([]interface{})[16].(float64),
			Callsign: v.([]interface{})[17].(string),
			Unknown2: v.([]interface{})[18].(float64),
		}
		*aircraft = append(*aircraft,a)
*/

// }}}
	
// {{{ db.ParseCurrentList

func (db *Fr24)ParseCurrentList(body []byte) ([]fdb.FlightSnapshot, error) {
	jsonMap := map[string]interface{}{}
	if err := json.Unmarshal(body, &jsonMap); err != nil { return nil, err }

	// Unpack the aircraft summary object
	ret := []fdb.FlightSnapshot{}
	for _,vRaw := range jsonMap["aircraft"].([]interface{}) {
		v := vRaw.([]interface{})
		fs := fdb.FlightSnapshot{
			Flight: fdb.Flight{
				EquipmentType: v[9].(string),
			},
			Trackpoint: fdb.Trackpoint{
				DataSource:    "fr24:"+v[8].(string),
				TimestampUTC:  time.Unix(int64(v[11].(float64)), 0).UTC(),
				Heading:       v[4].(float64),
				Latlong:       geo.Latlong{v[2].(float64), v[3].(float64)},
				GroundSpeed:   v[6].(float64),
				Altitude:      v[5].(float64),
				Squawk:        v[7].(string),
			},
		}

		if err := currentListEntry2FlightIdentity(v,&fs.Flight.Identity); err != nil { return nil, err }

		ret = append(ret, fs)
	}

	return ret,nil
}

// }}}
// {{{ db.ParseCurrentDetails

func (db *Fr24)ParseCurrentDetails(body []byte) (*LiveDetailsResponse, error) {
	jsonMap := map[string]interface{}{}
	if err := json.Unmarshal(body, &jsonMap); err != nil { return nil, err }

	ld := LiveDetailsResponse{		
		FlightNumber: jsonMap["flight"].(string),
		Status: jsonMap["status"].(string),
		ScheduledDepartureUTC: time.Unix(int64(jsonMap["dep_schd"].(float64)), 0).UTC(),
		ScheduledArrivalUTC:time.Unix(int64(jsonMap["arr_schd"].(float64)), 0).UTC(),
		ETAUTC:time.Unix(int64(jsonMap["eta"].(float64)), 0).UTC(),
	}
	
	return &ld, nil
}

// }}}
// {{{ db.ParsePlaybackTrack

// result.response.data.flight.{airline,identification,aircraft,airport,track}
// result.response.timestamp
// result.request

func (db *Fr24)ParsePlaybackTrack(body []byte) (*fdb.Flight, error) {
	r := FlightPlaybackResponse{}
	if err := json.Unmarshal(body, &r); err != nil { return nil, err }

	if r.Result.Response.Timestamp == 0 {
		return nil, fmt.Errorf("ParsePlayback: no response element (%s no longer exists)",
			r.Result.Request.FlightId)
	}
	
	// Note: we need the track before we can do the flight identifier (it needs a timestamp from the track data)
	track := fdb.Track{}
	for _,frame := range r.Result.Response.Data.Flight.Track {
		track = append(track, fdb.Trackpoint{
			DataSource: "fr24",
			TimestampUTC: time.Unix(int64(frame.Timestamp),0).UTC(),
			Heading: float64(frame.Heading),
			Latlong: geo.Latlong{float64(frame.Latitude),float64(frame.Longitude)},
			GroundSpeed: float64(frame.Speed.Kts),
			Altitude: float64(frame.Altitude.Feet),
			Squawk: frame.Squawk,
		})
	}
	
	f := fdb.Flight{
		EquipmentType: r.Result.Response.Data.Flight.Aircraft.Model.Code,
	}
	f.Tracks = map[string]*fdb.Track{}
	f.Tracks["fr24"] = &track
	
	if err := playbackTrack2FlightIdentity(r, &f.Identity); err != nil { return nil, err }

	return &f, nil
}

// }}}

// {{{ db.LookupCurrentList

// LookCurrentList returns a snapshot of what's currently in the box
func (db *Fr24)LookupCurrentList(box geo.LatlongBox) ([]fdb.FlightSnapshot, error) {
	if err := db.EnsureHostname(); err != nil {	return nil, err }
	bounds := fmt.Sprintf("%.3f,%.3f,%.3f,%.3f", box.NE.Lat, box.SW.Lat, box.SW.Long, box.NE.Long)

	if body,err := db.url2body(db.GetCurrentListUrl(bounds)); err != nil {
		return nil, err
	} else {
		//fmt.Printf("---Body---\n%s\n-------\n", body)
		return db.ParseCurrentList(body)
	}
}

// }}}
// {{{ db.LookupCurrentDetails

// LookupCurrentDetails gets some details about a flight currently in the air
func (db *Fr24)LookupCurrentDetails(fr24Id string) (*LiveDetailsResponse, error) {
	if body,err := db.url2body(db.GetCurrentDetailsUrl(fr24Id)); err != nil {
		return nil, err
	} else {
		return db.ParseCurrentDetails(body)
	}
}

// }}}
// {{{ db.LookupPlaybackTrack

func (db *Fr24)LookupPlaybackTrack(fr24Id string) (*fdb.Flight, error) {
	if body,err := db.url2body(db.GetPlaybackTrackUrl(fr24Id)); err != nil {
		return nil, err
	} else {
		f,err := db.ParsePlaybackTrack(body)
		
		// This has the scheduledDepartureUTC; but the timezone to deduce the date is within `body` :/
/*
		if ld,err := db.LookupLiveDetails(fr24Id); err2 != nil {
			return nil, err
		} else {
			// 
		}
*/
		return f,err
	}
}

// }}}
// {{{ db.LookupQuery

// LookupCurrentDetails gets some details about a flight currently in the air
func (db *Fr24)LookupQuery(q string) (Identifier, error) {
	body,err := db.url2body(db.GetQueryUrl(q))
	if err != nil { return Identifier{},err }

	resp := QueryResponse{}
	if err := json.Unmarshal(body, &resp); err != nil { return Identifier{},err }
	
	for _,r := range resp.Results {
		if r.Type == "live" {
			return Identifier{r.Id, r.Detail.Reg, r.Detail.Callsign, r.Detail.Flight}, nil
		}
	}
	return Identifier{}, fmt.Errorf("No live match found")
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
