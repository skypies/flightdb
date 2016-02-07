package ui

import(
	"fmt"
	"net/http"
	
	"google.golang.org/appengine"

	"github.com/skypies/adsb"
	fdb "github.com/skypies/flightdb2"
	"github.com/skypies/flightdb2/fgae"
)

func init() {
	http.HandleFunc("/fdb/add-frag", addFragHandler)
}

// dev_appserver.py --clear_datastore=yes ./ui.yaml

func addFragHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)
	db := fgae.FlightDB{C:c}

	msgs,_ := adsb.Base64DecodeMessages(r.FormValue("msgs"))
	frag := fdb.MessagesToADSBTrackFragment(msgs)
	if err := db.AddADSBTrackFragment(frag); err != nil {
		w.Write([]byte(fmt.Sprintf("Not OK; %s: %v\n", *frag, err)))
	} else {
		w.Write([]byte(fmt.Sprintf("OK! %s\n", *frag)))
	}
}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
