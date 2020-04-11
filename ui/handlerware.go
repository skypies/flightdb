package ui

import(
	"fmt"
	"html/template"
	"net/http"
	
	"golang.org/x/net/context"
	// "google.golang.org/ appengine/user"

	"github.com/skypies/util/gcp/ds"
	"github.com/skypies/util/widget"
	"github.com/skypies/flightdb/fgae"
)

// Some convenience combos
func WithCtxOpt(f widget.CtxMaker, ch widget.ContextHandler) widget.BaseHandler {
	// TODO: Delete this routine, once the memcache stuff is figured out better
	return widget.WithCtx(f, WithOpt( ch))
}
func WithFdbCtx(f widget.CtxMaker, fh FdbHandler) widget.BaseHandler {
	return widget.WithCtx(f, WithFdb(fh))
}
func WithFdbCtxOpt(f widget.CtxMaker, fh FdbHandler) widget.BaseHandler {
	return widget.WithCtx(f, WithOpt( WithFdb(fh)))
}
func WithFdbCtxTmpl(f widget.CtxMaker, t *template.Template, fh FdbHandler) widget.BaseHandler {
	return widget.WithCtx(f, widget.WithTemplates(t, WithFdb(fh)))
}
func WithFdbCtxOptTmpl(f widget.CtxMaker, t *template.Template, fh FdbHandler) widget.BaseHandler {
	return widget.WithCtx(f, widget.WithTemplates(t, WithOpt( WithFdb(fh))))
}
func WithFdbCtxOptTmplUser(f widget.CtxMaker, t *template.Template, fh FdbHandler) widget.BaseHandler {
	return widget.WithCtx(f, widget.WithTemplates(t, WithOpt( EnsureUser( WithFdb(fh)))))
}

// To prevent other libs colliding with us in the context.Value keyspace, use these private keys
type contextKey int
const(
	uiOptionsKey contextKey = iota
)

// Rather than stash/retrieve an FDB object from the context, we'll just pass it
// directly to a new handler type, that we'll use throughout ui/.
type FdbHandler func(fgae.FlightDB, http.ResponseWriter, *http.Request)

func WithFdb(fh FdbHandler) widget.ContextHandler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		p := ds.GetProviderOrPanic(ctx) // PANICs if not found
		fdb := fgae.New(ctx, p)
		fh(fdb, w, r)
	}
}

func WithOpt(ch widget.ContextHandler) widget.ContextHandler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		p := ds.GetProviderOrPanic(ctx) // PANICs if not found
		db := fgae.New(ctx, p)
		r.ParseForm()
		
		opt,err := FormValueUIOptions(db,r) // May go to datastore
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Transparent magic, to permamently record large sets of idspecs passed as POST
		// params and replace them with an ID to the stored set; this lets us provide
		// permalinks.
		if err := opt.MaybeLoadSaveResultset(db); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if opt.ResultsetID != "" {
			// Rejigger all the POST and GET data into a single GET URL, then add our new field.
			vals := widget.ExtractAllCGIArgs(r)
			vals.Del("idspec")
			vals.Set("resultset", opt.ResultsetID)
			opt.Permalink = widget.URLStringReplacingGETArgs(r,&vals)
		}

		// FIXME: get user from context, and implement login
		//if u := user.Current(ctx); u != nil {
		//	opt.UserEmail = u.Email
		//}
		// opt.LoginUrl,_ = user.LoginURL(ctx, r.URL.RequestURI()) // Also a re-login URL
		opt.LoginUrl = "https://duckduckgo/"

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
func (opt *UIOptions)MaybeLoadSaveResultset(db fgae.FlightDB) error {

	// We have a stub resultsetid, and some idstrings - store them.
	if opt.ResultsetID == "saveme" && len(opt.IdSpecStrings) > 0 {
		if keyid,err := idSpecSetSave(db, opt.IdSpecStrings); err != nil {
			return err
		} else {
			opt.ResultsetID = keyid
		}

	} else if opt.ResultsetID != "" && len(opt.IdSpecStrings) == 0 {
		if idspecstrings,err := idSpecSetLoad(db, opt.ResultsetID); err != nil {
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

// FIXME: Wow - we just piggyback off the options parsing. Should maybe not do that.
func EnsureUser(ch widget.ContextHandler) widget.ContextHandler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		opt,ok := GetUIOptions(ctx)
		if !ok {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("GetUIOptions was not OK, bailing\n"))
		} else if opt.UserEmail == "" {
			http.Redirect(w,r,opt.LoginUrl,http.StatusFound)
		} else {
			ch(ctx, w, r)
		}
	}
}
