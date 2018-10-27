// Package ref contains some reference lookups, implemented as singletons
// Consider moving this out of flightdb2/, so that other projects can use it more easily
package ref

import(
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"fmt"
	
	"context"
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
		log.Errorf(c, "airframecache: could not load: %v", err)
		return nil
	}

	ac := AirframeCache{Map:map[string]*fdb.Airframe{}}
	buf := bytes.NewBuffer(data)
	if gzipReader,err := gzip.NewReader(buf); err != nil {
		log.Errorf(c, "airframecache: could not gzip.NewReader: %v", err)
	} else if err := gob.NewDecoder(gzipReader).Decode(&ac); err != nil {
		//log.Errorf(c, "airframecache: could not decode: %v", err)
	} else if err := gzipReader.Close(); err != nil {
		log.Errorf(c, "airframecache: could not gzipReader.Close: %v", err)
	}

	return &ac
}

func (ac *AirframeCache)Persist(c context.Context) error {
	var buf bytes.Buffer

	gzipWriter := gzip.NewWriter(&buf)
	
	if err := gob.NewEncoder(gzipWriter).Encode(ac); err != nil {
		return err
	} else if err := gzipWriter.Close(); err != nil {
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

