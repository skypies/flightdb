package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/cloud/bigquery"

	"github.com/skypies/util/date"
	"github.com/skypies/util/gcs"
	"github.com/skypies/util/widget"

	"github.com/skypies/flightdb2/fgae"
)

func init() {
	http.HandleFunc("/backend/publish-flights", publishFlightsHandler)
}

var(
	// The bigquery dataset (dest) is an entirely different google cloud project.

	// This, the 'src' project, needs its service worker account to be
	// added as an 'editor' to the 'dest' project, so that we can submit
	// bigquery load requests.

	// Similarly, the service worker from the 'dest' project needs to be added to
	// the 'source' project, so that dest can read the GCS folders. I think.

	// This is in the 'src' project
	folderGCS = "bigquery-flights"

	// This is the 'dest' project
	bigqueryProject = "serfr0-1000"
	bigqueryDataset = "public"
	bigqueryTableName = "flights"
)

// {{{ publishFlightsHandler

// http://backend-dot-serfr0-fdb.appspot.com/backend/publish-flights?datestring=yesterday
// http://backend-dot-serfr0-fdb.appspot.com/backend/publish-flights?datestring=2015.09.15

// As well as writing the data into a file in Cloud Storage, it will submit a load
// request into BigQuery to load that file.

func publishFlightsHandler(w http.ResponseWriter, r *http.Request) {
	tStart := time.Now()

	ctx := appengine.NewContext(r)

	datestring := r.FormValue("datestring")
	if datestring == "yesterday" {
		datestring = date.NowInPdt().AddDate(0,0,-1).Format("2006.01.02")
	}

	filename := "flights-"+datestring+".json"
	log.Infof(ctx, "Starting /backend/publish-complaints: %s", filename)
	
	n,err := writeBigQueryFlightsGCSFile(r, datestring, folderGCS, filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := submitLoadJob(r, folderGCS, filename); err != nil {
		http.Error(w, "submitLoadJob failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(fmt.Sprintf("OK!\n%d flights written to gs://%s/%s and job sent - took %s\n",
		n, folderGCS, filename, time.Since(tStart))))
}

// }}}

// {{{ writeBigQueryFlightsGCSFile

// Returns number of records written (which is zero if the file already exists)
func writeBigQueryFlightsGCSFile(r *http.Request, datestring, foldername,filename string) (int,error) {
	ctx := appengine.NewContext(r)
	
	if exists,err := gcs.Exists(ctx, foldername, filename); err != nil {
		return 0,err
	} else if exists {
		return 0,nil
	}	
	gcsHandle,err := gcs.OpenRW(ctx, foldername, filename, "application/json")
	if err != nil {
		return 0,err
	}
	encoder := json.NewEncoder(gcsHandle.IOWriter())
	
	tags := widget.FormValueCommaSpaceSepStrings(r,"tags")
	s := date.Datestring2MidnightPdt(datestring)
	e := s.AddDate(0,0,1).Add(-1 * time.Second) // +23:59:59 (or 22:59 or 24:59 when going in/out DST)

	db := fgae.FlightDB{C:ctx}
	n := 0

	q := db.QueryForTimeRange(tags, s, e)
	iter := db.NewLongIterator(q)
	for {
		f,err := iter.NextWithErr();
		if err != nil {
			return 0,fmt.Errorf("iterator [%s,%s] failed at %s: %v", s,e, time.Now(), err)
		} else if f == nil {
			break // we're all done with this iterator
		}

		// A flight that straddles midnight will have timeslots either side, and so will end up
		// showing in two sets of results. In such cases, only include the flight in the first
		// day. So if the flight's timeslot is before our start time, skip it.
		slots := f.Timeslots()
		if len(slots)>0 && slots[0].Before(s) {
			continue
		}
		
		n++
		fbq := f.ForBigQuery()

		if err := encoder.Encode(fbq); err != nil {
			return 0,err
		}
	}

	if err := gcsHandle.Close(); err != nil {
		return 0, err
	}

	log.Infof(ctx, "GCS bigquery file '%s' successfully written", filename)

	return n,nil
}

// }}}

// {{{ submitLoadJob

func submitLoadJob(r *http.Request, gcsfolder, gcsfile string) error {
	ctx := appengine.NewContext(r)

	client,err := bigquery.NewClient(ctx, bigqueryProject)
	if err != nil {
		return fmt.Errorf("Creating bigquery client: %v", err)
	}

	gcsSrc := client.NewGCSReference(fmt.Sprintf("gs://%s/%s", gcsfolder, gcsfile))
	gcsSrc.SourceFormat = bigquery.JSON

	tableDest := &bigquery.Table{
		ProjectID: bigqueryProject,
		DatasetID: bigqueryDataset,
		TableID:   bigqueryTableName,
	}
	
	job,err := client.Copy(ctx, tableDest, gcsSrc, bigquery.WriteAppend)
	if err != nil {
		return fmt.Errorf("Submission of load job: %v", err)
	}

	time.Sleep(5 * time.Second)
	
	if status, err := job.Status(ctx); err != nil {
		return fmt.Errorf("Failure determining status: %v", err)
	} else if err := status.Err(); err != nil {
		detailedErrStr := ""
		for i,innerErr := range status.Errors {
			detailedErrStr += fmt.Sprintf(" [%2d] %v\n", i, innerErr)
		}
		log.Errorf(ctx, "BiqQuery LoadJob error: %v\n--\n%s", err, detailedErrStr)
		return fmt.Errorf("Job error: %v\n--\n%s", err, detailedErrStr)
	} else {
		log.Infof(ctx, "BiqQuery LoadJob status: done=%v, state=%s, %s",
			status.Done(), status.State, status)
	}
	
	return nil
}

// }}}


// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}

