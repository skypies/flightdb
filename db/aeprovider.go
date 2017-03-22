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

func (p AppengineDSProvider)FlattenQuery(in *Query) *datastore.Query {
	out := datastore.NewQuery(in.Kind)
	for _,filter := range in.Filters {
		out = out.Filter(filter.Field, filter.Value)
	}
	if in.OrderStr != "" { out = out.Order(in.OrderStr) }
	if in.LimitVal != 0  { out = out.Limit(in.LimitVal) }
	return out
}

func (p AppengineDSProvider)UnpackKeyers(in []Keyer) []*datastore.Key {
	out := []*datastore.Key{}
	for _,keyer := range in {
		out = append(out, keyer.(*datastore.Key))
	}
	return out
}

// The following functions implement FlightDBProvider.

func (p AppengineDSProvider)GetAll(ctx context.Context, q *Query, dst interface{}) ([]Keyer, error) {
	aeQuery := p.FlattenQuery(q)
	
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
