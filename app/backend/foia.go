package backend

import(
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	"google.golang.org/appengine"
	"google.golang.org/appengine/taskqueue"
	"google.golang.org/appengine/log"	

	"github.com/skypies/util/date"
	"github.com/skypies/util/widget"
	fdb "github.com/skypies/flightdb"
	"github.com/skypies/flightdb/faadata"
	"github.com/skypies/flightdb/fgae"
)

func init() {
	http.HandleFunc("/foia/load", foiaHandler)
	http.HandleFunc("/foia/enqueue", multiEnqueueHandler)
	//http.HandleFunc("/foia/rm", rmHandler)
}

// {{{ foiaHandler

// http://backend-dot-serfr0-fdb.appspot.com/foia/load?bucket=eastbay-foia&date=20160703

// Load up FOIA historical data from GCS, and add new flights into the DB
func foiaHandler(w http.ResponseWriter, r *http.Request) {
	ctx,_ := context.WithTimeout(appengine.NewContext(r), 10*time.Minute)

	datestr := r.FormValue("date")
	if datestr == "" {
		http.Error(w, "need 'date=20141231' arg", http.StatusBadRequest)
		return
	}

	bucketName := getBucket(w,r)
	if bucketName == "" { return }

	namepairs,err := getGCSFilenames(ctx,bucketName,datestr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	str := ""
	for _,pair := range namepairs {
		str += fmt.Sprintf("----------- %s|%s -----------\n", pair[0], pair[1])
		output,err := loadGCSFile(ctx, pair[0], pair[1])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		str += output
	}
	
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(fmt.Sprintf("OK\n%s", str)))
	log.Infof(ctx, "FOIA loader for date=%s\n%s", datestr, str)
}

// }}}
// {{{ multiEnqueueHandler

// eb-foia
// http://backend-dot-serfr0-fdb.appspot.com/foia/enqueue?bucket=eastbay-foia&date=range&range_from=2013/01/01&range_to=2016/06/24
// http://backend-dot-serfr0-fdb.appspot.com/foia/enqueue?bucket=eastbay-foia&date=range&range_from=2016/06/25&range_to=2016/10/31

// Writes them all into a batch queue
func multiEnqueueHandler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	str := ""

	bucketName := getBucket(w,r)
	if bucketName == "" { return }
	
	s,e,_ := widget.FormValueDateRange(r)
	days := date.IntermediateMidnights(s.Add(-1 * time.Second),e) // decrement start, to include it
	url := "/foia/load"
	
	for i,day := range days {
		dayStr := day.Format("20060102")
		thisUrl := fmt.Sprintf("%s?bucket=%s&date=%s", url, bucketName, dayStr)
		
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
	ctx,_ := context.WithTimeout(appengine.NewContext(r), 9*time.Minute)
	db := fgae.NewDB(ctx)
	
	n := 0
	
	max := widget.FormValueInt64(r,"n")
	if max == 0 { max = 50000 }
	q := 	db.NewQuery().ByTags([]string{"FOIA"}).Limit(int(max))

	tStart := time.Now()
	str := "starting ...\n\n"
	
	for {
		keyers,err := db.LookupAllKeys(q)	
		if err != nil {
			http.Error(w, fmt.Sprintf("GetAll(n=%d): %s", n, err.Error()), http.StatusInternalServerError)
			return
		}
		str += fmt.Sprintf("Found %d keys\n", len(keyers))

		if len(keyers)==0 { break }

		maxRm := 400
		for len(keyers)>maxRm {
			if err := db.DeleteAllKeys(keyers[0:maxRm-1]); err != nil {
				str += fmt.Sprintf("keys remain: %d\n", len(keyers))
				http.Error(w, err.Error()+"\n--\n"+str, http.StatusInternalServerError)
				return
			} else {
				n += maxRm
			}
			keyers = keyers[maxRm:]
		}
		if err = db.DeleteAllKeys(keyers); err != nil {
			str += fmt.Sprintf("keys remain2: %d\n", len(keyers))
			http.Error(w, err.Error()+"\n"+str, http.StatusInternalServerError)
			return
		} else {
			n += len(keyers)
		}
	}

	str += "\nKeys all deleted :O\nTime taken: " + time.Since(tStart).String() + "\n"
	
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(fmt.Sprintf("OK\n%s", str)))
}

// }}}

// {{{ getBucket

func getBucket(w http.ResponseWriter, r *http.Request) string {
	bucketName := r.FormValue("bucket")

	validBuckets := []string{"eastbay-foia", "mtv-foia"}

	for _,valid := range validBuckets {
		if bucketName == valid { return bucketName }
	}

	http.Error(w, fmt.Sprintf("bucket '%s' not recognized", bucketName), http.StatusBadRequest)
	return ""
}

// }}}
// {{{ getGCSReader

func getGCSReader(ctx context.Context, bucketName, fileName string) (io.Reader, error) {
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
	return gzipReader,nil
}

// }}}
// {{{ getGCSFilenames

// PA  naming: faa-foia     FOIA-2015-006790/Offload_track_table  /Offload_track_20150104.txt.gz
// RG  naming: rg-foia      2014                                  /Offload_track_IFR_20140104.txt.gz
// EB  naming: eastbay-foia eb-foia/2015                          /Offload_track_IFR_20150104.txt.gz
// MTV naming: mtv-foia     2010                                  /Offload_track_IFR_20100101.txt.gz
func getGCSFilenames(ctx context.Context, bucketName, datestr string) ([][]string, error) {
	frags := regexp.MustCompile("^(\\d{4})").FindStringSubmatch(datestr)
	if len(frags) == 0 {
		return nil, fmt.Errorf("date '%s' did not match YYYYMMDD", datestr)
	}
	year := frags[0]

	getFilePrefix := func()string { return "OH-NO-FILENAME-ERROR" }
	switch bucketName {
	case "eastbay-foia": getFilePrefix = func()string{
		return fmt.Sprintf("eb-foia/%s/Offload_track_IFR_%s", year, datestr)
	}
	case "mtv-foia": getFilePrefix = func()string{
		return fmt.Sprintf("%s/Offload_track_IFR_%s", year, datestr)
	}
	}

	log.Infof(ctx, "FOIAUPLOAD starting %s (%s)", datestr, getFilePrefix())

	client, err := storage.NewClient(ctx)
	if err != nil { return nil,err }

	bucket := client.Bucket(bucketName)
	q := &storage.Query{ Prefix: getFilePrefix() }

	//str := ""
	namepairs := [][]string{}
	it := bucket.Objects(ctx, q)
	for {
    oa, err := it.Next()
    if err == iterator.Done {
			break
    }
    if err != nil {
			return nil, fmt.Errorf("GCS-Readdir [gs://%s]%s': %v", bucketName, q.Prefix, err)
    }
		log.Infof(ctx, "%8db %s {%s}\n", oa.Size, oa.Updated.Format("2006.01.02"), oa.Name)
		namepairs = append(namepairs, []string{bucketName, oa.Name})
	}

	return namepairs,nil
}

// }}}
// {{{ loadGCSFile

func loadGCSFile(ctx context.Context, bucketName, filename string) (string, error) {
	src := bucketName+","+filename
	str := fmt.Sprintf("Flights loaded from %s\n", src)

	tStart := time.Now()

	ioReader,err := getGCSReader(ctx, bucketName, filename)
	if err != nil {
		err = fmt.Errorf("loadGCSFile(%s): %v", src, err)
		log.Errorf(ctx, "%v", err)
		return "", err
	}
	log.Infof(ctx, "opened %s, about to faadata.ReadFrom\n", src)

	n,str,err := faadata.ReadFrom(ctx, src, bucketName, ioReader, foiaIdempotentAdd)
	if err != nil {
		err = fmt.Errorf("loadGCSFile(%s): %v", src, err)
		log.Errorf(ctx, "%v", err)
		return "", err
	}
	
	str += fmt.Sprintf("-- %s all done, %d added, took %s\n", src, n, time.Since(tStart))

	log.Infof(ctx, "FOIAUPLOAD finished %s (%d flights added, took %s)\n", src, n, time.Since(tStart))
	log.Infof(ctx, str)
	
	return str,nil
}

// }}}

// {{{ foiaIdempotentAdd

func foiaIdempotentAdd(ctx context.Context, f *fdb.Flight) (bool, string, error) {
	db := fgae.NewDB(ctx)
	q := db.NewQuery().ByIdSpec(f.IdSpec()).ByTags([]string{"FOIA"})
	prefix := f.IdentityString()
	
	if flight,err := db.LookupFirst(q); err != nil {
		err = fmt.Errorf("foiaCallback(%s).GetFirst: %v", f.IdSpecString(), err)
		return false,fmt.Sprintf("ERROR lookup %s: %v\n", prefix, err), err

	} else if flight != nil {
		return false,fmt.Sprintf("already exists: %s (%s)\n", prefix, flight.IdentityString()), nil

	} else if err := db.PersistFlight(f); err != nil {
		err = fmt.Errorf("foiaCallback(%s).Persist: %v", f.IdSpecString(), err)
		return false, fmt.Sprintf("ERROR save %s: %v\n", prefix, err), err
	}

	return true, fmt.Sprintf("saved: %s\n", prefix), nil
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}