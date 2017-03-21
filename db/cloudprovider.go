package db

// https://godoc.org/cloud.google.com/go/datastore

import(
	"golang.org/x/net/context"
	"cloud.google.com/go/datastore" // cf. appengine/datastore
)

// CloudDSProvider implements the FlightDBProvider interface using the cloud datastore API,
// for use outside of appengine environments.
type CloudDSProvider struct {
	Project string
}

func (p CloudDSProvider)GetAll(ctx context.Context, q *Query, dst interface{}) ([]Keyer, error) {
	dsClient, err := datastore.NewClient(ctx, p.Project)
	if err != nil { return nil,err }

	dsQuery := datastore.NewQuery(q.Kind)
	for _,filter := range q.Filters {
		dsQuery = dsQuery.Filter(filter.Field, filter.Value)
	}
	if q.OrderStr != "" { dsQuery = dsQuery.Order(q.OrderStr) }
	if q.LimitVal != 0 { dsQuery = dsQuery.Limit(q.LimitVal) }

	keys,err := dsClient.GetAll(ctx, dsQuery, dst)
	keyers := []Keyer{}
	for _,k := range keys {
		keyers = append(keyers, Keyer(k))
	}
	return keyers,err
}


func (p CloudDSProvider)Put(ctx context.Context, keyer Keyer, src interface{}) (Keyer, error) {
	dsClient, err := datastore.NewClient(ctx, p.Project)
	if err != nil { return nil,err }

	key,error := dsClient.Put(ctx, keyer.(*datastore.Key), src)
	return Keyer(key), error
}	


func (p CloudDSProvider)DecodeKey(encoded string) (Keyer, error) {
	key, err := datastore.DecodeKey(encoded)
	return Keyer(key), err
}


func (p CloudDSProvider)NewNameKey(ctx context.Context, kind, name string, root Keyer) Keyer {
	key := datastore.NameKey(kind, name, root.(*datastore.Key))
	return Keyer(key)
}
func (p CloudDSProvider)NewIDKey(ctx context.Context, kind string, id int64, root Keyer) Keyer {
	key := datastore.IDKey(kind, id, root.(*datastore.Key))
	return Keyer(key)
}
