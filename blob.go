package flightdb2

import(
	"bytes"
	"encoding/gob"
	"time"
)

// An indexed flight blob is the thing we persist into datastore (or other blobstores)
type IndexedFlightBlob struct {
	Blob             []byte      `datastore:",noindex"`

	Icao24             string
	Ident              string  // Right now, this is the ADS-B callsign (SKW2848, or N1J421)
	LastUpdate         time.Time  // Used to identify most-recent instance of Icao24 for ADS-B
	Timeslots        []time.Time
	Tags             []string

}

func (f *Flight)ToBlob(d time.Duration) (*IndexedFlightBlob, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(f); err != nil {
		return nil,err
	}

	return &IndexedFlightBlob{
		Blob: buf.Bytes(),
		Icao24: f.IcaoId,
		Ident: f.Callsign,
		Timeslots: f.Timeslots(d),
		Tags: f.TagList(),
		LastUpdate: time.Now(),
	}, nil
}

func (blob *IndexedFlightBlob)ToFlight(key string) (*Flight, error) {
	buf := bytes.NewBuffer(blob.Blob)
	f := Flight{}
	err := gob.NewDecoder(buf).Decode(&f)

	// Various kinds of post-load fixups
	f.SetDatastoreKey(key)
	f.SetLastUpdate(blob.LastUpdate)
	
	return &f, err
}
