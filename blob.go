package flightdb

import(
	"bytes"
	"encoding/gob"
	"sort"
	"time"
)

const KWaypointTagPrefix = "^"

// An indexed flight blob is the thing we persist into datastore (or other blobstores)
type IndexedFlightBlob struct {
	Blob             []byte      `datastore:",noindex"`

	Icao24             string
	Ident              string  // Right now, this is the ADS-B callsign (SKW2848, or N1J421)
	LastUpdate         time.Time  // Used to identify most-recent instance of Icao24 for ADS-B
	Timeslots        []time.Time
	Tags             []string

	// DO NOT POPULATE
	Waypoints        []string //`datastore:",noindex"`
}

// Real tags, and things we want to search on
func (f *Flight)IndexTagList() []string {
	tags := f.TagList()
	for _,wp := range f.WaypointList() {
		tags = append(tags, KWaypointTagPrefix + wp)
	}
	sort.Strings(tags)
	return tags
}

func (f *Flight)ToBlob() (*IndexedFlightBlob, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(f); err != nil {
		return nil,err
	}
	
	return &IndexedFlightBlob{
		Blob: buf.Bytes(),
		Icao24: f.IcaoId,
		Ident: f.Callsign,
		Timeslots: f.Timeslots(),
		Tags: f.IndexTagList(),
		// Waypoints: f.WaypointList(),
		LastUpdate: time.Now(),
	}, nil
}

func (blob *IndexedFlightBlob)ToFlight(key string) (*Flight, error) {
	buf := bytes.NewBuffer(blob.Blob)
	f := BlankFlight()
	err := gob.NewDecoder(buf).Decode(&f)

	// Various kinds of post-load fixups
	f.SetDatastoreKey(key)
	f.SetLastUpdate(blob.LastUpdate)
	
	return &f, err
}
