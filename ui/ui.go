package ui

import(
	"fmt"
	"time"
	"net/http"
	"golang.org/x/net/context"
	"google.golang.org/appengine"

	"github.com/skypies/flightdb2/fgae"

	_ "github.com/skypies/flightdb2/analysis" // populate the reports registry
)

// A 'middleware' handler to parse out common fields, and stuff them into a context

/* Common code for pulling out a user session cookie, populating a Context, etc.
 * Users that aren't logged in will be redirected to the specified URL.

func init() {
  http.HandleFunc("/deb", UIOptionsHandler(debHandler))
}

func debHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	opt,ok := GetUIOptions(ctx)
	str := fmt.Sprintf("OK\nresultsetid=%s, ok=%v\n", opt.ResultsetID, ok) 
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(str))
}

 */

// To prevent other libs colliding in the context.Value keyspace, use this private key
type contextKey int
const uiOptionsKey contextKey = 0

type baseHandler    func(http.ResponseWriter, *http.Request)
type contextHandler func(context.Context, http.ResponseWriter, *http.Request)

func UIOptionsHandler(ch contextHandler) baseHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx,_ := context.WithTimeout(appengine.NewContext(r), 550 * time.Second)
		r.ParseForm()

		opt,err := FormValueUIOptions(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := opt.MaybeLoadSaveResultset(ctx); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if r.FormValue("debugoptions") != "" {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(fmt.Sprintf("OK\n%#v\n", opt)))
			return
		}

		// Call the underlying handler, with our shiny context
		ctx = context.WithValue(ctx, uiOptionsKey, opt)		
		ch(ctx, w, r)
	}
}

// Maybe load, or maybe save, all the idspecs as a resultset in datastore.
func (opt *UIOptions)MaybeLoadSaveResultset(ctx context.Context) error {

	// We have a stub resultsetid, and some idstrings - store them.
	if opt.ResultsetID == "saveme" && len(opt.IdSpecStrings) > 0 {
		if keyid,err := fgae.IdSpecSetSave(ctx, opt.IdSpecStrings); err != nil {
			return err
		} else {
			opt.ResultsetID = keyid
		}

	} else if opt.ResultsetID != "" && len(opt.IdSpecStrings) == 0 {
		if idspecstrings,err := fgae.IdSpecSetLoad(ctx, opt.ResultsetID); err != nil {
			return err
		} else {
			opt.IdSpecStrings = idspecstrings
		}
	}

	return nil
}

// Underlying handlers should call this to get their session object
func GetUIOptions(ctx context.Context) (UIOptions,bool) {
	opt, ok := ctx.Value(uiOptionsKey).(UIOptions)
	return opt, ok
}
