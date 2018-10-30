package fgae

import (
	"fmt"
	"strings"
	"testing"

	"golang.org/x/net/context"

	"google.golang.org/appengine"
	"google.golang.org/appengine/aetest" // Also used for testing Cloud API

	"github.com/skypies/util/dsprovider"
	fdb "github.com/skypies/flightdb"
	"github.com/skypies/flightdb/faadata" // for quick ascii loading of trackpoints
)

const appid = "mytestapp"

// {{{ loadFlights

func loadFlights(t *testing.T, data string) []*fdb.Flight {
	flights := []*fdb.Flight{}
	callback := func(ctx context.Context, f *fdb.Flight) (bool,string,error) {
		flights = append(flights,f)
		return true,"",nil
	}

	_,_,err := faadata.ReadFrom(context.TODO(), "testdata", strings.NewReader(data), callback)
	if err != nil { t.Fatal(err) }

	return flights
}

// }}}
// {{{ newConsistentContext

// A version of aetest.NewContext() that has a consistent datastore - so we can read our writes.
func newConsistentContext() (context.Context, func(), error) {
	inst, err := aetest.NewInstance(&aetest.Options{
		StronglyConsistentDatastore: true,
		AppID: appid,
	})
	if err != nil {
		return nil, nil, err
	}
	req, err := inst.NewRequest("GET", "/", nil)
	if err != nil {
		inst.Close()
		return nil, nil, err
	}
	ctx := appengine.NewContext(req)
	return ctx, func() {
		inst.Close()
	}, nil
}

// }}}

// {{{ testEverything

func testEverything(t *testing.T, p dsprovider.DatastoreProvider) {
	ctx, done, err := newConsistentContext()
	if err != nil { t.Fatal(err) }
	defer done()

	db := NewDB(ctx)
	db.Backend = p
	
	flights := loadFlights(t, fakeFlights)
	for _,f := range flights {
		if err := db.PersistFlight(f); err != nil { t.Fatal(err) }
	}
	
	run := func(expected int, q *FQuery) {
		if results,err := db.LookupAll(q); err != nil {
			t.Fatal(err)
		} else if len(results) != expected {
			t.Errorf("expected %d results, saw %d; query: %s", expected, len(results), q)
			for i,f := range results { fmt.Printf("result [%3d] %s\n", i, f) }
		}
	}

	run(len(flights), db.NewQuery())
	run(3,            db.NewQuery().Limit(3))
	run(1,            db.NewQuery().ByCallsign(flights[0].Callsign))

	// Now delete something
	first,err := db.LookupFirst(db.NewQuery())
	if err != nil || first == nil {
		t.Errorf("db.GetFirstByQuery: %v / %v\n", err, first)
	} else if keyer,err := db.Backend.DecodeKey(first.GetDatastoreKey()); err != nil {
		t.Errorf("p.DecodeKey: %v\n", err)
	} else if err := db.DeleteByKey(keyer); err != nil {
		t.Errorf("p.Delete: %v\n", err)
	}

	nExpected := len(flights)-1
	run(nExpected, db.NewQuery())

	// Now test the iterator
	n := 0
	fi := db.NewIterator(db.NewQuery())
		for fi.Iterate(ctx) {
		f := fi.Flight()
		fmt.Printf(" iterator result: %s\n", f)
		n++
	}
	if fi.Err() != nil {
		t.Errorf("test iterator err: %v\n", fi.Err())
	}

	if n != nExpected {
		t.Errorf("test expected to see %d, but saw %d\n", nExpected, n)
	}

}

// }}}

func TestEverything(t *testing.T) {
	testEverything(t, dsprovider.AppengineDSProvider{})
	// Sadly, the aetest framework hangs on the first Put from the cloud client
	// testEverything(t, dsprovider.CloudDSProvider{appid})
}

var (
	// {{{ fakeFlights

	fakeFlights = `
AIRCRAFT_ID,FLIGHT_INDEX,TRACK_INDEX,SOURCE_FACILITY,BEACON_CODE,DEP_APRT,ARR_APRT,ACFT_TYPE,LATITUDE,LONGITUDE,ALTITUDEx100ft,TRACK_POINT_DATE_UTC,TRACK_POINT_TIME_UTC
AAA1234,20170401260,20170401NCT1111AAA1234,NCT,1234,SFO,LAX,B707,37.62872,-122.36932,6,20170401,16:08:17
AAA1234,20170401260,20170401NCT1111AAA1234,NCT,1234,SFO,LAX,B707,37.63181,-122.36801,8,20170401,16:08:22
AAA1234,20170401260,20170401NCT1111AAA1234,NCT,1234,SFO,LAX,B707,37.63628,-122.36617,10,20170401,16:08:27
BBB1234,20170401467,20170401NCT2111BBB1234,NCT,2234,LAS,OAK,B707,36.97572,-118.81405,352,20170401,06:15:12
BBB1234,20170401467,20170401NCT2111BBB1234,NCT,2234,LAS,OAK,B747,36.97793,-118.82863,352,20170401,06:15:18
BBB1234,20170401467,20170401NCT2111BBB1234,NCT,2234,LAS,OAK,B747,36.98252,-118.84747,350,20170401,06:15:24
CCC1234,20170401107,20170401NCT3111CCC1234,NCT,3234,BOS,SFO,B753,40.3151,-118.25999,300,20170401,01:51:49
CCC1234,20170401107,20170401NCT3111CCC1234,NCT,3234,BOS,SFO,B753,40.30411,-118.29092,300,20170401,01:52:01
CCC1234,20170401107,20170401NCT3111CCC1234,NCT,3234,BOS,SFO,B753,40.29498,-118.32425,300,20170401,01:52:13
DDD1234,20170401960,20160415NCT4111DDD1234,NCT,4234,SFO,YVR,B78W,37.70271,-122.55082,54,20170401,21:16:03
DDD1234,20170401960,20160415NCT4111DDD1234,NCT,4234,SFO,YVR,B78W,37.70497,-122.55581,55,20170401,21:16:08
DDD1234,20170401960,20160415NCT4111DDD1234,NCT,4234,SFO,YVR,B78W,37.71036,-122.56377,58,20170401,21:16:13
EEE1234,20170401398,20160413NCT5111EEE1234,NCT,5234,MIA,OAK,C535,36.26174,-120.31039,340,20170401,19:10:42
EEE1234,20170401398,20160413NCT5111EEE1234,NCT,5234,MIA,OAK,C535,36.27094,-120.32021,340,20170401,19:10:49
EEE1234,20170401398,20160413NCT5111EEE1234,NCT,5234,MIA,OAK,C535,36.27987,-120.32931,340,20170401,19:10:54
FFF1234,20170401180,20160418NCT6111FFF1234,NCT,6234,ORD,SFO,A322,35.86058,-121.14301,320,20170401,02:15:15
FFF1234,20170401180,20160418NCT6111FFF1234,NCT,6234,ORD,SFO,A322,35.869,-121.15031,320,20170401,02:15:20
FFF1234,20170401180,20160418NCT6111FFF1234,NCT,6234,ORD,SFO,A322,35.87756,-121.15737,320,20170401,02:15:25
GGG1234,20170401899,20170401NCT7111GGG1234,NCT,7234,BTR,SFO,E180,40.09747,-123.06654,350,20170401,20:49:17
GGG1234,20170401899,20170401NCT7111GGG1234,NCT,7234,BTR,SFO,E180,40.07688,-123.06439,350,20170401,20:49:29
GGG1234,20170401899,20170401NCT7111GGG1234,NCT,7234,BTR,SFO,E180,40.05581,-123.06256,350,20170401,20:49:41
HHH1234,20170401828,20160411NCT8111HHH1234,NCT,8234,LPL,SFO,B762,37.82513,-119.44514,380,20170401,17:28:37
HHH1234,20170401828,20160411NCT8111HHH1234,NCT,8234,LPL,SFO,B762,37.8186,-119.4794,380,20170401,17:28:49
HHH1234,20170401828,20160411NCT8111HHH1234,NCT,8234,LPL,SFO,B762,37.81978,-119.50132,380,20170401,17:28:59
III1234,20170401607,20170401NCT9111III1234,NCT,9234,SFO,HYA,CRJ3,37.77443,-122.27459,67,20170401,00:39:30
III1234,20170401607,20170401NCT9111III1234,NCT,9234,SFO,HYA,CRJ3,37.78058,-122.27294,69,20170401,00:39:34
III1234,20170401607,20170401NCT9111III1234,NCT,9234,SFO,HYA,CRJ3,37.7868,-122.27138,70,20170401,00:39:39
`

	// }}}
)

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
