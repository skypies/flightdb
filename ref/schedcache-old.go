// Package ref contains some reference lookups, implemented as singletons
package ref

import(
	"bytes"
	"encoding/gob"
	"log"

	"golang.org/x/net/context"

	"github.com/skypies/util/ae"
)

func NewScheduleCache(ctx context.Context) *ScheduleCache {
	data,err := ae.LoadSingleton(ctx, kScheduleCacheSingletonName)
	if err != nil {
		log.Printf("schedcache: could not load: %v")
		return nil
	}

	buf := bytes.NewBuffer(data)
	ac := BlankScheduleCache()
	if err := gob.NewDecoder(buf).Decode(&ac); err != nil {
		log.Printf("airframecache: could not decode: %v", err)
	}

	return &ac
}

func (ac *ScheduleCache)Persist(ctx context.Context) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(ac); err != nil {
		return err
	}

	return ae.SaveSingleton(ctx, kScheduleCacheSingletonName, buf.Bytes())
}
