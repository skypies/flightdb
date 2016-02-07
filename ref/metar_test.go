package ref

// go test -v github.com/skypies/flightdb2/ref

import(
	"fmt"
	"testing"
	"time"
	"golang.org/x/net/context"
)

func TestLookupOnDemand(t *testing.T) {
	mc := NewMetarCache(context.Background(), "KSFO")
	fmt.Printf("Starting ...\n%s", mc.Archive)

	_,err := mc.Lookup(time.Now().UTC())
	if err != nil {
		t.Errorf("Lookup: %v\n", err)
	}
	fmt.Printf("Finishing ...\n%s", mc.Archive)
}
