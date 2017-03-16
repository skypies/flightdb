// Package ref contains some reference lookups, implemented as singletons
// Consider moving this out of flightdb2/, so that other projects can use it more easily
package ref

import(
	"bytes"
	"encoding/gob"
	"fmt"
	"time"
	
	"golang.org/x/net/context"
	"google.golang.org/appengine/log"

	fdb "github.com/skypies/flightdb"
	"github.com/skypies/util/gaeutil"
)

// Keep a snapshot of the IcaoId -> Schedule mapping info in memcache.
type ScheduleCache struct {
	LastUpdated time.Time
	Map map[string]*fdb.FlightSnapshot
}

func (ac ScheduleCache)String() string {
	str := fmt.Sprintf("--- schedule cache (%d entries, age %s) ---\n", len(ac.Map),
		time.Since(ac.LastUpdated))
	for _,v := range ac.Map {
		str += fmt.Sprintf(" %s\n", v)
	}
	return str
}

func BlankScheduleCache() *ScheduleCache {
	return &ScheduleCache{
		Map: map[string]*fdb.FlightSnapshot{},
	}
}

func NewScheduleCache(c context.Context) *ScheduleCache {
	data,err := gaeutil.LoadSingleton(c,"schedcache")
	if err != nil {
		log.Errorf(c, "schedcache: could not load: %v")
		return nil
	}

	buf := bytes.NewBuffer(data)
	ac := BlankScheduleCache()
	if err := gob.NewDecoder(buf).Decode(&ac); err != nil {
		log.Errorf(c, "airframecache: could not decode: %v", err)
	}

	return ac
}

func (ac *ScheduleCache)Persist(c context.Context) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(ac); err != nil {
		return err
	}

	return gaeutil.SaveSingleton(c,"schedcache", buf.Bytes())
}

func (ac *ScheduleCache)Get(id string) *fdb.FlightSnapshot {
	return ac.Map[id]
}

