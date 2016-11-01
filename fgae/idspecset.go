package fgae

// Routines to store report result sets in datastore, so we can have GET URLs that refer to many
// flights.

import(
	"google.golang.org/appengine/datastore"
	"golang.org/x/net/context"
)

// Wrap up in a datastore-friendly struct
type IdSpecSetStruct struct {
	IdSpecStrings []string `datastore:",noindex"`
}

// Returns a serialized key to look it up again
func IdSpecSetSave(ctx context.Context, idspecstrings []string) (string, error) {
	incompletekey := datastore.NewIncompleteKey(ctx, "IdSpecSet", nil)

	data := IdSpecSetStruct{IdSpecStrings:idspecstrings}
	
	if finalkey,err := datastore.Put(ctx, incompletekey, &data); err != nil {
		return "", err
	} else {
		return finalkey.Encode(), nil
	}
}

func IdSpecSetLoad(ctx context.Context, keystring string) ([]string, error) {
	key,err := datastore.DecodeKey(keystring)
	if err != nil { return nil, err }

	data := IdSpecSetStruct{}
  err = datastore.Get(ctx, key, &data)
	
	return data.IdSpecStrings, err
}
