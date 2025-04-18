package main

// You'll need something useful in $ENV{GOOGLE_APPLICATION_CREDENTIALS}

import(
	"flag"
	"fmt"
	"log"
	"os"
	"reflect"
	"time"

	"context"

	"github.com/skypies/adsb"
	"github.com/skypies/util/date"
	"github.com/skypies/util/gcp/ds"
	"github.com/skypies/util/gcp/gcs"

	fdb "github.com/skypies/flightdb"
	"github.com/skypies/flightdb/fgae"
)

var(
	ctx = context.Background()
	fVerbosity int
	fFoiaOnly bool
	fInPdt bool
	fLimit int
	fIcaoId string
	fCallsign string
	fArchiveFrom, fArchiveTo string
	archiveFoldername = "archived-flights"
)

// {{{ type timeType

// timeType is a time that implements flag.Value
type timeType time.Time
func (t *timeType) String() string { return date.InPdt(time.Time(*t)).Format(time.RFC3339) }
func (t *timeType) Set(value string) error {
	format := "2006-01-02T15:04:05"  // No zoned time.RFC3339, because ParseInPdt adds one
	if tm,err := date.ParseInPdt(format, value); err != nil {
		return err
	} else {
		*t = timeType(tm)
	}
	return nil
}

// }}}

// {{{ init

func init() {
	flag.IntVar(&fVerbosity, "v", 0, "verbosity level")
	flag.BoolVar(&fFoiaOnly, "foia", false, "FOIA data only")
	flag.BoolVar(&fInPdt, "pdt", true, "show timestamps in PDT")
	flag.IntVar(&fLimit, "limit", 40, "how many matches to retrieve")
	flag.StringVar(&fIcaoId, "icao", "", "ICAO id for airframe (6-digit hex)")
	flag.StringVar(&fCallsign, "callsign", "", "Callsign, or maybe registration, for a flight")

	flag.StringVar(&fArchiveFrom, "archivefrom", "", "2015.01.01")
	flag.StringVar(&fArchiveTo, "archiveto", "", "2015.01.02")

	for _,e := range []string{"GOOGLE_APPLICATION_CREDENTIALS"} {
		if os.Getenv(e) == "" {
			log.Fatal("You're gonna need $"+e)
		}
	}

	flag.Parse()
}

// }}}

// {{{ queryFromArgs

// Based on the various command line flags
func queryFromArgs() *fgae.FQuery {
	fq := fgae.NewFlightQuery()
	fq.Limit(fLimit)
	if fFoiaOnly {fq.ByTags([]string{"FOIA"}) }

	// last updated stuff
	//cutoff,err := time.Parse("2006-01-02 15:04 -0700 MST", "2017-01-01 04:00 -0700 PDT")
	//if false && err == nil  {
	//	q.Filter("LastUpdate > ", cutoff).Limit(100)
	//}

	if fIcaoId != "" { fq.ByIcaoId(adsb.IcaoId(fIcaoId)) }
	if fCallsign != "" { fq.ByCallsign(fCallsign) }

	fq.Order("-LastUpdate")
	
	return fq
}

// }}}
// {{{ runQuery

func runQuery(fq *fgae.FQuery) {
	fmt.Printf("Running query %s\n", fq)

	p,err := ds.NewCloudDSProvider(ctx,"serfr0-fdb")
	if err != nil { log.Fatal(err) }

	db := fgae.New(ctx,p)

	flights,err := db.LookupAll(fq)
	if err != nil { log.Fatal(err) }

	for i,f := range flights {
		s,_ := f.Times()
		if fInPdt { s = date.InPdt(s) }

		n := len(f.AnyTrack())
		str := fmt.Sprintf("%25.25s %s %4dpts %s", f.IdentityString(), s, n, f.IdSpecString())

		fmt.Printf("[%2d] %s\n", i, str)
	}
	fmt.Printf("\n")

	if fVerbosity > 0 {
		for i,f := range flights {
			str := fmt.Sprintf("----{ %d : %s }----\n", i, f.IdentityString())
			str += fmt.Sprintf("    idspec: %s    key %s\n", f.IdSpec(), f.GetDatastoreKey())
			str += fmt.Sprintf("    airframe: %s\n", f.Airframe.String())
			str += fmt.Sprintf("    index tags: %v\n", f.IndexTagList())
			str += fmt.Sprintf("    - Anytrack: %s\n", f.AnyTrack())
			
			for k,v := range f.Tracks {
				str += fmt.Sprintf("    - [%-7.7s] %s\n", k, v)
				if fVerbosity > 2 {
					for n,tp := range *v {
						str += fmt.Sprintf("      - [%3d] %s\n", n, tp)
					}
				}
			}
			for _,t := range f.Timeslots() {
				str += fmt.Sprintf("    ** timeslot: [%s] %s\n", t, date.InPdt(t))
			}
			
			if fVerbosity > 1 {
				str += fmt.Sprintf("---- DebugLog:-\n%s\n", f.DebugLog)
			}
			str += "\n"
			fmt.Print(str)
		}
	}
}

// }}}

// {{{ runArchiver

func runArchiver(sStr, eStr string) {

	s := date.Datestring2MidnightPdt(sStr)
	e := date.Datestring2MidnightPdt(eStr)

	if s.IsZero() {
		log.Fatal("need archivefrom")
	} else if e.IsZero() {
		log.Printf("(assuming single day of archiving)")
		e = s
	}
	// Nudge a second either way, else IntermediateMidnights will skip them
	s = s.Add(-1 * time.Second)
	e = e.Add(1 * time.Second)

	p,err := ds.NewCloudDSProvider(ctx,"serfr0-fdb")
	if err != nil { log.Fatal(err) }
	db := fgae.New(ctx,p)

	log.Printf("(archiving from %s - %s)\n", s, e)

	midnights := date.IntermediateMidnights(s, e)
	for _, m := range midnights {
		winS, winE := date.WindowForTime(m) // start/end timestamps for the 23-25h day that follows midnight

		log.Printf("%s: (starting)", m)

		// First, just delete any FOIA objects
		//log.Printf("%s: (looking fort FOIA objects)", m)
		fq := fgae.NewFlightQuery().ByTags([]string{"FOIA"}).ByTimeRange(winS, winE)
		keyers,err := db.LookupAllKeys(fq)
		if err != nil { log.Fatal(err) }
		//log.Printf("%s: (found %d FOIA objects)", m, len(keyers))

		if len(keyers) > 0 {
			log.Printf("%s: deleting %d FOIA flights from DB ...\n", m, len(keyers))
			if err := multiPassDeleteAllKeys(db, keyers); err != nil {
				log.Fatal(err)
			}
			log.Printf("%s: ... done\n", m)
		}

		// Now archive what remains
		//log.Printf("%s: (starting archiver)", m)
		if err := archiveOneDay(db, m, winS, winE); err != nil {
			log.Fatal(err)
		}
	}
}

// }}}
// {{{ archiveOneDay

func archiveOneDay(db fgae.FlightDB, midnight, s, e time.Time) error {
	ctx := db.Ctx()

	overwrite := false
	delete := true
	filename := midnight.Format("2006-01-02-flights")

	blobs := []fdb.IndexedFlightBlob{}
	keyers := []ds.Keyer{}
	q := fgae.NewFlightQuery().ByTimeRange(s,e)
	// log.Printf("archiver WTF (%s) %s, %s", filename, s, e)
	it := db.NewIterator(q)
	for it.Iterate(ctx) {
		f := it.Flight()

		// A flight that straddles midnight will have timeslots either
		// side, and so will end up showing in the results for two
		// different days. We don't want dupes in the aggregate output, so
		// we should only include the flight in one of them; we pick the
		// first day. So if the flight's first timeslot does not start
		// after our start-time, skip it.
		if slots := f.Timeslots(); len(slots)>0 && slots[0].Before(s) {
			continue
		}

		if blob, err := f.ToBlob(); err != nil {
			log.Fatal(err)
		} else {
			blobs = append(blobs, *blob)
			if keyer, err := db.Backend.DecodeKey(f.GetDatastoreKey()); err != nil {
				log.Fatal(err)
			} else {
				keyers = append(keyers, keyer)
			}
		}
	}
	if it.Err() != nil {
		return fmt.Errorf("iterator [%s,%s] failed at %s: %v", s,e, time.Now(), it.Err())
	}

	if len(blobs) == 0 {
		log.Printf("%s: no flights found, nothing to archive or verify against; skipping", filename)
		return nil
	}

	exists,err := gcs.Exists(ctx, archiveFoldername, filename);
	if err != nil {
		return err
	}	else if exists {
		log.Printf("%s: GCS file already existed", filename)
	}

	if !exists || overwrite {
		gcsHandle,err := gcs.OpenRW(ctx, archiveFoldername, filename, "application/octet-stream")
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("%s: writing %d blobs to %s", midnight, len(blobs), filename)

		if err := fdb.MarshalBlobSlice(blobs, gcsHandle.IOWriter()); err != nil {
			log.Fatal(err)
		}

		if err := gcsHandle.Close(); err != nil {
			return err
		}

		log.Printf("%s: GCS archive file successfully written (%d flights)", filename, len(blobs))
	}

	log.Printf("%s: verifying GCS archive file ...", filename)
	if err := verifyArchiveFlights(archiveFoldername, filename, blobs); err != nil {
		log.Fatal(err)
	}
	log.Printf("%s: ... GCS archive file successfully verified !", filename)

	if delete {
		log.Printf("%s: deleting %d flights from DB ...\n", filename, len(keyers))
		if err := multiPassDeleteAllKeys(db, keyers); err != nil {
			log.Fatal(err)
		}
		log.Printf("%s: ... done\n", filename)
	}

	return nil
}

// }}}
// {{{ verifyArchivedFlights

func verifyArchiveFlights(bucketname, filename string, origBlobs []fdb.IndexedFlightBlob) error {
	
	if exists,_ := gcs.Exists(ctx, bucketname, filename); !exists {
		return fmt.Errorf("can not find existing file %s/%s", bucketname, filename)
	}

	filehandle, err := gcs.OpenR(ctx, bucketname, filename)
	if err != nil {
		return err
	}
	defer filehandle.Close()

	rdr, err := filehandle.ToReader(ctx, bucketname, filename)
	if err != nil {
		return err
	}

	archivedBlobs, err := fdb.UnmarshalBlobSlice(rdr)
	if err != nil {
		return err
	}

	if len(archivedBlobs) != len(origBlobs) {
		return fmt.Errorf("%s/%s: count mismatch - orig=%d, archive=%d\n", bucketname, filename,
			len(origBlobs), len(archivedBlobs))
	}

	for i:=0; i<len(origBlobs); i++ {
		// This timestamp is tied to object creation, so nuke it to help with DeepEqual
		archivedBlobs[i].LastUpdate = origBlobs[i].LastUpdate

		// Can't simply DeepEqual the serialized blobs; the encoded binary strings will differ
		f1,_ := archivedBlobs[i].ToFlight("fakekey")
		f2,_ := origBlobs[i].ToFlight("fakekey")

		// Are all the blob fields (aside from the encoded flight) equal ?
		b1, b2 := origBlobs[i], archivedBlobs[i]
		b1.Blob, b2.Blob = []byte{}, []byte{}
		if ! reflect.DeepEqual(b1, b2) {
			return fmt.Errorf("%s: archived blob %d had different index fields\n\n%#v\n\n%#v\n\n",
				filename, i, b1, b2)
		}

		// Nopw check the decoded flight objects
		if ! reflect.DeepEqual(f1, f2) {
			// If this isn't an issue, make the output more readable
			if f1.DebugLog == f2.DebugLog {
				f1.DebugLog, f2.DebugLog = "", ""
			}

			return fmt.Errorf("%s: decoded FlightBlob %d did not match:-\n\n%#v\n\n\n%#v\n\n",
				filename, i, f1, f2)
		}
	}

	return nil
}

// }}}
// {{{ multiPassDeleteAllKeys

func multiPassDeleteAllKeys(db fgae.FlightDB, keyers []ds.Keyer) error {
	maxKeyersToDeleteInOneCall := 500 // May need to make multiple calls

	for len(keyers) > 0 {
		keyersToDelete := []ds.Keyer{}
		if len(keyers) <= maxKeyersToDeleteInOneCall {
			keyersToDelete, keyers = keyers, keyersToDelete
		} else {
			keyersToDelete, keyers = keyers[0:maxKeyersToDeleteInOneCall], keyers[maxKeyersToDeleteInOneCall:]
		}

		if err := db.DeleteAllKeys(keyersToDelete); err != nil {
			return err
		}
	}
	return nil
}

// }}}

func main() {
	if fArchiveFrom != "" {
		runArchiver(fArchiveFrom, fArchiveTo)
		return
	}

	if len(flag.Args()) == 0 {
		runQuery(queryFromArgs())
		return
	}

	// assume it's all idspecs ...
	for _,arg := range flag.Args() {
		if idspec,err := fdb.NewIdSpec(arg); err == nil {
			fmt.Printf("Idspec time: %s (%s)\n", idspec.Time, date.InPdt(idspec.Time))
			runQuery(queryFromArgs().ByIdSpec(idspec))
		} else {
			log.Fatal("bad idspec '%s': %v\n", arg, err)
		}
	}
}


// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
