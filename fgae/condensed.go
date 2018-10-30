package fgae

import(
	"bytes"
	"encoding/gob"
	"fmt"
	"time"

	"golang.org/x/net/context"

	"github.com/skypies/util/ae"
	"github.com/skypies/util/gcp/ds"

	fdb "github.com/skypies/flightdb"
)

// {{{ db.FetchCondensedFlights

func (flightdb FlightDB)FetchCondensedFlights(s,e time.Time, tags []string) ([]fdb.CondensedFlight,error,string) {
	key := fmt.Sprintf("condensed-%d:%d-%v", s.Unix(), e.Unix(), tags)
	str := fmt.Sprintf("FetchCondensedFlights\n* s: %s\n* e: %s\nt: %v\nk: %s\n\n", s, e, tags, key)

	coolOff := time.Hour*2 // time.Hour*25 {  // I can't remember why it needed to be 25h ?
	if time.Since(e) < coolOff {
		return []fdb.CondensedFlight{}, fmt.Errorf("Can't condense; inside cooling period %v", coolOff), str
	}
	
	// If we've already processed this day, load the object
	if data,err := ae.LoadSingletonFromDatastore(flightdb.Ctx(), key); err == nil {
		buf := bytes.NewBuffer(data)
		cfs := []fdb.CondensedFlight{}
		if err := gob.NewDecoder(buf).Decode(&cfs); err != nil {
			flightdb.Errorf("condense: could not decode: %v", err)
		}
		str += fmt.Sprintf("found %d OK\n", len(cfs))
		return cfs,nil,str

	} else if err != ae.ErrNoSuchEntityDS {
		return []fdb.CondensedFlight{}, fmt.Errorf("condense load: %v",err), str
	}

	// OK, do it the hard way
	cfs,err,fetchstr := fetchCondensedFlightsIndividually(flightdb.Ctx(), flightdb.Backend, s,e,tags);
	str += "--\n" + fetchstr + "--\n"
	if err != nil {
		return []fdb.CondensedFlight{}, err, str
	}
	str += fmt.Sprintf("raw lookup found %d matches\n", len(cfs))

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(cfs); err != nil {
		return []fdb.CondensedFlight{}, fmt.Errorf("condense encode: %v",err), str
	} else if err := ae.SaveSingletonToDatastore(flightdb.Ctx(), key, buf.Bytes()); err != nil {
		return []fdb.CondensedFlight{}, fmt.Errorf("condense save: %v",err), str
	}
	str += fmt.Sprintf("singleton persisted, %d bytes, with %d\n", buf.Len(), len(cfs))

	return cfs, nil, str
}

// }}}

// {{{ fetchCondensedFlightsIndividually

func fetchCondensedFlightsIndividually(ctx context.Context, p ds.DatastoreProvider, s,e time.Time, tags []string) ([]fdb.CondensedFlight,error,string) {
	str := "# individual lookup\n"

	ret := []fdb.CondensedFlight{}

	q := QueryForTimeRange(tags, s, e)
	it := NewFlightIterator(ctx, p, q)
	i := 0
	tStart := time.Now()
	for it.Iterate(ctx) {
		cf := it.Flight().Condense()
		ret = append(ret, *cf)
		if i<50 {
			str += fmt.Sprintf("# [%3d] %s\n", i, cf)
		}
		i++
	}
	if it.Err() != nil {
		return ret,it.Err(),str
	}

	str += fmt.Sprintf("# All done ! %d results, took %s\n", i, time.Since(tStart))
	return ret,nil,str
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
