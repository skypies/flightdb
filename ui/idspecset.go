package ui

// Routines to store report result sets in datastore, so we can have GET URLs that refer to many
// flights.

import(
	"github.com/skypies/flightdb/fgae"
)

// Wrap up in a datastore-friendly struct
type IdSpecSetStruct struct {
	IdSpecStrings []string `datastore:",noindex"`
}

// Returns a serialized key to look it up again
func idSpecSetSave(db fgae.FlightDB, idspecstrings []string) (string, error) {
	ctx := db.Ctx()
	incompletekey := db.Backend.NewIncompleteKey(ctx, "IdSpecSet", nil)

	data := IdSpecSetStruct{IdSpecStrings:idspecstrings}
	
	if finalkey,err := db.Backend.Put(ctx, incompletekey, &data); err != nil {
		return "", err
	} else {
		return finalkey.Encode(), nil
	}
}

func idSpecSetLoad(db fgae.FlightDB, keystring string) ([]string, error) {
	ctx := db.Ctx()
	keyer,err := db.Backend.DecodeKey(keystring)
	if err != nil { return nil, err }

	data := IdSpecSetStruct{}
  err = db.Backend.Get(ctx, keyer, &data)
	
	return data.IdSpecStrings, err
}
