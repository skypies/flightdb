package main

/* Dates uploaded ... NOTE that the data is loaded per-UTC day. So if
 * you want a full PDT day, you need to load the day you want, and the
 * day that follows it.

---- All over again, with RG data

http://backend-dot-serfr0-fdb.appspot.com/foia/load?date=20140516

Dates available:  2014/05/10-18, 2014/09/*, 2015/05/09-16, 2015/09/*, 2015/11/14-20, 2015/12/05-13

  201401*
  201412*
  201501*


20140510   (Fri Apr 15, 13:56)
20140511   (Fri Apr 15, 13:58)
20140512   (Fri Apr 15, 13:59)
20140513   (Fri Apr 15, 14:01)
20140514   (Thu Mar 31, 17:00)
20140515   (Thu Mar 31, 17:03)
20140516   (Mon Apr 11, 08:40)
20140517   (Fri Apr 15, 14:02)
20140518   (Fri Apr 15, 14:04)
20140519   (Fri Apr 15, 14:05)

201409*    (Sun Apr 17, 13:ish)
201411*    (Mon Apr 18, 15:ish)

20150509   (Fri Apr 15, 13:36) 
20150510   (Fri Apr 15, 13:38) 
20150511   (Fri Apr 15, 13:39) 
20150512   (Fri Apr 15, 13:$1) 
20150513   (Thu Mar 31, 16:48)
20150514   (Thu Mar 31, 16:57)
20150515   (Fri Apr 15, 13:43) 
20150516   (Fri Apr 15, 13:44) 
20150517   (Fri Apr 15, 13:45) 

201509*    (Sun Apr 17, 14:ish)


20151114   (Fri Apr 15, 12:34) - 1975 flights
20151115   (Fri Apr 15, 12:36) - 1715 flights
20151116   (Fri Apr 15, 12:39) - 2106 flights
20151117   (Fri Apr 15, 12:41) - 2151 flights
20151118   (Fri Apr 15, 12:43) - 2131 flights
20151119   (Fri Apr 15, 12:55) 
20151120   (Fri Apr 15, 12:58) 
20151121   (Fri Apr 15, 13:00) 

20151205   (Fri Apr 15, 12:31) - 1916 flights
20151206   (Fri Apr 15, 12:33) - 1620 flights
20151207   (Fri Apr 15, 09:38) - 2072 flights
20151208   (Fri Apr 15, 09:41) - 2050 flights
20151209   (Fri Apr 15, 09:44) - 2161 flights
20151210   (Fri Apr 15, 09:47) - 2071 flights
20151211   (Fri Apr 15, 09:51) - 2210 flights
20151212   (Fri Apr 15, 09:56) - 1902 flights
20151213   (Fri Apr 15, 09:59) - 1492 flights
20151214   (Fri Apr 15, 10:01) - 2111 flights

-- next up: something in aug/sep, both years.

 */

import(
	"compress/gzip"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/cloud/storage"

	"github.com/skypies/geo"
	fdb "github.com/skypies/flightdb2"
	"github.com/skypies/flightdb2/fgae"
)

func init() {
	http.HandleFunc("/foia/load", foiaHandler)
	//http.HandleFunc("/foia/rm", rmHandler)
}

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

// [0]AIRCRAFT_ID, [1]FLIGHT_INDEX, [2]TRACK_INDEX,
//   [3]SOURCE_FACILITY, [4]BEACON_CODE, [5]DEP_APRT, [6]ARR_APRT, [7]ACFT_TYPE,
//   [8]LATITUDE, [9]LONGITUDE, [10]ALTITUDEx100ft,
//   [11]TRACK_POINT_DATE_UTC, [12]TRACK_POINT_TIME_UTC
// VOI902,2015020103105708,20150201065937NCT1024VOI902,
//   NCT,1024,MMGL,OAK,A320,
//   37.69849,-122.21049,1,
//   20150201,07:24:04

func rowToFlight(row []string) fdb.Flight {
	f := fdb.Flight{
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

func rowToTrackpoint(row []string) fdb.Trackpoint {
	lat,_  := strconv.ParseFloat(row[8], 64)
	long,_ := strconv.ParseFloat(row[9], 64)
	alt,_  := strconv.ParseFloat(row[10], 64)

	t,_ := time.Parse("20060102 15:04:05 MST", row[11] + " " + row[12] + " UTC")
	
	tp := fdb.Trackpoint{
		DataSource:    "RG-FOIA",
		TimestampUTC:  t,
		Latlong:       geo.Latlong{Lat:lat, Long:long},
		Altitude:      alt * 100.0,
		Squawk:        row[4],
	}

	return tp
}

func addFlight(ctx context.Context, rows [][]string, debug string) (string, error) {
	if len(rows) == 0 { return "", fmt.Errorf("No rows!") }
//	str := fmt.Sprintf("%s : %d rows\n", rows[0][0], len(rows))

	t := fdb.Track{}
	for _,row := range rows {
		t = append(t, rowToTrackpoint(row))
	}

	sort.Sort(fdb.TrackByTimestampAscending(t))
	
	f := rowToFlight(rows[0])
	f.Tracks["FOIA"] = &t
	f.SetTag("FOIA")

	f.Analyse()
	f.DebugLog += debug
	
	str := ""

	if true {// f.Callsign == "AAL1544" {
		db := fgae.FlightDB{C:ctx}
		if err := db.PersistFlight(&f); err != nil {
			return "", err
		}
		str += fmt.Sprintf("* %s %v %v\n", f.Callsign, f.TagList(), f.WaypointList())
	}
	
	return str,nil
}

// PA naming: faa-foia  FOIA-2015-006790/Offload_track_table  /Offload_track_20150104.txt.gz
// RG naming: rg-foia   2014                                  /txt.Offload_track_IFR_20140104.gz
func doStorageJunk(ctx context.Context, date string) (string, error) {
	frags := regexp.MustCompile("^(\\d{4})").FindStringSubmatch(date)
	if len(frags) == 0 {
		return "", fmt.Errorf("date '%s' did not match YYYYMMDD", date)
	}

	dir := frags[0]
	bucketName := "rg-foia"

	tStart := time.Now()
	log.Infof(ctx, "FOIAUPLOAD starting %s (%s)", date, dir)
	
	client, err := storage.NewClient(ctx)
	if err != nil { return "",err }

	bucket := client.Bucket(bucketName)
	q := &storage.Query{
		//Delimiter: "/",
		Prefix: dir + "/Offload_track_IFR_"+date,
		MaxResults: 200,
	}

	objs,err := bucket.List(ctx, q)
	if err != nil { return "", fmt.Errorf("GCS-Readdir: %v", err) }

	str := ""
	names := []string{}
	for _,oa := range objs.Results {
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
		for {
			row,err := csvReader.Read()
			if err == io.EOF { break }
			if err != nil { return "", err }

			if len(rows)>0 && row[0] != rows[0][0] {
				thisDebug := fmt.Sprintf("%s:%d-%d", allDebug, i-len(rows), i-1)
				if deb,err := addFlight(ctx, rows, thisDebug); err != nil {
					log.Errorf(ctx, "FOIAUPLOAD ERR/Add %s %v\n%s", err, deb)
					return deb,err
				} else {
					str += deb
				}
				rows = [][]string{}
				nFlights++
			}

			rows = append(rows, row)
			i++
		}

		if len(rows)>0 {
			thisDebug := fmt.Sprintf("%s:%d-%d", allDebug, i-len(rows), i-1)
			if deb,err := addFlight(ctx, rows, thisDebug); err != nil {
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

func rmHandler(w http.ResponseWriter, r *http.Request) {
	c,_ := context.WithTimeout(appengine.NewContext(r), 9*time.Minute)
	//c := appengine.NewContext(r)
	db := fgae.FlightDB{C:c}

	q := 	db.NewQuery().ByTags([]string{"FOIA"}).Query.KeysOnly()

	tStart := time.Now()
	str := "starting ...\n\n"

	for {
		keys,err := q.GetAll(c,nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		str += fmt.Sprintf("Found %d keys\n", len(keys))

		if len(keys)==0 { break }

		maxRm := 400
		for len(keys)>maxRm {
			if err := datastore.DeleteMulti(c, keys[0:maxRm-1]); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			keys = keys[maxRm:]
		}
		if err = datastore.DeleteMulti(c, keys); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	str += "\nKeys all deleted :O\nTime taken: " + time.Since(tStart).String() + "\n"
	
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(fmt.Sprintf("OK\n%s", str)))
}
