// Package ref contains some reference lookups, implemented as singletons
package ref

import(
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"fmt"

	"golang.org/x/net/context"

	"google.golang.org/appengine/log"

	"github.com/skypies/util/ae"

	fdb "github.com/skypies/flightdb"
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

func NewAirframeCache(ctx context.Context) *AirframeCache {
	data,err := ae.LoadSingleton(ctx, "airframes")
	if err != nil {
		log.Errorf(ctx, "airframecache: could not load: %v", err)
		return nil
	}

	ac := AirframeCache{Map:map[string]*fdb.Airframe{}}
	buf := bytes.NewBuffer(data)
	if gzipReader,err := gzip.NewReader(buf); err != nil {
		log.Errorf(ctx, "airframecache: could not gzip.NewReader: %v", err)
	} else if err := gob.NewDecoder(gzipReader).Decode(&ac); err != nil {
		//db.Errorf("airframecache: could not decode: %v", err)
	} else if err := gzipReader.Close(); err != nil {
		log.Errorf(ctx, "airframecache: could not gzipReader.Close: %v", err)
	}

	return &ac
}

func (ac *AirframeCache)Persist(ctx context.Context) error {
	var buf bytes.Buffer

	gzipWriter := gzip.NewWriter(&buf)
	
	if err := gob.NewEncoder(gzipWriter).Encode(ac); err != nil {
		return err
	} else if err := gzipWriter.Close(); err != nil {
		return err
	}

	return ae.SaveSingleton(ctx, "airframes", buf.Bytes())
}

func (ac *AirframeCache)Get(id string) *fdb.Airframe {
	return ac.Map[id]
}

func (ac *AirframeCache)Set(af *fdb.Airframe) {
	ac.Map[af.Icao24] = af
}

