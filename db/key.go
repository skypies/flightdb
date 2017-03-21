package db

// Keyer is a very thin wrapper over keys for both cloud datastore and appengine datastore.
type Keyer interface {
	Encode() string
}
