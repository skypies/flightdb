package fgae

import(
	"bytes"
	"encoding/gob"
	"fmt"
	"time"

	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"

	"github.com/skypies/util/gaeutil"
	fdb "github.com/skypies/flightdb"
	backend "github.com/skypies/flightdb/db"
)

// {{{ db.FetchCondensedFlights

func (db FlightDB)FetchCondensedFlights(s,e time.Time, tags []string) ([]fdb.CondensedFlight,error,string) {
	key := fmt.Sprintf("condensed-%d:%d-%v", s.Unix(), e.Unix(), tags)
	str := fmt.Sprintf("Aloha!\n* s: %s\n* e: %s\nt: %v\nk: %s\n\n", s, e, tags, key)

	if time.Since(e) < time.Hour*25 {
		return []fdb.CondensedFlight{}, fmt.Errorf("Can't condense within 25h of now"), str
	}
	
	// If we've already processed this day, load the object
	if data,err := gaeutil.LoadSingletonFromDatastore(db.Ctx(), key); err == nil {
		buf := bytes.NewBuffer(data)
		cfs := []fdb.CondensedFlight{}
		if err := gob.NewDecoder(buf).Decode(&cfs); err != nil {
			log.Errorf(db.Ctx(), "condense: could not decode: %v", err)
		}
		str += fmt.Sprintf("found %d OK\n", len(cfs))
		return cfs,nil,str

	} else if err != datastore.ErrNoSuchEntity {
		return []fdb.CondensedFlight{}, fmt.Errorf("condense load: %v",err), str
	}

	// OK, do it the hard way
	cfs,err,fetchstr := db.FetchCondensedFlightsIndividually(s,e,tags);
	str += "--\n" + fetchstr + "--\n"
	if err != nil {
		return []fdb.CondensedFlight{}, err, str
	}
	str += fmt.Sprintf("raw lookup found %d matches\n", len(cfs))

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(cfs); err != nil {
		return []fdb.CondensedFlight{}, fmt.Errorf("condense encode: %v",err), str
	} else if err := gaeutil.SaveSingletonToDatastore(db.Ctx(), key, buf.Bytes()); err != nil {
		return []fdb.CondensedFlight{}, fmt.Errorf("condense save: %v",err), str
	}
	str += fmt.Sprintf("singleton persisted, %d bytes, with %d\n", buf.Len(), len(cfs))

	return cfs, nil, str
}

// }}}
// {{{ db.FetchCondensedFlightsIndividually

func (db FlightDB)FetchCondensedFlightsIndividually(s,e time.Time, tags []string) ([]fdb.CondensedFlight,error,string) {
	str := "# individual lookup\n"

	ret := []fdb.CondensedFlight{}
	
	q := backend.QueryForTimeRange(tags, s, e)
	iter := db.NewIterator(q)
	i := 0
	tStart := time.Now()
	for iter.Iterate() {
		if iter.Err() != nil { break }
		cf := iter.Flight().Condense()
		ret = append(ret, *cf)
		if i<50 {
			str += fmt.Sprintf("# [%3d] %s\n", i, cf)
		}
		i++
	}
	if iter.Err() != nil {
		return ret,iter.Err(),str
	}

	str += fmt.Sprintf("# All done ! %d results, took %s\n", i, time.Since(tStart))
	return ret,nil,str
}

/*

New tag - :NORCAL: ?
Or OR tags ?

singleton persisted, 106166 bytes, with  650 (elapsed:  6.388675654s) [:NORCAL]
singleton persisted, 336626 bytes, with 3396 (elapsed: 28.469733166s) []

 */

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
