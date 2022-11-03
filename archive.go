package flightdb

import(
	"encoding/gob"
	"io"
)

func MarshalBlobSlice(blobs []IndexedFlightBlob, w io.Writer) error {
	return gob.NewEncoder(w).Encode(blobs)
}

func UnmarshalBlobSlice(r io.Reader) ([]IndexedFlightBlob, error) {
	blobs := []IndexedFlightBlob{}

	if err := gob.NewDecoder(r).Decode(&blobs); err != nil {
		return nil, err
	}

	return blobs, nil
}
