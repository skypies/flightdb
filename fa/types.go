package fa

// {{{ FlightInfoExResponse

type FlightInfoExResponse struct {
	FlightInfoExResult FlightInfoExStruct `json:"FlightInfoExResult"`
}

type FlightInfoExStruct struct {
	Flights []FlightExStruct `json:"flights"`
	Nextoffset	int `json:"next_offset"`
}

type FlightExStruct struct {
	Actualarrivaltime	int `json:"actualarrivaltime"`
	Actualdeparturetime	int `json:"actualdeparturetime"`
	Aircrafttype	string `json:"aircrafttype"`
	Destination	string `json:"destination"`
	Destinationcity	string `json:"destinationCity"`
	Destinationname	string `json:"destinationName"`
	Diverted	string `json:"diverted"`
	Estimatedarrivaltime	int `json:"estimatedarrivaltime"`
	Faflightid	string `json:"faFlightID"`
	Filedairspeedkts	int `json:"filed_airspeed_kts"`
	Filedairspeedmach	string `json:"filed_airspeed_mach"`
	Filedaltitude	int `json:"filed_altitude"`
	Fileddeparturetime	int `json:"filed_departuretime"`
	Filedete	string `json:"filed_ete"`   // A duration string, e.g. "01:10:00"
	Filedtime	int `json:"filed_time"`     // The scheduled departure, as epoch
	Ident	string `json:"ident"`           // Callsign (ICAO3+Fnumber, usually)
	Origin	string `json:"origin"`
	Origincity	string `json:"originCity"`
	Originname	string `json:"originName"`
	Route	string `json:"route"`
}

// }}}
// {{{ GetHistoricalTrackResponse

type GetHistoricalTrackResponse struct {
	GetHistoricalTrackResult GetHistoricalTrackResult `json:"GetHistoricalTrackResult"`
}

type GetHistoricalTrackResult struct {
	Track []TrackStruct `json:"data"`
}

type TrackStruct struct {
	Altitude	int `json:"altitude"`
	Altitudechange	string `json:"altitudeChange"`
	Altitudestatus	string `json:"altitudeStatus"`
	Groundspeed	int `json:"groundspeed"`
	Latitude	float64 `json:"latitude"`
	Longitude	float64 `json:"longitude"`
	Timestamp	int `json:"timestamp"`
	Updatetype	string `json:"updateType"`
}

/* http://discussions.flightaware.com/post143420.html
 *
 * "TO" is oceanic position, "TP" is projected, "TZ" is radar position, "TA" is ADS-B position.
 * You really only care that it is not "TP".
 * 
 * The leading "T" is stripped from some of the return values for brevity.
 */
func (ts TrackStruct)DataCanBeTrusted() bool {
	return (ts.Updatetype == "TA" || ts.Updatetype == "A")
}

// }}}
// {{{ ArrivedResponse

type ArrivedResponse struct {
	ArrivedResult ArrivedStruct `json:"ArrivedResult"`
}

type ArrivedStruct struct {
	Arrivals []ArrivalFlightStruct `json:"arrivals"`
	Nextoffset	int `json:"next_offset"`
}

type ArrivalFlightStruct struct {
	Actualarrivaltime	int `json:"actualarrivaltime"`
	Actualdeparturetime	int `json:"actualdeparturetime"`
	Aircrafttype	string `json:"aircrafttype"`
	Destination	string `json:"destination"`
	Destinationcity	string `json:"destinationCity"`
	Destinationname	string `json:"destinationName"`
	Ident	string `json:"ident"`           // Callsign (ICAO3+Fnumber, usually)
	Origin	string `json:"origin"`
	Origincity	string `json:"originCity"`
	Originname	string `json:"originName"`
}

// }}}
// {{{ SearchResponse

type SearchResponse struct {
	SearchResult SearchStruct `json:"SearchResult"`
}

type SearchStruct struct {
	Aircraft []InFlightStruct `json:"aircraft"`
	Nextoffset	int `json:"next_offset"`
}

type InFlightStruct struct {
	FaFlightID        string  `json:"faFlightID"`        // "AAL209-1452816900-schedule-0000"
	Ident             string  `json:"ident"`             // "AAL209"
	Prefix            string  `json:"prefix"`            // ""
	EquipType         string  `json:"type"`              // "B738"
	Suffix            string  `json:"suffix"`            // "L"
	Origin            string  `json:"origin"`            // "KLAX"
	Destination       string  `json:"destination"`       // "KSFO"
	Timeout           string  `json:"timeout"`           // "ok"
	Timestamp         int     `json:"timestamp"`         // 1452993062
	DepartureTime     int     `json:"departureTime"`     // 1452990000
	FirstPositionTime int     `json:"firstPositionTime"` // 1452990179
	ArrivalTime       int     `json:"arrivalTime"`       // 0
	Longitude         float64 `json:"longitude"`         // -122.11666999999999916
	Latitude          float64 `json:"latitude"`          // 37.516669999999997742
	LowLongitude      float64 `json:"lowLongitude"`      // -122.11666999999999916
	LowLatitude       float64 `json:"lowLatitude"`       // 33.876390000000000668
	HighLongitude     float64 `json:"highLongitude"`     // -118.43332999999999799
	HighLatitude      float64 `json:"highLatitude"`      // 37.516669999999997742
	Groundspeed       int     `json:"groundspeed"`       // 177
	Altitude          int     `json:"altitude"`          // 44
	Heading           int     `json:"heading"`           // 332
	AltitudeStatus    string  `json:"altitudeStatus"`    // "C"
	UpdateType        string  `json:"updateType"`        // "TZ"
	AltitudeChange    string  `json:"altitudeChange"`    // "D"
	Waypoints         string  `json:"waypoints"`         // "33.933 -118.4 34"
}

// }}}

// {{{ FirehoseMessage

// See examples in https://flightaware.com/commercial/firehose/firehose_documentation.rvt
// A union type of five message flavors; we're only interested in the position flavor
type FirehoseMessage struct {
	Type              string  `json:"type"` // {position,flightplan,departure,arrival,cancellation}

	//// Fields for a Position Message, found in example prof
	Registration       string  `json:"reg"`        // N739MA   
	Squawk             string  `json:"squawk"`     // 1754
	Ident              string  `json:"ident"`      // BSK323
	GroundSpeed        int     `json:"gs"`         // 252
	TimestampEpoch     int     `json:"clock"`      // 1419271753
	AirOrGround        string  `json:"air_ground"` // A
	IcaoId             string  `json:"hexid"`      // A9ED8E
	Heading            int     `json:"heading"`    // 157
	Updatetype         string  `json:"updateType"` // A
	Lat                float64 `json:"lat"`        // 38.71405
	Long               float64 `json:"lon"`        // -92.22064
	Altitude           int     `json:"alt"`        // 4800
	FlightawareId      string  `json:"id"`         // BSK323-1419245889-42-0

	//// Extra fields found in docs
  ReceiverName       string `json:"facility_name"`
  ReceiverId         string `json:"facility_hash"`

	AirSpeed           int    `json:"airspeed_kts"` // Exciting !! Does anything populate it ?
	AltitudeBarometric int    `json:"baro_alt"`
	AltitudeGPS        int    `json:"gps_alt"`
	ETAEpoch           int    `json:"eta"`
}

// Position.UpdateType: Specifies source of message A for ADS-B, Z for
// radar, O for transoceanic, P for estimated, D for datalink, M for
// Multilateration (MLAT)

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
