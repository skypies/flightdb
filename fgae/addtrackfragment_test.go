package fgae

// cd flightdb/fgae && goapp test (note it takes about 4 minutes to run)

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"google.golang.org/appengine/aetest"

	fdb "github.com/skypies/flightdb"
	"github.com/skypies/flightdb/fgae"
)

/* Misordered Frags

Originally, this (culled) sequence of 196 TrackFragments (totalling
1294 Trackpoints) would generate many instances instead of a single
instance:

*** A5BB1B@1383402847:1583409465 [A5BB1B@1383402847:1583409465] (heavily trimmed, see below)
Found 20 flight objects:-
[00] 105 points, start=2017.01.03 00:37:15, 5m15s, 54.4KM (310 deg), src=ADSB/CulverCity
[01] 335 points, start=2017.01.03 00:37:18, 45m14s, 523.5KM (319 deg), src=ADSB/CulverCity
[02] 152 points, start=2017.01.03 00:37:20, 14m19s, 161.4KM (310 deg), src=ADSB/CulverCity
[03]  23 points, start=2017.01.03 00:37:44, 5m38s, 58.9KM (310 deg), src=ADSB/CulverCity
[04]  53 points, start=2017.01.03 00:38:00, 8m17s, 90.5KM (310 deg), src=ADSB/CulverCity
[05]  24 points, start=2017.01.03 00:38:16, 5m1s, 52.6KM (310 deg), src=ADSB/CulverCity
[06]   8 points, start=2017.01.03 00:38:24, 7s, 1.3KM (308 deg), src=ADSB/CulverCity
[07] 127 points, start=2017.01.03 00:38:38, 11m9s, 125.7KM (310 deg), src=ADSB/CulverCity
[08]  39 points, start=2017.01.03 00:39:59, 11m37s, 133.6KM (310 deg), src=ADSB/CulverCity
[09]   6 points, start=2017.01.03 00:40:15, 3s, 0.6KM (307 deg), src=ADSB/CulverCity
[10] 177 points, start=2017.01.03 00:55:15, 2m33s, 30.2KM (310 deg), src=ADSB/ScottsValley3
[11]  58 points, start=2017.01.03 00:56:26, 12m30s, 150.1KM (321 deg), src=ADSB/ScottsValley3
[12]  15 points, start=2017.01.03 01:06:16, 12s, 2.5KM (328 deg), src=ADSB/ScottsValley3
[13]  11 points, start=2017.01.03 01:10:12, 6s, 1.4KM (325 deg), src=ADSB/ScottsValley3
[14]  11 points, start=2017.01.03 01:15:09, 5s, 1.2KM (327 deg), src=ADSB/ScottsValley3
[15]  18 points, start=2017.01.03 01:16:36, 11s, 2.3KM (328 deg), src=ADSB/ScottsValley3
[16]  15 points, start=2017.01.03 01:23:39, 16s, 2.8KM (330 deg), src=ADSB/ScottsValley3
[17]  45 points, start=2017.01.03 01:27:46, 21s, 3.5KM (331 deg), src=ADSB/ScottsValleyLite
[18]  75 points, start=2017.01.03 01:34:48, 2m19s, 15.3KM (331 deg), src=ADSB/Saratoga
[19]   6 points, start=2017.01.03 01:36:38, 4s, 0.5KM (350 deg), src=ADSB/Saratoga

They arrived out of order from PubSub; when a prefix frag appeared out
of order, it was deemed 'not a plausible extension', and triggered a
new instance. And when the next preceding frag appeared, it would
happen again, etc. This set also contained 'No space overlap, despite
time overlap':

  [19]  6 points, start=2017.01.03 01:36:38, 4s, 0.5KM (350 deg), src=ADSB/Saratoga
	-- AddFrag [A5BB17/ASA235]A 2017-01-15 19:21:52.136 +0000 UTC: was not plausible, so new flight
	-- Outcome=5[1]
	t1: Track:   66 points, start=2017.01.03 01:34:48, 2m19s, 15.3KM (331 deg), src=ADSB/Saratoga
	t2: Track:    6 points, start=2017.01.03 01:36:38, 4s, 0.5KM (350 deg), src=ADSB/Saratoga
	t1:  2017-01-03 01:34:48.795 +0000 UTC  ->  2017-01-03 01:37:08.544 +0000 UTC
	t2:  2017-01-03 01:36:38.631 +0000 UTC  ->  2017-01-03 01:36:42.684 +0000 UTC
	t2 is entirely contained within t1
	Overlap: from 2017-01-03 01:36:38.631 +0000 UTC, for 4.053s
	* OverlapA: Track:    5 points, start=2017.01.03 01:36:38, 22s, 2.5KM (3 deg), src=ADSB/Saratoga
	* OverlapB: Track:    6 points, start=2017.01.03 01:36:38, 4s, 0.5KM (350 deg), src=ADSB/Saratoga
	* space comparison: [1], 0.000000
	No space overlap, despite time overlap

But since the logic for adding fragments was beefed up, this sequence
should now generate a single flight !

 */

func TestMisorderedFrags(t *testing.T) {
	ctx, done, err := aetest.NewContext()
	if err != nil { t.Fatal(err) }
	defer done()

	db := fgae.NewDB(ctx)
	
	idspec,_ := fdb.NewIdSpec("A5BB1B@1483403847:1483407465")  // Has to match the frags
	
	if results,err := db.LookupAll(db.NewQuery().ByIdSpec(idspec)); err != nil {
		t.Fatal(err)
	} else if len(results) != 0 {
		t.Errorf("Expected no flight objects, but found %d:-\n", len(results))
		for i,f := range results { fmt.Printf("[%02d] %s\n", i, f) }
	}
	
	frags := []fdb.TrackFragment{}
	if err := json.NewDecoder(strings.NewReader(MisorderedFragsJSON)).Decode(&frags); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("(found %d frags)\n", len(frags))

	nPts := 0
	for _,frag := range frags {
		time.Sleep(1000 * time.Millisecond) // the test datastore gets confused if we go any quicker
		if err := db.AddTrackFragment(&frag); err != nil {
			t.Fatal(err)
		}
		nPts += len(frag.Track)
	}

	results,err := db.LookupAll(db.NewQuery().ByIdSpec(idspec))
	if err != nil { t.Fatal(err) }

	if len(results) != 1 {
		fmt.Printf("Found %d flight objects:-\n", len(results))
		for i,f := range results { fmt.Printf("[%02d] %s\n", i, f) }
		t.Errorf("Expected a single flight object, but found %d.", len(results))

	} else {
		f := results[0]
		track := f.AnyTrack()
		if len(track) != nPts+1 {
			t.Errorf("Expected the single flight to have %d Trackpoints, found %d\n", nPts, len(track))
		}
	}

	t.Errorf("SHOW ME THE STDOUT\n")
}

var (
  // http://localhost:8080/fdb/snarf?idspec=A5BB1B@1483403847:1483407465
	// http://localhost:8080/fdb/debug2?idspec=A5BB1B@1483403847:1483407465&json=1
	// Of the 865 frags, trim as follows:
	//  events = events[:777]
	//  events = append(events[:300], events[759:]...)
	//  events = append(events[:143], events[261:]...)
	MisorderedFragsJSON = `
[
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:38.743Z",
        "Lat": 33.01607,
        "Long": -117.61304,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:43.061Z",
        "Lat": 33.19922,
        "Long": -117.8699,
        "Altitude": 33975,
        "GroundSpeed": 346,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:43.581Z",
        "Lat": 33.19975,
        "Long": -117.87142,
        "Altitude": 33950,
        "GroundSpeed": 346,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:44.131Z",
        "Lat": 33.20031,
        "Long": -117.87147,
        "Altitude": 33950,
        "GroundSpeed": 346,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:44.621Z",
        "Lat": 33.20087,
        "Long": -117.87292,
        "Altitude": 33925,
        "GroundSpeed": 346,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:45.021Z",
        "Lat": 33.20128,
        "Long": -117.87292,
        "Altitude": 33925,
        "GroundSpeed": 346,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:46.032Z",
        "Lat": 33.20231,
        "Long": -117.87439,
        "Altitude": 33900,
        "GroundSpeed": 346,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:46.582Z",
        "Lat": 33.20288,
        "Long": -117.87578,
        "Altitude": 33875,
        "GroundSpeed": 347,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:47.063Z",
        "Lat": 33.20338,
        "Long": -117.87578,
        "Altitude": 33875,
        "GroundSpeed": 347,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:47.623Z",
        "Lat": 33.20399,
        "Long": -117.8773,
        "Altitude": 33850,
        "GroundSpeed": 347,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:48.123Z",
        "Lat": 33.2045,
        "Long": -117.8773,
        "Altitude": 33825,
        "GroundSpeed": 347,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:43.001Z",
        "Lat": 33.32721,
        "Long": -118.05145,
        "Altitude": 30750,
        "GroundSpeed": 368,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:48.284Z",
        "Lat": 33.33307,
        "Long": -118.06084,
        "Altitude": 30650,
        "GroundSpeed": 371,
        "Heading": 311,
        "VerticalRate": -1088,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:48.844Z",
        "Lat": 33.33366,
        "Long": -118.06095,
        "Altitude": 30650,
        "GroundSpeed": 371,
        "Heading": 311,
        "VerticalRate": -1024,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:49.374Z",
        "Lat": 33.33424,
        "Long": -118.06239,
        "Altitude": 30650,
        "GroundSpeed": 371,
        "Heading": 311,
        "VerticalRate": -1024,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:49.825Z",
        "Lat": 33.33475,
        "Long": -118.0625,
        "Altitude": 30625,
        "GroundSpeed": 371,
        "Heading": 311,
        "VerticalRate": -1024,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:50.385Z",
        "Lat": 33.33536,
        "Long": -118.06397,
        "Altitude": 30625,
        "GroundSpeed": 372,
        "Heading": 311,
        "VerticalRate": -960,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:50.925Z",
        "Lat": 33.336,
        "Long": -118.06408,
        "Altitude": 30625,
        "GroundSpeed": 372,
        "Heading": 311,
        "VerticalRate": -960,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:52.535Z",
        "Lat": 33.33778,
        "Long": -118.06721,
        "Altitude": 30600,
        "GroundSpeed": 372,
        "Heading": 311,
        "VerticalRate": -960,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:53.026Z",
        "Lat": 33.33833,
        "Long": -118.06721,
        "Altitude": 30600,
        "GroundSpeed": 372,
        "Heading": 311,
        "VerticalRate": -960,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:53.426Z",
        "Lat": 33.33876,
        "Long": -118.06884,
        "Altitude": 30600,
        "GroundSpeed": 374,
        "Heading": 311,
        "VerticalRate": -896,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:53.996Z",
        "Lat": 33.33941,
        "Long": -118.06878,
        "Altitude": 30575,
        "GroundSpeed": 374,
        "Heading": 311,
        "VerticalRate": -896,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:55.467Z",
        "Lat": 33.34104,
        "Long": -118.07186,
        "Altitude": 30550,
        "GroundSpeed": 374,
        "Heading": 311,
        "VerticalRate": -896,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:56.027Z",
        "Lat": 33.34169,
        "Long": -118.07197,
        "Altitude": 30550,
        "GroundSpeed": 374,
        "Heading": 311,
        "VerticalRate": -896,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:56.527Z",
        "Lat": 33.34222,
        "Long": -118.07358,
        "Altitude": 30550,
        "GroundSpeed": 374,
        "Heading": 311,
        "VerticalRate": -896,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:57.057Z",
        "Lat": 33.34282,
        "Long": -118.07364,
        "Altitude": 30550,
        "GroundSpeed": 374,
        "Heading": 311,
        "VerticalRate": -896,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:58.588Z",
        "Lat": 33.34456,
        "Long": -118.07672,
        "Altitude": 30525,
        "GroundSpeed": 375,
        "Heading": 311,
        "VerticalRate": -832,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:58.998Z",
        "Lat": 33.34502,
        "Long": -118.07672,
        "Altitude": 30525,
        "GroundSpeed": 375,
        "Heading": 311,
        "VerticalRate": -832,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:59.398Z",
        "Lat": 33.34546,
        "Long": -118.07831,
        "Altitude": 30500,
        "GroundSpeed": 376,
        "Heading": 311,
        "VerticalRate": -832,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:59.878Z",
        "Lat": 33.34597,
        "Long": -118.07825,
        "Altitude": 30500,
        "GroundSpeed": 376,
        "Heading": 311,
        "VerticalRate": -832,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:00.429Z",
        "Lat": 33.34662,
        "Long": -118.07996,
        "Altitude": 30500,
        "GroundSpeed": 376,
        "Heading": 311,
        "VerticalRate": -768,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:00.879Z",
        "Lat": 33.34712,
        "Long": -118.07985,
        "Altitude": 30500,
        "GroundSpeed": 376,
        "Heading": 311,
        "VerticalRate": -768,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:41.011Z",
        "Lat": 33.46147,
        "Long": -118.242,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:41.452Z",
        "Lat": 33.46193,
        "Long": -118.24372,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:41.881Z",
        "Lat": 33.46245,
        "Long": -118.24355,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:42.472Z",
        "Lat": 33.46312,
        "Long": -118.24519,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:43.062Z",
        "Lat": 33.46381,
        "Long": -118.24519,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:44.153Z",
        "Lat": 33.46501,
        "Long": -118.24697,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:44.653Z",
        "Lat": 33.46559,
        "Long": -118.2486,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:45.203Z",
        "Lat": 33.46623,
        "Long": -118.24849,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:46.314Z",
        "Lat": 33.46747,
        "Long": -118.25178,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:46.844Z",
        "Lat": 33.46806,
        "Long": -118.25184,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:47.364Z",
        "Lat": 33.46868,
        "Long": -118.2533,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:47.774Z",
        "Lat": 33.46915,
        "Long": -118.25336,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:52.946Z",
        "Lat": 33.47502,
        "Long": -118.26161,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:53.516Z",
        "Lat": 33.47567,
        "Long": -118.26317,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:02.4Z",
        "Lat": 33.48578,
        "Long": -118.27782,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:02.84Z",
        "Lat": 33.48628,
        "Long": -118.27787,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:03.391Z",
        "Lat": 33.48689,
        "Long": -118.27943,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:03.871Z",
        "Lat": 33.48745,
        "Long": -118.27937,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:04.431Z",
        "Lat": 33.48807,
        "Long": -118.28112,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:04.931Z",
        "Lat": 33.48862,
        "Long": -118.28112,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:05.461Z",
        "Lat": 33.48921,
        "Long": -118.28262,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:05.972Z",
        "Lat": 33.48982,
        "Long": -118.28268,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:06.412Z",
        "Lat": 33.49031,
        "Long": -118.28436,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:06.993Z",
        "Lat": 33.49095,
        "Long": -118.2843,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:07.802Z",
        "Lat": 33.49191,
        "Long": -118.28598,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:08.833Z",
        "Lat": 33.49306,
        "Long": -118.28765,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:09.283Z",
        "Lat": 33.49359,
        "Long": -118.28918,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:09.694Z",
        "Lat": 33.49406,
        "Long": -118.28923,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:10.214Z",
        "Lat": 33.49461,
        "Long": -118.29078,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:10.654Z",
        "Lat": 33.49512,
        "Long": -118.29095,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:11.134Z",
        "Lat": 33.49567,
        "Long": -118.29089,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:11.594Z",
        "Lat": 33.4962,
        "Long": -118.29249,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:12.114Z",
        "Lat": 33.4968,
        "Long": -118.29249,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:12.654Z",
        "Lat": 33.49741,
        "Long": -118.29419,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:13.105Z",
        "Lat": 33.49791,
        "Long": -118.29413,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:13.615Z",
        "Lat": 33.49848,
        "Long": -118.29574,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:14.045Z",
        "Lat": 33.49899,
        "Long": -118.29579,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:15.126Z",
        "Lat": 33.5002,
        "Long": -118.29738,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:18.227Z",
        "Lat": 33.50372,
        "Long": -118.30386,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:18.757Z",
        "Lat": 33.50432,
        "Long": -118.30397,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:19.167Z",
        "Lat": 33.50478,
        "Long": -118.3038,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:19.717Z",
        "Lat": 33.50541,
        "Long": -118.30555,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:20.258Z",
        "Lat": 33.50601,
        "Long": -118.30715,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:20.658Z",
        "Lat": 33.50647,
        "Long": -118.30721,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:21.199Z",
        "Lat": 33.50711,
        "Long": -118.30715,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:21.649Z",
        "Lat": 33.5076,
        "Long": -118.30885,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:22.549Z",
        "Lat": 33.50862,
        "Long": -118.31045,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:23.019Z",
        "Lat": 33.50917,
        "Long": -118.31045,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:23.509Z",
        "Lat": 33.50974,
        "Long": -118.31205,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:32.854Z",
        "Lat": 33.65762,
        "Long": -118.52365,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:33.424Z",
        "Lat": 33.65829,
        "Long": -118.52526,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:33.954Z",
        "Lat": 33.6589,
        "Long": -118.52526,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:34.384Z",
        "Lat": 33.65936,
        "Long": -118.52684,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:34.944Z",
        "Lat": 33.66,
        "Long": -118.5269,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:35.525Z",
        "Lat": 33.66067,
        "Long": -118.52852,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:36.105Z",
        "Lat": 33.66136,
        "Long": -118.52852,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:36.566Z",
        "Lat": 33.66188,
        "Long": -118.53009,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:37.126Z",
        "Lat": 33.66252,
        "Long": -118.5302,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:38.126Z",
        "Lat": 33.66365,
        "Long": -118.53184,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:38.676Z",
        "Lat": 33.66431,
        "Long": -118.5334,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:39.547Z",
        "Lat": 33.66527,
        "Long": -118.53516,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:40.037Z",
        "Lat": 33.66583,
        "Long": -118.53516,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:40.577Z",
        "Lat": 33.66646,
        "Long": -118.53671,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:41.057Z",
        "Lat": 33.66701,
        "Long": -118.53682,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:41.628Z",
        "Lat": 33.66765,
        "Long": -118.53842,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:42.037Z",
        "Lat": 33.66816,
        "Long": -118.53842,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:42.478Z",
        "Lat": 33.66866,
        "Long": -118.53996,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:43.389Z",
        "Lat": 33.6697,
        "Long": -118.54162,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:43.898Z",
        "Lat": 33.67026,
        "Long": -118.54168,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:44.438Z",
        "Lat": 33.6709,
        "Long": -118.54338,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:44.939Z",
        "Lat": 33.67145,
        "Long": -118.54338,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:45.449Z",
        "Lat": 33.67207,
        "Long": -118.54506,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:45.919Z",
        "Lat": 33.67258,
        "Long": -118.545,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:46.489Z",
        "Lat": 33.67323,
        "Long": -118.54657,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:49.171Z",
        "Lat": 33.6763,
        "Long": -118.54988,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:49.701Z",
        "Lat": 33.67694,
        "Long": -118.55156,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:50.101Z",
        "Lat": 33.6774,
        "Long": -118.55156,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:50.611Z",
        "Lat": 33.67795,
        "Long": -118.55319,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:51.012Z",
        "Lat": 33.67841,
        "Long": -118.55319,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:52.182Z",
        "Lat": 33.67975,
        "Long": -118.55478,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:52.702Z",
        "Lat": 33.68037,
        "Long": -118.55649,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:53.182Z",
        "Lat": 33.68092,
        "Long": -118.55644,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:53.613Z",
        "Lat": 33.68143,
        "Long": -118.55816,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:54.153Z",
        "Lat": 33.68203,
        "Long": -118.55822,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:29.348Z",
        "Lat": 33.72239,
        "Long": -118.61761,
        "Altitude": 30000,
        "GroundSpeed": 387,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:29.928Z",
        "Lat": 33.72305,
        "Long": -118.61761,
        "Altitude": 30000,
        "GroundSpeed": 387,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:30.818Z",
        "Lat": 33.72409,
        "Long": -118.61922,
        "Altitude": 30000,
        "GroundSpeed": 387,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:31.398Z",
        "Lat": 33.72477,
        "Long": -118.62087,
        "Altitude": 30000,
        "GroundSpeed": 387,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:31.858Z",
        "Lat": 33.72528,
        "Long": -118.62093,
        "Altitude": 30000,
        "GroundSpeed": 387,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:32.31Z",
        "Lat": 33.72578,
        "Long": -118.62252,
        "Altitude": 30000,
        "GroundSpeed": 387,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:32.749Z",
        "Lat": 33.72629,
        "Long": -118.62258,
        "Altitude": 30000,
        "GroundSpeed": 387,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:46.875Z",
        "Lat": 33.74245,
        "Long": -118.64562,
        "Altitude": 30000,
        "GroundSpeed": 387,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:47.285Z",
        "Lat": 33.74292,
        "Long": -118.64731,
        "Altitude": 30000,
        "GroundSpeed": 387,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:47.785Z",
        "Lat": 33.74348,
        "Long": -118.64731,
        "Altitude": 30000,
        "GroundSpeed": 387,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:48.226Z",
        "Lat": 33.744,
        "Long": -118.64892,
        "Altitude": 30000,
        "GroundSpeed": 387,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:16.324Z",
        "Lat": 32.99377,
        "Long": -117.5825,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:50.789Z",
        "Lat": 33.1463,
        "Long": -117.79596,
        "Altitude": 35375,
        "GroundSpeed": 327,
        "Heading": 311,
        "VerticalRate": -1472,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:51.829Z",
        "Lat": 33.14729,
        "Long": -117.79737,
        "Altitude": 35350,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": -1472,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:52.409Z",
        "Lat": 33.14786,
        "Long": -117.79876,
        "Altitude": 35350,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": -1472,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:52.85Z",
        "Lat": 33.14832,
        "Long": -117.79882,
        "Altitude": 35325,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": -1472,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:53.33Z",
        "Lat": 33.14878,
        "Long": -117.80017,
        "Altitude": 35325,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": -1472,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:53.751Z",
        "Lat": 33.1492,
        "Long": -117.80017,
        "Altitude": 35300,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": -1472,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:54.32Z",
        "Lat": 33.14973,
        "Long": -117.80156,
        "Altitude": 35300,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": -1472,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:54.78Z",
        "Lat": 33.15019,
        "Long": -117.80151,
        "Altitude": 35275,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": -1472,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:55.251Z",
        "Lat": 33.15064,
        "Long": -117.80292,
        "Altitude": 35275,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": -1472,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:57.202Z",
        "Lat": 33.15257,
        "Long": -117.80431,
        "Altitude": 35225,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": -1472,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:57.782Z",
        "Lat": 33.15315,
        "Long": -117.80567,
        "Altitude": 35225,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": -1536,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:58.243Z",
        "Lat": 33.15358,
        "Long": -117.80706,
        "Altitude": 35200,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": -1536,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:58.692Z",
        "Lat": 33.15404,
        "Long": -117.80706,
        "Altitude": 35200,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": -1536,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:10.118Z",
        "Lat": 33.2916,
        "Long": -118.00056,
        "Altitude": 31625,
        "GroundSpeed": 360,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:11.018Z",
        "Lat": 33.2926,
        "Long": -118.00206,
        "Altitude": 31600,
        "GroundSpeed": 359,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:11.568Z",
        "Lat": 33.29318,
        "Long": -118.00353,
        "Altitude": 31600,
        "GroundSpeed": 359,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:12.069Z",
        "Lat": 33.29369,
        "Long": -118.00353,
        "Altitude": 31575,
        "GroundSpeed": 359,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:12.578Z",
        "Lat": 33.29425,
        "Long": -118.00508,
        "Altitude": 31575,
        "GroundSpeed": 359,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:14.082Z",
        "Lat": 33.29588,
        "Long": -118.00662,
        "Altitude": 31525,
        "GroundSpeed": 359,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:15.08Z",
        "Lat": 33.29695,
        "Long": -118.00816,
        "Altitude": 31500,
        "GroundSpeed": 360,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:15.6Z",
        "Lat": 33.29751,
        "Long": -118.0097,
        "Altitude": 31475,
        "GroundSpeed": 360,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:17.17Z",
        "Lat": 33.29919,
        "Long": -118.01118,
        "Altitude": 31450,
        "GroundSpeed": 360,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:17.631Z",
        "Lat": 33.29965,
        "Long": -118.01278,
        "Altitude": 31425,
        "GroundSpeed": 360,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:06:16.395Z",
        "Lat": 34.99686,
        "Long": -120.15192,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:06:16.955Z",
        "Lat": 34.99768,
        "Long": -120.15198,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:06:17.956Z",
        "Lat": 34.99923,
        "Long": -120.15318,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:06:18.467Z",
        "Lat": 35.00002,
        "Long": -120.15439,
        "Altitude": 30000,
        "GroundSpeed": 396,
        "Heading": 328,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:06:19.396Z",
        "Lat": 35.00146,
        "Long": -120.15553,
        "Altitude": 30000,
        "GroundSpeed": 396,
        "Heading": 328,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:06:19.987Z",
        "Lat": 35.00239,
        "Long": -120.15553,
        "Altitude": 30000,
        "GroundSpeed": 396,
        "Heading": 328,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:06:20.567Z",
        "Lat": 35.00327,
        "Long": -120.15685,
        "Altitude": 30000,
        "GroundSpeed": 396,
        "Heading": 328,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:06:24.129Z",
        "Lat": 35.00877,
        "Long": -120.16045,
        "Altitude": 30000,
        "GroundSpeed": 396,
        "Heading": 328,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:06:24.559Z",
        "Lat": 35.00945,
        "Long": -120.16167,
        "Altitude": 30000,
        "GroundSpeed": 396,
        "Heading": 328,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:06:24.979Z",
        "Lat": 35.01009,
        "Long": -120.16173,
        "Altitude": 30000,
        "GroundSpeed": 396,
        "Heading": 328,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:06:25.409Z",
        "Lat": 35.01073,
        "Long": -120.16285,
        "Altitude": 30000,
        "GroundSpeed": 396,
        "Heading": 328,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:06:25.81Z",
        "Lat": 35.01133,
        "Long": -120.16285,
        "Altitude": 30000,
        "GroundSpeed": 396,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:06:26.98Z",
        "Lat": 35.01315,
        "Long": -120.16414,
        "Altitude": 30000,
        "GroundSpeed": 396,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:06:28.011Z",
        "Lat": 35.01473,
        "Long": -120.16525,
        "Altitude": 30000,
        "GroundSpeed": 396,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:06:29.06Z",
        "Lat": 35.01636,
        "Long": -120.16655,
        "Altitude": 30000,
        "GroundSpeed": 396,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:15.014Z",
        "Lat": 33.11124,
        "Long": -117.74614,
        "Altitude": 36000,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:15.554Z",
        "Lat": 33.11177,
        "Long": -117.74754,
        "Altitude": 36000,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:16.484Z",
        "Lat": 33.11266,
        "Long": -117.74883,
        "Altitude": 36000,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:16.994Z",
        "Lat": 33.11316,
        "Long": -117.74888,
        "Altitude": 36000,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:17.395Z",
        "Lat": 33.11354,
        "Long": -117.75023,
        "Altitude": 36000,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:18.565Z",
        "Lat": 33.11472,
        "Long": -117.75157,
        "Altitude": 36000,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:24.107Z",
        "Lat": 33.00149,
        "Long": -117.59216,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:24.597Z",
        "Lat": 33.00197,
        "Long": -117.59354,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:25.167Z",
        "Lat": 33.00256,
        "Long": -117.59348,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:28.158Z",
        "Lat": 33.00554,
        "Long": -117.59766,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:29.199Z",
        "Lat": 33.00655,
        "Long": -117.59908,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:30.13Z",
        "Lat": 33.00749,
        "Long": -117.60051,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:30.72Z",
        "Lat": 33.00806,
        "Long": -117.60189,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:31.3Z",
        "Lat": 33.00866,
        "Long": -117.60326,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:00.016Z",
        "Lat": 32.97756,
        "Long": -117.55881,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:00.457Z",
        "Lat": 32.97803,
        "Long": -117.56014,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:00.887Z",
        "Lat": 32.97844,
        "Long": -117.56019,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:01.367Z",
        "Lat": 32.97891,
        "Long": -117.56156,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:01.877Z",
        "Lat": 32.97942,
        "Long": -117.56156,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 64,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:02.417Z",
        "Lat": 32.97995,
        "Long": -117.56299,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:03.428Z",
        "Lat": 32.98096,
        "Long": -117.56442,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:03.898Z",
        "Lat": 32.98142,
        "Long": -117.56436,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:04.469Z",
        "Lat": 32.98201,
        "Long": -117.56569,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:05.479Z",
        "Lat": 32.98301,
        "Long": -117.56711,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:06.03Z",
        "Lat": 32.98356,
        "Long": -117.56716,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:06.61Z",
        "Lat": 32.98412,
        "Long": -117.56854,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:07.68Z",
        "Lat": 32.98519,
        "Long": -117.56991,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:14.653Z",
        "Lat": 32.99208,
        "Long": -117.57975,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:15.183Z",
        "Lat": 32.99263,
        "Long": -117.57964,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:15.773Z",
        "Lat": 32.9932,
        "Long": -117.58101,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:20.196Z",
        "Lat": 32.99762,
        "Long": -117.58661,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:21.986Z",
        "Lat": 32.99939,
        "Long": -117.58936,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:23.572Z",
        "Lat": 33.06051,
        "Long": -117.67546,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:25.103Z",
        "Lat": 33.06203,
        "Long": -117.67687,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:26.193Z",
        "Lat": 33.06312,
        "Long": -117.67826,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:26.703Z",
        "Lat": 33.06358,
        "Long": -117.67967,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:28.304Z",
        "Lat": 33.06519,
        "Long": -117.68242,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:28.874Z",
        "Lat": 33.06573,
        "Long": -117.68236,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:30.485Z",
        "Lat": 33.06734,
        "Long": -117.68516,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:30.975Z",
        "Lat": 33.06779,
        "Long": -117.68516,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:31.475Z",
        "Lat": 33.06829,
        "Long": -117.68656,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:32.566Z",
        "Lat": 33.0694,
        "Long": -117.68791,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:33.016Z",
        "Lat": 33.06981,
        "Long": -117.68802,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:34.116Z",
        "Lat": 33.0709,
        "Long": -117.68936,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:34.656Z",
        "Lat": 33.07146,
        "Long": -117.69071,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:35.177Z",
        "Lat": 33.07196,
        "Long": -117.69071,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:17.266Z",
        "Lat": 33.23509,
        "Long": -117.9217,
        "Altitude": 33050,
        "GroundSpeed": 353,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:18.175Z",
        "Lat": 33.23606,
        "Long": -117.9217,
        "Altitude": 33025,
        "GroundSpeed": 353,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:20.237Z",
        "Lat": 33.23822,
        "Long": -117.92626,
        "Altitude": 32975,
        "GroundSpeed": 354,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:20.707Z",
        "Lat": 33.23872,
        "Long": -117.92626,
        "Altitude": 32975,
        "GroundSpeed": 354,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:21.277Z",
        "Lat": 33.23932,
        "Long": -117.92775,
        "Altitude": 32950,
        "GroundSpeed": 354,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:21.793Z",
        "Lat": 33.23988,
        "Long": -117.92775,
        "Altitude": 32925,
        "GroundSpeed": 354,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:22.837Z",
        "Lat": 33.24097,
        "Long": -117.92933,
        "Altitude": 32900,
        "GroundSpeed": 354,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:23.958Z",
        "Lat": 33.24216,
        "Long": -117.93083,
        "Altitude": 32875,
        "GroundSpeed": 355,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:54.087Z",
        "Lat": 33.47632,
        "Long": -118.26311,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:54.677Z",
        "Lat": 33.47699,
        "Long": -118.26475,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:55.697Z",
        "Lat": 33.47813,
        "Long": -118.26642,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:56.188Z",
        "Lat": 33.47869,
        "Long": -118.26642,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:57.249Z",
        "Lat": 33.4799,
        "Long": -118.26967,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:58.789Z",
        "Lat": 33.48166,
        "Long": -118.27128,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:59.329Z",
        "Lat": 33.48228,
        "Long": -118.27298,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:00.389Z",
        "Lat": 33.48349,
        "Long": -118.27452,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:00.939Z",
        "Lat": 33.48409,
        "Long": -118.27463,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:01.43Z",
        "Lat": 33.48465,
        "Long": -118.27623,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:01.83Z",
        "Lat": 33.48512,
        "Long": -118.27623,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:16.227Z",
        "Lat": 33.50148,
        "Long": -118.30056,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:17.296Z",
        "Lat": 33.50267,
        "Long": -118.30224,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:10:12.105Z",
        "Lat": 35.35968,
        "Long": -120.43862,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:10:12.685Z",
        "Lat": 35.3606,
        "Long": -120.43997,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:10:13.226Z",
        "Lat": 35.36142,
        "Long": -120.44106,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:10:14.155Z",
        "Lat": 35.36285,
        "Long": -120.44113,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:10:14.666Z",
        "Lat": 35.36362,
        "Long": -120.44237,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:10:15.656Z",
        "Lat": 35.36513,
        "Long": -120.44358,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:10:16.233Z",
        "Lat": 35.36604,
        "Long": -120.44472,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:10:16.756Z",
        "Lat": 35.36682,
        "Long": -120.44483,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:10:17.238Z",
        "Lat": 35.36755,
        "Long": -120.44598,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:10:18.208Z",
        "Lat": 35.36904,
        "Long": -120.44598,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:10:18.688Z",
        "Lat": 35.3698,
        "Long": -120.44718,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:20.47Z",
        "Lat": 32.93834,
        "Long": -117.50438,
        "Altitude": 36000,
        "GroundSpeed": 332,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:21.54Z",
        "Lat": 32.93939,
        "Long": -117.5059,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:24.162Z",
        "Lat": 32.94199,
        "Long": -117.50853,
        "Altitude": 36000,
        "GroundSpeed": 332,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:50.423Z",
        "Lat": 32.96809,
        "Long": -117.5463,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:50.833Z",
        "Lat": 32.96851,
        "Long": -117.5463,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:51.403Z",
        "Lat": 32.96904,
        "Long": -117.54766,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:51.933Z",
        "Lat": 32.9696,
        "Long": -117.54771,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:52.364Z",
        "Lat": 32.97002,
        "Long": -117.54899,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:52.924Z",
        "Lat": 32.97057,
        "Long": -117.5491,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:54.084Z",
        "Lat": 32.97169,
        "Long": -117.5504,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:54.545Z",
        "Lat": 32.97217,
        "Long": -117.5519,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:55.605Z",
        "Lat": 32.97323,
        "Long": -117.55326,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:56.105Z",
        "Lat": 32.97369,
        "Long": -117.55326,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:57.066Z",
        "Lat": 32.97464,
        "Long": -117.55459,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:57.801Z",
        "Lat": 33.03496,
        "Long": -117.63936,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:59.292Z",
        "Lat": 33.03645,
        "Long": -117.64216,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:00.362Z",
        "Lat": 33.03749,
        "Long": -117.64347,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:00.822Z",
        "Lat": 33.03795,
        "Long": -117.64358,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:01.943Z",
        "Lat": 33.0391,
        "Long": -117.64502,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:02.873Z",
        "Lat": 33.04001,
        "Long": -117.64627,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:05.265Z",
        "Lat": 33.04236,
        "Long": -117.65046,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:21.631Z",
        "Lat": 33.05861,
        "Long": -117.67277,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:22.152Z",
        "Lat": 33.05912,
        "Long": -117.67277,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:35.727Z",
        "Lat": 33.07248,
        "Long": -117.69216,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:36.617Z",
        "Lat": 33.07338,
        "Long": -117.69357,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:37.188Z",
        "Lat": 33.07393,
        "Long": -117.69351,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:38.178Z",
        "Lat": 33.0749,
        "Long": -117.69485,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:38.658Z",
        "Lat": 33.07539,
        "Long": -117.69626,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:39.219Z",
        "Lat": 33.07592,
        "Long": -117.69765,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:39.718Z",
        "Lat": 33.07644,
        "Long": -117.69771,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:41.2Z",
        "Lat": 33.07787,
        "Long": -117.69906,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:42.19Z",
        "Lat": 33.07886,
        "Long": -117.70046,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:43.071Z",
        "Lat": 33.07974,
        "Long": -117.70181,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:43.62Z",
        "Lat": 33.08025,
        "Long": -117.70326,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:44.071Z",
        "Lat": 33.08072,
        "Long": -117.7032,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:45.441Z",
        "Lat": 33.08207,
        "Long": -117.70606,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:47.442Z",
        "Lat": 33.08402,
        "Long": -117.70881,
        "Altitude": 36000,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": 64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:47.892Z",
        "Lat": 33.08449,
        "Long": -117.70875,
        "Altitude": 36000,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": 64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:48.403Z",
        "Lat": 33.08496,
        "Long": -117.71016,
        "Altitude": 36000,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": 64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:10.747Z",
        "Lat": 33.16599,
        "Long": -117.82397,
        "Altitude": 34875,
        "GroundSpeed": 334,
        "Heading": 311,
        "VerticalRate": -1792,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:11.698Z",
        "Lat": 33.16698,
        "Long": -117.82534,
        "Altitude": 34850,
        "GroundSpeed": 334,
        "Heading": 311,
        "VerticalRate": -1792,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:12.128Z",
        "Lat": 33.1674,
        "Long": -117.82534,
        "Altitude": 34825,
        "GroundSpeed": 334,
        "Heading": 311,
        "VerticalRate": -1792,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:13.668Z",
        "Lat": 33.16893,
        "Long": -117.82814,
        "Altitude": 34775,
        "GroundSpeed": 336,
        "Heading": 311,
        "VerticalRate": -1792,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:14.169Z",
        "Lat": 33.16945,
        "Long": -117.82814,
        "Altitude": 34775,
        "GroundSpeed": 336,
        "Heading": 311,
        "VerticalRate": -1792,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:14.609Z",
        "Lat": 33.16992,
        "Long": -117.82958,
        "Altitude": 34750,
        "GroundSpeed": 336,
        "Heading": 311,
        "VerticalRate": -1792,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:15.719Z",
        "Lat": 33.17103,
        "Long": -117.831,
        "Altitude": 34725,
        "GroundSpeed": 336,
        "Heading": 311,
        "VerticalRate": -1792,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:16.179Z",
        "Lat": 33.1715,
        "Long": -117.83095,
        "Altitude": 34725,
        "GroundSpeed": 336,
        "Heading": 311,
        "VerticalRate": -1792,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:16.76Z",
        "Lat": 33.17207,
        "Long": -117.83238,
        "Altitude": 34700,
        "GroundSpeed": 337,
        "Heading": 311,
        "VerticalRate": -1792,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:17.33Z",
        "Lat": 33.17266,
        "Long": -117.83386,
        "Altitude": 34675,
        "GroundSpeed": 337,
        "Heading": 311,
        "VerticalRate": -1792,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:18.291Z",
        "Lat": 33.17363,
        "Long": -117.83524,
        "Altitude": 34650,
        "GroundSpeed": 338,
        "Heading": 311,
        "VerticalRate": -1792,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:18.691Z",
        "Lat": 33.17404,
        "Long": -117.83524,
        "Altitude": 34650,
        "GroundSpeed": 338,
        "Heading": 311,
        "VerticalRate": -1792,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:20.131Z",
        "Lat": 33.1755,
        "Long": -117.83672,
        "Altitude": 34600,
        "GroundSpeed": 339,
        "Heading": 311,
        "VerticalRate": -1792,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:20.712Z",
        "Lat": 33.1761,
        "Long": -117.83815,
        "Altitude": 34575,
        "GroundSpeed": 339,
        "Heading": 311,
        "VerticalRate": -1792,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:56.807Z",
        "Lat": 33.21359,
        "Long": -117.8905,
        "Altitude": 33600,
        "GroundSpeed": 349,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:57.237Z",
        "Lat": 33.21404,
        "Long": -117.89199,
        "Altitude": 33600,
        "GroundSpeed": 349,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:58.317Z",
        "Lat": 33.21519,
        "Long": -117.89352,
        "Altitude": 33575,
        "GroundSpeed": 350,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:58.898Z",
        "Lat": 33.21579,
        "Long": -117.89346,
        "Altitude": 33550,
        "GroundSpeed": 350,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:59.417Z",
        "Lat": 33.21633,
        "Long": -117.89496,
        "Altitude": 33525,
        "GroundSpeed": 350,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:59.988Z",
        "Lat": 33.21693,
        "Long": -117.8949,
        "Altitude": 33525,
        "GroundSpeed": 350,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:00.489Z",
        "Lat": 33.21744,
        "Long": -117.89643,
        "Altitude": 33500,
        "GroundSpeed": 350,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:00.928Z",
        "Lat": 33.2179,
        "Long": -117.89648,
        "Altitude": 33500,
        "GroundSpeed": 350,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:04.39Z",
        "Lat": 33.22156,
        "Long": -117.90236,
        "Altitude": 33400,
        "GroundSpeed": 351,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:04.84Z",
        "Lat": 33.22202,
        "Long": -117.90242,
        "Altitude": 33400,
        "GroundSpeed": 351,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:05.27Z",
        "Lat": 33.22247,
        "Long": -117.90382,
        "Altitude": 33375,
        "GroundSpeed": 351,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:06.212Z",
        "Lat": 33.22345,
        "Long": -117.90527,
        "Altitude": 33350,
        "GroundSpeed": 351,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:11.353Z",
        "Lat": 33.22885,
        "Long": -117.91284,
        "Altitude": 33225,
        "GroundSpeed": 352,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:11.913Z",
        "Lat": 33.22945,
        "Long": -117.91278,
        "Altitude": 33200,
        "GroundSpeed": 352,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:12.413Z",
        "Lat": 33.22998,
        "Long": -117.91428,
        "Altitude": 33200,
        "GroundSpeed": 352,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:12.913Z",
        "Lat": 33.23053,
        "Long": -117.91428,
        "Altitude": 33175,
        "GroundSpeed": 352,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:13.464Z",
        "Lat": 33.23108,
        "Long": -117.9157,
        "Altitude": 33150,
        "GroundSpeed": 352,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:13.874Z",
        "Lat": 33.2315,
        "Long": -117.91576,
        "Altitude": 33150,
        "GroundSpeed": 353,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:14.334Z",
        "Lat": 33.23199,
        "Long": -117.9173,
        "Altitude": 33150,
        "GroundSpeed": 353,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:15.364Z",
        "Lat": 33.23308,
        "Long": -117.91873,
        "Altitude": 33100,
        "GroundSpeed": 353,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:16.415Z",
        "Lat": 33.23419,
        "Long": -117.92027,
        "Altitude": 33075,
        "GroundSpeed": 353,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:25.398Z",
        "Lat": 33.2437,
        "Long": -117.9338,
        "Altitude": 32850,
        "GroundSpeed": 355,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:25.799Z",
        "Lat": 33.24412,
        "Long": -117.9338,
        "Altitude": 32825,
        "GroundSpeed": 355,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:26.389Z",
        "Lat": 33.24477,
        "Long": -117.93527,
        "Altitude": 32825,
        "GroundSpeed": 355,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:26.97Z",
        "Lat": 33.24536,
        "Long": -117.93527,
        "Altitude": 32800,
        "GroundSpeed": 355,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:27.49Z",
        "Lat": 33.24593,
        "Long": -117.93678,
        "Altitude": 32775,
        "GroundSpeed": 356,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:27.97Z",
        "Lat": 33.24644,
        "Long": -117.93678,
        "Altitude": 32775,
        "GroundSpeed": 356,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:16.615Z",
        "Lat": 33.36488,
        "Long": -118.10561,
        "Altitude": 30300,
        "GroundSpeed": 379,
        "Heading": 311,
        "VerticalRate": -768,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:17.576Z",
        "Lat": 33.36599,
        "Long": -118.10712,
        "Altitude": 30300,
        "GroundSpeed": 381,
        "Heading": 311,
        "VerticalRate": -768,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:18.716Z",
        "Lat": 33.36726,
        "Long": -118.1088,
        "Altitude": 30275,
        "GroundSpeed": 381,
        "Heading": 311,
        "VerticalRate": -768,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:19.757Z",
        "Lat": 33.36846,
        "Long": -118.11048,
        "Altitude": 30275,
        "GroundSpeed": 381,
        "Heading": 311,
        "VerticalRate": -768,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:20.317Z",
        "Lat": 33.36909,
        "Long": -118.11204,
        "Altitude": 30250,
        "GroundSpeed": 381,
        "Heading": 311,
        "VerticalRate": -768,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:20.817Z",
        "Lat": 33.36964,
        "Long": -118.11204,
        "Altitude": 30250,
        "GroundSpeed": 381,
        "Heading": 311,
        "VerticalRate": -768,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:21.868Z",
        "Lat": 33.37088,
        "Long": -118.11357,
        "Altitude": 30225,
        "GroundSpeed": 381,
        "Heading": 311,
        "VerticalRate": -768,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:22.278Z",
        "Lat": 33.37134,
        "Long": -118.11522,
        "Altitude": 30225,
        "GroundSpeed": 381,
        "Heading": 311,
        "VerticalRate": -768,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:22.738Z",
        "Lat": 33.37184,
        "Long": -118.11528,
        "Altitude": 30225,
        "GroundSpeed": 381,
        "Heading": 311,
        "VerticalRate": -768,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:23.229Z",
        "Lat": 33.37241,
        "Long": -118.11682,
        "Altitude": 30225,
        "GroundSpeed": 381,
        "Heading": 311,
        "VerticalRate": -768,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:23.769Z",
        "Lat": 33.37302,
        "Long": -118.11682,
        "Altitude": 30225,
        "GroundSpeed": 381,
        "Heading": 311,
        "VerticalRate": -768,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:24.25Z",
        "Lat": 33.37358,
        "Long": -118.11846,
        "Altitude": 30200,
        "GroundSpeed": 381,
        "Heading": 311,
        "VerticalRate": -768,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:24.819Z",
        "Lat": 33.37422,
        "Long": -118.11852,
        "Altitude": 30200,
        "GroundSpeed": 381,
        "Heading": 311,
        "VerticalRate": -768,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:25.67Z",
        "Lat": 33.37521,
        "Long": -118.12007,
        "Altitude": 30200,
        "GroundSpeed": 381,
        "Heading": 311,
        "VerticalRate": -768,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:26.099Z",
        "Lat": 33.37567,
        "Long": -118.12007,
        "Altitude": 30175,
        "GroundSpeed": 381,
        "Heading": 311,
        "VerticalRate": -768,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:27.09Z",
        "Lat": 33.37683,
        "Long": -118.1217,
        "Altitude": 30175,
        "GroundSpeed": 381,
        "Heading": 311,
        "VerticalRate": -768,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:27.63Z",
        "Lat": 33.37744,
        "Long": -118.12337,
        "Altitude": 30175,
        "GroundSpeed": 381,
        "Heading": 311,
        "VerticalRate": -768,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:28.13Z",
        "Lat": 33.378,
        "Long": -118.12337,
        "Altitude": 30150,
        "GroundSpeed": 381,
        "Heading": 311,
        "VerticalRate": -768,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:29.112Z",
        "Lat": 33.37912,
        "Long": -118.12495,
        "Altitude": 30150,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -768,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:30.193Z",
        "Lat": 33.38038,
        "Long": -118.12657,
        "Altitude": 30125,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -768,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:40.986Z",
        "Lat": 33.39272,
        "Long": -118.14439,
        "Altitude": 30050,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -256,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:41.536Z",
        "Lat": 33.39336,
        "Long": -118.14608,
        "Altitude": 30050,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -256,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:42.076Z",
        "Lat": 33.39397,
        "Long": -118.14596,
        "Altitude": 30050,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -256,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:43.437Z",
        "Lat": 33.39555,
        "Long": -118.14933,
        "Altitude": 30050,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -192,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:43.897Z",
        "Lat": 33.39606,
        "Long": -118.14927,
        "Altitude": 30050,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -128,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:44.877Z",
        "Lat": 33.3972,
        "Long": -118.15087,
        "Altitude": 30050,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -128,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:45.408Z",
        "Lat": 33.39779,
        "Long": -118.15252,
        "Altitude": 30050,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -128,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:45.829Z",
        "Lat": 33.3983,
        "Long": -118.15258,
        "Altitude": 30050,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -128,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:46.298Z",
        "Lat": 33.3988,
        "Long": -118.15411,
        "Altitude": 30050,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -128,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:47.199Z",
        "Lat": 33.39986,
        "Long": -118.15411,
        "Altitude": 30025,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -128,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:48.27Z",
        "Lat": 33.40109,
        "Long": -118.15735,
        "Altitude": 30025,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -128,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:08.637Z",
        "Lat": 33.42444,
        "Long": -118.18998,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:09.658Z",
        "Lat": 33.42562,
        "Long": -118.19165,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:10.138Z",
        "Lat": 33.42618,
        "Long": -118.19159,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:11.129Z",
        "Lat": 33.42732,
        "Long": -118.19323,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:11.589Z",
        "Lat": 33.42786,
        "Long": -118.19479,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:12.059Z",
        "Lat": 33.42837,
        "Long": -118.1949,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:13.119Z",
        "Lat": 33.42961,
        "Long": -118.19647,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:13.55Z",
        "Lat": 33.43009,
        "Long": -118.19815,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:14.01Z",
        "Lat": 33.43061,
        "Long": -118.19804,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:14.59Z",
        "Lat": 33.4313,
        "Long": -118.19965,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:15.051Z",
        "Lat": 33.43181,
        "Long": -118.19976,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:15.5Z",
        "Lat": 33.43233,
        "Long": -118.20129,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:15.941Z",
        "Lat": 33.43284,
        "Long": -118.20134,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:13.41Z",
        "Lat": 33.56654,
        "Long": -118.39365,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:13.911Z",
        "Lat": 33.5671,
        "Long": -118.39365,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:14.44Z",
        "Lat": 33.56772,
        "Long": -118.39534,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:14.952Z",
        "Lat": 33.56831,
        "Long": -118.3954,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:15.481Z",
        "Lat": 33.56891,
        "Long": -118.39691,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:16.001Z",
        "Lat": 33.56952,
        "Long": -118.39691,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:16.581Z",
        "Lat": 33.57019,
        "Long": -118.39854,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:17.041Z",
        "Lat": 33.57069,
        "Long": -118.39865,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:18.542Z",
        "Lat": 33.57243,
        "Long": -118.40196,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:18.962Z",
        "Lat": 33.57289,
        "Long": -118.40179,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:19.533Z",
        "Lat": 33.57357,
        "Long": -118.40355,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:51:22.685Z",
        "Lat": 33.85236,
        "Long": -118.80509,
        "Altitude": 30000,
        "GroundSpeed": 389,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:51:23.095Z",
        "Lat": 33.85286,
        "Long": -118.80514,
        "Altitude": 30000,
        "GroundSpeed": 389,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:51:24.346Z",
        "Lat": 33.85428,
        "Long": -118.80851,
        "Altitude": 30000,
        "GroundSpeed": 389,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:51:24.816Z",
        "Lat": 33.85483,
        "Long": -118.80851,
        "Altitude": 30000,
        "GroundSpeed": 389,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:51:25.287Z",
        "Lat": 33.85539,
        "Long": -118.8101,
        "Altitude": 30000,
        "GroundSpeed": 389,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:51:25.717Z",
        "Lat": 33.85586,
        "Long": -118.81016,
        "Altitude": 30000,
        "GroundSpeed": 389,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:51:26.967Z",
        "Lat": 33.8573,
        "Long": -118.81176,
        "Altitude": 30000,
        "GroundSpeed": 389,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:51:27.557Z",
        "Lat": 33.858,
        "Long": -118.81353,
        "Altitude": 30000,
        "GroundSpeed": 389,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:51:28.138Z",
        "Lat": 33.85865,
        "Long": -118.81353,
        "Altitude": 30000,
        "GroundSpeed": 389,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:51:28.718Z",
        "Lat": 33.85931,
        "Long": -118.81518,
        "Altitude": 30000,
        "GroundSpeed": 389,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:51:29.168Z",
        "Lat": 33.85986,
        "Long": -118.81506,
        "Altitude": 30000,
        "GroundSpeed": 389,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:51:29.678Z",
        "Lat": 33.86042,
        "Long": -118.81674,
        "Altitude": 30000,
        "GroundSpeed": 389,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:51:30.219Z",
        "Lat": 33.86105,
        "Long": -118.81843,
        "Altitude": 30000,
        "GroundSpeed": 389,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:51:38.762Z",
        "Lat": 33.8709,
        "Long": -118.83188,
        "Altitude": 30000,
        "GroundSpeed": 389,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:51:39.223Z",
        "Lat": 33.87141,
        "Long": -118.83345,
        "Altitude": 30000,
        "GroundSpeed": 389,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:51:39.803Z",
        "Lat": 33.87211,
        "Long": -118.8335,
        "Altitude": 30000,
        "GroundSpeed": 389,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:51:40.263Z",
        "Lat": 33.87263,
        "Long": -118.83519,
        "Altitude": 30000,
        "GroundSpeed": 389,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:15:09.73Z",
        "Lat": 35.81333,
        "Long": -120.80298,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:15:10.212Z",
        "Lat": 35.81403,
        "Long": -120.80298,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:15:10.791Z",
        "Lat": 35.81493,
        "Long": -120.80418,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:15:11.221Z",
        "Lat": 35.81561,
        "Long": -120.80544,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:15:11.681Z",
        "Lat": 35.81631,
        "Long": -120.80555,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:15:12.261Z",
        "Lat": 35.81721,
        "Long": -120.80669,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:15:12.718Z",
        "Lat": 35.8179,
        "Long": -120.80669,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:15:13.172Z",
        "Lat": 35.81859,
        "Long": -120.80675,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:15:13.612Z",
        "Lat": 35.81929,
        "Long": -120.80794,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:15:15.162Z",
        "Lat": 35.82166,
        "Long": -120.80921,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:15:15.722Z",
        "Lat": 35.82251,
        "Long": -120.8104,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3221"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:59.237Z",
        "Lat": 33.09566,
        "Long": -117.72534,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:00.337Z",
        "Lat": 33.09673,
        "Long": -117.7268,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:01.328Z",
        "Lat": 33.09771,
        "Long": -117.7282,
        "Altitude": 36000,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:01.909Z",
        "Lat": 33.09827,
        "Long": -117.72815,
        "Altitude": 36000,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:02.488Z",
        "Lat": 33.09888,
        "Long": -117.72949,
        "Altitude": 36000,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:02.949Z",
        "Lat": 33.09933,
        "Long": -117.7296,
        "Altitude": 36000,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:03.509Z",
        "Lat": 33.09985,
        "Long": -117.73089,
        "Altitude": 36000,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:04.009Z",
        "Lat": 33.10036,
        "Long": -117.73089,
        "Altitude": 36000,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:05.059Z",
        "Lat": 33.10139,
        "Long": -117.73224,
        "Altitude": 36000,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:06.42Z",
        "Lat": 33.10272,
        "Long": -117.7351,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:38.533Z",
        "Lat": 33.13426,
        "Long": -117.77937,
        "Altitude": 35675,
        "GroundSpeed": 327,
        "Heading": 311,
        "VerticalRate": -1344,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:42.325Z",
        "Lat": 33.13802,
        "Long": -117.78492,
        "Altitude": 35575,
        "GroundSpeed": 327,
        "Heading": 311,
        "VerticalRate": -1536,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:36.374Z",
        "Lat": 33.38745,
        "Long": -118.13785,
        "Altitude": 30075,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -640,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:36.804Z",
        "Lat": 33.38791,
        "Long": -118.13791,
        "Altitude": 30075,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -576,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:37.345Z",
        "Lat": 33.38857,
        "Long": -118.13957,
        "Altitude": 30075,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -576,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:38.365Z",
        "Lat": 33.38974,
        "Long": -118.14109,
        "Altitude": 30050,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -576,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:38.935Z",
        "Lat": 33.39038,
        "Long": -118.1412,
        "Altitude": 30050,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -576,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:39.455Z",
        "Lat": 33.39099,
        "Long": -118.14288,
        "Altitude": 30050,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -448,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:39.866Z",
        "Lat": 33.39145,
        "Long": -118.14271,
        "Altitude": 30050,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -384,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:48.789Z",
        "Lat": 33.40169,
        "Long": -118.15735,
        "Altitude": 30025,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -128,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:49.81Z",
        "Lat": 33.40286,
        "Long": -118.15902,
        "Altitude": 30025,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -128,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:02.345Z",
        "Lat": 33.41725,
        "Long": -118.18021,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:02.925Z",
        "Lat": 33.41789,
        "Long": -118.18021,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:03.505Z",
        "Lat": 33.41859,
        "Long": -118.18178,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:03.956Z",
        "Lat": 33.41911,
        "Long": -118.18189,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:04.466Z",
        "Lat": 33.41968,
        "Long": -118.18334,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:05.006Z",
        "Lat": 33.42032,
        "Long": -118.18345,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:05.506Z",
        "Lat": 33.42088,
        "Long": -118.18503,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:06.036Z",
        "Lat": 33.42148,
        "Long": -118.18509,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:51:31.25Z",
        "Lat": 33.86224,
        "Long": -118.82011,
        "Altitude": 30000,
        "GroundSpeed": 389,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:51:32.29Z",
        "Lat": 33.86343,
        "Long": -118.82179,
        "Altitude": 30000,
        "GroundSpeed": 389,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:51:32.81Z",
        "Lat": 33.86403,
        "Long": -118.82179,
        "Altitude": 30000,
        "GroundSpeed": 389,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:51:33.39Z",
        "Lat": 33.8647,
        "Long": -118.82343,
        "Altitude": 30000,
        "GroundSpeed": 389,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:51:33.85Z",
        "Lat": 33.86522,
        "Long": -118.82355,
        "Altitude": 30000,
        "GroundSpeed": 389,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:51:34.43Z",
        "Lat": 33.86591,
        "Long": -118.82515,
        "Altitude": 30000,
        "GroundSpeed": 389,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:51:34.871Z",
        "Lat": 33.86641,
        "Long": -118.8251,
        "Altitude": 30000,
        "GroundSpeed": 389,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:51:35.351Z",
        "Lat": 33.86699,
        "Long": -118.82687,
        "Altitude": 30000,
        "GroundSpeed": 389,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:51:35.771Z",
        "Lat": 33.86745,
        "Long": -118.82675,
        "Altitude": 30000,
        "GroundSpeed": 389,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:51:36.341Z",
        "Lat": 33.8681,
        "Long": -118.82852,
        "Altitude": 30000,
        "GroundSpeed": 389,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:16:36.267Z",
        "Lat": 35.94635,
        "Long": -120.91112,
        "Altitude": 29975,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": -128,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:16:36.766Z",
        "Lat": 35.94713,
        "Long": -120.91118,
        "Altitude": 29975,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": -64,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:16:37.317Z",
        "Lat": 35.94796,
        "Long": -120.91237,
        "Altitude": 29975,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": -64,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:16:37.897Z",
        "Lat": 35.94885,
        "Long": -120.91237,
        "Altitude": 29975,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": -64,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:16:38.968Z",
        "Lat": 35.95047,
        "Long": -120.91364,
        "Altitude": 29975,
        "GroundSpeed": 394,
        "Heading": 328,
        "VerticalRate": -64,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:16:39.878Z",
        "Lat": 35.95187,
        "Long": -120.91488,
        "Altitude": 29975,
        "GroundSpeed": 394,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:16:40.278Z",
        "Lat": 35.95248,
        "Long": -120.9161,
        "Altitude": 29975,
        "GroundSpeed": 394,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:16:40.869Z",
        "Lat": 35.9534,
        "Long": -120.91616,
        "Altitude": 29975,
        "GroundSpeed": 394,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:16:41.446Z",
        "Lat": 35.9543,
        "Long": -120.91734,
        "Altitude": 29975,
        "GroundSpeed": 394,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:16:43.42Z",
        "Lat": 35.95727,
        "Long": -120.91973,
        "Altitude": 29975,
        "GroundSpeed": 393,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:16:43.9Z",
        "Lat": 35.95802,
        "Long": -120.91973,
        "Altitude": 30000,
        "GroundSpeed": 393,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:16:44.44Z",
        "Lat": 35.95885,
        "Long": -120.92102,
        "Altitude": 30000,
        "GroundSpeed": 392,
        "Heading": 328,
        "VerticalRate": 64,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:16:45.01Z",
        "Lat": 35.95972,
        "Long": -120.92096,
        "Altitude": 30000,
        "GroundSpeed": 392,
        "Heading": 328,
        "VerticalRate": 64,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:16:45.53Z",
        "Lat": 35.96049,
        "Long": -120.92219,
        "Altitude": 30000,
        "GroundSpeed": 392,
        "Heading": 328,
        "VerticalRate": 64,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:16:45.941Z",
        "Lat": 35.96109,
        "Long": -120.92225,
        "Altitude": 29975,
        "GroundSpeed": 391,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:16:46.471Z",
        "Lat": 35.96191,
        "Long": -120.92348,
        "Altitude": 30000,
        "GroundSpeed": 391,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:16:47.061Z",
        "Lat": 35.96278,
        "Long": -120.92348,
        "Altitude": 30000,
        "GroundSpeed": 390,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:16:47.611Z",
        "Lat": 35.96365,
        "Long": -120.92464,
        "Altitude": 30000,
        "GroundSpeed": 390,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3221"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:44.61Z",
        "Lat": 32.96233,
        "Long": -117.53789,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:45.141Z",
        "Lat": 32.96288,
        "Long": -117.53795,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:46.101Z",
        "Lat": 32.96383,
        "Long": -117.53931,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:47.041Z",
        "Lat": 32.96475,
        "Long": -117.54075,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:47.582Z",
        "Lat": 32.96527,
        "Long": -117.54205,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:48.032Z",
        "Lat": 32.96573,
        "Long": -117.54211,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:48.462Z",
        "Lat": 32.96613,
        "Long": -117.54344,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:48.932Z",
        "Lat": 32.96663,
        "Long": -117.54349,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:58.556Z",
        "Lat": 32.97615,
        "Long": -117.55734,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:59.036Z",
        "Lat": 32.97661,
        "Long": -117.55745,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:59.607Z",
        "Lat": 32.97719,
        "Long": -117.55881,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:08.08Z",
        "Lat": 32.98557,
        "Long": -117.56991,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:08.991Z",
        "Lat": 32.9865,
        "Long": -117.57129,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:09.581Z",
        "Lat": 32.98706,
        "Long": -117.57266,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:08.528Z",
        "Lat": 33.2899,
        "Long": -117.99904,
        "Altitude": 31675,
        "GroundSpeed": 359,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:09.087Z",
        "Lat": 33.2905,
        "Long": -117.99899,
        "Altitude": 31650,
        "GroundSpeed": 359,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:09.647Z",
        "Lat": 33.29113,
        "Long": -118.00062,
        "Altitude": 31650,
        "GroundSpeed": 360,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:18.171Z",
        "Lat": 33.30026,
        "Long": -118.01273,
        "Altitude": 31400,
        "GroundSpeed": 360,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:19.152Z",
        "Lat": 33.3013,
        "Long": -118.01437,
        "Altitude": 31375,
        "GroundSpeed": 361,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:19.681Z",
        "Lat": 33.30189,
        "Long": -118.01581,
        "Altitude": 31375,
        "GroundSpeed": 361,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:21.092Z",
        "Lat": 33.30341,
        "Long": -118.01733,
        "Altitude": 31325,
        "GroundSpeed": 361,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:21.662Z",
        "Lat": 33.30398,
        "Long": -118.01884,
        "Altitude": 31325,
        "GroundSpeed": 361,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:23.013Z",
        "Lat": 33.30547,
        "Long": -118.02047,
        "Altitude": 31275,
        "GroundSpeed": 362,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:18.889Z",
        "Lat": 32.93678,
        "Long": -117.50164,
        "Altitude": 36000,
        "GroundSpeed": 332,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:19.33Z",
        "Lat": 32.9372,
        "Long": -117.5031,
        "Altitude": 36000,
        "GroundSpeed": 332,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:24.612Z",
        "Lat": 32.94246,
        "Long": -117.51004,
        "Altitude": 36000,
        "GroundSpeed": 332,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:25.172Z",
        "Lat": 32.94301,
        "Long": -117.50999,
        "Altitude": 36000,
        "GroundSpeed": 332,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:26.253Z",
        "Lat": 32.94411,
        "Long": -117.51279,
        "Altitude": 36000,
        "GroundSpeed": 332,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:26.773Z",
        "Lat": 32.94461,
        "Long": -117.51273,
        "Altitude": 36000,
        "GroundSpeed": 332,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:27.323Z",
        "Lat": 32.94516,
        "Long": -117.51419,
        "Altitude": 36000,
        "GroundSpeed": 332,
        "Heading": 311,
        "VerticalRate": 64,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:39.488Z",
        "Lat": 32.95726,
        "Long": -117.5309,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:43.18Z",
        "Lat": 32.96091,
        "Long": -117.53509,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:43.68Z",
        "Lat": 32.96141,
        "Long": -117.53645,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:10.161Z",
        "Lat": 32.98766,
        "Long": -117.57266,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:11.162Z",
        "Lat": 32.98865,
        "Long": -117.57404,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:12.612Z",
        "Lat": 32.99007,
        "Long": -117.57689,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:13.152Z",
        "Lat": 32.99062,
        "Long": -117.57695,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:32.671Z",
        "Lat": 33.01003,
        "Long": -117.60463,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:33.091Z",
        "Lat": 33.01044,
        "Long": -117.60463,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:33.921Z",
        "Lat": 33.01126,
        "Long": -117.60606,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:34.862Z",
        "Lat": 33.01218,
        "Long": -117.60738,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:35.291Z",
        "Lat": 33.01261,
        "Long": -117.60881,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:35.712Z",
        "Lat": 33.01303,
        "Long": -117.60881,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:37.713Z",
        "Lat": 33.01503,
        "Long": -117.61161,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:40.824Z",
        "Lat": 33.01813,
        "Long": -117.61578,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:41.334Z",
        "Lat": 33.01862,
        "Long": -117.61722,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:41.865Z",
        "Lat": 33.01913,
        "Long": -117.61711,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:42.965Z",
        "Lat": 33.02023,
        "Long": -117.61848,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:45.346Z",
        "Lat": 33.02262,
        "Long": -117.62277,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:08.356Z",
        "Lat": 33.04546,
        "Long": -117.65473,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:13.818Z",
        "Lat": 33.05088,
        "Long": -117.66161,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:14.238Z",
        "Lat": 33.05127,
        "Long": -117.66297,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:17.85Z",
        "Lat": 33.05484,
        "Long": -117.66722,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:18.37Z",
        "Lat": 33.05534,
        "Long": -117.66852,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:19.991Z",
        "Lat": 33.05698,
        "Long": -117.66991,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:57.336Z",
        "Lat": 33.0938,
        "Long": -117.72265,
        "Altitude": 36000,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:57.807Z",
        "Lat": 33.09427,
        "Long": -117.7226,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:58.296Z",
        "Lat": 33.09471,
        "Long": -117.724,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:07.44Z",
        "Lat": 33.10376,
        "Long": -117.7365,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:08.421Z",
        "Lat": 33.10474,
        "Long": -117.73784,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:08.861Z",
        "Lat": 33.10515,
        "Long": -117.73773,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:09.771Z",
        "Lat": 33.10604,
        "Long": -117.73913,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:10.282Z",
        "Lat": 33.10657,
        "Long": -117.74053,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:10.822Z",
        "Lat": 33.10707,
        "Long": -117.74059,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:24.728Z",
        "Lat": 33.12076,
        "Long": -117.75998,
        "Altitude": 35950,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": -512,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:26.628Z",
        "Lat": 33.12263,
        "Long": -117.76273,
        "Altitude": 35925,
        "GroundSpeed": 327,
        "Heading": 311,
        "VerticalRate": -768,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:06.206Z",
        "Lat": 33.16144,
        "Long": -117.81688,
        "Altitude": 35000,
        "GroundSpeed": 332,
        "Heading": 311,
        "VerticalRate": -1792,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:08.267Z",
        "Lat": 33.16351,
        "Long": -117.82106,
        "Altitude": 34925,
        "GroundSpeed": 333,
        "Heading": 311,
        "VerticalRate": -1792,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:09.227Z",
        "Lat": 33.16447,
        "Long": -117.82248,
        "Altitude": 34900,
        "GroundSpeed": 333,
        "Heading": 311,
        "VerticalRate": -1792,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:09.767Z",
        "Lat": 33.16502,
        "Long": -117.82243,
        "Altitude": 34900,
        "GroundSpeed": 333,
        "Heading": 311,
        "VerticalRate": -1792,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:23.653Z",
        "Lat": 33.17913,
        "Long": -117.84249,
        "Altitude": 34500,
        "GroundSpeed": 341,
        "Heading": 311,
        "VerticalRate": -1792,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:24.203Z",
        "Lat": 33.17969,
        "Long": -117.84244,
        "Altitude": 34475,
        "GroundSpeed": 341,
        "Heading": 311,
        "VerticalRate": -1792,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:25.073Z",
        "Lat": 33.18059,
        "Long": -117.8438,
        "Altitude": 34450,
        "GroundSpeed": 341,
        "Heading": 311,
        "VerticalRate": -1792,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:25.664Z",
        "Lat": 33.18118,
        "Long": -117.84524,
        "Altitude": 34450,
        "GroundSpeed": 342,
        "Heading": 311,
        "VerticalRate": -1792,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:28.395Z",
        "Lat": 33.18402,
        "Long": -117.84963,
        "Altitude": 34375,
        "GroundSpeed": 342,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:40.97Z",
        "Lat": 33.19702,
        "Long": -117.86699,
        "Altitude": 34025,
        "GroundSpeed": 346,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:41.42Z",
        "Lat": 33.19752,
        "Long": -117.8685,
        "Altitude": 34025,
        "GroundSpeed": 346,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:42.52Z",
        "Lat": 33.19867,
        "Long": -117.87001,
        "Altitude": 34000,
        "GroundSpeed": 346,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:48.653Z",
        "Lat": 33.20503,
        "Long": -117.8788,
        "Altitude": 33825,
        "GroundSpeed": 348,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:49.604Z",
        "Lat": 33.20604,
        "Long": -117.88022,
        "Altitude": 33800,
        "GroundSpeed": 348,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:50.184Z",
        "Lat": 33.20664,
        "Long": -117.88016,
        "Altitude": 33775,
        "GroundSpeed": 348,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:38.364Z",
        "Lat": 33.25749,
        "Long": -117.95339,
        "Altitude": 32500,
        "GroundSpeed": 358,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:39.306Z",
        "Lat": 33.2585,
        "Long": -117.95499,
        "Altitude": 32450,
        "GroundSpeed": 358,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:42.426Z",
        "Lat": 33.26184,
        "Long": -117.95955,
        "Altitude": 32375,
        "GroundSpeed": 358,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:42.846Z",
        "Lat": 33.2623,
        "Long": -117.9596,
        "Altitude": 32375,
        "GroundSpeed": 358,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:43.937Z",
        "Lat": 33.26344,
        "Long": -117.9611,
        "Altitude": 32325,
        "GroundSpeed": 359,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:44.496Z",
        "Lat": 33.26404,
        "Long": -117.96257,
        "Altitude": 32325,
        "GroundSpeed": 359,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:44.967Z",
        "Lat": 33.26454,
        "Long": -117.96251,
        "Altitude": 32300,
        "GroundSpeed": 359,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:46.012Z",
        "Lat": 33.26567,
        "Long": -117.96407,
        "Altitude": 32275,
        "GroundSpeed": 359,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:52.52Z",
        "Lat": 33.27269,
        "Long": -117.97471,
        "Altitude": 32100,
        "GroundSpeed": 359,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:53.351Z",
        "Lat": 33.27358,
        "Long": -117.97629,
        "Altitude": 32075,
        "GroundSpeed": 359,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:53.941Z",
        "Lat": 33.27424,
        "Long": -117.97629,
        "Altitude": 32075,
        "GroundSpeed": 359,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:54.521Z",
        "Lat": 33.27484,
        "Long": -117.97778,
        "Altitude": 32050,
        "GroundSpeed": 359,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:55.101Z",
        "Lat": 33.27548,
        "Long": -117.97773,
        "Altitude": 32025,
        "GroundSpeed": 359,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:55.552Z",
        "Lat": 33.27596,
        "Long": -117.97932,
        "Altitude": 32025,
        "GroundSpeed": 359,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:29.556Z",
        "Lat": 33.31255,
        "Long": -118.03134,
        "Altitude": 31100,
        "GroundSpeed": 364,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:32.056Z",
        "Lat": 33.31525,
        "Long": -118.03442,
        "Altitude": 31025,
        "GroundSpeed": 364,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:33.477Z",
        "Lat": 33.31678,
        "Long": -118.03739,
        "Altitude": 31000,
        "GroundSpeed": 364,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:34.658Z",
        "Lat": 33.3181,
        "Long": -118.03903,
        "Altitude": 30950,
        "GroundSpeed": 365,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:35.178Z",
        "Lat": 33.31865,
        "Long": -118.03892,
        "Altitude": 30950,
        "GroundSpeed": 365,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:35.749Z",
        "Lat": 33.3193,
        "Long": -118.04058,
        "Altitude": 30925,
        "GroundSpeed": 365,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:36.249Z",
        "Lat": 33.31984,
        "Long": -118.04205,
        "Altitude": 30925,
        "GroundSpeed": 365,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:36.799Z",
        "Lat": 33.32043,
        "Long": -118.04205,
        "Altitude": 30900,
        "GroundSpeed": 365,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:43:38.96Z",
        "Lat": 33.32277,
        "Long": -118.04513,
        "Altitude": 30850,
        "GroundSpeed": 365,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:06.721Z",
        "Lat": 33.35367,
        "Long": -118.08951,
        "Altitude": 30425,
        "GroundSpeed": 378,
        "Heading": 311,
        "VerticalRate": -768,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:07.301Z",
        "Lat": 33.35435,
        "Long": -118.09109,
        "Altitude": 30425,
        "GroundSpeed": 378,
        "Heading": 311,
        "VerticalRate": -768,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:44:07.822Z",
        "Lat": 33.35491,
        "Long": -118.09114,
        "Altitude": 30400,
        "GroundSpeed": 378,
        "Heading": 311,
        "VerticalRate": -768,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:26.575Z",
        "Lat": 33.44499,
        "Long": -118.21915,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:27.155Z",
        "Lat": 33.44568,
        "Long": -118.21926,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:27.555Z",
        "Lat": 33.44611,
        "Long": -118.22091,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:28.086Z",
        "Lat": 33.44671,
        "Long": -118.22079,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:28.936Z",
        "Lat": 33.44769,
        "Long": -118.2225,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:29.496Z",
        "Lat": 33.44834,
        "Long": -118.22405,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:30.067Z",
        "Lat": 33.44899,
        "Long": -118.22399,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:30.977Z",
        "Lat": 33.45003,
        "Long": -118.22563,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:31.417Z",
        "Lat": 33.45053,
        "Long": -118.22735,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:31.967Z",
        "Lat": 33.45113,
        "Long": -118.2273,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:32.988Z",
        "Lat": 33.45232,
        "Long": -118.22893,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:34.139Z",
        "Lat": 33.45365,
        "Long": -118.23066,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:34.598Z",
        "Lat": 33.45415,
        "Long": -118.23223,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:35.099Z",
        "Lat": 33.45474,
        "Long": -118.23223,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:36.019Z",
        "Lat": 33.45579,
        "Long": -118.23385,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:36.939Z",
        "Lat": 33.4568,
        "Long": -118.23552,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:37.519Z",
        "Lat": 33.45747,
        "Long": -118.23711,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:37.96Z",
        "Lat": 33.45798,
        "Long": -118.23716,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:38.45Z",
        "Lat": 33.45854,
        "Long": -118.23882,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:39.031Z",
        "Lat": 33.45918,
        "Long": -118.23876,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:39.451Z",
        "Lat": 33.4597,
        "Long": -118.24041,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:39.911Z",
        "Lat": 33.46021,
        "Long": -118.24036,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:48.274Z",
        "Lat": 33.46971,
        "Long": -118.25497,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:45:49.815Z",
        "Lat": 33.47148,
        "Long": -118.25667,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:52.301Z",
        "Lat": 33.54245,
        "Long": -118.3593,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:53.812Z",
        "Lat": 33.54419,
        "Long": -118.36092,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:54.232Z",
        "Lat": 33.54465,
        "Long": -118.36255,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:54.702Z",
        "Lat": 33.5452,
        "Long": -118.36255,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:55.173Z",
        "Lat": 33.54575,
        "Long": -118.3625,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:55.603Z",
        "Lat": 33.54619,
        "Long": -118.36418,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:56.113Z",
        "Lat": 33.5468,
        "Long": -118.36418,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:56.674Z",
        "Lat": 33.54744,
        "Long": -118.36575,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:57.723Z",
        "Lat": 33.54862,
        "Long": -118.3675,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:58.215Z",
        "Lat": 33.54918,
        "Long": -118.36905,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:58.784Z",
        "Lat": 33.54982,
        "Long": -118.36905,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:59.235Z",
        "Lat": 33.55034,
        "Long": -118.3707,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:46:59.654Z",
        "Lat": 33.55085,
        "Long": -118.37082,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:10.749Z",
        "Lat": 33.56351,
        "Long": -118.38884,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:11.33Z",
        "Lat": 33.56416,
        "Long": -118.39045,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:11.91Z",
        "Lat": 33.56482,
        "Long": -118.39039,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:12.36Z",
        "Lat": 33.56534,
        "Long": -118.39209,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:26.926Z",
        "Lat": 33.582,
        "Long": -118.41502,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:27.396Z",
        "Lat": 33.58255,
        "Long": -118.41665,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:27.827Z",
        "Lat": 33.58306,
        "Long": -118.41671,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:28.376Z",
        "Lat": 33.58369,
        "Long": -118.41827,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:29.387Z",
        "Lat": 33.58483,
        "Long": -118.41991,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:29.838Z",
        "Lat": 33.58534,
        "Long": -118.41997,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:30.367Z",
        "Lat": 33.58598,
        "Long": -118.42152,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:30.888Z",
        "Lat": 33.58658,
        "Long": -118.42157,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:31.417Z",
        "Lat": 33.58716,
        "Long": -118.42323,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:32.408Z",
        "Lat": 33.58832,
        "Long": -118.42488,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:33.478Z",
        "Lat": 33.58953,
        "Long": -118.42644,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:34.979Z",
        "Lat": 33.59125,
        "Long": -118.42813,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:35.559Z",
        "Lat": 33.59191,
        "Long": -118.42987,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:35.96Z",
        "Lat": 33.59237,
        "Long": -118.42987,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:36.49Z",
        "Lat": 33.59299,
        "Long": -118.43144,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:36.91Z",
        "Lat": 33.59349,
        "Long": -118.43144,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:37.34Z",
        "Lat": 33.59396,
        "Long": -118.43313,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:37.9Z",
        "Lat": 33.59461,
        "Long": -118.43307,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:38.471Z",
        "Lat": 33.59528,
        "Long": -118.43469,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:53.057Z",
        "Lat": 33.61198,
        "Long": -118.45784,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:54.137Z",
        "Lat": 33.61323,
        "Long": -118.45945,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:55.768Z",
        "Lat": 33.61509,
        "Long": -118.4626,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:56.178Z",
        "Lat": 33.61556,
        "Long": -118.46266,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:47:56.678Z",
        "Lat": 33.61615,
        "Long": -118.46429,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:17.237Z",
        "Lat": 33.63972,
        "Long": -118.49882,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:17.687Z",
        "Lat": 33.64023,
        "Long": -118.49882,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:18.097Z",
        "Lat": 33.6407,
        "Long": -118.49882,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:18.617Z",
        "Lat": 33.64133,
        "Long": -118.50055,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:58.885Z",
        "Lat": 33.68747,
        "Long": -118.56636,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:59.365Z",
        "Lat": 33.68799,
        "Long": -118.56794,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:48:59.825Z",
        "Lat": 33.68855,
        "Long": -118.56806,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:00.246Z",
        "Lat": 33.68903,
        "Long": -118.56967,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:00.705Z",
        "Lat": 33.68953,
        "Long": -118.56967,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:01.135Z",
        "Lat": 33.69003,
        "Long": -118.56972,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:01.686Z",
        "Lat": 33.69069,
        "Long": -118.57132,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:02.216Z",
        "Lat": 33.69127,
        "Long": -118.57292,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:02.636Z",
        "Lat": 33.69177,
        "Long": -118.57303,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:07.188Z",
        "Lat": 33.69699,
        "Long": -118.57953,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:07.638Z",
        "Lat": 33.69754,
        "Long": -118.58128,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:08.058Z",
        "Lat": 33.698,
        "Long": -118.58133,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:08.618Z",
        "Lat": 33.69864,
        "Long": -118.58289,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:09.149Z",
        "Lat": 33.69923,
        "Long": -118.58295,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:09.709Z",
        "Lat": 33.69991,
        "Long": -118.58454,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:10.179Z",
        "Lat": 33.70042,
        "Long": -118.58448,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:10.71Z",
        "Lat": 33.70102,
        "Long": -118.5862,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:15.111Z",
        "Lat": 33.7061,
        "Long": -118.59276,
        "Altitude": 30000,
        "GroundSpeed": 387,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:15.942Z",
        "Lat": 33.70703,
        "Long": -118.59449,
        "Altitude": 30000,
        "GroundSpeed": 387,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:16.412Z",
        "Lat": 33.70757,
        "Long": -118.59607,
        "Altitude": 30000,
        "GroundSpeed": 387,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:16.882Z",
        "Lat": 33.70811,
        "Long": -118.59601,
        "Altitude": 30000,
        "GroundSpeed": 387,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:17.313Z",
        "Lat": 33.70862,
        "Long": -118.59776,
        "Altitude": 30000,
        "GroundSpeed": 387,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:17.733Z",
        "Lat": 33.70908,
        "Long": -118.59776,
        "Altitude": 30000,
        "GroundSpeed": 387,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:18.563Z",
        "Lat": 33.71004,
        "Long": -118.59932,
        "Altitude": 30000,
        "GroundSpeed": 387,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:35.8Z",
        "Lat": 33.7298,
        "Long": -118.62745,
        "Altitude": 30000,
        "GroundSpeed": 387,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:36.231Z",
        "Lat": 33.73027,
        "Long": -118.62908,
        "Altitude": 30000,
        "GroundSpeed": 387,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:49:36.781Z",
        "Lat": 33.73091,
        "Long": -118.62908,
        "Altitude": 30000,
        "GroundSpeed": 387,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:50:42.789Z",
        "Lat": 33.80649,
        "Long": -118.7385,
        "Altitude": 30000,
        "GroundSpeed": 388,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:50:43.349Z",
        "Lat": 33.80712,
        "Long": -118.74029,
        "Altitude": 30000,
        "GroundSpeed": 388,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:50:43.929Z",
        "Lat": 33.80777,
        "Long": -118.74023,
        "Altitude": 30000,
        "GroundSpeed": 388,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:50:44.509Z",
        "Lat": 33.80846,
        "Long": -118.74192,
        "Altitude": 30000,
        "GroundSpeed": 388,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:50:44.97Z",
        "Lat": 33.80896,
        "Long": -118.7418,
        "Altitude": 30000,
        "GroundSpeed": 388,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:50:45.53Z",
        "Lat": 33.80963,
        "Long": -118.74361,
        "Altitude": 30000,
        "GroundSpeed": 388,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:50:46.031Z",
        "Lat": 33.81019,
        "Long": -118.74361,
        "Altitude": 30000,
        "GroundSpeed": 388,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:50:46.53Z",
        "Lat": 33.81075,
        "Long": -118.74528,
        "Altitude": 30000,
        "GroundSpeed": 388,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:50:47.08Z",
        "Lat": 33.81139,
        "Long": -118.74517,
        "Altitude": 30000,
        "GroundSpeed": 388,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:50:47.491Z",
        "Lat": 33.81187,
        "Long": -118.74687,
        "Altitude": 30000,
        "GroundSpeed": 388,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:50:47.951Z",
        "Lat": 33.81238,
        "Long": -118.74693,
        "Altitude": 30000,
        "GroundSpeed": 388,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:50:48.441Z",
        "Lat": 33.81294,
        "Long": -118.74859,
        "Altitude": 30000,
        "GroundSpeed": 388,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:50:57.035Z",
        "Lat": 33.82283,
        "Long": -118.76193,
        "Altitude": 30000,
        "GroundSpeed": 388,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:50:57.575Z",
        "Lat": 33.82346,
        "Long": -118.76347,
        "Altitude": 30000,
        "GroundSpeed": 388,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:50:58.105Z",
        "Lat": 33.82406,
        "Long": -118.76352,
        "Altitude": 30000,
        "GroundSpeed": 388,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:50:58.505Z",
        "Lat": 33.82452,
        "Long": -118.76518,
        "Altitude": 30000,
        "GroundSpeed": 388,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:50:59.015Z",
        "Lat": 33.82512,
        "Long": -118.76518,
        "Altitude": 30000,
        "GroundSpeed": 388,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:50:59.415Z",
        "Lat": 33.8256,
        "Long": -118.76684,
        "Altitude": 30000,
        "GroundSpeed": 388,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:51:00.006Z",
        "Lat": 33.82625,
        "Long": -118.76684,
        "Altitude": 30000,
        "GroundSpeed": 388,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:51:00.586Z",
        "Lat": 33.82695,
        "Long": -118.76848,
        "Altitude": 30000,
        "GroundSpeed": 388,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:52:10.946Z",
        "Lat": 33.90788,
        "Long": -118.88524,
        "Altitude": 30000,
        "GroundSpeed": 388,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:52:11.526Z",
        "Lat": 33.90856,
        "Long": -118.88689,
        "Altitude": 30000,
        "GroundSpeed": 388,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:52:12.056Z",
        "Lat": 33.90916,
        "Long": -118.88695,
        "Altitude": 30000,
        "GroundSpeed": 388,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:52:12.577Z",
        "Lat": 33.90976,
        "Long": -118.88849,
        "Altitude": 30000,
        "GroundSpeed": 388,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:52:12.987Z",
        "Lat": 33.91026,
        "Long": -118.88849,
        "Altitude": 30000,
        "GroundSpeed": 388,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:52:13.487Z",
        "Lat": 33.91084,
        "Long": -118.89015,
        "Altitude": 30000,
        "GroundSpeed": 388,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:52:22.992Z",
        "Lat": 33.9218,
        "Long": -118.90514,
        "Altitude": 30000,
        "GroundSpeed": 388,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:52:23.441Z",
        "Lat": 33.92229,
        "Long": -118.90686,
        "Altitude": 30000,
        "GroundSpeed": 388,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:52:24.001Z",
        "Lat": 33.92294,
        "Long": -118.9068,
        "Altitude": 30000,
        "GroundSpeed": 388,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:52:27.042Z",
        "Lat": 33.92642,
        "Long": -118.91187,
        "Altitude": 30000,
        "GroundSpeed": 388,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:52:28.053Z",
        "Lat": 33.9276,
        "Long": -118.9135,
        "Altitude": 30000,
        "GroundSpeed": 388,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:52:38.828Z",
        "Lat": 33.93997,
        "Long": -118.93177,
        "Altitude": 30000,
        "GroundSpeed": 387,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:52:39.768Z",
        "Lat": 33.94105,
        "Long": -118.9333,
        "Altitude": 30000,
        "GroundSpeed": 387,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:52:40.918Z",
        "Lat": 33.9424,
        "Long": -118.93502,
        "Altitude": 30000,
        "GroundSpeed": 387,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:52:42.009Z",
        "Lat": 33.94366,
        "Long": -118.93667,
        "Altitude": 30000,
        "GroundSpeed": 387,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:52:42.95Z",
        "Lat": 33.94473,
        "Long": -118.93838,
        "Altitude": 30000,
        "GroundSpeed": 387,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:52:43.439Z",
        "Lat": 33.94529,
        "Long": -118.94005,
        "Altitude": 30000,
        "GroundSpeed": 387,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:52:43.97Z",
        "Lat": 33.94589,
        "Long": -118.93993,
        "Altitude": 30000,
        "GroundSpeed": 387,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:52:44.93Z",
        "Lat": 33.94698,
        "Long": -118.94169,
        "Altitude": 30000,
        "GroundSpeed": 387,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:52:45.871Z",
        "Lat": 33.94808,
        "Long": -118.94325,
        "Altitude": 30000,
        "GroundSpeed": 387,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:52:46.411Z",
        "Lat": 33.94867,
        "Long": -118.94499,
        "Altitude": 30000,
        "GroundSpeed": 387,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:53:16.174Z",
        "Lat": 33.98267,
        "Long": -118.99303,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:53:19.024Z",
        "Lat": 33.98589,
        "Long": -118.99813,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:53:19.444Z",
        "Lat": 33.98635,
        "Long": -118.99984,
        "Altitude": 30000,
        "GroundSpeed": 386,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:54:10.557Z",
        "Lat": 34.04434,
        "Long": -119.08473,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:54:11.136Z",
        "Lat": 34.04498,
        "Long": -119.08468,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:54:11.577Z",
        "Lat": 34.04547,
        "Long": -119.08625,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:54:13.528Z",
        "Lat": 34.0477,
        "Long": -119.08968,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:54:14.078Z",
        "Lat": 34.04831,
        "Long": -119.08957,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:55:31.34Z",
        "Lat": 34.1361,
        "Long": -119.21906,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:31.85Z",
        "Lat": 34.20444,
        "Long": -119.31862,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:32.35Z",
        "Lat": 34.20502,
        "Long": -119.32032,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:33.44Z",
        "Lat": 34.20626,
        "Long": -119.32194,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:33.95Z",
        "Lat": 34.20682,
        "Long": -119.32194,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:34.54Z",
        "Lat": 34.20749,
        "Long": -119.32363,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:35.551Z",
        "Lat": 34.20863,
        "Long": -119.32531,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:56:36.698Z",
        "Lat": 34.20996,
        "Long": -119.32688,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:38.222Z",
        "Lat": 34.21166,
        "Long": -119.33012,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:52.772Z",
        "Lat": 34.29556,
        "Long": -119.45289,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:53.351Z",
        "Lat": 34.2962,
        "Long": -119.45469,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:53.931Z",
        "Lat": 34.29685,
        "Long": -119.45463,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:54.381Z",
        "Lat": 34.29735,
        "Long": -119.45625,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:55.432Z",
        "Lat": 34.29853,
        "Long": -119.45795,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:55.932Z",
        "Lat": 34.29908,
        "Long": -119.45795,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:17:52.389Z",
        "Lat": 36.05516,
        "Long": -121.00571,
        "Altitude": 30000,
        "GroundSpeed": 368,
        "Heading": 323,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:17:52.919Z",
        "Lat": 36.05589,
        "Long": -121.00565,
        "Altitude": 30000,
        "GroundSpeed": 368,
        "Heading": 323,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:17:53.439Z",
        "Lat": 36.05657,
        "Long": -121.00698,
        "Altitude": 30000,
        "GroundSpeed": 368,
        "Heading": 323,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:17:53.849Z",
        "Lat": 36.05713,
        "Long": -121.00698,
        "Altitude": 30000,
        "GroundSpeed": 368,
        "Heading": 323,
        "VerticalRate": 0,
        "Squawk": "3221"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:17:54.349Z",
        "Lat": 36.05782,
        "Long": -121.00822,
        "Altitude": 30000,
        "GroundSpeed": 368,
        "Heading": 323,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:17:54.86Z",
        "Lat": 36.0585,
        "Long": -121.00828,
        "Altitude": 30000,
        "GroundSpeed": 368,
        "Heading": 323,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:17:55.437Z",
        "Lat": 36.05927,
        "Long": -121.0095,
        "Altitude": 30000,
        "GroundSpeed": 368,
        "Heading": 323,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:17:55.89Z",
        "Lat": 36.05988,
        "Long": -121.00961,
        "Altitude": 30000,
        "GroundSpeed": 368,
        "Heading": 323,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:17:56.391Z",
        "Lat": 36.06056,
        "Long": -121.01086,
        "Altitude": 30000,
        "GroundSpeed": 368,
        "Heading": 323,
        "VerticalRate": 0,
        "Squawk": "3221"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:17:56.888Z",
        "Lat": 36.06125,
        "Long": -121.01074,
        "Altitude": 30000,
        "GroundSpeed": 368,
        "Heading": 323,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:17:57.478Z",
        "Lat": 36.06206,
        "Long": -121.01213,
        "Altitude": 30000,
        "GroundSpeed": 368,
        "Heading": 323,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:17:57.991Z",
        "Lat": 36.06276,
        "Long": -121.01201,
        "Altitude": 30000,
        "GroundSpeed": 368,
        "Heading": 323,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:17:58.581Z",
        "Lat": 36.06354,
        "Long": -121.01343,
        "Altitude": 30000,
        "GroundSpeed": 368,
        "Heading": 323,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:17:59.021Z",
        "Lat": 36.06413,
        "Long": -121.01337,
        "Altitude": 30000,
        "GroundSpeed": 368,
        "Heading": 323,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:17:59.592Z",
        "Lat": 36.0649,
        "Long": -121.01464,
        "Altitude": 30000,
        "GroundSpeed": 368,
        "Heading": 323,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:18:00.162Z",
        "Lat": 36.06569,
        "Long": -121.0147,
        "Altitude": 30000,
        "GroundSpeed": 368,
        "Heading": 323,
        "VerticalRate": 0,
        "Squawk": "3221"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:18:00.749Z",
        "Lat": 36.06647,
        "Long": -121.01595,
        "Altitude": 30000,
        "GroundSpeed": 368,
        "Heading": 323,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:18:01.233Z",
        "Lat": 36.06714,
        "Long": -121.01721,
        "Altitude": 30000,
        "GroundSpeed": 368,
        "Heading": 323,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:18:01.773Z",
        "Lat": 36.06788,
        "Long": -121.01727,
        "Altitude": 30000,
        "GroundSpeed": 368,
        "Heading": 323,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:18:02.263Z",
        "Lat": 36.06853,
        "Long": -121.01852,
        "Altitude": 30000,
        "GroundSpeed": 368,
        "Heading": 323,
        "VerticalRate": 0,
        "Squawk": "3221"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:18:02.763Z",
        "Lat": 36.06921,
        "Long": -121.01858,
        "Altitude": 30000,
        "GroundSpeed": 368,
        "Heading": 323,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:18:03.854Z",
        "Lat": 36.07068,
        "Long": -121.01984,
        "Altitude": 30000,
        "GroundSpeed": 368,
        "Heading": 323,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:18:57.913Z",
        "Lat": 36.14381,
        "Long": -121.08915,
        "Altitude": 29200,
        "GroundSpeed": 365,
        "Heading": 323,
        "VerticalRate": -2304,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:18:58.374Z",
        "Lat": 36.14447,
        "Long": -121.09051,
        "Altitude": 29200,
        "GroundSpeed": 365,
        "Heading": 323,
        "VerticalRate": -2304,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:18:58.804Z",
        "Lat": 36.14502,
        "Long": -121.09045,
        "Altitude": 29175,
        "GroundSpeed": 365,
        "Heading": 323,
        "VerticalRate": -2368,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:18:59.357Z",
        "Lat": 36.14576,
        "Long": -121.09178,
        "Altitude": 29150,
        "GroundSpeed": 365,
        "Heading": 323,
        "VerticalRate": -2368,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:18:59.884Z",
        "Lat": 36.14646,
        "Long": -121.09178,
        "Altitude": 29125,
        "GroundSpeed": 365,
        "Heading": 323,
        "VerticalRate": -2368,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:19:00.304Z",
        "Lat": 36.14703,
        "Long": -121.09297,
        "Altitude": 29125,
        "GroundSpeed": 365,
        "Heading": 323,
        "VerticalRate": -2368,
        "Squawk": "3221"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:19:00.778Z",
        "Lat": 36.14767,
        "Long": -121.09308,
        "Altitude": 29100,
        "GroundSpeed": 365,
        "Heading": 323,
        "VerticalRate": -2368,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:19:04.286Z",
        "Lat": 36.15239,
        "Long": -121.09812,
        "Altitude": 28975,
        "GroundSpeed": 365,
        "Heading": 323,
        "VerticalRate": -2368,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:19:04.856Z",
        "Lat": 36.15312,
        "Long": -121.09812,
        "Altitude": 28950,
        "GroundSpeed": 364,
        "Heading": 323,
        "VerticalRate": -2368,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:19:05.306Z",
        "Lat": 36.15372,
        "Long": -121.09931,
        "Altitude": 28925,
        "GroundSpeed": 364,
        "Heading": 323,
        "VerticalRate": -2368,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:19:05.727Z",
        "Lat": 36.15428,
        "Long": -121.09937,
        "Altitude": 28900,
        "GroundSpeed": 364,
        "Heading": 323,
        "VerticalRate": -2368,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:19:06.287Z",
        "Lat": 36.15504,
        "Long": -121.10069,
        "Altitude": 28875,
        "GroundSpeed": 364,
        "Heading": 323,
        "VerticalRate": -2368,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:19:16.235Z",
        "Lat": 36.16827,
        "Long": -121.11334,
        "Altitude": 28500,
        "GroundSpeed": 362,
        "Heading": 323,
        "VerticalRate": -2368,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:19:16.821Z",
        "Lat": 36.16905,
        "Long": -121.11334,
        "Altitude": 28475,
        "GroundSpeed": 362,
        "Heading": 323,
        "VerticalRate": -2432,
        "Squawk": "3221"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:19:17.401Z",
        "Lat": 36.16983,
        "Long": -121.11463,
        "Altitude": 28450,
        "GroundSpeed": 362,
        "Heading": 323,
        "VerticalRate": -2432,
        "Squawk": "3221"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:19:17.985Z",
        "Lat": 36.17058,
        "Long": -121.11474,
        "Altitude": 28425,
        "GroundSpeed": 362,
        "Heading": 323,
        "VerticalRate": -2432,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:19:18.435Z",
        "Lat": 36.1712,
        "Long": -121.11591,
        "Altitude": 28400,
        "GroundSpeed": 362,
        "Heading": 323,
        "VerticalRate": -2432,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:19:18.945Z",
        "Lat": 36.17189,
        "Long": -121.11591,
        "Altitude": 28375,
        "GroundSpeed": 362,
        "Heading": 323,
        "VerticalRate": -2432,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:19:19.485Z",
        "Lat": 36.17258,
        "Long": -121.11726,
        "Altitude": 28350,
        "GroundSpeed": 362,
        "Heading": 323,
        "VerticalRate": -2432,
        "Squawk": "3221"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:19:19.985Z",
        "Lat": 36.17323,
        "Long": -121.11726,
        "Altitude": 28350,
        "GroundSpeed": 362,
        "Heading": 323,
        "VerticalRate": -2432,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:19:20.516Z",
        "Lat": 36.17395,
        "Long": -121.11843,
        "Altitude": 28325,
        "GroundSpeed": 362,
        "Heading": 323,
        "VerticalRate": -2432,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:19:21.026Z",
        "Lat": 36.17464,
        "Long": -121.11843,
        "Altitude": 28300,
        "GroundSpeed": 362,
        "Heading": 323,
        "VerticalRate": -2432,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:19:21.556Z",
        "Lat": 36.17533,
        "Long": -121.11977,
        "Altitude": 28275,
        "GroundSpeed": 362,
        "Heading": 323,
        "VerticalRate": -2432,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:19:22.077Z",
        "Lat": 36.17602,
        "Long": -121.11977,
        "Altitude": 28250,
        "GroundSpeed": 362,
        "Heading": 323,
        "VerticalRate": -2432,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:19:22.654Z",
        "Lat": 36.17679,
        "Long": -121.12101,
        "Altitude": 28250,
        "GroundSpeed": 362,
        "Heading": 323,
        "VerticalRate": -2432,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:19:23.114Z",
        "Lat": 36.17738,
        "Long": -121.12101,
        "Altitude": 28225,
        "GroundSpeed": 362,
        "Heading": 323,
        "VerticalRate": -2432,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:19:23.695Z",
        "Lat": 36.17816,
        "Long": -121.12222,
        "Altitude": 28200,
        "GroundSpeed": 362,
        "Heading": 323,
        "VerticalRate": -2432,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:19:24.137Z",
        "Lat": 36.17877,
        "Long": -121.12228,
        "Altitude": 28175,
        "GroundSpeed": 362,
        "Heading": 323,
        "VerticalRate": -2432,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:19:24.618Z",
        "Lat": 36.1794,
        "Long": -121.12352,
        "Altitude": 28150,
        "GroundSpeed": 362,
        "Heading": 323,
        "VerticalRate": -2368,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:19:25.038Z",
        "Lat": 36.17995,
        "Long": -121.12352,
        "Altitude": 28150,
        "GroundSpeed": 362,
        "Heading": 323,
        "VerticalRate": -2368,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:19:25.608Z",
        "Lat": 36.18073,
        "Long": -121.12479,
        "Altitude": 28125,
        "GroundSpeed": 362,
        "Heading": 323,
        "VerticalRate": -2368,
        "Squawk": "3221"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:19:26.068Z",
        "Lat": 36.18133,
        "Long": -121.12485,
        "Altitude": 28100,
        "GroundSpeed": 362,
        "Heading": 323,
        "VerticalRate": -2368,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:19:26.538Z",
        "Lat": 36.18196,
        "Long": -121.1261,
        "Altitude": 28075,
        "GroundSpeed": 362,
        "Heading": 323,
        "VerticalRate": -2368,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:19:27.596Z",
        "Lat": 36.18333,
        "Long": -121.12731,
        "Altitude": 28050,
        "GroundSpeed": 361,
        "Heading": 323,
        "VerticalRate": -2368,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:19:28.486Z",
        "Lat": 36.18452,
        "Long": -121.12867,
        "Altitude": 28000,
        "GroundSpeed": 361,
        "Heading": 323,
        "VerticalRate": -2368,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:19:29.067Z",
        "Lat": 36.1853,
        "Long": -121.12862,
        "Altitude": 27975,
        "GroundSpeed": 361,
        "Heading": 323,
        "VerticalRate": -2368,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:19:29.528Z",
        "Lat": 36.18589,
        "Long": -121.12988,
        "Altitude": 27975,
        "GroundSpeed": 361,
        "Heading": 323,
        "VerticalRate": -2368,
        "Squawk": "3221"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:19:29.977Z",
        "Lat": 36.1865,
        "Long": -121.12994,
        "Altitude": 27950,
        "GroundSpeed": 361,
        "Heading": 323,
        "VerticalRate": -2368,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:19:30.417Z",
        "Lat": 36.18709,
        "Long": -121.13113,
        "Altitude": 27925,
        "GroundSpeed": 361,
        "Heading": 323,
        "VerticalRate": -2368,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:19:30.9Z",
        "Lat": 36.18773,
        "Long": -121.13108,
        "Altitude": 27925,
        "GroundSpeed": 361,
        "Heading": 323,
        "VerticalRate": -2368,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:19:31.411Z",
        "Lat": 36.18841,
        "Long": -121.13233,
        "Altitude": 27900,
        "GroundSpeed": 361,
        "Heading": 323,
        "VerticalRate": -2368,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:21:11.153Z",
        "Lat": 36.31801,
        "Long": -121.25553,
        "Altitude": 24100,
        "GroundSpeed": 347,
        "Heading": 323,
        "VerticalRate": -2176,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:21:11.673Z",
        "Lat": 36.31866,
        "Long": -121.25675,
        "Altitude": 24075,
        "GroundSpeed": 347,
        "Heading": 323,
        "VerticalRate": -2112,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:21:12.203Z",
        "Lat": 36.31936,
        "Long": -121.25669,
        "Altitude": 24075,
        "GroundSpeed": 347,
        "Heading": 323,
        "VerticalRate": -2112,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:21:12.691Z",
        "Lat": 36.31998,
        "Long": -121.25788,
        "Altitude": 24050,
        "GroundSpeed": 347,
        "Heading": 323,
        "VerticalRate": -2112,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:21:13.261Z",
        "Lat": 36.32071,
        "Long": -121.25914,
        "Altitude": 24025,
        "GroundSpeed": 347,
        "Heading": 323,
        "VerticalRate": -2112,
        "Squawk": "3221"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:21:13.714Z",
        "Lat": 36.32127,
        "Long": -121.25914,
        "Altitude": 24000,
        "GroundSpeed": 347,
        "Heading": 323,
        "VerticalRate": -2112,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:21:14.134Z",
        "Lat": 36.32182,
        "Long": -121.25914,
        "Altitude": 24000,
        "GroundSpeed": 347,
        "Heading": 323,
        "VerticalRate": -2112,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:21:14.694Z",
        "Lat": 36.32254,
        "Long": -121.26034,
        "Altitude": 23975,
        "GroundSpeed": 346,
        "Heading": 323,
        "VerticalRate": -2112,
        "Squawk": "3221"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:21:17.295Z",
        "Lat": 36.32583,
        "Long": -121.26399,
        "Altitude": 23875,
        "GroundSpeed": 346,
        "Heading": 323,
        "VerticalRate": -2112,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:21:17.745Z",
        "Lat": 36.32639,
        "Long": -121.26405,
        "Altitude": 23875,
        "GroundSpeed": 346,
        "Heading": 323,
        "VerticalRate": -2112,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:21:18.225Z",
        "Lat": 36.32699,
        "Long": -121.26516,
        "Altitude": 23850,
        "GroundSpeed": 346,
        "Heading": 323,
        "VerticalRate": -2112,
        "Squawk": "3221"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:21:18.813Z",
        "Lat": 36.32776,
        "Long": -121.2652,
        "Altitude": 23825,
        "GroundSpeed": 345,
        "Heading": 323,
        "VerticalRate": -2112,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:21:19.393Z",
        "Lat": 36.32848,
        "Long": -121.26639,
        "Altitude": 23825,
        "GroundSpeed": 345,
        "Heading": 323,
        "VerticalRate": -2112,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:21:19.876Z",
        "Lat": 36.32909,
        "Long": -121.26651,
        "Altitude": 23800,
        "GroundSpeed": 345,
        "Heading": 323,
        "VerticalRate": -2112,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:21:20.354Z",
        "Lat": 36.32968,
        "Long": -121.26766,
        "Altitude": 23775,
        "GroundSpeed": 345,
        "Heading": 323,
        "VerticalRate": -2112,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:21:20.763Z",
        "Lat": 36.33023,
        "Long": -121.26772,
        "Altitude": 23775,
        "GroundSpeed": 345,
        "Heading": 323,
        "VerticalRate": -2112,
        "Squawk": "3221"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:21:21.184Z",
        "Lat": 36.33073,
        "Long": -121.26772,
        "Altitude": 23750,
        "GroundSpeed": 345,
        "Heading": 323,
        "VerticalRate": -2112,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:21:21.617Z",
        "Lat": 36.33132,
        "Long": -121.26884,
        "Altitude": 23725,
        "GroundSpeed": 345,
        "Heading": 323,
        "VerticalRate": -2176,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:22:27.815Z",
        "Lat": 36.41316,
        "Long": -121.34738,
        "Altitude": 21425,
        "GroundSpeed": 332,
        "Heading": 323,
        "VerticalRate": -2048,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:22:28.345Z",
        "Lat": 36.41382,
        "Long": -121.34857,
        "Altitude": 21400,
        "GroundSpeed": 332,
        "Heading": 323,
        "VerticalRate": -2048,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:22:28.855Z",
        "Lat": 36.41441,
        "Long": -121.34863,
        "Altitude": 21400,
        "GroundSpeed": 332,
        "Heading": 323,
        "VerticalRate": -2048,
        "Squawk": "3221"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:22:30.683Z",
        "Lat": 36.41666,
        "Long": -121.35098,
        "Altitude": 21325,
        "GroundSpeed": 331,
        "Heading": 323,
        "VerticalRate": -2048,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:22:31.163Z",
        "Lat": 36.41725,
        "Long": -121.35098,
        "Altitude": 21325,
        "GroundSpeed": 331,
        "Heading": 323,
        "VerticalRate": -2048,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:22:31.714Z",
        "Lat": 36.41791,
        "Long": -121.35206,
        "Altitude": 21300,
        "GroundSpeed": 330,
        "Heading": 323,
        "VerticalRate": -2048,
        "Squawk": "3221"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:22:32.164Z",
        "Lat": 36.41847,
        "Long": -121.35212,
        "Altitude": 21275,
        "GroundSpeed": 330,
        "Heading": 323,
        "VerticalRate": -2048,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:22:32.574Z",
        "Lat": 36.41895,
        "Long": -121.35332,
        "Altitude": 21275,
        "GroundSpeed": 330,
        "Heading": 323,
        "VerticalRate": -2048,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:22:33.094Z",
        "Lat": 36.41959,
        "Long": -121.35321,
        "Altitude": 21250,
        "GroundSpeed": 330,
        "Heading": 323,
        "VerticalRate": -2048,
        "Squawk": "3221"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:22:33.535Z",
        "Lat": 36.4201,
        "Long": -121.3544,
        "Altitude": 21225,
        "GroundSpeed": 330,
        "Heading": 323,
        "VerticalRate": -2048,
        "Squawk": "3221"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:15.748Z",
        "Lat": 32.93361,
        "Long": -117.49755,
        "Altitude": 36000,
        "GroundSpeed": 332,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:16.199Z",
        "Lat": 32.93408,
        "Long": -117.49743,
        "Altitude": 36000,
        "GroundSpeed": 332,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:16.698Z",
        "Lat": 32.93459,
        "Long": -117.49883,
        "Altitude": 36000,
        "GroundSpeed": 332,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:17.199Z",
        "Lat": 32.93509,
        "Long": -117.49883,
        "Altitude": 36000,
        "GroundSpeed": 332,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:17.789Z",
        "Lat": 32.93566,
        "Long": -117.50024,
        "Altitude": 36000,
        "GroundSpeed": 332,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:30.255Z",
        "Lat": 32.94809,
        "Long": -117.51839,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:31.314Z",
        "Lat": 32.94912,
        "Long": -117.5198,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:32.355Z",
        "Lat": 32.95015,
        "Long": -117.52125,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:34.186Z",
        "Lat": 32.952,
        "Long": -117.52255,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:37:42.089Z",
        "Lat": 32.95982,
        "Long": -117.5337,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:47.907Z",
        "Lat": 33.02514,
        "Long": -117.62551,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:48.337Z",
        "Lat": 33.02559,
        "Long": -117.62693,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:48.798Z",
        "Lat": 33.02605,
        "Long": -117.62688,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:49.377Z",
        "Lat": 33.02662,
        "Long": -117.62826,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:50.728Z",
        "Lat": 33.02797,
        "Long": -117.62968,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:51.21Z",
        "Lat": 33.02843,
        "Long": -117.631,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:52.278Z",
        "Lat": 33.02948,
        "Long": -117.63237,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:52.839Z",
        "Lat": 33.03003,
        "Long": -117.63248,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:53.309Z",
        "Lat": 33.03054,
        "Long": -117.63381,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:53.779Z",
        "Lat": 33.031,
        "Long": -117.63381,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:54.76Z",
        "Lat": 33.03195,
        "Long": -117.63517,
        "Altitude": 36000,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:55.26Z",
        "Lat": 33.03244,
        "Long": -117.63656,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:55.77Z",
        "Lat": 33.03296,
        "Long": -117.63656,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:56.221Z",
        "Lat": 33.03342,
        "Long": -117.63792,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:38:56.751Z",
        "Lat": 33.03392,
        "Long": -117.63809,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:06.205Z",
        "Lat": 33.04329,
        "Long": -117.65051,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:07.315Z",
        "Lat": 33.04441,
        "Long": -117.65332,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:09.886Z",
        "Lat": 33.04697,
        "Long": -117.65601,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:10.807Z",
        "Lat": 33.04788,
        "Long": -117.65742,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:11.317Z",
        "Lat": 33.04836,
        "Long": -117.65887,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:12.308Z",
        "Lat": 33.04935,
        "Long": -117.66022,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:12.748Z",
        "Lat": 33.0498,
        "Long": -117.66028,
        "Altitude": 36000,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:15.798Z",
        "Lat": 33.05283,
        "Long": -117.66436,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:17.26Z",
        "Lat": 33.05428,
        "Long": -117.66711,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:49.343Z",
        "Lat": 33.08593,
        "Long": -117.71161,
        "Altitude": 36000,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": 64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:49.913Z",
        "Lat": 33.08649,
        "Long": -117.71156,
        "Altitude": 36000,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": 64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:53.014Z",
        "Lat": 33.08954,
        "Long": -117.7157,
        "Altitude": 36000,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:53.545Z",
        "Lat": 33.09003,
        "Long": -117.71716,
        "Altitude": 36000,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:55.045Z",
        "Lat": 33.09151,
        "Long": -117.71851,
        "Altitude": 36000,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:39:55.445Z",
        "Lat": 33.09194,
        "Long": -117.71991,
        "Altitude": 36000,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:11.322Z",
        "Lat": 33.10758,
        "Long": -117.74199,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:12.303Z",
        "Lat": 33.10854,
        "Long": -117.74333,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:12.873Z",
        "Lat": 33.10909,
        "Long": -117.74328,
        "Altitude": 36000,
        "GroundSpeed": 329,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:19.085Z",
        "Lat": 33.11522,
        "Long": -117.75163,
        "Altitude": 36000,
        "GroundSpeed": 328,
        "Heading": 311,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:27.089Z",
        "Lat": 33.12309,
        "Long": -117.76267,
        "Altitude": 35925,
        "GroundSpeed": 327,
        "Heading": 311,
        "VerticalRate": -768,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:27.519Z",
        "Lat": 33.1235,
        "Long": -117.76413,
        "Altitude": 35900,
        "GroundSpeed": 327,
        "Heading": 311,
        "VerticalRate": -896,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:28.069Z",
        "Lat": 33.12401,
        "Long": -117.76402,
        "Altitude": 35900,
        "GroundSpeed": 327,
        "Heading": 311,
        "VerticalRate": -896,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:28.599Z",
        "Lat": 33.12456,
        "Long": -117.76547,
        "Altitude": 35900,
        "GroundSpeed": 327,
        "Heading": 311,
        "VerticalRate": -896,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:29.02Z",
        "Lat": 33.12497,
        "Long": -117.76547,
        "Altitude": 35875,
        "GroundSpeed": 327,
        "Heading": 311,
        "VerticalRate": -896,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:29.49Z",
        "Lat": 33.12541,
        "Long": -117.76682,
        "Altitude": 35875,
        "GroundSpeed": 327,
        "Heading": 311,
        "VerticalRate": -896,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:43.266Z",
        "Lat": 33.13891,
        "Long": -117.78622,
        "Altitude": 35550,
        "GroundSpeed": 327,
        "Heading": 311,
        "VerticalRate": -1536,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:43.696Z",
        "Lat": 33.13933,
        "Long": -117.78627,
        "Altitude": 35550,
        "GroundSpeed": 327,
        "Heading": 311,
        "VerticalRate": -1536,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:44.546Z",
        "Lat": 33.14017,
        "Long": -117.78772,
        "Altitude": 35525,
        "GroundSpeed": 327,
        "Heading": 311,
        "VerticalRate": -1536,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:47.147Z",
        "Lat": 33.14273,
        "Long": -117.79047,
        "Altitude": 35450,
        "GroundSpeed": 327,
        "Heading": 311,
        "VerticalRate": -1472,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:48.198Z",
        "Lat": 33.14375,
        "Long": -117.79177,
        "Altitude": 35425,
        "GroundSpeed": 327,
        "Heading": 311,
        "VerticalRate": -1472,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:48.698Z",
        "Lat": 33.14424,
        "Long": -117.79316,
        "Altitude": 35425,
        "GroundSpeed": 327,
        "Heading": 311,
        "VerticalRate": -1472,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:49.229Z",
        "Lat": 33.14477,
        "Long": -117.79451,
        "Altitude": 35400,
        "GroundSpeed": 327,
        "Heading": 311,
        "VerticalRate": -1472,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:49.738Z",
        "Lat": 33.14524,
        "Long": -117.79463,
        "Altitude": 35400,
        "GroundSpeed": 327,
        "Heading": 311,
        "VerticalRate": -1472,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:40:59.613Z",
        "Lat": 33.15492,
        "Long": -117.80847,
        "Altitude": 35175,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": -1536,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:00.123Z",
        "Lat": 33.15543,
        "Long": -117.80853,
        "Altitude": 35150,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": -1536,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:00.684Z",
        "Lat": 33.15601,
        "Long": -117.80986,
        "Altitude": 35150,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": -1600,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:01.245Z",
        "Lat": 33.15655,
        "Long": -117.81127,
        "Altitude": 35125,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": -1600,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:02.184Z",
        "Lat": 33.15748,
        "Long": -117.81127,
        "Altitude": 35100,
        "GroundSpeed": 330,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:02.614Z",
        "Lat": 33.15788,
        "Long": -117.81271,
        "Altitude": 35100,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:03.164Z",
        "Lat": 33.15843,
        "Long": -117.8126,
        "Altitude": 35075,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:04.174Z",
        "Lat": 33.15944,
        "Long": -117.81402,
        "Altitude": 35050,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:04.625Z",
        "Lat": 33.1599,
        "Long": -117.81552,
        "Altitude": 35025,
        "GroundSpeed": 331,
        "Heading": 311,
        "VerticalRate": -1792,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:21.642Z",
        "Lat": 33.17708,
        "Long": -117.83964,
        "Altitude": 34550,
        "GroundSpeed": 339,
        "Heading": 311,
        "VerticalRate": -1792,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:22.223Z",
        "Lat": 33.17766,
        "Long": -117.84095,
        "Altitude": 34550,
        "GroundSpeed": 339,
        "Heading": 311,
        "VerticalRate": -1792,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:22.643Z",
        "Lat": 33.17807,
        "Long": -117.84106,
        "Altitude": 34525,
        "GroundSpeed": 339,
        "Heading": 311,
        "VerticalRate": -1792,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:30.035Z",
        "Lat": 33.18569,
        "Long": -117.85113,
        "Altitude": 34325,
        "GroundSpeed": 343,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:30.555Z",
        "Lat": 33.18626,
        "Long": -117.85259,
        "Altitude": 34300,
        "GroundSpeed": 343,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:31.466Z",
        "Lat": 33.18718,
        "Long": -117.85393,
        "Altitude": 34275,
        "GroundSpeed": 344,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:32.557Z",
        "Lat": 33.18832,
        "Long": -117.85551,
        "Altitude": 34250,
        "GroundSpeed": 344,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:33.007Z",
        "Lat": 33.18878,
        "Long": -117.8554,
        "Altitude": 34250,
        "GroundSpeed": 344,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:33.507Z",
        "Lat": 33.18928,
        "Long": -117.85684,
        "Altitude": 34225,
        "GroundSpeed": 344,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:34.007Z",
        "Lat": 33.18984,
        "Long": -117.85684,
        "Altitude": 34225,
        "GroundSpeed": 344,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:35.108Z",
        "Lat": 33.19098,
        "Long": -117.85831,
        "Altitude": 34175,
        "GroundSpeed": 344,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:36.138Z",
        "Lat": 33.19202,
        "Long": -117.85981,
        "Altitude": 34150,
        "GroundSpeed": 344,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:37.869Z",
        "Lat": 33.19384,
        "Long": -117.86262,
        "Altitude": 34125,
        "GroundSpeed": 344,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:38.349Z",
        "Lat": 33.19432,
        "Long": -117.86418,
        "Altitude": 34100,
        "GroundSpeed": 346,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:38.889Z",
        "Lat": 33.19487,
        "Long": -117.86413,
        "Altitude": 34100,
        "GroundSpeed": 346,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:39.379Z",
        "Lat": 33.19538,
        "Long": -117.86553,
        "Altitude": 34075,
        "GroundSpeed": 346,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:39.879Z",
        "Lat": 33.19589,
        "Long": -117.86553,
        "Altitude": 34050,
        "GroundSpeed": 346,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:40.44Z",
        "Lat": 33.19647,
        "Long": -117.8671,
        "Altitude": 34050,
        "GroundSpeed": 346,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:50.594Z",
        "Lat": 33.20709,
        "Long": -117.88165,
        "Altitude": 33775,
        "GroundSpeed": 348,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:50.994Z",
        "Lat": 33.2075,
        "Long": -117.88165,
        "Altitude": 33750,
        "GroundSpeed": 348,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:51.474Z",
        "Lat": 33.20799,
        "Long": -117.88313,
        "Altitude": 33750,
        "GroundSpeed": 348,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:52.885Z",
        "Lat": 33.20947,
        "Long": -117.88462,
        "Altitude": 33700,
        "GroundSpeed": 349,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:54.326Z",
        "Lat": 33.21098,
        "Long": -117.88759,
        "Altitude": 33675,
        "GroundSpeed": 349,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:55.306Z",
        "Lat": 33.212,
        "Long": -117.88907,
        "Altitude": 33650,
        "GroundSpeed": 349,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:41:55.846Z",
        "Lat": 33.2126,
        "Long": -117.88913,
        "Altitude": 33625,
        "GroundSpeed": 349,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:06.701Z",
        "Lat": 33.22398,
        "Long": -117.90533,
        "Altitude": 33350,
        "GroundSpeed": 351,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:07.232Z",
        "Lat": 33.22452,
        "Long": -117.90679,
        "Altitude": 33325,
        "GroundSpeed": 351,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:07.641Z",
        "Lat": 33.22498,
        "Long": -117.9069,
        "Altitude": 33325,
        "GroundSpeed": 352,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:08.212Z",
        "Lat": 33.22554,
        "Long": -117.90824,
        "Altitude": 33300,
        "GroundSpeed": 352,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:08.741Z",
        "Lat": 33.22614,
        "Long": -117.90835,
        "Altitude": 33300,
        "GroundSpeed": 352,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:09.172Z",
        "Lat": 33.22659,
        "Long": -117.90829,
        "Altitude": 33275,
        "GroundSpeed": 352,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:10.313Z",
        "Lat": 33.22778,
        "Long": -117.91132,
        "Altitude": 33250,
        "GroundSpeed": 352,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:28.4Z",
        "Lat": 33.24687,
        "Long": -117.93829,
        "Altitude": 32750,
        "GroundSpeed": 357,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:28.94Z",
        "Lat": 33.24747,
        "Long": -117.93834,
        "Altitude": 32750,
        "GroundSpeed": 357,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:29.42Z",
        "Lat": 33.24798,
        "Long": -117.93986,
        "Altitude": 32725,
        "GroundSpeed": 357,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:29.96Z",
        "Lat": 33.24854,
        "Long": -117.93975,
        "Altitude": 32725,
        "GroundSpeed": 357,
        "Heading": 311,
        "VerticalRate": -1728,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:31.051Z",
        "Lat": 33.24971,
        "Long": -117.94142,
        "Altitude": 32675,
        "GroundSpeed": 357,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:42:31.461Z",
        "Lat": 33.25012,
        "Long": -117.94277,
        "Altitude": 32675,
        "GroundSpeed": 357,
        "Heading": 311,
        "VerticalRate": -1664,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:23:39.925Z",
        "Lat": 36.49989,
        "Long": -121.43013,
        "Altitude": 19000,
        "GroundSpeed": 324,
        "Heading": 327,
        "VerticalRate": -640,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:23:40.505Z",
        "Lat": 36.50066,
        "Long": -121.43114,
        "Altitude": 19000,
        "GroundSpeed": 324,
        "Heading": 328,
        "VerticalRate": -512,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:23:40.946Z",
        "Lat": 36.50121,
        "Long": -121.4312,
        "Altitude": 19000,
        "GroundSpeed": 324,
        "Heading": 328,
        "VerticalRate": -512,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:23:41.426Z",
        "Lat": 36.5018,
        "Long": -121.43218,
        "Altitude": 19000,
        "GroundSpeed": 324,
        "Heading": 328,
        "VerticalRate": -512,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:23:41.846Z",
        "Lat": 36.50235,
        "Long": -121.43218,
        "Altitude": 19000,
        "GroundSpeed": 324,
        "Heading": 328,
        "VerticalRate": -384,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:23:42.416Z",
        "Lat": 36.50308,
        "Long": -121.43314,
        "Altitude": 19000,
        "GroundSpeed": 324,
        "Heading": 329,
        "VerticalRate": -256,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:23:42.874Z",
        "Lat": 36.50368,
        "Long": -121.4332,
        "Altitude": 19000,
        "GroundSpeed": 324,
        "Heading": 329,
        "VerticalRate": -256,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:23:46.336Z",
        "Lat": 36.50816,
        "Long": -121.43692,
        "Altitude": 19000,
        "GroundSpeed": 323,
        "Heading": 330,
        "VerticalRate": -64,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:23:46.785Z",
        "Lat": 36.50876,
        "Long": -121.43698,
        "Altitude": 19000,
        "GroundSpeed": 324,
        "Heading": 331,
        "VerticalRate": -64,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:23:47.226Z",
        "Lat": 36.50934,
        "Long": -121.43785,
        "Altitude": 19000,
        "GroundSpeed": 324,
        "Heading": 331,
        "VerticalRate": -64,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:23:47.706Z",
        "Lat": 36.50994,
        "Long": -121.43785,
        "Altitude": 19000,
        "GroundSpeed": 324,
        "Heading": 331,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:23:55.292Z",
        "Lat": 36.52,
        "Long": -121.44509,
        "Altitude": 19000,
        "GroundSpeed": 324,
        "Heading": 332,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:23:55.792Z",
        "Lat": 36.52065,
        "Long": -121.44509,
        "Altitude": 19000,
        "GroundSpeed": 324,
        "Heading": 332,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:23:56.362Z",
        "Lat": 36.52139,
        "Long": -121.44596,
        "Altitude": 19000,
        "GroundSpeed": 324,
        "Heading": 332,
        "VerticalRate": 0,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:23:56.86Z",
        "Lat": 36.52208,
        "Long": -121.44596,
        "Altitude": 19000,
        "GroundSpeed": 324,
        "Heading": 332,
        "VerticalRate": 64,
        "Squawk": "3221"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:26.687Z",
        "Lat": 34.19865,
        "Long": -119.31023,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:27.238Z",
        "Lat": 34.19928,
        "Long": -119.31192,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:56:28.335Z",
        "Lat": 34.20053,
        "Long": -119.31365,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:28.878Z",
        "Lat": 34.20113,
        "Long": -119.31371,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:29.808Z",
        "Lat": 34.20216,
        "Long": -119.31524,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:30.309Z",
        "Lat": 34.20273,
        "Long": -119.3169,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:30.819Z",
        "Lat": 34.20332,
        "Long": -119.3169,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:31.399Z",
        "Lat": 34.20398,
        "Long": -119.31856,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:56:31.836Z",
        "Lat": 34.20444,
        "Long": -119.31862,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:56:34.527Z",
        "Lat": 34.20749,
        "Long": -119.32363,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:56:35.537Z",
        "Lat": 34.20863,
        "Long": -119.32531,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:59:07.222Z",
        "Lat": 34.37901,
        "Long": -119.57731,
        "Altitude": 30000,
        "GroundSpeed": 381,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:59:08.162Z",
        "Lat": 34.38008,
        "Long": -119.57743,
        "Altitude": 30000,
        "GroundSpeed": 381,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:59:09.195Z",
        "Lat": 34.38121,
        "Long": -119.57895,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:59:09.606Z",
        "Lat": 34.38167,
        "Long": -119.58063,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:59:10.696Z",
        "Lat": 34.3829,
        "Long": -119.58226,
        "Altitude": 30000,
        "GroundSpeed": 382,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:01:52.654Z",
        "Lat": 34.58647,
        "Long": -119.83029,
        "Altitude": 30000,
        "GroundSpeed": 403,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "7135"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T01:01:53.062Z",
        "Lat": 34.58711,
        "Long": -119.83029,
        "Altitude": 30000,
        "GroundSpeed": 403,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:01:53.545Z",
        "Lat": 34.58789,
        "Long": -119.83149,
        "Altitude": 30000,
        "GroundSpeed": 403,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "7135"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:01:54.015Z",
        "Lat": 34.58859,
        "Long": -119.83149,
        "Altitude": 30000,
        "GroundSpeed": 403,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "7135"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:01:54.445Z",
        "Lat": 34.58926,
        "Long": -119.83276,
        "Altitude": 30000,
        "GroundSpeed": 403,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "7135"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:01:56.036Z",
        "Lat": 34.59176,
        "Long": -119.83395,
        "Altitude": 30000,
        "GroundSpeed": 403,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "7135"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T01:01:56.554Z",
        "Lat": 34.59261,
        "Long": -119.83522,
        "Altitude": 30000,
        "GroundSpeed": 403,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "6537"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:01:57.627Z",
        "Lat": 34.59427,
        "Long": -119.83641,
        "Altitude": 30000,
        "GroundSpeed": 403,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "7135"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:01:58.076Z",
        "Lat": 34.59497,
        "Long": -119.83646,
        "Altitude": 30000,
        "GroundSpeed": 403,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "7135"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:01:58.497Z",
        "Lat": 34.59563,
        "Long": -119.83758,
        "Altitude": 30000,
        "GroundSpeed": 403,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "7135"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:01:59.057Z",
        "Lat": 34.59654,
        "Long": -119.83763,
        "Altitude": 30000,
        "GroundSpeed": 403,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "7135"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:01:59.588Z",
        "Lat": 34.59734,
        "Long": -119.83881,
        "Altitude": 30000,
        "GroundSpeed": 402,
        "Heading": 328,
        "VerticalRate": -64,
        "Squawk": "7135"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:02:00.618Z",
        "Lat": 34.59897,
        "Long": -119.84004,
        "Altitude": 30000,
        "GroundSpeed": 402,
        "Heading": 328,
        "VerticalRate": -64,
        "Squawk": "7135"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:02:01.148Z",
        "Lat": 34.59979,
        "Long": -119.8401,
        "Altitude": 30000,
        "GroundSpeed": 402,
        "Heading": 328,
        "VerticalRate": -64,
        "Squawk": "7135"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:05:58.258Z",
        "Lat": 34.96889,
        "Long": -120.13006,
        "Altitude": 30000,
        "GroundSpeed": 396,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:05:59.428Z",
        "Lat": 34.97069,
        "Long": -120.13132,
        "Altitude": 30000,
        "GroundSpeed": 396,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:05:59.949Z",
        "Lat": 34.97148,
        "Long": -120.13132,
        "Altitude": 30000,
        "GroundSpeed": 396,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:06:00.429Z",
        "Lat": 34.97223,
        "Long": -120.13253,
        "Altitude": 30000,
        "GroundSpeed": 396,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:06:00.859Z",
        "Lat": 34.97287,
        "Long": -120.13253,
        "Altitude": 30000,
        "GroundSpeed": 396,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:06:01.399Z",
        "Lat": 34.97372,
        "Long": -120.13367,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:06:01.879Z",
        "Lat": 34.97446,
        "Long": -120.13378,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:06:02.42Z",
        "Lat": 34.9753,
        "Long": -120.13494,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:06:02.99Z",
        "Lat": 34.97617,
        "Long": -120.13488,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:06:03.51Z",
        "Lat": 34.97698,
        "Long": -120.13613,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:06:04.45Z",
        "Lat": 34.97841,
        "Long": -120.1374,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:06:05.04Z",
        "Lat": 34.97932,
        "Long": -120.1374,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:06:05.591Z",
        "Lat": 34.98019,
        "Long": -120.13853,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:06:06.131Z",
        "Lat": 34.98103,
        "Long": -120.13859,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:06:06.612Z",
        "Lat": 34.98175,
        "Long": -120.13976,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:06:07.072Z",
        "Lat": 34.98248,
        "Long": -120.13987,
        "Altitude": 30000,
        "GroundSpeed": 395,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:08:51.451Z",
        "Lat": 35.23571,
        "Long": -120.34091,
        "Altitude": 30000,
        "GroundSpeed": 396,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:08:51.881Z",
        "Lat": 35.23637,
        "Long": -120.34091,
        "Altitude": 30000,
        "GroundSpeed": 396,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:08:52.431Z",
        "Lat": 35.23718,
        "Long": -120.34212,
        "Altitude": 30000,
        "GroundSpeed": 396,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:08:52.932Z",
        "Lat": 35.23796,
        "Long": -120.34212,
        "Altitude": 30000,
        "GroundSpeed": 396,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:08:53.441Z",
        "Lat": 35.23874,
        "Long": -120.3433,
        "Altitude": 30000,
        "GroundSpeed": 396,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:08:53.892Z",
        "Lat": 35.23944,
        "Long": -120.34324,
        "Altitude": 30000,
        "GroundSpeed": 396,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:08:54.422Z",
        "Lat": 35.24025,
        "Long": -120.34452,
        "Altitude": 30000,
        "GroundSpeed": 396,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:08:54.942Z",
        "Lat": 35.24107,
        "Long": -120.34452,
        "Altitude": 30000,
        "GroundSpeed": 396,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:08:55.473Z",
        "Lat": 35.24191,
        "Long": -120.34582,
        "Altitude": 30000,
        "GroundSpeed": 396,
        "Heading": 328,
        "VerticalRate": 0,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:08:56.463Z",
        "Lat": 35.24341,
        "Long": -120.34704,
        "Altitude": 30000,
        "GroundSpeed": 396,
        "Heading": 328,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:08:56.963Z",
        "Lat": 35.24419,
        "Long": -120.34704,
        "Altitude": 30000,
        "GroundSpeed": 396,
        "Heading": 328,
        "VerticalRate": -64,
        "Squawk": "3201"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:08:57.534Z",
        "Lat": 35.24507,
        "Long": -120.34821,
        "Altitude": 30000,
        "GroundSpeed": 396,
        "Heading": 328,
        "VerticalRate": -64,
        "Squawk": "3201"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:15.837Z",
        "Lat": 34.11855,
        "Long": -119.1925,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 311,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:16.238Z",
        "Lat": 34.119,
        "Long": -119.19409,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:17.408Z",
        "Lat": 34.12032,
        "Long": -119.19577,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:18.409Z",
        "Lat": 34.12143,
        "Long": -119.1974,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:20.399Z",
        "Lat": 34.12372,
        "Long": -119.20071,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:20.97Z",
        "Lat": 34.12436,
        "Long": -119.20082,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:21.49Z",
        "Lat": 34.12493,
        "Long": -119.20235,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:21.9Z",
        "Lat": 34.1254,
        "Long": -119.2024,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:22.43Z",
        "Lat": 34.12601,
        "Long": -119.20407,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:23.02Z",
        "Lat": 34.12669,
        "Long": -119.20407,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:55:23.557Z",
        "Lat": 34.12731,
        "Long": -119.20584,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:55:24.097Z",
        "Lat": 34.12791,
        "Long": -119.20572,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "3013"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:25.051Z",
        "Lat": 34.12898,
        "Long": -119.20743,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:25.471Z",
        "Lat": 34.12945,
        "Long": -119.20916,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:26.362Z",
        "Lat": 34.13045,
        "Long": -119.2108,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:27.442Z",
        "Lat": 34.13168,
        "Long": -119.21242,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:34.375Z",
        "Lat": 34.13951,
        "Long": -119.22408,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:35.406Z",
        "Lat": 34.14071,
        "Long": -119.22564,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:36.446Z",
        "Lat": 34.14189,
        "Long": -119.22733,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:38.547Z",
        "Lat": 34.14427,
        "Long": -119.23064,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:39.507Z",
        "Lat": 34.14532,
        "Long": -119.23227,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:42.108Z",
        "Lat": 34.1483,
        "Long": -119.23559,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:42.538Z",
        "Lat": 34.14876,
        "Long": -119.23725,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:42.959Z",
        "Lat": 34.14926,
        "Long": -119.23731,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:55:43.376Z",
        "Lat": 34.14974,
        "Long": -119.23885,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:43.789Z",
        "Lat": 34.15016,
        "Long": -119.23885,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:44.379Z",
        "Lat": 34.15086,
        "Long": -119.2405,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:50.672Z",
        "Lat": 34.15796,
        "Long": -119.25059,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:51.253Z",
        "Lat": 34.15864,
        "Long": -119.25219,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:52.173Z",
        "Lat": 34.15966,
        "Long": -119.2523,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:52.593Z",
        "Lat": 34.16016,
        "Long": -119.25379,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:53.623Z",
        "Lat": 34.16129,
        "Long": -119.2555,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:54.093Z",
        "Lat": 34.16185,
        "Long": -119.25545,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:54.573Z",
        "Lat": 34.1624,
        "Long": -119.25726,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:55.154Z",
        "Lat": 34.16304,
        "Long": -119.25721,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:55.585Z",
        "Lat": 34.16352,
        "Long": -119.25877,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:56.045Z",
        "Lat": 34.16404,
        "Long": -119.25888,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:56.625Z",
        "Lat": 34.16469,
        "Long": -119.26052,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:55:57.975Z",
        "Lat": 34.16622,
        "Long": -119.26226,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:00.086Z",
        "Lat": 34.1686,
        "Long": -119.2654,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:03.017Z",
        "Lat": 34.17192,
        "Long": -119.27044,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:03.997Z",
        "Lat": 34.17302,
        "Long": -119.27204,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:05.048Z",
        "Lat": 34.17421,
        "Long": -119.2738,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:06.038Z",
        "Lat": 34.17535,
        "Long": -119.27547,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:06.538Z",
        "Lat": 34.1759,
        "Long": -119.27711,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:07.109Z",
        "Lat": 34.17654,
        "Long": -119.27705,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:07.61Z",
        "Lat": 34.17712,
        "Long": -119.27874,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:08.07Z",
        "Lat": 34.17763,
        "Long": -119.27885,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:08.61Z",
        "Lat": 34.17824,
        "Long": -119.28036,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:56:09.177Z",
        "Lat": 34.17888,
        "Long": -119.28036,
        "Altitude": 30000,
        "GroundSpeed": 385,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "3013"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:10.12Z",
        "Lat": 34.17996,
        "Long": -119.28205,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:10.971Z",
        "Lat": 34.18089,
        "Long": -119.28383,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:11.531Z",
        "Lat": 34.18154,
        "Long": -119.28537,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:12.101Z",
        "Lat": 34.18219,
        "Long": -119.28537,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:12.511Z",
        "Lat": 34.18263,
        "Long": -119.28703,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:13.452Z",
        "Lat": 34.18373,
        "Long": -119.28875,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:14.002Z",
        "Lat": 34.18433,
        "Long": -119.28869,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:14.562Z",
        "Lat": 34.18497,
        "Long": -119.29045,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:15.023Z",
        "Lat": 34.18547,
        "Long": -119.29039,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:16.173Z",
        "Lat": 34.1868,
        "Long": -119.29213,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:16.633Z",
        "Lat": 34.1873,
        "Long": -119.2937,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:17.133Z",
        "Lat": 34.18785,
        "Long": -119.2937,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:18.053Z",
        "Lat": 34.18889,
        "Long": -119.29539,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:19.994Z",
        "Lat": 34.19108,
        "Long": -119.29865,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:20.485Z",
        "Lat": 34.19165,
        "Long": -119.30026,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:21.485Z",
        "Lat": 34.19276,
        "Long": -119.30191,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:21.945Z",
        "Lat": 34.19332,
        "Long": -119.30202,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:22.495Z",
        "Lat": 34.19394,
        "Long": -119.30362,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:23.046Z",
        "Lat": 34.19453,
        "Long": -119.30368,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:23.916Z",
        "Lat": 34.1955,
        "Long": -119.30529,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:24.506Z",
        "Lat": 34.19618,
        "Long": -119.30693,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:25.096Z",
        "Lat": 34.19687,
        "Long": -119.30693,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:25.617Z",
        "Lat": 34.19746,
        "Long": -119.30861,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:37.191Z",
        "Lat": 34.21051,
        "Long": -119.32683,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:37.732Z",
        "Lat": 34.2111,
        "Long": -119.32858,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:39.813Z",
        "Lat": 34.21343,
        "Long": -119.33184,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:40.823Z",
        "Lat": 34.21458,
        "Long": -119.33355,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:41.364Z",
        "Lat": 34.2152,
        "Long": -119.33527,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:41.904Z",
        "Lat": 34.2158,
        "Long": -119.33516,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:42.974Z",
        "Lat": 34.21701,
        "Long": -119.33697,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:43.465Z",
        "Lat": 34.21757,
        "Long": -119.33859,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:44.434Z",
        "Lat": 34.21866,
        "Long": -119.34022,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:45.905Z",
        "Lat": 34.22032,
        "Long": -119.34179,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:46.966Z",
        "Lat": 34.2215,
        "Long": -119.34359,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:47.496Z",
        "Lat": 34.22209,
        "Long": -119.34511,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:48.006Z",
        "Lat": 34.22269,
        "Long": -119.34511,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:49.027Z",
        "Lat": 34.22383,
        "Long": -119.34684,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:56:50.868Z",
        "Lat": 34.22589,
        "Long": -119.3502,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:56:51.715Z",
        "Lat": 34.22688,
        "Long": -119.35181,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:56:52.675Z",
        "Lat": 34.22795,
        "Long": -119.35339,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": -64,
        "Squawk": "6537"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:56:53.155Z",
        "Lat": 34.2285,
        "Long": -119.35351,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": -64,
        "Squawk": "6537"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:56:53.616Z",
        "Lat": 34.22902,
        "Long": -119.35513,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": -64,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:56:54.136Z",
        "Lat": 34.22963,
        "Long": -119.35518,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": -64,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:56:54.676Z",
        "Lat": 34.23019,
        "Long": -119.3567,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": -64,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:56:56.067Z",
        "Lat": 34.23177,
        "Long": -119.3585,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": -64,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:56:56.576Z",
        "Lat": 34.23235,
        "Long": -119.36007,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": -64,
        "Squawk": "6537"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:56:57.146Z",
        "Lat": 34.23299,
        "Long": -119.36018,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": -64,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:56:57.727Z",
        "Lat": 34.23368,
        "Long": -119.36176,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:56:59.318Z",
        "Lat": 34.23545,
        "Long": -119.36508,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:56:59.758Z",
        "Lat": 34.23596,
        "Long": -119.36514,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:00.249Z",
        "Lat": 34.23651,
        "Long": -119.36674,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:00.778Z",
        "Lat": 34.23711,
        "Long": -119.36668,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:01.189Z",
        "Lat": 34.23756,
        "Long": -119.36668,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:01.739Z",
        "Lat": 34.23819,
        "Long": -119.36846,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:03.22Z",
        "Lat": 34.23982,
        "Long": -119.37166,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:57:03.683Z",
        "Lat": 34.24033,
        "Long": -119.37166,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:57:04.113Z",
        "Lat": 34.24085,
        "Long": -119.37166,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:57:04.583Z",
        "Lat": 34.24136,
        "Long": -119.37329,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:57:05.544Z",
        "Lat": 34.24243,
        "Long": -119.37504,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:06.06Z",
        "Lat": 34.24303,
        "Long": -119.3751,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:57:06.484Z",
        "Lat": 34.24352,
        "Long": -119.3766,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:57:07.054Z",
        "Lat": 34.24416,
        "Long": -119.37671,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:57:07.585Z",
        "Lat": 34.24476,
        "Long": -119.3783,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:08.561Z",
        "Lat": 34.24585,
        "Long": -119.38008,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:09.722Z",
        "Lat": 34.24718,
        "Long": -119.38168,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:10.182Z",
        "Lat": 34.24769,
        "Long": -119.38162,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:10.742Z",
        "Lat": 34.24832,
        "Long": -119.38338,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:11.243Z",
        "Lat": 34.24885,
        "Long": -119.38494,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:11.743Z",
        "Lat": 34.24941,
        "Long": -119.38505,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:12.293Z",
        "Lat": 34.25006,
        "Long": -119.38663,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:12.703Z",
        "Lat": 34.25052,
        "Long": -119.38663,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:13.654Z",
        "Lat": 34.2516,
        "Long": -119.38837,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:14.194Z",
        "Lat": 34.25221,
        "Long": -119.38826,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:57:14.688Z",
        "Lat": 34.25272,
        "Long": -119.38989,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:57:15.258Z",
        "Lat": 34.25337,
        "Long": -119.39157,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:15.655Z",
        "Lat": 34.25383,
        "Long": -119.39169,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:57:16.108Z",
        "Lat": 34.25435,
        "Long": -119.39157,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:16.525Z",
        "Lat": 34.25482,
        "Long": -119.3933,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:57:17.018Z",
        "Lat": 34.25537,
        "Long": -119.39325,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:17.515Z",
        "Lat": 34.25593,
        "Long": -119.39495,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:57:18.069Z",
        "Lat": 34.25653,
        "Long": -119.39501,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:18.556Z",
        "Lat": 34.25711,
        "Long": -119.39667,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:19.536Z",
        "Lat": 34.25821,
        "Long": -119.39827,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:57:20.12Z",
        "Lat": 34.25882,
        "Long": -119.39821,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:20.617Z",
        "Lat": 34.2594,
        "Long": -119.39992,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:57:21.131Z",
        "Lat": 34.25995,
        "Long": -119.39992,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:57:21.68Z",
        "Lat": 34.26058,
        "Long": -119.40147,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:57:22.261Z",
        "Lat": 34.26123,
        "Long": -119.40323,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:22.787Z",
        "Lat": 34.26183,
        "Long": -119.40317,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:57:23.331Z",
        "Lat": 34.26245,
        "Long": -119.40491,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:57:23.731Z",
        "Lat": 34.26291,
        "Long": -119.40491,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:57:24.641Z",
        "Lat": 34.26393,
        "Long": -119.40659,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:57:25.236Z",
        "Lat": 34.26459,
        "Long": -119.40811,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:57:25.812Z",
        "Lat": 34.26524,
        "Long": -119.40817,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:57:26.332Z",
        "Lat": 34.26581,
        "Long": -119.40984,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:26.799Z",
        "Lat": 34.26636,
        "Long": -119.40984,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:27.77Z",
        "Lat": 34.26743,
        "Long": -119.41143,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:57:28.804Z",
        "Lat": 34.2686,
        "Long": -119.41315,
        "Altitude": 30000,
        "GroundSpeed": 384,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:57:29.374Z",
        "Lat": 34.26924,
        "Long": -119.41492,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:57:29.894Z",
        "Lat": 34.2698,
        "Long": -119.41475,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:30.821Z",
        "Lat": 34.27089,
        "Long": -119.41651,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:57:31.425Z",
        "Lat": 34.27152,
        "Long": -119.41813,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:31.962Z",
        "Lat": 34.27218,
        "Long": -119.41824,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:57:32.995Z",
        "Lat": 34.27332,
        "Long": -119.41976,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:33.442Z",
        "Lat": 34.27381,
        "Long": -119.4215,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:57:33.875Z",
        "Lat": 34.27432,
        "Long": -119.42156,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:35.833Z",
        "Lat": 34.27651,
        "Long": -119.42482,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:36.723Z",
        "Lat": 34.27753,
        "Long": -119.42643,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:37.194Z",
        "Lat": 34.27803,
        "Long": -119.42643,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:37.624Z",
        "Lat": 34.27851,
        "Long": -119.42814,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:38.134Z",
        "Lat": 34.27911,
        "Long": -119.42814,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:38.695Z",
        "Lat": 34.27972,
        "Long": -119.42974,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:57:39.758Z",
        "Lat": 34.28093,
        "Long": -119.43146,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:40.805Z",
        "Lat": 34.2821,
        "Long": -119.43305,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:41.257Z",
        "Lat": 34.2826,
        "Long": -119.43472,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:41.676Z",
        "Lat": 34.28307,
        "Long": -119.43466,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:42.766Z",
        "Lat": 34.2843,
        "Long": -119.43646,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:43.326Z",
        "Lat": 34.28493,
        "Long": -119.43804,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:43.797Z",
        "Lat": 34.28549,
        "Long": -119.43804,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:44.327Z",
        "Lat": 34.28604,
        "Long": -119.43972,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:44.837Z",
        "Lat": 34.28664,
        "Long": -119.43972,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:45.767Z",
        "Lat": 34.28768,
        "Long": -119.44147,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:46.358Z",
        "Lat": 34.28833,
        "Long": -119.44308,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:57:46.951Z",
        "Lat": 34.28902,
        "Long": -119.44308,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:47.418Z",
        "Lat": 34.28954,
        "Long": -119.44468,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T00:57:48.742Z",
        "Lat": 34.29099,
        "Long": -119.44639,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "CulverCity",
        "TimestampUTC": "2017-01-03T00:57:49.159Z",
        "Lat": 34.29149,
        "Long": -119.44639,
        "Altitude": 30000,
        "GroundSpeed": 383,
        "Heading": 310,
        "VerticalRate": 0,
        "Squawk": "6537"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:34:48.795Z",
        "Lat": 37.287,
        "Long": -121.96975,
        "Altitude": 4275,
        "GroundSpeed": 259,
        "Heading": 332,
        "VerticalRate": -1344,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:34:51.746Z",
        "Lat": 37.28997,
        "Long": -121.97201,
        "Altitude": 4250,
        "GroundSpeed": 252,
        "Heading": 331,
        "VerticalRate": -512,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:34:52.327Z",
        "Lat": 37.29053,
        "Long": -121.97279,
        "Altitude": 4250,
        "GroundSpeed": 252,
        "Heading": 331,
        "VerticalRate": -512,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:34:53.207Z",
        "Lat": 37.2914,
        "Long": -121.97273,
        "Altitude": 4250,
        "GroundSpeed": 251,
        "Heading": 330,
        "VerticalRate": -448,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:34:55.768Z",
        "Lat": 37.29388,
        "Long": -121.97517,
        "Altitude": 4225,
        "GroundSpeed": 249,
        "Heading": 328,
        "VerticalRate": -512,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:16.136Z",
        "Lat": 37.31143,
        "Long": -121.99314,
        "Altitude": 4025,
        "GroundSpeed": 238,
        "Heading": 320,
        "VerticalRate": -512,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:16.687Z",
        "Lat": 37.3119,
        "Long": -121.99412,
        "Altitude": 4025,
        "GroundSpeed": 238,
        "Heading": 320,
        "VerticalRate": -512,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:17.167Z",
        "Lat": 37.31232,
        "Long": -121.99412,
        "Altitude": 4025,
        "GroundSpeed": 238,
        "Heading": 320,
        "VerticalRate": -512,
        "Squawk": ""
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:17.729Z",
        "Lat": 37.31278,
        "Long": -121.99493,
        "Altitude": 4025,
        "GroundSpeed": 238,
        "Heading": 320,
        "VerticalRate": -448,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:18.228Z",
        "Lat": 37.31319,
        "Long": -121.99587,
        "Altitude": 4000,
        "GroundSpeed": 238,
        "Heading": 320,
        "VerticalRate": -448,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:18.757Z",
        "Lat": 37.31364,
        "Long": -121.99593,
        "Altitude": 4000,
        "GroundSpeed": 237,
        "Heading": 320,
        "VerticalRate": -384,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:19.268Z",
        "Lat": 37.31404,
        "Long": -121.99679,
        "Altitude": 4000,
        "GroundSpeed": 237,
        "Heading": 320,
        "VerticalRate": -384,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:20.288Z",
        "Lat": 37.31493,
        "Long": -121.99762,
        "Altitude": 4000,
        "GroundSpeed": 237,
        "Heading": 320,
        "VerticalRate": -320,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:20.698Z",
        "Lat": 37.31525,
        "Long": -121.99762,
        "Altitude": 4000,
        "GroundSpeed": 236,
        "Heading": 320,
        "VerticalRate": -256,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:21.098Z",
        "Lat": 37.31557,
        "Long": -121.99762,
        "Altitude": 4000,
        "GroundSpeed": 236,
        "Heading": 320,
        "VerticalRate": -256,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:22.129Z",
        "Lat": 37.31641,
        "Long": -121.99858,
        "Altitude": 4000,
        "GroundSpeed": 236,
        "Heading": 320,
        "VerticalRate": -192,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:22.579Z",
        "Lat": 37.3168,
        "Long": -121.99949,
        "Altitude": 4000,
        "GroundSpeed": 236,
        "Heading": 320,
        "VerticalRate": -192,
        "Squawk": ""
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:22.99Z",
        "Lat": 37.31712,
        "Long": -121.99943,
        "Altitude": 4000,
        "GroundSpeed": 236,
        "Heading": 320,
        "VerticalRate": -128,
        "Squawk": ""
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:23.51Z",
        "Lat": 37.31758,
        "Long": -122.00037,
        "Altitude": 4000,
        "GroundSpeed": 236,
        "Heading": 320,
        "VerticalRate": -128,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:23.95Z",
        "Lat": 37.31795,
        "Long": -122.00037,
        "Altitude": 4000,
        "GroundSpeed": 235,
        "Heading": 320,
        "VerticalRate": -128,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:24.43Z",
        "Lat": 37.31831,
        "Long": -122.00125,
        "Altitude": 4000,
        "GroundSpeed": 235,
        "Heading": 320,
        "VerticalRate": -128,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:24.89Z",
        "Lat": 37.31868,
        "Long": -122.00125,
        "Altitude": 4000,
        "GroundSpeed": 233,
        "Heading": 320,
        "VerticalRate": -64,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:25.95Z",
        "Lat": 37.31958,
        "Long": -122.00216,
        "Altitude": 3975,
        "GroundSpeed": 233,
        "Heading": 320,
        "VerticalRate": -64,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:26.401Z",
        "Lat": 37.31992,
        "Long": -122.003,
        "Altitude": 3975,
        "GroundSpeed": 233,
        "Heading": 320,
        "VerticalRate": 0,
        "Squawk": ""
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:44.929Z",
        "Lat": 37.33461,
        "Long": -122.01878,
        "Altitude": 3700,
        "GroundSpeed": 224,
        "Heading": 320,
        "VerticalRate": -1216,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:45.469Z",
        "Lat": 37.33503,
        "Long": -122.01965,
        "Altitude": 3700,
        "GroundSpeed": 224,
        "Heading": 320,
        "VerticalRate": -1216,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:45.949Z",
        "Lat": 37.33541,
        "Long": -122.01965,
        "Altitude": 3675,
        "GroundSpeed": 224,
        "Heading": 320,
        "VerticalRate": -1216,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:46.93Z",
        "Lat": 37.33617,
        "Long": -122.02053,
        "Altitude": 3650,
        "GroundSpeed": 224,
        "Heading": 320,
        "VerticalRate": -1216,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:47.37Z",
        "Lat": 37.33652,
        "Long": -122.02138,
        "Altitude": 3650,
        "GroundSpeed": 223,
        "Heading": 320,
        "VerticalRate": -1216,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:47.799Z",
        "Lat": 37.33685,
        "Long": -122.02133,
        "Altitude": 3650,
        "GroundSpeed": 223,
        "Heading": 320,
        "VerticalRate": -1216,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:48.28Z",
        "Lat": 37.33722,
        "Long": -122.02222,
        "Altitude": 3625,
        "GroundSpeed": 223,
        "Heading": 320,
        "VerticalRate": -1216,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:48.79Z",
        "Lat": 37.33763,
        "Long": -122.02217,
        "Altitude": 3625,
        "GroundSpeed": 223,
        "Heading": 320,
        "VerticalRate": -1216,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:49.331Z",
        "Lat": 37.33806,
        "Long": -122.02306,
        "Altitude": 3625,
        "GroundSpeed": 223,
        "Heading": 320,
        "VerticalRate": -1216,
        "Squawk": ""
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:49.831Z",
        "Lat": 37.33843,
        "Long": -122.02306,
        "Altitude": 3600,
        "GroundSpeed": 223,
        "Heading": 320,
        "VerticalRate": -1152,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:50.341Z",
        "Lat": 37.33882,
        "Long": -122.02392,
        "Altitude": 3600,
        "GroundSpeed": 223,
        "Heading": 320,
        "VerticalRate": -1152,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:50.811Z",
        "Lat": 37.33919,
        "Long": -122.02392,
        "Altitude": 3575,
        "GroundSpeed": 223,
        "Heading": 320,
        "VerticalRate": -1152,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:51.381Z",
        "Lat": 37.33964,
        "Long": -122.02485,
        "Altitude": 3575,
        "GroundSpeed": 223,
        "Heading": 320,
        "VerticalRate": -1152,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:51.892Z",
        "Lat": 37.34006,
        "Long": -122.02479,
        "Altitude": 3575,
        "GroundSpeed": 223,
        "Heading": 320,
        "VerticalRate": -1088,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:35:52.392Z",
        "Lat": 37.34042,
        "Long": -122.02561,
        "Altitude": 3550,
        "GroundSpeed": 223,
        "Heading": 320,
        "VerticalRate": -1088,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:36:14.041Z",
        "Lat": 37.35715,
        "Long": -122.0436,
        "Altitude": 3125,
        "GroundSpeed": 219,
        "Heading": 319,
        "VerticalRate": -1088,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:36:14.601Z",
        "Lat": 37.35759,
        "Long": -122.04437,
        "Altitude": 3125,
        "GroundSpeed": 219,
        "Heading": 319,
        "VerticalRate": -1088,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:36:15.071Z",
        "Lat": 37.35791,
        "Long": -122.04443,
        "Altitude": 3125,
        "GroundSpeed": 219,
        "Heading": 319,
        "VerticalRate": -1088,
        "Squawk": ""
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:36:15.601Z",
        "Lat": 37.35831,
        "Long": -122.04527,
        "Altitude": 3100,
        "GroundSpeed": 219,
        "Heading": 319,
        "VerticalRate": -1088,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:36:16.112Z",
        "Lat": 37.35873,
        "Long": -122.04527,
        "Altitude": 3100,
        "GroundSpeed": 219,
        "Heading": 319,
        "VerticalRate": -1088,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:36:16.562Z",
        "Lat": 37.35905,
        "Long": -122.04613,
        "Altitude": 3075,
        "GroundSpeed": 219,
        "Heading": 319,
        "VerticalRate": -1088,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:36:17.042Z",
        "Lat": 37.35942,
        "Long": -122.04613,
        "Altitude": 3075,
        "GroundSpeed": 219,
        "Heading": 319,
        "VerticalRate": -1088,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:36:17.632Z",
        "Lat": 37.35989,
        "Long": -122.047,
        "Altitude": 3050,
        "GroundSpeed": 219,
        "Heading": 319,
        "VerticalRate": -1152,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:36:18.213Z",
        "Lat": 37.36031,
        "Long": -122.04778,
        "Altitude": 3050,
        "GroundSpeed": 219,
        "Heading": 319,
        "VerticalRate": -1152,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:36:18.693Z",
        "Lat": 37.36066,
        "Long": -122.04776,
        "Altitude": 3050,
        "GroundSpeed": 218,
        "Heading": 319,
        "VerticalRate": -1216,
        "Squawk": ""
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:36:19.173Z",
        "Lat": 37.36102,
        "Long": -122.04788,
        "Altitude": 3025,
        "GroundSpeed": 218,
        "Heading": 319,
        "VerticalRate": -1216,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:36:19.584Z",
        "Lat": 37.36134,
        "Long": -122.04873,
        "Altitude": 3025,
        "GroundSpeed": 218,
        "Heading": 319,
        "VerticalRate": -1216,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:36:20.004Z",
        "Lat": 37.36166,
        "Long": -122.04867,
        "Altitude": 3025,
        "GroundSpeed": 218,
        "Heading": 319,
        "VerticalRate": -1152,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:36:20.433Z",
        "Lat": 37.36198,
        "Long": -122.04952,
        "Altitude": 3000,
        "GroundSpeed": 218,
        "Heading": 319,
        "VerticalRate": -1152,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:36:20.834Z",
        "Lat": 37.3623,
        "Long": -122.04952,
        "Altitude": 3000,
        "GroundSpeed": 218,
        "Heading": 319,
        "VerticalRate": -1152,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:36:21.304Z",
        "Lat": 37.36264,
        "Long": -122.05034,
        "Altitude": 3000,
        "GroundSpeed": 218,
        "Heading": 319,
        "VerticalRate": -1152,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:36:35.71Z",
        "Lat": 37.37507,
        "Long": -122.05954,
        "Altitude": 2725,
        "GroundSpeed": 218,
        "Heading": 340,
        "VerticalRate": -1152,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:36:36.291Z",
        "Lat": 37.37567,
        "Long": -122.05992,
        "Altitude": 2725,
        "GroundSpeed": 218,
        "Heading": 340,
        "VerticalRate": -1152,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:36:36.75Z",
        "Lat": 37.37608,
        "Long": -122.05998,
        "Altitude": 2725,
        "GroundSpeed": 218,
        "Heading": 342,
        "VerticalRate": -1152,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:36:37.201Z",
        "Lat": 37.37654,
        "Long": -122.05992,
        "Altitude": 2700,
        "GroundSpeed": 218,
        "Heading": 342,
        "VerticalRate": -1152,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:36:37.641Z",
        "Lat": 37.37698,
        "Long": -122.06031,
        "Altitude": 2700,
        "GroundSpeed": 218,
        "Heading": 342,
        "VerticalRate": -1152,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:36:38.121Z",
        "Lat": 37.37744,
        "Long": -122.06031,
        "Altitude": 2700,
        "GroundSpeed": 218,
        "Heading": 344,
        "VerticalRate": -1152,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:36:40.224Z",
        "Lat": 37.37956,
        "Long": -122.06114,
        "Altitude": 2650,
        "GroundSpeed": 218,
        "Heading": 347,
        "VerticalRate": -1024,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:36:40.693Z",
        "Lat": 37.38002,
        "Long": -122.06114,
        "Altitude": 2650,
        "GroundSpeed": 219,
        "Heading": 349,
        "VerticalRate": -1024,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:36:41.122Z",
        "Lat": 37.38043,
        "Long": -122.06114,
        "Altitude": 2625,
        "GroundSpeed": 219,
        "Heading": 349,
        "VerticalRate": -1024,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:37:00.15Z",
        "Lat": 37.39942,
        "Long": -122.05936,
        "Altitude": 2300,
        "GroundSpeed": 216,
        "Heading": 17,
        "VerticalRate": -832,
        "Squawk": ""
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:37:00.731Z",
        "Lat": 37.39998,
        "Long": -122.05892,
        "Altitude": 2300,
        "GroundSpeed": 216,
        "Heading": 17,
        "VerticalRate": -832,
        "Squawk": ""
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:37:01.15Z",
        "Lat": 37.40039,
        "Long": -122.05892,
        "Altitude": 2275,
        "GroundSpeed": 215,
        "Heading": 18,
        "VerticalRate": -832,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:37:01.611Z",
        "Lat": 37.40077,
        "Long": -122.05846,
        "Altitude": 2275,
        "GroundSpeed": 215,
        "Heading": 20,
        "VerticalRate": -832,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:37:03.152Z",
        "Lat": 37.40222,
        "Long": -122.05799,
        "Altitude": 2250,
        "GroundSpeed": 215,
        "Heading": 21,
        "VerticalRate": -896,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:37:03.582Z",
        "Lat": 37.40258,
        "Long": -122.05745,
        "Altitude": 2250,
        "GroundSpeed": 215,
        "Heading": 21,
        "VerticalRate": -896,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:37:04.172Z",
        "Lat": 37.40314,
        "Long": -122.05745,
        "Altitude": 2225,
        "GroundSpeed": 215,
        "Heading": 23,
        "VerticalRate": -896,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:37:05.282Z",
        "Lat": 37.40407,
        "Long": -122.05631,
        "Altitude": 2200,
        "GroundSpeed": 215,
        "Heading": 23,
        "VerticalRate": -896,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:37:06.353Z",
        "Lat": 37.40501,
        "Long": -122.05571,
        "Altitude": 2200,
        "GroundSpeed": 215,
        "Heading": 23,
        "VerticalRate": -896,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:37:07.433Z",
        "Lat": 37.40593,
        "Long": -122.05512,
        "Altitude": 2175,
        "GroundSpeed": 215,
        "Heading": 23,
        "VerticalRate": -896,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:37:08.544Z",
        "Lat": 37.40689,
        "Long": -122.05442,
        "Altitude": 2150,
        "GroundSpeed": 214,
        "Heading": 29,
        "VerticalRate": -832,
        "Squawk": ""
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:36:38.631Z",
        "Lat": 37.37796,
        "Long": -122.06062,
        "Altitude": 2675,
        "GroundSpeed": 218,
        "Heading": 345,
        "VerticalRate": -1088,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:36:39.192Z",
        "Lat": 37.37851,
        "Long": -122.06062,
        "Altitude": 2675,
        "GroundSpeed": 218,
        "Heading": 345,
        "VerticalRate": -1088,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:36:39.752Z",
        "Lat": 37.37907,
        "Long": -122.06091,
        "Altitude": 2650,
        "GroundSpeed": 218,
        "Heading": 347,
        "VerticalRate": -1024,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:36:41.672Z",
        "Lat": 37.38098,
        "Long": -122.06133,
        "Altitude": 2625,
        "GroundSpeed": 219,
        "Heading": 349,
        "VerticalRate": -1024,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:36:42.173Z",
        "Lat": 37.38149,
        "Long": -122.06133,
        "Altitude": 2625,
        "GroundSpeed": 218,
        "Heading": 350,
        "VerticalRate": -1024,
        "Squawk": ""
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "Saratoga",
        "TimestampUTC": "2017-01-03T01:36:42.684Z",
        "Lat": 37.38203,
        "Long": -122.06155,
        "Altitude": 2600,
        "GroundSpeed": 218,
        "Heading": 350,
        "VerticalRate": -1024,
        "Squawk": ""
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:27:46.827Z",
        "Lat": 36.81729,
        "Long": -121.64732,
        "Altitude": 13825,
        "GroundSpeed": 310,
        "Heading": 332,
        "VerticalRate": -1920,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:27:47.227Z",
        "Lat": 36.81779,
        "Long": -121.64816,
        "Altitude": 13800,
        "GroundSpeed": 310,
        "Heading": 332,
        "VerticalRate": -1920,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:27:47.698Z",
        "Lat": 36.81835,
        "Long": -121.64816,
        "Altitude": 13800,
        "GroundSpeed": 310,
        "Heading": 332,
        "VerticalRate": -1920,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:27:48.17Z",
        "Lat": 36.81896,
        "Long": -121.64822,
        "Altitude": 13775,
        "GroundSpeed": 310,
        "Heading": 332,
        "VerticalRate": -1920,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:27:48.601Z",
        "Lat": 36.81949,
        "Long": -121.64904,
        "Altitude": 13775,
        "GroundSpeed": 310,
        "Heading": 332,
        "VerticalRate": -1920,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:27:49.018Z",
        "Lat": 36.82004,
        "Long": -121.64904,
        "Altitude": 13750,
        "GroundSpeed": 310,
        "Heading": 332,
        "VerticalRate": -1920,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:27:49.45Z",
        "Lat": 36.82059,
        "Long": -121.64992,
        "Altitude": 13750,
        "GroundSpeed": 310,
        "Heading": 332,
        "VerticalRate": -1920,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:27:49.848Z",
        "Lat": 36.8211,
        "Long": -121.64992,
        "Altitude": 13725,
        "GroundSpeed": 310,
        "Heading": 332,
        "VerticalRate": -1920,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:27:50.438Z",
        "Lat": 36.82182,
        "Long": -121.65081,
        "Altitude": 13700,
        "GroundSpeed": 310,
        "Heading": 332,
        "VerticalRate": -1920,
        "Squawk": "3221"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:27:51.019Z",
        "Lat": 36.82256,
        "Long": -121.65075,
        "Altitude": 13700,
        "GroundSpeed": 310,
        "Heading": 332,
        "VerticalRate": -1920,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:27:51.599Z",
        "Lat": 36.82329,
        "Long": -121.65161,
        "Altitude": 13675,
        "GroundSpeed": 310,
        "Heading": 332,
        "VerticalRate": -1920,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:27:52.049Z",
        "Lat": 36.82384,
        "Long": -121.65167,
        "Altitude": 13650,
        "GroundSpeed": 310,
        "Heading": 332,
        "VerticalRate": -1920,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:27:52.559Z",
        "Lat": 36.82448,
        "Long": -121.65253,
        "Altitude": 13650,
        "GroundSpeed": 310,
        "Heading": 332,
        "VerticalRate": -1920,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:27:53.1Z",
        "Lat": 36.82516,
        "Long": -121.65247,
        "Altitude": 13625,
        "GroundSpeed": 310,
        "Heading": 332,
        "VerticalRate": -1920,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:27:53.6Z",
        "Lat": 36.8258,
        "Long": -121.65336,
        "Altitude": 13600,
        "GroundSpeed": 310,
        "Heading": 332,
        "VerticalRate": -1920,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:27:54.133Z",
        "Lat": 36.82645,
        "Long": -121.65336,
        "Altitude": 13600,
        "GroundSpeed": 309,
        "Heading": 332,
        "VerticalRate": -1856,
        "Squawk": "3221"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:27:54.643Z",
        "Lat": 36.82713,
        "Long": -121.65424,
        "Altitude": 13575,
        "GroundSpeed": 309,
        "Heading": 332,
        "VerticalRate": -1856,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:27:55.173Z",
        "Lat": 36.82777,
        "Long": -121.6543,
        "Altitude": 13550,
        "GroundSpeed": 309,
        "Heading": 332,
        "VerticalRate": -1856,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:27:55.693Z",
        "Lat": 36.82845,
        "Long": -121.65518,
        "Altitude": 13550,
        "GroundSpeed": 309,
        "Heading": 332,
        "VerticalRate": -1856,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:27:56.271Z",
        "Lat": 36.82915,
        "Long": -121.65602,
        "Altitude": 13525,
        "GroundSpeed": 309,
        "Heading": 332,
        "VerticalRate": -1856,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:27:56.734Z",
        "Lat": 36.82974,
        "Long": -121.65602,
        "Altitude": 13500,
        "GroundSpeed": 309,
        "Heading": 332,
        "VerticalRate": -1856,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:27:57.312Z",
        "Lat": 36.83045,
        "Long": -121.65687,
        "Altitude": 13500,
        "GroundSpeed": 309,
        "Heading": 332,
        "VerticalRate": -1856,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:27:57.752Z",
        "Lat": 36.83101,
        "Long": -121.65687,
        "Altitude": 13475,
        "GroundSpeed": 309,
        "Heading": 332,
        "VerticalRate": -1856,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:27:58.233Z",
        "Lat": 36.83162,
        "Long": -121.65768,
        "Altitude": 13475,
        "GroundSpeed": 309,
        "Heading": 332,
        "VerticalRate": -1856,
        "Squawk": "3221"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:27:58.652Z",
        "Lat": 36.83217,
        "Long": -121.65773,
        "Altitude": 13450,
        "GroundSpeed": 309,
        "Heading": 332,
        "VerticalRate": -1856,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:27:59.225Z",
        "Lat": 36.8329,
        "Long": -121.65854,
        "Altitude": 13425,
        "GroundSpeed": 309,
        "Heading": 332,
        "VerticalRate": -1856,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:27:59.682Z",
        "Lat": 36.83348,
        "Long": -121.65862,
        "Altitude": 13425,
        "GroundSpeed": 309,
        "Heading": 332,
        "VerticalRate": -1856,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:28:00.155Z",
        "Lat": 36.83404,
        "Long": -121.65862,
        "Altitude": 13400,
        "GroundSpeed": 309,
        "Heading": 332,
        "VerticalRate": -1856,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:28:00.636Z",
        "Lat": 36.83464,
        "Long": -121.65945,
        "Altitude": 13400,
        "GroundSpeed": 309,
        "Heading": 332,
        "VerticalRate": -1856,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:28:01.217Z",
        "Lat": 36.83537,
        "Long": -121.65945,
        "Altitude": 13375,
        "GroundSpeed": 309,
        "Heading": 332,
        "VerticalRate": -1856,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:28:01.646Z",
        "Lat": 36.83595,
        "Long": -121.66032,
        "Altitude": 13350,
        "GroundSpeed": 309,
        "Heading": 332,
        "VerticalRate": -1920,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:28:02.103Z",
        "Lat": 36.83651,
        "Long": -121.66032,
        "Altitude": 13350,
        "GroundSpeed": 309,
        "Heading": 332,
        "VerticalRate": -1920,
        "Squawk": "3221"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:28:02.684Z",
        "Lat": 36.83725,
        "Long": -121.66122,
        "Altitude": 13325,
        "GroundSpeed": 309,
        "Heading": 332,
        "VerticalRate": -1920,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:28:03.144Z",
        "Lat": 36.8378,
        "Long": -121.66117,
        "Altitude": 13325,
        "GroundSpeed": 309,
        "Heading": 332,
        "VerticalRate": -1920,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:28:03.594Z",
        "Lat": 36.83837,
        "Long": -121.66201,
        "Altitude": 13300,
        "GroundSpeed": 309,
        "Heading": 332,
        "VerticalRate": -1856,
        "Squawk": "3221"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:28:04.034Z",
        "Lat": 36.83893,
        "Long": -121.66207,
        "Altitude": 13275,
        "GroundSpeed": 309,
        "Heading": 332,
        "VerticalRate": -1856,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:28:04.515Z",
        "Lat": 36.83954,
        "Long": -121.66288,
        "Altitude": 13275,
        "GroundSpeed": 309,
        "Heading": 332,
        "VerticalRate": -1856,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:28:05.024Z",
        "Lat": 36.84018,
        "Long": -121.66288,
        "Altitude": 13250,
        "GroundSpeed": 309,
        "Heading": 332,
        "VerticalRate": -1856,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:28:05.587Z",
        "Lat": 36.84088,
        "Long": -121.66382,
        "Altitude": 13225,
        "GroundSpeed": 309,
        "Heading": 332,
        "VerticalRate": -1856,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:28:06.145Z",
        "Lat": 36.84158,
        "Long": -121.66377,
        "Altitude": 13225,
        "GroundSpeed": 309,
        "Heading": 332,
        "VerticalRate": -1856,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:28:06.615Z",
        "Lat": 36.84219,
        "Long": -121.6646,
        "Altitude": 13200,
        "GroundSpeed": 309,
        "Heading": 332,
        "VerticalRate": -1856,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:28:07.088Z",
        "Lat": 36.84279,
        "Long": -121.66466,
        "Altitude": 13175,
        "GroundSpeed": 309,
        "Heading": 332,
        "VerticalRate": -1856,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:28:07.519Z",
        "Lat": 36.8433,
        "Long": -121.66546,
        "Altitude": 13175,
        "GroundSpeed": 309,
        "Heading": 332,
        "VerticalRate": -1920,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:28:08.068Z",
        "Lat": 36.844,
        "Long": -121.66552,
        "Altitude": 13150,
        "GroundSpeed": 309,
        "Heading": 332,
        "VerticalRate": -1920,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:28:08.566Z",
        "Lat": 36.84462,
        "Long": -121.66637,
        "Altitude": 13150,
        "GroundSpeed": 309,
        "Heading": 332,
        "VerticalRate": -1920,
        "Squawk": "3221"
      }
    ],
    "DataSystem": "A"
  },
  {
    "IcaoId": "A5BB1B",
    "Callsign": "ASA235",
    "Track": [
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:28:09.079Z",
        "Lat": 36.84526,
        "Long": -121.66637,
        "Altitude": 13125,
        "GroundSpeed": 309,
        "Heading": 332,
        "VerticalRate": -1920,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:28:09.526Z",
        "Lat": 36.84586,
        "Long": -121.66721,
        "Altitude": 13100,
        "GroundSpeed": 309,
        "Heading": 332,
        "VerticalRate": -1920,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValleyLite",
        "TimestampUTC": "2017-01-03T01:28:10.057Z",
        "Lat": 36.84652,
        "Long": -121.66721,
        "Altitude": 13100,
        "GroundSpeed": 308,
        "Heading": 332,
        "VerticalRate": -1920,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:28:10.58Z",
        "Lat": 36.84718,
        "Long": -121.66809,
        "Altitude": 13075,
        "GroundSpeed": 308,
        "Heading": 332,
        "VerticalRate": -1920,
        "Squawk": "3221"
      },
      {
        "DataSource": "ADSB",
        "ReceiverName": "ScottsValley3",
        "TimestampUTC": "2017-01-03T01:28:11.11Z",
        "Lat": 36.84782,
        "Long": -121.66803,
        "Altitude": 13050,
        "GroundSpeed": 308,
        "Heading": 332,
        "VerticalRate": -1920,
        "Squawk": "3221"
      }
    ],
    "DataSystem": "A"
  }
]
`
)
