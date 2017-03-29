package fgae

import(
	"fmt"
	"strings"
	"golang.org/x/net/context"

	"github.com/skypies/util/dsprovider"
	fdb "github.com/skypies/flightdb"
)

const	kRestrictorSetKind = "RSet"

func userToRootKey(ctx context.Context, p dsprovider.DatastoreProvider, user string) dsprovider.Keyer {
	return p.NewNameKey(ctx, kRestrictorSetKind, strings.ToLower(user), nil)
}

// {{{ db.LookupRestrictorSets

func (flightdb *FlightDB)LookupRestrictorSets(userEmail string) ([]fdb.GeoRestrictorSet, error) {
	q := dsprovider.NewQuery(kRestrictorSetKind).
		Ancestor(userToRootKey(flightdb.Ctx(), flightdb.Backend, userEmail))

	blobs := []fdb.IndexedRestrictorSetBlob{}
	keyers, err := flightdb.Backend.GetAll(flightdb.Ctx(), q, &blobs)
	if err != nil {
		return nil, err
	}

	rsets := []fdb.GeoRestrictorSet{}
	for i,blob := range blobs {
		if rset,err := blob.ToRestrictorSet(keyers[i].Encode()); err != nil {
			return nil, err
		} else {
			rsets = append(rsets, *rset)
		}
	}

	return rsets, nil
}

// }}}
// {{{ db.PersistRestrictorSet

func (flightdb *FlightDB)PersistRestrictorSet(grs fdb.GeoRestrictorSet) error {
	strings.ToLower(grs.User)

	// Default to an incomplete key (i.e. a new thing), but overwrite if we have a real key
	keyer := flightdb.Backend.NewIncompleteKey(flightdb.Ctx(), kRestrictorSetKind,
		userToRootKey(flightdb.Ctx(), flightdb.Backend, grs.User))

	if grs.DSKey != "" {
		var err error
		keyer,err = flightdb.Backend.DecodeKey(grs.DSKey)
		if err != nil {
			return fmt.Errorf("PersistRestrictorSet[%s]: bad key '%s': %v", grs, grs.DSKey, err)
		}
	}

	blob,err := grs.ToBlob()
	if err != nil {
		return fmt.Errorf("PersistRestrictorSet[%s]: ToBlob err: %v", grs, err)
	}

	_,err = flightdb.Backend.Put(flightdb.Ctx(), keyer, blob)
	if err != nil {
		return fmt.Errorf("PersistRestrictorSet[%s]: Put err: %v", grs, err)
	}

	return nil
}

// }}}
// {{{ db.LoadRestrictorSet

func (flightdb *FlightDB)LoadRestrictorSet(dskey string) (fdb.GeoRestrictorSet, error) {
	p := flightdb.Backend
	blob := fdb.IndexedRestrictorSetBlob{}
	
	if keyer,err := p.DecodeKey(dskey); err != nil {
		return fdb.GeoRestrictorSet{}, fmt.Errorf("LoadRestrictorSet '%s' : %v", dskey, err)
	} else if err := p.Get(flightdb.Ctx(), keyer, &blob); err != nil {
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

func (flightdb *FlightDB)DeleteRestrictorSet(dskey string) (error) {
	p := flightdb.Backend

	if keyer,err := p.DecodeKey(dskey); err != nil {
		return fmt.Errorf("DeleteRestrictorSet '%s' : %v", dskey, err)
	} else if err := p.Delete(flightdb.Ctx(), keyer); err != nil {
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
