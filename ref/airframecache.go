package ref

import(
	"fmt"

	"context"

	"github.com/skypies/util/singleton"
	
	fdb "github.com/skypies/flightdb"
)

const kAirframeCacheSingletonName = "airframes"

// We build a big map, from Icao24 ADS-B Mode-S identifiers, to static data about the physical
// airframe that is flying. This is built up over time.
type AirframeCache struct {
	Map map[string]*fdb.Airframe
}

func BlankAirframeCache() AirframeCache {
	return AirframeCache{Map:map[string]*fdb.Airframe{}}
}

func (ac *AirframeCache)Get(id string) *fdb.Airframe { return ac.Map[id] }
func (ac *AirframeCache)Set(af *fdb.Airframe)        { ac.Map[af.Icao24] = af }

func (ac AirframeCache)String() string {
	str := fmt.Sprintf("--- airframe cache (%d entries) ---\n", len(ac.Map))
	for _,v := range ac.Map {
		str += fmt.Sprintf(" %s\n", v)
	}
	return str
}

func LoadAirframeCache(ctx context.Context, sp singleton.SingletonProvider) (*AirframeCache, error) {
	airframes := BlankAirframeCache()
	err := sp.ReadSingleton(ctx, kAirframeCacheSingletonName, singleton.GzipReader, &airframes)
	return &airframes, err
}

func (ac *AirframeCache)SaveAirframeCache(ctx context.Context, sp singleton.SingletonProvider) error {
	return sp.WriteSingleton(ctx, kAirframeCacheSingletonName, singleton.GzipWriter, ac)
}
