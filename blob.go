package flightdb

import(
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"io"
	"sort"
	"time"
)

const KWaypointTagPrefix = "^"

type BlobEncoding int64
const(
	AsGob BlobEncoding = iota
	AsGzippedGob
)

var DefaultBlobEncoding = AsGzippedGob // Try and save some datastore GB-months.

// An indexed flight blob is the thing we persist into datastore (or other blobstores)
type IndexedFlightBlob struct {
	Blob             []byte      `datastore:",noindex"`
	BlobEncoding       BlobEncoding

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
	encoding := DefaultBlobEncoding

	var buf bytes.Buffer
	var writer io.Writer
	var closeFunc func() error = func()error{return nil}

	switch encoding {
	case AsGob:
		writer = &buf
	case AsGzippedGob:
		gzipWriter := gzip.NewWriter(&buf)
		closeFunc = gzipWriter.Close
		writer = gzipWriter
	default:
		return nil, fmt.Errorf("Unrecognized blobencoding '%v'", encoding)
	}

	if err := gob.NewEncoder(writer).Encode(f); err != nil {
		return nil,err
	}
	if err := closeFunc(); err != nil {
		return nil,err
	}
	
	return &IndexedFlightBlob{
		Blob: buf.Bytes(),
		BlobEncoding: encoding,
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

	var reader io.Reader
	var closeFunc func() error = func()error{return nil}

	encoding := blob.BlobEncoding
	switch encoding {
	case AsGob: reader = buf
	case AsGzippedGob:
		if gzipReader,err := gzip.NewReader(buf); err != nil {
			return &f, err
		} else {
			reader = gzipReader
			closeFunc = gzipReader.Close
		}
	default:
		return nil, fmt.Errorf("Unrecognized blobencoding '%v'", encoding)
	}

	if err := gob.NewDecoder(reader).Decode(&f); err != nil {
		return nil, err
	}

	if err := closeFunc(); err != nil {
		return nil, err
	}

	// Various kinds of post-load fixups
	f.SetDatastoreKey(key)
	f.SetLastUpdate(blob.LastUpdate)
	// TODO(abw) - retain details about encoding ?

	return &f, nil
}
