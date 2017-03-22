package db

import(
	"fmt"
	"golang.org/x/net/context"
	fdb "github.com/skypies/flightdb"
)

/*

 it := db.NewFlightIterator(ctx, p, q)
 for it.Iterate(ctx) {
   f := it.Flight()
   fmt.Printf("%s\n", f)
 }
 if it.Err() != nil {
   return it.Err()
 }

*/

// We don't bother with the datastore Iterator implementations; they
// time out after 60s, concluding that the result set is likelly
// stale. So we just grab all the keys, and then work through them
// individually. The goal isn't for efficient batched fetches, it is
// to avoid timeouts.
type FlightIterator struct {
	p        DatastoreProvider

	keyers []Keyer // The full result set
	i        int
	val     *fdb.Flight  // Consider a more general interface{}, maybe decodes to Flight on demand
	err      error
}

// Snarf down all the keys from the get go.
func NewFlightIterator(ctx context.Context, p DatastoreProvider, q *Query) *FlightIterator {
	keyers,err := GetKeysByQuery(ctx, p, q)
	it := FlightIterator{
		p: p,
		keyers: keyers,
		err: err,
	}
	return &it
}

func (it *FlightIterator)Iterate(ctx context.Context) bool {
	if it.err != nil { return false }
	it.val,it.err = it.nextWithErr(ctx)
	return (it.val != nil && it.err == nil)
}
func (it *FlightIterator)Flight() *fdb.Flight { return it.val }
func (it *FlightIterator)Err() error {
	if it.err == nil { return nil }
	return fmt.Errorf("flightiterator: %v", it.err)
}

func (it *FlightIterator)nextWithErr(ctx context.Context) (*fdb.Flight, error) {
	if it.err != nil { return nil, it.err }

	if it.i >= len(it.keyers) {
		return nil,nil // We're all done !
	}

	keyer := it.keyers[it.i]
	it.i++

	return GetByKey(ctx, it.p, keyer)
}
