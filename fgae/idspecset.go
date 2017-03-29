package fgae

// Routines to store report result sets in datastore, so we can have GET URLs that refer to many
// flights.

import(
	"golang.org/x/net/context"
	"github.com/skypies/util/dsprovider"
)

// Wrap up in a datastore-friendly struct
type IdSpecSetStruct struct {
	IdSpecStrings []string `datastore:",noindex"`
}

// Returns a serialized key to look it up again
func IdSpecSetSave(ctx context.Context, idspecstrings []string) (string, error) {
	p := dsprovider.AppengineDSProvider{}
	incompletekey := p.NewIncompleteKey(ctx, "IdSpecSet", nil)

	data := IdSpecSetStruct{IdSpecStrings:idspecstrings}
	
	if finalkey,err := p.Put(ctx, incompletekey, &data); err != nil {
		return "", err
	} else {
		return finalkey.Encode(), nil
	}
}

func IdSpecSetLoad(ctx context.Context, keystring string) ([]string, error) {
	p := dsprovider.AppengineDSProvider{}
	keyer,err := p.DecodeKey(keystring)
	if err != nil { return nil, err }

	data := IdSpecSetStruct{}
  err = p.Get(ctx, keyer, &data)
	
	return data.IdSpecStrings, err
}
