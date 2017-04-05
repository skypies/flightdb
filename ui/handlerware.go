package ui

import(
	"fmt"
	"html/template"
	"net/http"
	"time"
	
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/user"

	"github.com/skypies/util/widget"
	"github.com/skypies/flightdb/fgae"
)

/* Common code for pulling out a user session cookie, populating a Context, etc.

import "github.com/skypies/flightdb/ui"

func init() {
  http.HandleFunc("/foo", ui.WithCtxOpt(fooHandler))
  http.HandleFunc("/bar", ui.WithCtxOptTmpl(Templates, barHandler))
  //http.HandleFunc("/bar", ui.WithCtxOptTmplUser(Templates, barHandler)) // must be logged in
}

func fooHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	opt,ok := GetUIOptions(ctx)
	str := fmt.Sprintf("OK\nresultsetid=%s, ok=%v\n", opt.ResultsetID, ok) 
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(str))
}

func barHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	opt,ok := GetUIOptions(ctx)
  templates,ok := GetTemplates(ctx)
  templates.ExecuteTemplate(w, "bar-template", params)
}

 */

type baseHandler    func(http.ResponseWriter, *http.Request)
type contextHandler func(context.Context, http.ResponseWriter, *http.Request)

// To prevent other libs colliding with us in the context.Value keyspace, use these private keys
type contextKey int
const(
	uiOptionsKey contextKey = iota
	templatesKey
)

// Some convenience combos
func WithCtxOpt(ch contextHandler) baseHandler {
	return WithCtx(WithOpt(ch))
}
func WithCtxTmpl(t *template.Template, ch contextHandler) baseHandler {
	return WithCtx(WithTmpl(t,ch))
}
func WithCtxOptTmpl(t *template.Template, ch contextHandler) baseHandler {
	return WithCtx(WithTmpl(t,WithOpt(ch)))
}
func WithCtxOptTmplUser(t *template.Template, ch contextHandler) baseHandler {
	return WithCtx(WithTmpl(t,EnsureUser(WithOpt(ch))))
}

// Outermost wrapper; all other wrappers take (and return) contexthandlers
func WithCtx(ch contextHandler) baseHandler {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx,_ := context.WithTimeout(appengine.NewContext(r), 550 * time.Second)
		ch(ctx,w,r)
	}
}

func WithTmpl(t *template.Template, ch contextHandler) contextHandler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		ctx = context.WithValue(ctx, templatesKey, t)		
		ch(ctx, w, r)
	}
}

func WithOpt(ch contextHandler) contextHandler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		
		opt,err := FormValueUIOptions(ctx,r)  // May go to datastore
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Transparent magic, to permamently record large sets of idspecs passed as POST
		// params and replace them with an ID to the stored set; this lets us provide
		// permalinks.
		if err := opt.MaybeLoadSaveResultset(ctx); err != nil {
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

		if u := user.Current(ctx); u != nil {
			opt.UserEmail = u.Email
		}
		opt.LoginUrl,_ = user.LoginURL(ctx, r.URL.RequestURI()) // Also a re-login URL

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

// Underlying handlers should call this to get their session object
func GetTemplates(ctx context.Context) (*template.Template, bool) {
	tmpl, ok := ctx.Value(templatesKey).(*template.Template)
	return tmpl, ok
}

func EnsureUser(ch contextHandler) contextHandler {
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
