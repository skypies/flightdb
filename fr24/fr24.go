package fr24

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"time"

	"github.com/skypies/adsb"
	fdb "github.com/skypies/flightdb"
	"github.com/skypies/pi/airspace"
	"github.com/skypies/geo"
)

const(
	kURLBalancers      = "www.flightradar24.com/balance.json"
	kURLQuery          = "www.flightradar24.com/v1/search/web/find"
	kURLPlaybackTrack  = "mobile.api.fr24.com/common/v1/flight-playback.json"
  kURLHistoryList    = "api.flightradar24.com/common/v1/flight/list.json"
)

var ErrNotInLiveDB = fmt.Errorf("No longer in live DB")
var ErrNotFound = fmt.Errorf("Not found anywhere")
var ErrBadInput = fmt.Errorf("Not enough data to work with")

// {{{ Fr24{}

type Fr24 struct {
	Client *http.Client
	host    string
	Prefix  string
}

func NewFr24(c *http.Client) (*Fr24, error) {
	if c == nil {
		c = &http.Client{}
	}
	db := Fr24{Client: c, host:"data-live.flightradar24.com"}
	return &db, nil
}

// }}}
// {{{ Get*Url

// id is a fr24 internal hex ID
func (fr *Fr24) GetPlaybackTrackUrl(id string) string {
	return fmt.Sprintf("%s%s?flightId=%s", fr.Prefix, kURLPlaybackTrack, id)
}

// adsb=1&mlat=1&flarm=1&faa=1&estimated=1&air=1&gnd=1&vehicles=1&gliders=1&array=1
//  faa=1 - includes T-F5M flights, which aren't always FAA; use epoch to judge freshness
func (fr *Fr24) GetCurrentListUrl(bounds string) string {
	return fmt.Sprintf("%s/zones/fcgi/feed.json?array=1&bounds=%s&faa=1", fr.host, bounds)
}
func (fr *Fr24) GetCurrentDetailsUrl(id string) string {
	return fmt.Sprintf("%s/_external/planedata_json.1.3.php?f=%s", fr.host, id)
}
func (fr *Fr24) GetQueryUrl(query string) string {
	return fmt.Sprintf("%s?query=%s&limit=8", kURLQuery, query)
}
func (fr *Fr24) GetLookupHistoryUrl(reg, iataFlightNumber string) string {
	if reg != "" {
		return fmt.Sprintf("%s%s?query=%s&fetchBy=reg", fr.Prefix, kURLHistoryList, reg)
	} else {
		return fmt.Sprintf("%s%s?query=%s&fetchBy=flight", fr.Prefix, kURLHistoryList, iataFlightNumber)
	}
}

// }}}

// {{{ fr24.url2{resp,body,jsonMap}

func (fr *Fr24) url2resp(url string) (resp *http.Response, err error) {
	if resp,err = fr.Client.Get("https://" + url); err != nil {
		return
	}
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf ("Bad status: %v", resp.Status)
	}
	return
}

func (fr *Fr24) Url2Body(url string) (body []byte, err error) {
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

	//i.Registration = flight.Aircraft.Identification.Registration
	i.IcaoId       = flight.Aircraft.Identification.ModeS
	i.Callsign     = fdb.NewCallsign(id.Callsign).String()

	return nil
}

// }}}
// {{{ currentListEntry2FlightIdentity

func currentListEntry2FlightIdentity(v []interface{}, id *fdb.Identity) {
	id.Origin, id.Destination = v[12].(string), v[13].(string)

	if id.ForeignKeys == nil { id.ForeignKeys = map[string]string{} }
	id.ForeignKeys["fr24"] = v[0].(string)

  id.IcaoId = v[1].(string)
	id.Callsign = fdb.NewCallsign(v[17].(string)).String()

	if flightnumber := v[14].(string); flightnumber != "" {	
		// FR24 copies callsigns of the form {[A-Z][0-9]+} into the flightnumber field. Undo that.
		if ! regexp.MustCompile("^[CN][0-9]+$").MatchString(flightnumber) {
			id.ParseIata(flightnumber) // Ignore errors
		}
	}
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

// ["cc51d4f",  "AD41CD",   36.3006,   -121.2479,  326,
//  22216,      376,        "1064",    "T-MLAT2",  "B737"
//  "N953WN",   1489779000, "LAX",     "SAT",      "WN1699",
//  0,          -2176,      "SWA1699", 0]

func (db *Fr24)ParseCurrentList(body []byte) ([]fdb.FlightSnapshot, error) {
	jsonMap := map[string]interface{}{}
	if err := json.Unmarshal(body, &jsonMap); err != nil { return nil, err }
	
	// Unpack the aircraft summary object
	ret := []fdb.FlightSnapshot{}
	for _,vRaw := range jsonMap["aircraft"].([]interface{}) {
		v := vRaw.([]interface{})
		fs := fdb.FlightSnapshot{
			Flight: fdb.Flight{
				Airframe: fdb.Airframe{
					Registration: v[10].(string),
					EquipmentType: v[9].(string),
				},
			},
			Trackpoint: fdb.Trackpoint{
				DataSource:    "fr24",
				ReceiverName:  v[8].(string),  // e.g. "T-MLAT", or "T-F5M"
				TimestampUTC:  time.Unix(int64(v[11].(float64)), 0).UTC(),
				Heading:       v[4].(float64),
				Latlong:       geo.Latlong{v[2].(float64), v[3].(float64)},
				GroundSpeed:   v[6].(float64),
				VerticalRate:  v[16].(float64),
				Altitude:      v[5].(float64),
				Squawk:        v[7].(string),
			},
		}

		// The flightnumber, if present, takes precedence over any number we parse out of the
		// callsign.
		currentListEntry2FlightIdentity(v,&fs.Flight.Identity)
		fs.Flight.ParseCallsign()
		
		ret = append(ret, fs)
	}

	return ret,nil
}

// }}}
// {{{ db.ParseCurrentDetails

func (db *Fr24)ParseCurrentDetails(body []byte) (*CurrentDetailsResponse, error) {
	jsonMap := map[string]interface{}{}
	if err := json.Unmarshal(body, &jsonMap); err != nil { return nil, err }

	// This block has panic()ed - interface conversion: interface is nil, not string
	ld := CurrentDetailsResponse{		
		FlightNumber: jsonMap["flight"].(string),
		Status: jsonMap["status"].(string),
		ScheduledDepartureUTC: time.Unix(int64(jsonMap["dep_schd"].(float64)), 0).UTC(),
		ScheduledArrivalUTC:time.Unix(int64(jsonMap["arr_schd"].(float64)), 0).UTC(),
		OriginTZOffset:jsonMap["from_tz_offset"].(string),
		DestinationTZOffset:jsonMap["to_tz_offset"].(string),
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
	
	// Note: need track before we parse flight identifier (it needs a timestamp from the track data)
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
		Airframe: fdb.Airframe{
			Registration:  r.Result.Response.Data.Flight.Aircraft.Identification.Registration,
			EquipmentType: r.Result.Response.Data.Flight.Aircraft.Model.Code,
		},
	}
	f.Tracks = map[string]*fdb.Track{}
	f.Tracks["fr24"] = &track
	
	if err := playbackTrack2FlightIdentity(r, &f.Identity); err != nil { return nil, err }

	f.ParseCallsign()
	
	return &f, nil
}

// }}}

// {{{ db.LookupCurrentList

// LookCurrentList returns a snapshot of what's currently in the box.
// This is used to populate the mapview, with just enough info for tooltips on the aircraft.
func (db *Fr24)LookupCurrentList(box geo.LatlongBox) ([]fdb.FlightSnapshot, error) {
	bounds := fmt.Sprintf("%.3f,%.3f,%.3f,%.3f", box.NE.Lat, box.SW.Lat, box.SW.Long, box.NE.Long)

	if body,err := db.Url2Body(db.GetCurrentListUrl(bounds)); err != nil {
		return nil, err
	} else {
		//fmt.Printf("---Body---\n%s\n-------\n", body)
		return db.ParseCurrentList(body)
	}
}

// }}}
// {{{ db.LookupCurrentDetails

// LookupCurrentDetails gets just a few details about a flight currently in the air.
// It's what pops up in the panel when you click on a plane on the map.
func (db *Fr24)LookupCurrentDetails(fr24Id string) (*CurrentDetailsResponse, error) {
	if body,err := db.Url2Body(db.GetCurrentDetailsUrl(fr24Id)); err != nil {
		return nil, err
	} else {
		return db.ParseCurrentDetails(body)
	}
}

// }}}
// {{{ db.LookupPlaybackTrack

// Given an fr24Id, this call fetches its flight track.
func (db *Fr24)LookupPlaybackTrack(fr24Id string) (*fdb.Flight, error) {
	if body,err := db.Url2Body(db.GetPlaybackTrackUrl(fr24Id)); err != nil {
		return nil, err
	} else {
		return db.ParsePlaybackTrack(body)
	}
}

// }}}
// {{{ db.LookupQuery

// This is the as-you-type instant results thing in the mainquery box. It's nice and cheap.
// If you're searching for a callsign/flightnumber/registration that is "live", you can get
// the fr24Id foriegn key. Things are only live for up to 10m after they land though.
func (db *Fr24)LookupQuery(q string) (Identifier, error) {
	body,err := db.Url2Body(db.GetQueryUrl(q))
	if err != nil { return Identifier{},err }

	resp := QueryResponse{}
	if err := json.Unmarshal(body, &resp); err != nil { return Identifier{},err }
	
	for _,r := range resp.Results {
		if r.Type == "live" {
			return Identifier{
				Fr24:r.Id,
				Reg:r.Detail.Reg,
				Callsign:r.Detail.Callsign,
				IATAFlightNumber:r.Detail.Flight,
			}, nil
		}
	}
	return Identifier{}, ErrNotInLiveDB
}

// }}}
// {{{ db.LookupHistory

// This is the heavyweight history lookup, giving schedule data for all legs flown:
// (a) given a registration ID, lists all flights it has flown recently;
// (b) given an IATA flightnumber, lists all instances of that flight (with various registrations)
//
// http://www.flightradar24.com/reg/n980uy
// http://www.flightradar24.com/flight/aa1799

// http://www.flightradar24.com/data/airplanes/hp-1830cmp  ???

func (db *Fr24)LookupHistory(reg,iataflightnumber string) ([]Identifier, error) {
	resp := LookupHistoryResponse{}

	if body,err := db.Url2Body(db.GetLookupHistoryUrl(reg,iataflightnumber)); err != nil {
		return nil, err
	} else if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	out := []Identifier{}
	for _, row := range resp.Result.Response.Data {
		out = append(out, Identifier{
			Fr24: row.Identification.Id,
			IATAFlightNumber: row.Identification.Number.Default,
			Callsign: row.Identification.Callsign,
			Reg: row.Aircraft.Registration,
			Orig: row.Airport.Origin.Code.Iata,
			Dest: row.Airport.Destination.Code.Iata,
			DepartureEpoch: int64(row.Time.Scheduled.Departure),
			DepartureTimeLocation: row.Airport.Origin.Timezone.Name,
		})
	}
	
	//for _,id := range out { fmt.Printf("* %s\n", id)}

	return out, nil
}

// }}}

// {{{ db.GetFr24Id

func (db *Fr24)GetFr24Id(f *fdb.Flight) (string, string, error) {
	str := fmt.Sprintf("** ID lookup for %v\n", f.IdentityString())

	if f.Airframe.Registration == "" {
		return "", str+"** flight had no registration\n", ErrBadInput
	//} else if len(f.Identity.Registration) > 7 {
	//	return "", str+"** flight's registration was too long or fr24\n", ErrBadInput
	}

	// The callsign as observed via ADS-B can be a poor match for the post-processed one that
	// fr24 usess; so we should post-process too.
	callsign := f.NormalizedCallsignString()
	
	// Approach 1: fast, not always avail
	str += fmt.Sprintf("** url1: %s\n", db.GetQueryUrl(f.Registration))
	id,err := db.LookupQuery(f.Registration)	
	if err == nil {
		if fdb.CallsignStringsEqual(callsign, id.Callsign) {
			return id.Fr24, str+fmt.Sprintf("** found via query: %v\n", id), nil
		} else {
			str += fmt.Sprintf("** query had mismatch(it had %s, we had %s\n", id.Callsign, callsign)
		}
	} else if err != ErrNotInLiveDB {
		return "", str+"** lookup error\n", err
	}

	// Approach 2: slower
	str += fmt.Sprintf("** url2: %s\n", db.GetLookupHistoryUrl(f.Registration,""))
	ids,err := db.LookupHistory(f.Registration, "")
	if err != nil {
		return "", str, err
	}
	for _,id := range ids {
		if fdb.CallsignStringsEqual(callsign, id.Callsign) {
			return id.Fr24, str+fmt.Sprintf("** found via history: %v\n", id), nil
		}
	}
	return "", str+fmt.Sprintf("** not found at all"), ErrNotFound
}

// }}}

// {{{ SnapshotToAircraftData

// This would ideally be in flightdb2/snapshot.go, but pi/airspace depends on fdb
func SnapshotToAircraftData(fs fdb.FlightSnapshot) airspace.AircraftData {	
	msg := adsb.CompositeMsg{
		Msg: adsb.Msg{
			Type: "MSG", // Default; ADSB.
			Icao24: adsb.IcaoId(fs.IcaoId),
			GeneratedTimestampUTC: fs.Trackpoint.TimestampUTC,
			Callsign: fs.Flight.NormalizedCallsignString(),
			Altitude: int64(fs.Trackpoint.Altitude),
			GroundSpeed: int64(fs.Trackpoint.GroundSpeed),
			VerticalRate: int64(fs.Trackpoint.VerticalRate),
			Track: int64(fs.Trackpoint.Heading),
			Position: fs.Trackpoint.Latlong,
		},
		ReceiverName: fs.Trackpoint.ReceiverName,
	}

	af := fs.Flight.Airframe
	af.Icao24 = string(fs.IcaoId)

	// Hack up some fake 'message types' ...
	if fs.Trackpoint.DataSource == "fr24" {
		if tf5m := regexp.MustCompile("T-F5M").FindString(msg.ReceiverName); tf5m != "" {
			msg.Type = "T-F5M"
		} else if mlat := regexp.MustCompile("MLAT").FindString(msg.ReceiverName); mlat != "" {
			msg.Type = "MLAT"
		}
	}
	
	return airspace.AircraftData{
 		Msg: &msg,
		Airframe: af,
		Schedule: fs.Schedule,
		NumMessagesSeen: 1,
		Source: fs.Trackpoint.DataSource,
	}
}

// }}}
// {{{ db.FetchAirspace, FetchAirspace

func FetchAirspace(client *http.Client, box geo.LatlongBox) (*airspace.Airspace, error) {
	db,err := NewFr24(client)
	if err != nil {
		return nil,err
	}
	return db.FetchAirspace(box)
}

func (db *Fr24)FetchAirspace(box geo.LatlongBox) (*airspace.Airspace, error) {
	as := airspace.NewAirspace()

	snapshots,err := db.LookupCurrentList(box)
	if err != nil {
		return nil,err
	}

	counts := map[string]int{}
	for _,snap := range snapshots {
		if snap.Altitude < 10 { continue } // fr24 has lots of aircraft on the ground

		// An airspace usually uses IcaoID as a key. But we want to be able to support
		// an airspace containing skypi & fr24 data; so we prefix the fr24 data.
		// Plus, for fr24, we need to be able to support FAA data which has no IcaoID.
		// So, we use a distinct key, and pretend it is an IcaoID.
		key := snap.IcaoId
		if key == "" {
			// E.g. BC7[HWD:FUK]. Comes via fr24/T-F5M, so no IcaoID. Callsign is often equip type :/
			counts[snap.Callsign]++
			key = fmt.Sprintf("X%s%02d", snap.Callsign, counts[snap.Callsign])
		}
		key = "EE"+key // Prefix, to avoid collisions

		
		as.Aircraft[adsb.IcaoId(key)] = SnapshotToAircraftData(snap)
	}
	
	return &as,nil
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
