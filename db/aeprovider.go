package db

// https://cloud.google.com/appengine/docs/standard/go/datastore/reference

import(
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
)

// AppengineDSProvider implements the FlightDBProvider interface using the appengine datastore API,
// for use inside appengine environments.
type AppengineDSProvider struct {
}

func (p AppengineDSProvider)GetAll(ctx context.Context, q *Query, dst interface{}) ([]Keyer, error) {
	aeQuery := datastore.NewQuery(q.Kind)
	for _,filter := range q.Filters {
		aeQuery = aeQuery.Filter(filter.Field, filter.Value)
	}
	if q.OrderStr != "" { aeQuery = aeQuery.Order(q.OrderStr) }
	if q.LimitVal != 0 { aeQuery = aeQuery.Limit(q.LimitVal) }

	keys,err := aeQuery.GetAll(ctx, dst)
	keyers := []Keyer{}
	for _,k := range keys {
		keyers = append(keyers, Keyer(k))
	}
	return keyers,err
}

func (p AppengineDSProvider)Put(ctx context.Context, keyer Keyer, src interface{}) (Keyer, error) {
	key,error := datastore.Put(ctx, keyer.(*datastore.Key), src)
	return Keyer(key), error
}	

func (p AppengineDSProvider)DecodeKey(encoded string) (Keyer, error) {
	key, err := datastore.DecodeKey(encoded)
	return Keyer(key), err
}

func (p AppengineDSProvider)NewNameKey(ctx context.Context, kind, name string, root Keyer) Keyer {
	key := datastore.NewKey(ctx, kind, name, 0, root.(*datastore.Key))
	return Keyer(key)
}
func (p AppengineDSProvider)NewIDKey(ctx context.Context, kind string, id int64, root Keyer) Keyer {
	key := datastore.NewKey(ctx, kind, "", id, root.(*datastore.Key))
	return Keyer(key)
}
