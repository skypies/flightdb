// Package ref contains some reference lookups, implemented as singletons
// Consider moving this out of flightdb2/, so that other projects can use it more easily
package ref

import(
	"bytes"
	"encoding/gob"
	"fmt"
	
	"golang.org/x/net/context"
	"google.golang.org/appengine/log"

	fdb "github.com/skypies/flightdb"
	"github.com/skypies/util/gaeutil"
)

// We build a big map, from Icao24 ADS-B Mode-S identifiers, to static data about the physical
// airframe that is flying.
type AirframeCache struct {
	Map map[string]*fdb.Airframe
}

func (ac AirframeCache)String() string {
	str := fmt.Sprintf("--- airframe cache (%d entries) ---\n", len(ac.Map))
	for _,v := range ac.Map {
		str += fmt.Sprintf(" %s\n", v)
	}
	return str
}

func NewAirframeCache(c context.Context) *AirframeCache {
	data,err := gaeutil.LoadSingleton(c,"airframes")
	if err != nil {
		log.Errorf(c, "airframecache: could not load: %v")
		return nil
	}

	buf := bytes.NewBuffer(data)
	ac := AirframeCache{Map:map[string]*fdb.Airframe{}}
	if err := gob.NewDecoder(buf).Decode(&ac); err != nil {
		//log.Errorf(c, "airframecache: could not decode: %v", err)
	}

	return &ac
}

func (ac *AirframeCache)Persist(c context.Context) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(ac); err != nil {
		return err
	}

	return gaeutil.SaveSingleton(c,"airframes", buf.Bytes())
}

func (ac *AirframeCache)Get(id string) *fdb.Airframe {
	return ac.Map[id]
}

func (ac *AirframeCache)Set(af *fdb.Airframe) {
	ac.Map[af.Icao24] = af
}

