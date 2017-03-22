package ui

import(
	"fmt"
	"net/http"
	"sort"
	"strings"

	"golang.org/x/net/context"

	"google.golang.org/appengine/log"

	"github.com/skypies/geo"
	"github.com/skypies/geo/sfo"
	"github.com/skypies/util/widget"
	fdb "github.com/skypies/flightdb"
	"github.com/skypies/flightdb/fgae"
	"github.com/skypies/flightdb/ref"
)

var uriStem = "/fdb/restrictors"

func init() {
	http.HandleFunc("/fdb/restrictors/list", UIOptionsHandler(EnsureLoggedIn(rListHandler)))

	http.HandleFunc("/fdb/restrictors/grs/new", UIOptionsHandler(EnsureLoggedIn(rGrsNewHandler)))
	http.HandleFunc("/fdb/restrictors/grs/delete",UIOptionsHandler(EnsureLoggedIn(rGrsDeleteHandler)))
	http.HandleFunc("/fdb/restrictors/grs/edit", UIOptionsHandler(EnsureLoggedIn(rGrsEditHandler)))
	http.HandleFunc("/fdb/restrictors/grs/view", UIOptionsHandler(EnsureLoggedIn(rGrsViewHandler)))

	http.HandleFunc("/fdb/restrictors/gr/new", UIOptionsHandler(EnsureLoggedIn(rGrNewHandler)))
	http.HandleFunc("/fdb/restrictors/gr/edit", UIOptionsHandler(EnsureLoggedIn(rGrEditHandler)))
	http.HandleFunc("/fdb/restrictors/gr/delete", UIOptionsHandler(EnsureLoggedIn(rGrDeleteHandler)))
}

// {{{ rListHandler

func rListHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	opt,_ := GetUIOptions(ctx)
	db := fgae.NewDB(ctx)

	if rsets,err := db.LookupRestrictorSets(opt.UserEmail); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		params := map[string]interface{}{
			"UIOptions": opt,
			"URIStem": uriStem,
			"RestrictorSets": rsets,
		}
		if err := templates.ExecuteTemplate(w, "restrictors-list", params); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// }}}

// {{{ rGrsNewHandler

// rGrsnewHandler    - () conjure empty grs, render [grs-edit]
func rGrsNewHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	opt,_ := GetUIOptions(ctx)	

	params := map[string]interface{}{
		"Title": "New Restrictor Set",
		"URIStem": uriStem,
		"UIOptions": opt,
		"GRS": fdb.GeoRestrictorSet{},
	}

	if err := templates.ExecuteTemplate(w, "restrictors-grs-edit", params); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// }}}
// {{{ rGrsEditHandler

// rGrsEditHandler   - (key [,form]) load it; if form, edit&save, chain to ./list; else render [grs-edit]
func rGrsEditHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	opt,_ := GetUIOptions(ctx)	
	db := fgae.NewDB(ctx)

	grs := fdb.GeoRestrictorSet{User:opt.UserEmail}
	maybeLoadGRSDSKey(ctx, r, &grs)	// If we have a key, load it up to populate the grs

	// If no form data, display the grs in an edit form
	if r.FormValue("name") == "" {
		params := map[string]interface{}{
			"Title": "Edit Restrictor Set",
			"URIStem": uriStem,
			"UIOptions": opt,
			"GRS": grs,
		}
		if err := templates.ExecuteTemplate(w, "restrictors-grs-edit", params); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Parse out the grs from the form
	grs.Name = strings.ToLower(r.FormValue("name"))
	switch r.FormValue("combinationlogic") {
	case "any": grs.Logic = fdb.CombinationLogicAny
	case "all": grs.Logic = fdb.CombinationLogicAll
	}
	grs.Tags = widget.FormValueCommaSpaceSepStrings(r,"tags")
	sort.Strings(grs.Tags)
	for i,tag := range grs.Tags { grs.Tags[i] = strings.ToUpper(tag) }
	
	if err := db.PersistRestrictorSet(grs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// All done; go to the list handler now.
	http.Redirect(w,r, uriStem+"/list", http.StatusFound)
	//rListHandler(ctx,w,r)	
}

// }}}
// {{{ rGrsDeleteHandler

// rGrsDeleteHandler - (key) delete it, chain to ./list
func rGrsDeleteHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	db := fgae.NewDB(ctx)

	key := r.FormValue("grs_dskey")
	if key == "" {
		http.Error(w, "/grs/delete - no grs_dskey", http.StatusBadRequest)
		return
	}

	if err := db.DeleteRestrictorSet(key); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// All done; go to the list handler now.
	rListHandler(ctx,w,r)	
}

// }}}
// {{{ rGrsViewHandler

// rGrsViewHandler- (key [,idspec]) load it, go to [grs-mapview]
func rGrsViewHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	grs,err := formValueDSKey(ctx, r)
	if err != nil {
		http.Error(w, fmt.Sprintf("rGrViewHandler, err: %v", err), http.StatusBadRequest)
		return
	}
	
	editUrl := fmt.Sprintf("%s/grs/edit?grs_dskey=%s", uriStem, grs.DSKey)
	legend := fmt.Sprintf("[<a target=\"_blank\" href=\"%s\">edit</a>]\n\n%s", editUrl, grs.String())
	legend = strings.Replace(legend, "\n", "<br/>", -1)
	
	ms := NewMapShapes()

	for _,gr := range grs.R {
		for _,line := range gr.ToLines() {
			ms.AddLine(MapLine{Start:line.From, End:line.To, Color:"#ff0808"})
		}
		for _,circle := range gr.ToCircles() {
			x := circle
			ms.AddCircle(MapCircle{C:&x, Color:"#ff0808"})
		}
	}	

	flights,err := formValueFlightsViaIdspecs(ctx, r)
	if err != nil {
		http.Error(w, fmt.Sprintf("rGrViewHandler, idspecs err: %v", err), http.StatusBadRequest)
		return
	}
	for _,f := range flights {
		outcome := f.GetIntersectableTrack().SatisfiesRestrictorSet(grs)
		log.Infof(ctx, fmt.Sprintf("** outcome:-\n%s\n\n%s\n", outcome, outcome.Debug()))
		ms.Points = append(ms.Points, flightToRestrictedMapPoints(f, grs)...)
		legend += fmt.Sprintf("<br/>Final outcome: <b>satsfies=%v</b><br/>", outcome.Satisfies(grs.Logic))
		for i,o := range outcome.Outcomes {
			legend += fmt.Sprintf("+(%02d) satisfied=%v [%d,%d]<br/>", i, o.Satisfies, o.I, o.J)
		}
	}

	p1,p2 := geo.FormValueLatlong(r, "pos1"),geo.FormValueLatlong(r, "pos2")
	if !p1.IsNil() && !p2.IsNil() {
		ms.AddLine(MapLine{Start:p1, End:p2, Color:"#0808ff"})
/*		log.Infof("Hand-intersecting the problem case\n")
		for _,gr := range grs.R {
			
		}*/
	}

	params := map[string]interface{}{
		"Legend": legend,
		"Waypoints": WaypointMapVar(sfo.KFixes),
		"Shapes": ms,
	}
	getGoogleMapsParams(r, params)
	params["MapsAPIKey"] = "AIzaSyDZd-t_YjSNGKmtmh6eR4Bt6eRR_w72b18"
	
	if err := templates.ExecuteTemplate(w, "map", params); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// }}}

// {{{ rGrNewHandler

func rGrNewHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	opt,_ := GetUIOptions(ctx)	

	grs,err := formValueDSKey(ctx, r)
	if err != nil {
		http.Error(w, fmt.Sprintf("rGrNewHandler, err: %v", err), http.StatusBadRequest)
		return
	}
	
	params := map[string]interface{}{
		"URIStem": uriStem,
		"UIOptions": opt,
		"Waypoints": sfo.ListWaypoints(),
		"GRS": grs,
		"GRIndex":len(grs.R),
	}

	fdb.BlankGeoRestrictorIntoParams(params)
	
	if err := templates.ExecuteTemplate(w, "restrictors-gr-edit", params); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// }}}
// {{{ rGrEditHandler

// rGrEditHandler    - (key,index [,form]) if form, edit&save, chain to ./grs/edit; else render [gr-edit]
func rGrEditHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	opt,_ := GetUIOptions(ctx)	
	db := fgae.NewDB(ctx)

	grs,err := formValueDSKey(ctx, r)
	if err != nil {
		http.Error(w, fmt.Sprintf("rGrNewHandler, err: %v", err), http.StatusBadRequest)
		return
	}

	grIndex := int(widget.FormValueInt64(r, "gr_index"))
	if grIndex > len(grs.R) {
		http.Error(w, fmt.Sprintf("rGrEditHandler, index too big (%d>%d)", grIndex,len(grs.R)),
			http.StatusBadRequest)
		return
	}

	// No form - fetch & display
	if r.FormValue("gr_type") == "" {
		if grIndex >= len(grs.R) {
			http.Error(w, fmt.Sprintf("rGrEditHandler, index too big (%d>=%d)", grIndex,len(grs.R)),
				http.StatusBadRequest)
			return
		}
		params := map[string]interface{}{
			"URIStem": uriStem,
			"UIOptions": opt,
			"Waypoints": sfo.ListWaypoints(),
			"GRS": grs,
			"GR": grs.R[grIndex],
			"GRIndex":grIndex,
		}
		fdb.GeoRestrictorIntoParams(grs.R[grIndex], params)
		//http.Error(w, fmt.Sprintf("WTF\n%#v\n", params), http.StatusInternalServerError)
		
		if err := templates.ExecuteTemplate(w, "restrictors-gr-edit", params); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	gr,err := fdb.FormValueGeoRestrictor(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("rGrEditHandler, parse err: %v", err), http.StatusBadRequest)
		return
	}

	if grIndex == len(grs.R) {
		grs.R = append(grs.R, gr)
	} else {
		grs.R[grIndex] = gr
	}

	if err := db.PersistRestrictorSet(grs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w,r, uriStem+"/grs/edit?grs_dskey="+grs.DSKey, http.StatusFound)
}

// }}}
// {{{ rGrDeleteHandler

// rGrDeleteHandler  - (key,index)
func rGrDeleteHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	db := fgae.NewDB(ctx)

	grs,err := formValueDSKey(ctx, r)
	if err != nil {
		http.Error(w, fmt.Sprintf("rGrNewHandler, err: %v", err), http.StatusBadRequest)
		return
	}

	grIndex := int(widget.FormValueInt64(r, "gr_index"))
	if grIndex > len(grs.R) {
		http.Error(w, fmt.Sprintf("rGrDeleteHandler, index too big (%d>%d)", grIndex,len(grs.R)),
			http.StatusBadRequest)
		return
	}

	grs.R = append(grs.R[:grIndex], grs.R[grIndex+1:]...) // chop out grIndex

	if err := db.PersistRestrictorSet(grs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w,r, uriStem+"/grs/edit?grs_dskey="+grs.DSKey, http.StatusFound)
}

// }}}

// {{{ rDebHandler

func rDebHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	opt,_ := GetUIOptions(ctx)

	r.ParseForm()
	str := r.Form
	
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(fmt.Sprintf("OK!\n\nOptions:\n%#v\n\nForm:\n%s\n", opt, str)))
}

// }}}

// {{{ maybeLoadGRSDSKey

// Can't find a nice way to centralize this, as it needs fdb & fgae :/
func maybeLoadGRSDSKey(ctx context.Context, r *http.Request, grs *fdb.GeoRestrictorSet) (error) {
	db := fgae.NewDB(ctx)

	// TODO: move to grs_dskey or something
	if dskey := r.FormValue("grs_dskey"); dskey == "" {
		return nil
	} else	if grsIn,err := db.LoadRestrictorSet(dskey); err != nil {
		return err
	} else {
		*grs = grsIn
		return nil
	}
}

// }}}
// {{{ formValueDSKey

func formValueDSKey(ctx context.Context, r *http.Request) (fdb.GeoRestrictorSet, error) {
	opt,_ := GetUIOptions(ctx)	
	grs := fdb.GeoRestrictorSet{User:opt.UserEmail}

	err := maybeLoadGRSDSKey(ctx, r, &grs)
	return grs,err
}

// }}}
// {{{ formValueFlightsViaIdspecs

func formValueFlightsViaIdspecs(ctx context.Context, r *http.Request) ([]*fdb.Flight, error) {
	opt,_ := GetUIOptions(ctx)	
	db := fgae.NewDB(ctx)

	// This whole Airframe cache thing should be automatic, and upstream from here.
	airframes := ref.NewAirframeCache(ctx)

	idspecs,_ := opt.IdSpecs()
	
	flights := []*fdb.Flight{}
	for _,idspec := range idspecs {
			f,err := db.LookupMostRecent(db.NewQuery().ByIdSpec(idspec))
			if err != nil {
				return flights,err
			}
		if af := airframes.Get(f.IcaoId); af != nil { f.OverlayAirframe(*af) }
		flights = append(flights, f)
	}

	return flights,nil
}

// }}}

// {{{ flightToRestrictedMapPoints

func flightToRestrictedMapPoints(f *fdb.Flight, grs fdb.GeoRestrictorSet) []MapPoint {
	if tName, t := f.PreferredTrack([]string{"FOIA", "ADSB", "MLAT"}); tName == "" {
		return nil
	} else {
		t.PostProcess()  // Move upstream ?
		return TrackToMapPoints(&t, "", "", ByADSBReceiver)
	}
}

// }}}
			
// {{{ notes

// To handle a new kind of restriction:
//  * Declare a struct and the required methods to geo/restrictions.go
//  * Add a bunch of form fields to ui/templates/restrictors-gr-edit-form.html; make sure to ...
//     * tag the table with id="shortname" so it can be hidden/exposed
//     * add an <option> to the <select>
//     * use template params to initialize the form elements
//  * Add a corresponding stanza to GeoRestrictorAsValues(), expressing as key/val pairs
//  * Add a corresponding stanza to FormValueGeoRestrictor(), parsing key/val pairs
//  * Add a bunch of good tests to intersectabletrack_test.go

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
