package fgae

import(
	"fmt"
	"strings"
	"google.golang.org/appengine/datastore"

	//"github.com/skypies/geo"
	fdb "github.com/skypies/flightdb"
)

const	kRestrictorSetKind = "RSet"

func (db *FlightDB)userToRootKey(user string) *datastore.Key {
	return datastore.NewKey(db.Ctx(), kRestrictorSetKind, strings.ToLower(user), 0, nil)
}

// {{{ db.LookupRestrictorSets

func (db *FlightDB)LookupRestrictorSets(userEmail string) ([]fdb.GeoRestrictorSet, error) {
	q := datastore.NewQuery(kRestrictorSetKind).
		Ancestor(db.userToRootKey(userEmail))
//		Filter("User = ", strings.ToLower(userEmail))

	blobs := []fdb.IndexedRestrictorSetBlob{}
	keys, err := q.GetAll(db.Ctx(), &blobs)
	if err != nil {
		return nil, err
	}

	rsets := []fdb.GeoRestrictorSet{}
	for i,blob := range blobs {
		if rset,err := blob.ToRestrictorSet(keys[i].Encode()); err != nil {
			return nil, err
		} else {
			rsets = append(rsets, *rset)
		}
	}

	return rsets, nil
}

// }}}
// {{{ db.PersistRestrictorSet

func (db *FlightDB)PersistRestrictorSet(grs fdb.GeoRestrictorSet) error {
	strings.ToLower(grs.User)

	key := datastore.NewIncompleteKey(db.Ctx(), kRestrictorSetKind, db.userToRootKey(grs.User))

	if grs.DSKey != "" {
		var err error
		key,err = datastore.DecodeKey(grs.DSKey)
		if err != nil {
			return fmt.Errorf("PersistRestrictorSet[%s]: bad key '%s': %v", grs, grs.DSKey, err)
		}
	}

	blob,err := grs.ToBlob()
	if err != nil {
		return fmt.Errorf("PersistRestrictorSet[%s]: ToBlob err: %v", grs, err)
	}

	_,err = datastore.Put(db.Ctx(), key, blob)
	if err != nil {
		return fmt.Errorf("PersistRestrictorSet[%s]: Put err: %v", grs, err)
	}

	return nil
}

// }}}
// {{{ db.LoadRestrictorSet

func (db *FlightDB)LoadRestrictorSet(dskey string) (fdb.GeoRestrictorSet, error) {
	blob := fdb.IndexedRestrictorSetBlob{}

	if key,err := datastore.DecodeKey(dskey); err != nil {
		return fdb.GeoRestrictorSet{}, fmt.Errorf("LoadRestrictorSet '%s' : %v", dskey, err)
	} else if err := datastore.Get(db.Ctx(), key, &blob); err != nil {
		return fdb.GeoRestrictorSet{}, fmt.Errorf("LoadRestrictorSet '%s' : %v", dskey, err)
	}

	if grs,err := blob.ToRestrictorSet(dskey); err != nil {
		return fdb.GeoRestrictorSet{}, fmt.Errorf("LoadRestrictorSet '%s' : %v", dskey, err)
	} else {
		return *grs, nil
	}
}

// }}}
// {{{ db.DeleteRestrictorSet

func (db *FlightDB)DeleteRestrictorSet(dskey string) (error) {
	if key,err := datastore.DecodeKey(dskey); err != nil {
		return fmt.Errorf("DeleteRestrictorSet '%s' : %v", dskey, err)
	} else if err := datastore.Delete(db.Ctx(), key); err != nil {
		return fmt.Errorf("DeleteRestrictorSet '%s' : %v", dskey, err)
	}
	return nil
}

// }}}


// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
