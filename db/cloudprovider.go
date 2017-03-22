package db

// https://godoc.org/cloud.google.com/go/datastore

import(
	"fmt"
	"golang.org/x/net/context"
	"cloud.google.com/go/datastore"
)

// CloudDSProvider implements the DatastoreProvider interface using the cloud datastore API,
// for use outside of appengine environments.
type CloudDSProvider struct {
	Project string
}

func (p CloudDSProvider)flattenQuery(in *Query) *datastore.Query {
	out := datastore.NewQuery(in.Kind)
	if in.AncestorKeyer != nil { out = out.Ancestor(in.AncestorKeyer.(*datastore.Key)) }
	for _,filter := range in.Filters {
		out = out.Filter(filter.Field, filter.Value)
	}
	if in.OrderStr != "" { out = out.Order(in.OrderStr) }
	if in.KeysOnlyVal    { out = out.KeysOnly() }
	return out
}

func  (p CloudDSProvider)unpackKeyer(in Keyer) *datastore.Key {
	if in == nil { return nil }
	return in.(*datastore.Key)
}
func (p CloudDSProvider)unpackKeyers(in []Keyer) []*datastore.Key {
	out := []*datastore.Key{}
	for _,keyer := range in {
		out = append(out, p.unpackKeyer(keyer))
	}
	return out
}

func (p CloudDSProvider)GetAll(ctx context.Context, q *Query, dst interface{}) ([]Keyer, error) {
	dsClient, err := datastore.NewClient(ctx, p.Project)
	if err != nil { return nil, fmt.Errorf("GetAll{cloud}: %v\nQuery: %s", err, q) }

	dsQuery := p.flattenQuery(q)

	keys,err := dsClient.GetAll(ctx, dsQuery, dst)
	if err != nil {
		return nil, fmt.Errorf("GetAll{cloud}: %v", err)
	}

	keyers := []Keyer{}
	for _,k := range keys {
		keyers = append(keyers, Keyer(k))
	}
	return keyers,nil
}


func (p CloudDSProvider)Get(ctx context.Context, keyer Keyer, dst interface{}) error {
	dsClient, err := datastore.NewClient(ctx, p.Project)
	if err != nil { return err }
	return dsClient.Get(ctx, keyer.(*datastore.Key), dst)
}

func (p CloudDSProvider)Put(ctx context.Context, keyer Keyer, src interface{}) (Keyer, error) {
	dsClient, err := datastore.NewClient(ctx, p.Project)
	if err != nil { return nil,err }

	key,error := dsClient.Put(ctx, keyer.(*datastore.Key), src)
	return Keyer(key), error
}	
func (p CloudDSProvider)Delete(ctx context.Context, keyer Keyer) error {
	dsClient, err := datastore.NewClient(ctx, p.Project)
	if err != nil { return err }
	return dsClient.Delete(ctx, keyer.(*datastore.Key))
}	
func (p CloudDSProvider)DeleteMulti(ctx context.Context, keyers []Keyer) error {
	dsClient, err := datastore.NewClient(ctx, p.Project)
	if err != nil { return err }
	return dsClient.DeleteMulti(ctx, p.unpackKeyers(keyers))
}	


func (p CloudDSProvider)DecodeKey(encoded string) (Keyer, error) {
	key, err := datastore.DecodeKey(encoded)
	return Keyer(key), err
}


func (p CloudDSProvider)NewIncompleteKey(ctx context.Context, kind, name string, root Keyer) Keyer {
	key := datastore.IncompleteKey(kind, p.unpackKeyer(root))
	return Keyer(key)
}
func (p CloudDSProvider)NewNameKey(ctx context.Context, kind, name string, root Keyer) Keyer {
	key := datastore.NameKey(kind, name, p.unpackKeyer(root))
	return Keyer(key)
}
func (p CloudDSProvider)NewIDKey(ctx context.Context, kind string, id int64, root Keyer) Keyer {
	key := datastore.IDKey(kind, id, p.unpackKeyer(root))
	return Keyer(key)
}
