package fgae

import(
	"golang.org/x/net/context"
	"github.com/skypies/util/dsprovider"
	fdb "github.com/skypies/flightdb"
)

// A shim on the dsprovider iterator that can talk flights
type FlightIterator dsprovider.Iterator

func NewFlightIterator(ctx context.Context, p dsprovider.DatastoreProvider, fq *FQuery) *FlightIterator {
	it := dsprovider.NewIterator(ctx, p, (*dsprovider.Query)(fq), fdb.IndexedFlightBlob{})
	return (*FlightIterator)(it)
}

func (fi *FlightIterator)Iterate(ctx context.Context) bool {
	it := (*dsprovider.Iterator)(fi)
	return it.Iterate(ctx)
}

func (fi *FlightIterator)Err() error {
	it := (*dsprovider.Iterator)(fi)
	return it.Err()
}

func (fi *FlightIterator)Flight() *fdb.Flight {
	blob := fdb.IndexedFlightBlob{}

	it := (*dsprovider.Iterator)(fi)
	keyer := it.Val(&blob)

	f, err := blob.ToFlight(keyer.Encode())
	if err != nil {
		it.SetErr(err)
		return nil
	}

	return f
}
