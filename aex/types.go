package aex

type LiveQueryResponse struct {
	Aircraft []AExAircraft `json:"acList"`
}

// https://www.adsbexchange.com/datafields/
type AExAircraft struct {
	Alt      float64 `json:Alt`      // 8550
	AltT     float64 `json:AltT`     // 0
	Bad      bool    `json:Bad`      // false
	Brng     float64 `json:Brng`     // 321.7
	CMsgs    float64 `json:CMsgs`    // 123
	CNum     string  `json:CNum`     // "33459"
	Call     string  `json:Call`     // "UAL1572"
	CallSus  bool    `json:CallSus`  // false
	Cou      string  `json:Cou`      // "United States"
	Dst      float64 `json:Dst`      // 8.71
	EngMount float64 `json:EngMount` // 0
	EngType  float64 `json:EngType`  // 3
	Engines  string  `json:Engines`  // "2"
	FSeen    string  `json:FSeen`    // "/Date(1505618493755)/"
	FlightsCount  float64 `json:FlightsCount` // 0
	From     string  `json:From`     // "KBOS General Edward Lawrence Logan, Boston, United States"
	GAlt     float64 `json:GAlt`     // 8514
	Gnd      bool    `json:Gnd`      // false
	HasPic   bool    `json:HasPic`   // false
	HasSig   bool    `json:HasSig`   // true
	Help     bool    `json:Help`     // false
	Icao     string  `json:Icao`     // "AAA5AE"
	Id       float64 `json:Id`       // 1.1183534e+07
	InHg     float64 `json:InHg`     // 29.8844624
	Interested  bool `json:Interested` // false
	Lat      float64 `json:Lat`      // 37.680267
	Long     float64 `json:Long`     // -122.436842
	Man      string  `json:Man`      // "Boeing"
	Mdl      string  `json:Mdl`      // "2008 BOEING 737-824"
	Mil      bool    `json:Mil`      // false
	Mlat     bool    `json:Mlat`     // false
	Op       string  `json:Op`       // "United Airlines"
	OpIcao   string  `json:OpIcao`   // "UAL"
	PosTime  float64 `json:PosTime`  // 1.50561864888e+12
	Rcvr     float64 `json:Rcvr`     // 1
	Reg      string  `json:Reg`      // "N78511"
	Sig      float64 `json:Sig`      // 69
	Spd      float64 `json:Spd`      // 268.6
	SpdTyp   float64 `json:SpdTyp`   // 0
	Species  float64 `json:Species`  // 1
	Sqk      string  `json:Sqk`      // "3261"
	Stops  []string  `json:Stops`    // ["KSFO San Francisco, United States"]
	TAlt     float64 `json:TAlt`     // 19008
	TSecs    float64 `json:TSecs`    // 155
	TTrk     float64 `json:TTrk`     // 345.9375
	Tisb     bool    `json:Tisb`     // false
	To       string  `json:To`       // "KSAN San Diego, United States"
	Trak     float64 `json:Trak`     // 196.9
	TrkH     bool    `json:TrkH`     // false
	Trt      float64 `json:Trt`      // 5
	Type     string  `json:Type`     // "B738"
	Vsi      float64 `json:Vsi`      // 3712
	VsiT     float64 `json:VsiT`     // 1
	WTC      float64 `json:WTC`      // 2
	Year     string  `json:Year`     // "2008"
}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
