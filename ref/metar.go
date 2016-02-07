// Package ref contains some reference lookups, implemented as singletons
// Consider moving this out of flightdb2/, so that other projects can use it more easily
package ref


import(
	"bytes"
	"encoding/gob"
	"fmt"
	"time"
	"net/http"
	
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
	"google.golang.org/appengine/memcache"

	"github.com/skypies/util/gaeutil"
	"github.com/skypies/util/widget"

	//fdb "github.com/skypies/flightdb2"
	"github.com/skypies/flightdb2/metar"
)

type MetarCache struct {
	C context.Context
	Station string
	metar.Archive

	Log string
}
func (mc MetarCache)String() string {
	return fmt.Sprintf("{metar ondemand cache}\n%s\n{log}\n%s\n", mc.Archive, mc.Log)
}

// THIS IS ALL INSUFFICIENT as an ambient background cache - because it is based on days,
// and will get confused by the current day (e.g. never memcache the current day)
// Revisit later, maybe ...

// {{{ NewMetarCache

func NewMetarCache(c context.Context, station string) *MetarCache {
	archive := metar.NewArchive()
	mc := MetarCache{
		C: c,
		Archive: *archive,
		Station: station,
	}
	return &mc
}

// }}}

// {{{ mc.populate

func (mc *MetarCache)populate(tMidnight time.Time) error {
	mc.Log += "* populate\n"
	client := urlfetch.Client(mc.C)
	reports, err := metar.FetchReportsFromNOAA(client, mc.Station, tMidnight,
		tMidnight.AddDate(0,0,1).Add(-1*time.Second))
	if err != nil { return err }

	for _,r := range reports {
		mc.Add(r)
	}
	mc.Log += fmt.Sprintf("* added?:-\n%v\n", reports)
	return nil
}

// }}}
// {{{ mc.reportsToMemcache

func (mc *MetarCache)reportsToMemcache(key string, r *[24]metar.Report) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(*r); err != nil {
		return err
	}
	return gaeutil.SaveSingletonToMemcache(mc.C, key, buf.Bytes())
}

// }}}
// {{{ mc.reportsFromMemcache

func (mc *MetarCache)reportsFromMemcache(key string) ([24]metar.Report, error) {
	data,err := gaeutil.LoadSingletonFromMemcache(mc.C, key)
	if err != nil {
		return [24]metar.Report{}, err
	} 

	buf := bytes.NewBuffer(data)
	reports := [24]metar.Report{}
	if err := gob.NewDecoder(buf).Decode(&reports); err != nil {
		return [24]metar.Report{}, err
	}

	return reports, nil
}

// }}}

// {{{ mc.Lookup

func (mc *MetarCache)Lookup(t time.Time) (*metar.Report, error) {
	tMidnight := mc.AtUTCMidnight(t)

	mc.Log += fmt.Sprintf("** lookup for %s\n", tMidnight)

	if _,exists := mc.Reports[tMidnight]; !exists {
		memKey := tMidnight.Format("metar-20060102Z")
		mc.Log += fmt.Sprintf("* not exist {%s}\n", memKey)

		if reports,err := mc.reportsFromMemcache(memKey); err == memcache.ErrCacheMiss {
			mc.Log += "* not found\n"
			// Not found.
			if err := mc.populate(tMidnight); err != nil { return nil, err }
			if mc.Reports[tMidnight] == nil {
				mc.Log += fmt.Sprintf("** FFS:-\n%s", mc)
				return nil, nil // fmt.Errorf("populate failed")				
			}
			if err := mc.reportsToMemcache(memKey, mc.Reports[tMidnight]); err != nil { return nil, err }

		} else if err != nil {
			return nil,err

		} else {
			// Found in memcache
			mc.Log += fmt.Sprintf("* found!\n%v**\n", reports)
			mc.Reports[tMidnight] = &reports
		}
	}

	return mc.Archive.Lookup(t), nil
}

// }}}

// {{{ metarHandler

func init() {
	http.HandleFunc("/ref/metar", metarHandler)
}

func metarHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	mc := NewMetarCache(c, "KSFO")

	rep,err := mc.Lookup(widget.FormValueEpochTime(r, "t"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(fmt.Sprintf("OK\n\n* %s\n\n%s\n", rep, mc)))
	return
}

// }}}


// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
