package main

import(
	"compress/gzip"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/taskqueue"
	"google.golang.org/appengine/log"	

	"github.com/skypies/geo"
	"github.com/skypies/util/date"
	"github.com/skypies/util/widget"
	fdb "github.com/skypies/flightdb2"
	"github.com/skypies/flightdb2/fgae"
)

func init() {
	http.HandleFunc("/foia/load", foiaHandler)
	http.HandleFunc("/foia/enqueue", multiEnqueueHandler)
	http.HandleFunc("/foia/rm", rmHandler)
	http.HandleFunc("/foia/verify", verifyHandler)
}

// {{{ foiaHandler

// http://backend-dot-serfr0-fdb.appspot.com/foia/load?date=20160703

// Load up FOIA historical data from GCS, and add new flights into the DB
func foiaHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	//c,_ := context.WithTimeout(appengine.NewContext(r), 9*time.Minute)
	// db := FlightDB{C:c}

	date := r.FormValue("date")
	if date == "" {
		http.Error(w, "need 'date=20141231' arg", http.StatusInternalServerError)
		return
	}

	str,err := doStorageJunk(c, date)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(fmt.Sprintf("OK\n%s", str)))
}

// }}}
// {{{ multiEnqueueHandler

// eb-foia
// http://backend-dot-serfr0-fdb.appspot.com/foia/enqueue?date=range&range_from=2013/01/01&range_to=2016/06/24

// Writes them all into a batch queue
func multiEnqueueHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	str := ""

	s,e,_ := widget.FormValueDateRange(r)
	days := date.IntermediateMidnights(s.Add(-1 * time.Second),e) // decrement start, to include it
	url := "/foia/load"
	
	for i,day := range days {
		dayStr := day.Format("20060102")
		thisUrl := fmt.Sprintf("%s?date=%s", url, dayStr)
		
		t := taskqueue.NewPOSTTask(thisUrl, map[string][]string{})

		// Give ourselves a few minutes to get the tasks posted ...
		t.Delay = 2*time.Minute + time.Duration(i)*15*time.Second

		if _,err := taskqueue.Add(ctx, t, "bigbatch"); err != nil {
			log.Errorf(ctx, "multiEnqueueHandler: enqueue: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		str += " * posting for " + thisUrl + "\n"
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(fmt.Sprintf("OK, enqueued %d\n--\n%s", len(days), str)))
}

// }}}
// {{{ rmHandler

func rmHandler(w http.ResponseWriter, r *http.Request) {
	c,_ := context.WithTimeout(appengine.NewContext(r), 9*time.Minute)
	//c := appengine.NewContext(r)
	db := fgae.FlightDB{C:c}

	n := 0
	
	max := widget.FormValueInt64(r,"n")
	if max == 0 { max = 50000 }
	q := 	db.NewQuery().ByTags([]string{"FOIA"}).Query.Limit(int(max)).KeysOnly()

	tStart := time.Now()
	str := "starting ...\n\n"
	
	for {
		keys,err := q.GetAll(c,nil)
		if err != nil {
			http.Error(w, fmt.Sprintf("GetAll (n=%d): %s", n, err.Error()), http.StatusInternalServerError)
			return
		}
		str += fmt.Sprintf("Found %d keys\n", len(keys))

		if len(keys)==0 { break }

		maxRm := 400
		for len(keys)>maxRm {
			if err := datastore.DeleteMulti(c, keys[0:maxRm-1]); err != nil {
				str += fmt.Sprintf("keys remain: %d\n", len(keys))
				http.Error(w, err.Error()+"\n--\n"+str, http.StatusInternalServerError)
				return
			} else {
				n += maxRm
			}
			keys = keys[maxRm:]
		}
		if err = datastore.DeleteMulti(c, keys); err != nil {
			str += fmt.Sprintf("keys remain2: %d\n", len(keys))
			http.Error(w, err.Error()+"\n"+str, http.StatusInternalServerError)
			return
		} else {
			n += len(keys)
		}
	}

	str += "\nKeys all deleted :O\nTime taken: " + time.Since(tStart).String() + "\n"
	
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(fmt.Sprintf("OK\n%s", str)))
}

// }}}
// {{{ verifyHandler

// Examine FOIA historical data from GCS, but do nothing to the DB
func verifyHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	date := r.FormValue("date")
	if date == "" {
		http.Error(w, "need 'date=20141231' arg", http.StatusInternalServerError)
		return
	}
	
	str,err := doStorageJunk(c, date)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(fmt.Sprintf("OK\n%s", str)))
}

// }}}

// {{{ getCSVReader

func getCSVReader(ctx context.Context, bucketName, fileName string) (*csv.Reader, error) {
	client, err := storage.NewClient(ctx)
	if err != nil { return nil, err }

	bucket := client.Bucket(bucketName)
	gcsReader,err := bucket.Object(fileName).NewReader(ctx)

	if err != nil {
		return nil, fmt.Errorf("GCS-Open %s|%s: %v", bucketName, fileName, err)
	}
	gzipReader,err := gzip.NewReader(gcsReader)
	if err != nil {
		return nil, fmt.Errorf("GCS-Open+GZ %s|%s: %v", bucketName, fileName, err)
	}
	csvReader := csv.NewReader(gzipReader)

	return csvReader, nil
}

// }}}
// {{{ rowToFlightSkeleton

// [0]AIRCRAFT_ID, [1]FLIGHT_INDEX, [2]TRACK_INDEX,
//   [3]SOURCE_FACILITY, [4]BEACON_CODE, [5]DEP_APRT, [6]ARR_APRT, [7]ACFT_TYPE,
//   [8]LATITUDE, [9]LONGITUDE, [10]ALTITUDEx100ft,
//   [11]TRACK_POINT_DATE_UTC, [12]TRACK_POINT_TIME_UTC
// VOI902,2015020103105708,20150201065937NCT1024VOI902,
//   NCT,1024,MMGL,OAK,A320,
//   37.69849,-122.21049,1,
//   20150201,07:24:04

func rowToFlightSkeleton(row []string) *fdb.Flight {
	f := &fdb.Flight{
		Identity: fdb.Identity{
			Callsign: row[0],
			ForeignKeys: map[string]string{
				"FAA": row[2],
			},
			Schedule: fdb.Schedule{
				Origin: row[5],
				Destination: row[6],
			},
		},
		Airframe: fdb.Airframe{
			EquipmentType: row[7],
		},
		Tracks: map[string]*fdb.Track{},
		Tags: map[string]int{},
		Waypoints: map[string]time.Time{},
	}

	f.ParseCallsign()

	return f
}

// }}}
// {{{ rowToTrackpoint

func rowToTrackpoint(row []string) fdb.Trackpoint {
	lat,_  := strconv.ParseFloat(row[8], 64)
	long,_ := strconv.ParseFloat(row[9], 64)
	alt,_  := strconv.ParseFloat(row[10], 64)

	t,_ := time.Parse("20060102 15:04:05 MST", row[11] + " " + row[12] + " UTC")
	
	tp := fdb.Trackpoint{
		DataSource:    "EB-FOIA", // Sigh; this should have gone in as EB-FOIA
		TimestampUTC:  t,
		Latlong:       geo.Latlong{Lat:lat, Long:long},
		Altitude:      alt * 100.0,
		Squawk:        row[4],
	}

	return tp
}

// }}}

// {{{ rowsToFlight

func rowsToFlight(rows [][]string, debug string) (*fdb.Flight, error) {
	if len(rows) == 0 { return nil, fmt.Errorf("No rows!") }
//	str := fmt.Sprintf("%s : %d rows\n", rows[0][0], len(rows))

	t := fdb.Track{}
	for _,row := range rows {
		t = append(t, rowToTrackpoint(row))
	}

	sort.Sort(fdb.TrackByTimestampAscending(t))
	
	f := rowToFlightSkeleton(rows[0])
	f.Tracks["FOIA"] = &t
	f.SetTag("FOIA")

	f.Analyse()
	f.DebugLog += debug

	return f, nil
}

// }}}
// {{{ addFlight

func addFlight(ctx context.Context, rows [][]string, tStart time.Time, debug string) (string, error) {
	if len(rows) == 0 { return "", fmt.Errorf("No rows!") }
//	str := fmt.Sprintf("%s : %d rows\n", rows[0][0], len(rows))

	t := fdb.Track{}
	for _,row := range rows {
		t = append(t, rowToTrackpoint(row))
	}

	sort.Sort(fdb.TrackByTimestampAscending(t))
	
	f := rowToFlightSkeleton(rows[0])
	f.Tracks["FOIA"] = &t
	f.SetTag("FOIA")

	tStartAnalyse := time.Now()
	f.Analyse()
	f.DebugLog += debug

	f.DebugLog += fmt.Sprintf("** full load+parse: %dms (analyse: %dms)\n",
		time.Since(tStart).Nanoseconds() / 1000000,
		time.Since(tStartAnalyse).Nanoseconds() / 1000000)
	
	str := ""

	if true {// f.Callsign == "AAL1544" {
		db := fgae.FlightDB{C:ctx}
		if err := db.PersistFlight(f); err != nil {
			return "", err
		}
		// str += fmt.Sprintf("* %s %v %v\n", f.Callsign, f.TagList(), f.WaypointList())
	}
	
	return str,nil
}

// }}}
// {{{ rowsAreSameFlight

// The more recent FOIA data has data that looks like this on consecutive lines ...

// 376147: QXE17,2016051028797150,20160510235032NCT6624QXE17,NCT,6624,EUG,SJC,DH8D,37.34841,-121.91391,3,20160511,00:40:59
// 376148: QXE17,2016051028797150,20160510235032NCT6624QXE17,NCT,6624,EUG,SJC,DH8D,37.35002,-121.91558,3,20160511,00:41:04
// 376149: QXE17,2016051028735155,20160510011647NCT4514QXE17,NCT,4514,SJC,RNO,DH8D,37.36278,-121.92703,6,20160510,01:16:47
// 376150: QXE17,2016051028735155,20160510011647NCT4514QXE17,NCT,4514,SJC,RNO,DH8D,37.3649,-121.92945,9,20160510,01:16:52

// ... so the flight number isn't enough to disambiguate. So use the
// FAA's FLIGHT_INDEX value; if that also changes, then it's a
// separate flight, even if the flightnumber doesn't change.

func rowsAreSameFlight(r1, r2 []string) bool {
	return r1[0] == r2[0] && r1[1] == r2[1]
}

// }}}
// {{{ doStorageJunk

// PA naming: faa-foia     FOIA-2015-006790/Offload_track_table  /Offload_track_20150104.txt.gz
// RG naming: rg-foia      2014                                  /Offload_track_IFR_20140104.txt.gz
// EB naming: eastbay-foia eb-foia/2015                          /Offload_track_IFR_20150104.txt.gz
func doStorageJunk(ctx context.Context, date string) (string, error) {
	frags := regexp.MustCompile("^(\\d{4})").FindStringSubmatch(date)
	if len(frags) == 0 {
		return "", fmt.Errorf("date '%s' did not match YYYYMMDD", date)
	}

	dir := "eb-foia/" + frags[0]  // Strip off the year from the datestring
	bucketName := "eastbay-foia"

	tStart := time.Now()
	log.Infof(ctx, "FOIAUPLOAD starting %s (%s)", date, dir)
	
	client, err := storage.NewClient(ctx)
	if err != nil { return "",err }

	bucket := client.Bucket(bucketName)
	q := &storage.Query{
		//Delimiter: "/",
		Prefix: dir + "/Offload_track_IFR_"+date,
	}

	str := ""
	names := []string{}
	it := bucket.Objects(ctx, q)
	for {
    oa, err := it.Next()
    if err == iterator.Done {
			break
    }
    if err != nil {
			return "", fmt.Errorf("GCS-Readdir [gs://%s]%s': %v", bucketName, q.Prefix, err)
    }
		str += fmt.Sprintf("%8db %s {%s}\n", oa.Size, oa.Updated.Format("2006.01.02"), oa.Name)
		names = append(names, oa.Name)
	}

	nFlights := 0
	for _,filename := range names {
		str += fmt.Sprintf("Flights loaded from %s|%s\n", bucketName, filename)
		allDebug := fmt.Sprintf("Flights loaded from %s|%s", bucketName, filename)
		csvReader,err := getCSVReader(ctx, bucketName, filename)
		if err != nil {
			log.Errorf(ctx, "FOIAUPLOAD ERR/CSV %s %v", err)
			return "", err
		}

		csvReader.Read() // Discard header row

		rows := [][]string{}		
		i := 1
		tStart := time.Now()
		for {
			row,err := csvReader.Read()
			if err == io.EOF { break }
			if err != nil { return "", err }

			// If this row appears to be a different flight than the one we're accumulating, flush
			if len(rows)>0 && !rowsAreSameFlight(row, rows[0]) {
				thisDebug := fmt.Sprintf("%s:%d-%d", allDebug, i-len(rows), i-1)
				if deb,err := addFlight(ctx, rows, tStart, thisDebug); err != nil {
					log.Errorf(ctx, "FOIAUPLOAD ERR/Add %s %v\n%s", err, deb)
					return deb,err
				} else {
					str += deb
				}
				tStart = time.Now()
				rows = [][]string{}
				nFlights++
			}

			rows = append(rows, row)
			i++
		}

		if len(rows)>0 {
			thisDebug := fmt.Sprintf("%s:%d-%d", allDebug, i-len(rows), i-1)
			if deb,err := addFlight(ctx, rows, tStart, thisDebug); err != nil {
				log.Errorf(ctx, "FOIAUPLOAD ERR/Add %s %v\n%s", err, deb)
				return deb,err
			} else {
				str += deb
			}
			nFlights++
		}
		str += fmt.Sprintf("-- File read, %d rows\n", i)
	}

	str += fmt.Sprintf("-- %s all done, %d flights, took %s\n", date, nFlights, time.Since(tStart))
	log.Infof(ctx, "FOIAUPLOAD finished %s (%d flights added, took %s)\n%s",
		date, nFlights, time.Since(tStart), str)
	
	return str,nil
}

// }}}

// {{{ processFoiaDatafile

type FoiaFlightFunction func(context.Context, *fdb.Flight) (string, error)

// PA naming: faa-foia  FOIA-2015-006790/Offload_track_table  /Offload_track_20150104.txt.gz
// RG naming: rg-foia   2014                                  /txt.Offload_track_IFR_20140104.gz
func processFoiaDatafile(ctx context.Context, date string, foiaFunc FoiaFlightFunction) (string,error) {
	frags := regexp.MustCompile("^(\\d{4})").FindStringSubmatch(date)
	if len(frags) == 0 {
		return "", fmt.Errorf("date '%s' did not match YYYYMMDD", date)
	}

	dir := frags[0]
	bucketName := "rg-foia"

	tStart := time.Now()
	
	client, err := storage.NewClient(ctx)
	if err != nil { return "",err }

	bucket := client.Bucket(bucketName)
	q := &storage.Query{
		//Delimiter: "/",
		Prefix: dir + "/Offload_track_IFR_"+date,
	}

	str := ""
	names := []string{}
	it := bucket.Objects(ctx, q)
	for {
    oa, err := it.Next()
    if err == iterator.Done {
			break
    }
    if err != nil {
			return "", fmt.Errorf("GCS-Readdir [gs://%s]%s': %v", bucketName, q.Prefix, err)
    }
		str += fmt.Sprintf("%8db %s {%s}\n", oa.Size, oa.Updated.Format("2006.01.02"), oa.Name)
		names = append(names, oa.Name)
	}

	//objs,err := bucket.List(ctx, q)
	//if err != nil { return "", fmt.Errorf("GCS-Readdir: %v", err) }
	//for _,oa := range objs.Results {
	//	str += fmt.Sprintf("%8db %s {%s}\n", oa.Size, oa.Updated.Format("2006.01.02"), oa.Name)
	//	names = append(names, oa.Name)
	//}

	nFlights := 0
	for _,filename := range names {
		str += fmt.Sprintf("Flights loaded from %s|%s\n", bucketName, filename)
		allDebug := fmt.Sprintf("Flights loaded from %s|%s", bucketName, filename)
		csvReader,err := getCSVReader(ctx, bucketName, filename)
		if err != nil {
			log.Errorf(ctx, "FOIAPROC ERR/CSV %s %v", err)
			return "", err
		}

		csvReader.Read() // Discard header row

		rows := [][]string{}		
		i := 1
		for {
			row,err := csvReader.Read()
			if err == io.EOF { break }
			if err != nil { return "", err }

			if len(rows)>0 && row[0] != rows[0][0] {
				thisDebug := fmt.Sprintf("%s:%d-%d", allDebug, i-len(rows), i-1)

				if f,err := rowsToFlight(rows, thisDebug); err != nil {
					log.Errorf(ctx, "FOIAPROC ERR/parse %v", err)
					return "", err
				} else if deb,err := foiaFunc(ctx, f); err != nil {
					log.Errorf(ctx, "FOIAPROC ERR/callback %v\n%s", err, deb)
					str += fmt.Sprintf("**PROB %v\n%s", err, deb)
				} else {
					str += fmt.Sprintf("* %s %v %v\n", f.Callsign, f.TagList(), f.WaypointList())
				}
				
				rows = [][]string{}
				nFlights++
			}

			rows = append(rows, row)
			i++
		}

		if len(rows)>0 {
			thisDebug := fmt.Sprintf("%s:%d-%d", allDebug, i-len(rows), i-1)
			if deb,err := addFlight(ctx, rows, time.Now(), thisDebug); err != nil {
				log.Errorf(ctx, "FOIAPROC ERR/Add %s %v\n%s", err, deb)
				return deb,err
			} else {
				str += deb
			}
			nFlights++
		}
		str += fmt.Sprintf("-- File read, %d rows\n", i)
	}

	str += fmt.Sprintf("-- %s all done, %d flights, took %s\n", date, nFlights, time.Since(tStart))
	log.Infof(ctx, "FOIAPROC finished %s (%d flights added, took %s)\n%s",
		date, nFlights, time.Since(tStart), str)
	
	return str,nil
}

// }}}


// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
