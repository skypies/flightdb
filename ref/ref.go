// Package ref contains some reference lookups, implemented as singletons
package ref

import(
	"bytes"
	"encoding/gob"
	"fmt"
	
	"golang.org/x/net/context"
	"google.golang.org/appengine/log"

	"github.com/skypies/util/gaeutil"
)

// An Airframe is a thing that flies. We use Icao24 (ADS-B Mode-S) identifiers to identify
// them. If/when we learn anything about a particular airframe, we store it here, and cache
// it indefinitely.
type Airframe struct {
	Icao24        string
	Registration  string
	EquipmentType string
}

type AirframeCache struct {
	Map map[string]*Airframe
}

func (ac AirframeCache)String() string {
	str := fmt.Sprintf("--- airframe cache (%d entries) ---\n", len(ac.Map))
	for _,v := range ac.Map {
		str += fmt.Sprintf(" [%s] %10.10s %s\n", v.Icao24, v.Registration, v.EquipmentType)
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
	ac := AirframeCache{Map:map[string]*Airframe{}}
	if err := gob.NewDecoder(buf).Decode(&ac); err != nil {
		log.Errorf(c, "airframecache: could not decode: %v", err)
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

func (ac *AirframeCache)Get(id string) *Airframe {
	return ac.Map[id]
}

func (ac *AirframeCache)Set(af *Airframe) {
	ac.Map[af.Icao24] = af
}

