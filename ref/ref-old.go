// Package ref contains some reference lookups, implemented as singletons
package ref

import(
	"bytes"
	"compress/gzip"
	"encoding/gob"

	"golang.org/x/net/context"

	"google.golang.org/appengine/log"

	"github.com/skypies/util/ae"
)

func NewAirframeCache(ctx context.Context) *AirframeCache {
	data,err := ae.LoadSingleton(ctx, kAirframeCacheSingletonName)
	if err != nil {
		log.Errorf(ctx, "airframecache: could not load: %v", err)
		return nil
	}

	ac := BlankAirframeCache()
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

	return ae.SaveSingleton(ctx, kAirframeCacheSingletonName, buf.Bytes())
}
