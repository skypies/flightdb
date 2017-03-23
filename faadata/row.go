package faadata

import(
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/skypies/geo"
	fdb "github.com/skypies/flightdb"
)

// {{{ notes

/* FAA data comes in gzip'ed CSV rows.

The headers vary, depending on the data dump; so we turn row into a
map from header name to value.

Much of the data looks like this:

[0]AIRCRAFT_ID, [1]FLIGHT_INDEX, [2]TRACK_INDEX,
  [3]SOURCE_FACILITY, [4]BEACON_CODE, [5]DEP_APRT, [6]ARR_APRT, [7]ACFT_TYPE,
  [8]LATITUDE, [9]LONGITUDE, [10]ALTITUDEx100ft,
  [11]TRACK_POINT_DATE_UTC, [12]TRACK_POINT_TIME_UTC

E.g.:

VOI902,2015020103105708,20150201065937NCT1024VOI902,
  NCT,1024,MMGL,OAK,A320,
  37.69849,-122.21049,1,
  20150201,07:24:04

More recent data has a USER_CLASS field (with values from [COG])

[0]AIRCRAFT_ID, [1]FLIGHT_INDEX, [2]TRACK_INDEX,
  [3]SOURCE_FACILITY, [4]BEACON_CODE, [5]DEP_APRT, [6]ARR_APRT, [7]ACFT_TYPE,
  [8]USER_CLASS, [9]LATITUDE, [10]LONGITUDE, [11]ALTITUDEx100ft,
  [12]TRACK_POINT_DATE_UTC, [13]TRACK_POINT_TIME_UTC

 */

// }}}

type RowReader struct {
	csvreader  *csv.Reader
	headers   []string
}

func NewRowReader(ioreader io.Reader) *RowReader {
	rdr := RowReader{
		csvreader: csv.NewReader(ioreader),
	}
	rdr.headers,_ = rdr.csvreader.Read() // Discard err, we'll get it when we try to get next row
	return &rdr
}

// {{{ rdr.Read()

func (r *RowReader)Read() (Row,error) {
	m := map[string]string{}

	vals,err := r.csvreader.Read()
	if err != nil {
		return m,err
	} else if len(r.headers) != len(vals) {
		return m, fmt.Errorf("header/val mismatch (%d/%d)", len(r.headers), len(vals))
	}

	for i,_ := range vals {
		m[r.headers[i]] = vals[i]
	}

	return m,nil
}

// }}}

type Row map[string]string

// {{{ row.ToFlightSkeleton

func (r Row)ToFlightSkeleton() *fdb.Flight {	
	f := &fdb.Flight{
		Identity: fdb.Identity{
			Callsign: r["AIRCRAFT_ID"],
			ForeignKeys: map[string]string{
				"FAA": r["TRACK_INDEX"],
			},
			Schedule: fdb.Schedule{
				Origin: r["DEP_APRT"],
				Destination: r["ARR_APRT"],
			},
		},
		Airframe: fdb.Airframe{
			EquipmentType: r["ACFT_TYPE"],
		},
		Tracks: map[string]*fdb.Track{},
		Tags: map[string]int{},
		Waypoints: map[string]time.Time{},
	}

	f.ParseCallsign()

	return f
}

// }}}
// {{{ row.ToTrackpoint

func (r Row)ToTrackpoint() fdb.Trackpoint {
	lat,_  := strconv.ParseFloat(r["LATITUDE"], 64)
	long,_ := strconv.ParseFloat(r["LONGITUDE"], 64)
	alt,_  := strconv.ParseFloat(r["ALTITUDEx100ft"], 64)

	tStr := fmt.Sprintf("%s %s UTC", r["TRACK_POINT_DATE_UTC"], r["TRACK_POINT_TIME_UTC"])
	t,_ := time.Parse("20060102 15:04:05 MST", tStr)
	
	tp := fdb.Trackpoint{
		DataSource:    "EB-FOIA", // Make configurable ?
		TimestampUTC:  t,
		Latlong:       geo.Latlong{Lat:lat, Long:long},
		Altitude:      alt * 100.0,
		Squawk:        r["BEACON_CODE"],
	}

	return tp
}

// }}}
// {{{ row.FromSameFlightAs

// Some FOIA data has data that looks like this on consecutive lines ...

// 376147: QXE17,2016051028797150,20160510235032NCT6624QXE17,NCT,6624,EUG,SJC,DH8D,37.34841,-121.91391,3,20160511,00:40:59
// 376148: QXE17,2016051028797150,20160510235032NCT6624QXE17,NCT,6624,EUG,SJC,DH8D,37.35002,-121.91558,3,20160511,00:41:04
// 376149: QXE17,2016051028735155,20160510011647NCT4514QXE17,NCT,4514,SJC,RNO,DH8D,37.36278,-121.92703,6,20160510,01:16:47
// 376150: QXE17,2016051028735155,20160510011647NCT4514QXE17,NCT,4514,SJC,RNO,DH8D,37.3649,-121.92945,9,20160510,01:16:52

// ... so the flight number isn't enough to disambiguate distinct
// flights. Use the FAA's FLIGHT_INDEX value; if that also changes,
// then it's a separate flight, even if the flightnumber doesn't
// change.

func (r1 Row)FromSameFlightAs(r2 Row) bool {
	return r1["AIRCRAFT_ID"] == r2["AIRCRAFT_ID"] && r1["FLIGHT_INDEX"] == r2["FLIGHT_INDEX"]
}

// }}}


// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
