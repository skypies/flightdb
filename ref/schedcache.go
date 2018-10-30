// Package ref contains some reference lookups, implemented as singletons
package ref

import(
	"bytes"
	"encoding/gob"
	"fmt"
	"time"

	"golang.org/x/net/context"

	"google.golang.org/appengine/log"

	"github.com/skypies/util/ae"

	fdb "github.com/skypies/flightdb"
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

func NewScheduleCache(ctx context.Context) *ScheduleCache {
	data,err := ae.LoadSingleton(ctx, "schedcache")
	if err != nil {
		log.Errorf(ctx, "schedcache: could not load: %v")
		return nil
	}

	buf := bytes.NewBuffer(data)
	ac := BlankScheduleCache()
	if err := gob.NewDecoder(buf).Decode(&ac); err != nil {
		log.Errorf(ctx, "airframecache: could not decode: %v", err)
	}

	return ac
}

func (ac *ScheduleCache)Persist(ctx context.Context) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(ac); err != nil {
		return err
	}

	return ae.SaveSingleton(ctx, "schedcache", buf.Bytes())
}

func (ac *ScheduleCache)Get(id string) *fdb.FlightSnapshot {
	return ac.Map[id]
}
