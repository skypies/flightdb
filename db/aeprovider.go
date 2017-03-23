package db

// https://cloud.google.com/appengine/docs/standard/go/datastore/reference

import(
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
)

// AppengineDSProvider implements the DatastoreProvider interface using the appengine datastore API,
// for use inside appengine environments.
type AppengineDSProvider struct {
}

func (p AppengineDSProvider)FlattenQuery(in *Query) *datastore.Query {
	out := datastore.NewQuery(in.Kind)
	if in.AncestorKeyer != nil { out = out.Ancestor(in.AncestorKeyer.(*datastore.Key)) }
	for _,filter := range in.Filters {
		out = out.Filter(filter.Field, filter.Value)
	}
	if in.OrderStr != "" { out = out.Order(in.OrderStr) }
	if in.KeysOnlyVal    { out = out.KeysOnly() }
	if in.LimitVal != 0  { out = out.Limit(in.LimitVal) }
	return out
}

func (p AppengineDSProvider)unpackKeyer(in Keyer) *datastore.Key {
	if in == nil { return nil }
	return in.(*datastore.Key)
}
func (p AppengineDSProvider)unpackKeyers(in []Keyer) []*datastore.Key {
	out := []*datastore.Key{}
	for _,keyer := range in {
		out = append(out, p.unpackKeyer(keyer))
	}
	return out
}


func (p AppengineDSProvider)GetAll(ctx context.Context, q *Query, dst interface{}) ([]Keyer, error) {
	aeQuery := p.FlattenQuery(q)	
	keys,err := aeQuery.GetAll(ctx, dst)
	if err != nil { return nil, fmt.Errorf("GetAll{AE}: %v\nQuery: %s", err, q) }

	keyers := []Keyer{}
	for _,k := range keys {
		keyers = append(keyers, Keyer(k))
	}
	return keyers,nil
}

func (p AppengineDSProvider)Get(ctx context.Context, keyer Keyer, dst interface{}) error {
	return datastore.Get(ctx, p.unpackKeyer(keyer), dst)
}

func (p AppengineDSProvider)Put(ctx context.Context, keyer Keyer, src interface{}) (Keyer, error) {
	key,error := datastore.Put(ctx, p.unpackKeyer(keyer), src)
	return Keyer(key), error
}	
func (p AppengineDSProvider)Delete(ctx context.Context, keyer Keyer) error {
	return datastore.Delete(ctx, p.unpackKeyer(keyer))
}	
func (p AppengineDSProvider)DeleteMulti(ctx context.Context, keyers []Keyer) error {
	return datastore.DeleteMulti(ctx, p.unpackKeyers(keyers))
}	

func (p AppengineDSProvider)DecodeKey(encoded string) (Keyer, error) {
	key, err := datastore.DecodeKey(encoded)
	return Keyer(key), err
}
func (p AppengineDSProvider)NewIncompleteKey(ctx context.Context, kind string, root Keyer) Keyer {
	key := datastore.NewIncompleteKey(ctx, kind, p.unpackKeyer(root))
	return Keyer(key)
}
func (p AppengineDSProvider)NewNameKey(ctx context.Context, kind, name string, root Keyer) Keyer {
	key := datastore.NewKey(ctx, kind, name, 0, p.unpackKeyer(root))
	return Keyer(key)
}
func (p AppengineDSProvider)NewIDKey(ctx context.Context, kind string, id int64, root Keyer) Keyer {
	key := datastore.NewKey(ctx, kind, "", id, p.unpackKeyer(root))
	return Keyer(key)
}
