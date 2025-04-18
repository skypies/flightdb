package fgae

import(
	"context"
	"github.com/skypies/util/gcp/ds"
	fdb "github.com/skypies/flightdb"
)

// A shim on the dsprovider iterator that can talk flights
type FlightIterator ds.Iterator

func NewFlightIterator(ctx context.Context, p ds.DatastoreProvider, fq *FQuery) *FlightIterator {
	it := ds.NewIterator(ctx, p, (*ds.Query)(fq), fdb.IndexedFlightBlob{})
	return (*FlightIterator)(it)
}

func (fi *FlightIterator)Iterate(ctx context.Context) bool {
	it := (*ds.Iterator)(fi)
	return it.Iterate(ctx)
}

func (fi *FlightIterator)Err() error {
	it := (*ds.Iterator)(fi)
	return it.Err()
}

func (fi *FlightIterator)Flight() *fdb.Flight {
	blob := fdb.IndexedFlightBlob{}

	it := (*ds.Iterator)(fi)
	keyer := it.Val(&blob)

	f, err := blob.ToFlight(keyer.Encode())
	if err != nil {
		it.SetErr(err)
		return nil
	}

	return f
}
