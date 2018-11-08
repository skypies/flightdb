// Package ref contains some reference lookups, implemented as singletons
package ref

import(
	"fmt"
	"time"

	"golang.org/x/net/context"
	"github.com/skypies/util/singleton"

	fdb "github.com/skypies/flightdb"
)

const kScheduleCacheSingletonName = "schedcache"

// Keep a snapshot of the IcaoId -> Schedule mapping info in memcache.
type ScheduleCache struct {
	LastUpdated time.Time
	Map map[string]*fdb.FlightSnapshot
}

func (ac *ScheduleCache)Get(id string) *fdb.FlightSnapshot { return ac.Map[id] }

func (ac ScheduleCache)String() string {
	str := fmt.Sprintf("--- schedule cache (%d entries, age %s) ---\n", len(ac.Map),
		time.Since(ac.LastUpdated))
	for _,v := range ac.Map {
		str += fmt.Sprintf(" %s\n", v)
	}
	return str
}

func BlankScheduleCache() ScheduleCache {
	return ScheduleCache{
		Map: map[string]*fdb.FlightSnapshot{},
	}
}

func LoadScheduleCache(ctx context.Context, sp singleton.SingletonProvider) (*ScheduleCache, error) {
	schedcache := BlankScheduleCache()
	err := sp.ReadSingleton(ctx, kScheduleCacheSingletonName, nil, &schedcache)
	return &schedcache, err
}

func (sc *ScheduleCache)SaveScheduleCache(ctx context.Context, sp singleton.SingletonProvider) error {
	return sp.WriteSingleton(ctx, kScheduleCacheSingletonName, nil, sc)
}
