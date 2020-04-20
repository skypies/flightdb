package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"
	
	"golang.org/x/net/context"

	"github.com/skypies/adsb"
	"github.com/skypies/geo"
	"github.com/skypies/geo/sfo"
	"github.com/skypies/pi/airspace"

	"github.com/skypies/util/gcp/ds"
	"github.com/skypies/util/gcp/singleton"
	hw "github.com/skypies/util/handlerware"

	"github.com/skypies/flightdb/aex"
	"github.com/skypies/flightdb/config"
	"github.com/skypies/flightdb/fr24"
	"github.com/skypies/flightdb/ref"
)

var(
	kMaxStaleDuration = time.Second * 30
	kMaxStaleScheduleDuration = time.Minute * 20
	airspaceSingletonName = "consolidated-airspace"
	swimAirspaceSingletonName = "swim-airspace"
)

// {{{ RealtimeAirspaceHandler

// RealtimeAirspaceHandler is a context handler that renders a google
// maps page, which will start polling for realtime flight positions.
// Requires the "map-poller" template and friends. If passed the
// `json=1` argument, will return a JSON rendering of the current
// state of the sky. It expects to be able to extract templates from
// the context.
//
// /?json=1&box_sw_lat=36.1&box_sw_long=-122.2&box_ne_lat=37.1&box_ne_long=-121.5
//  &comp=1      (for fr24)
//  &icao=AF1212 (for limiting heatmaps to one aircraft)
func RealtimeAirspaceHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if r.FormValue("json") != "" {
		jsonOutputHandler(w,r)
		return
	}

	// FIXME: get scheme://host from the request, *or* just go relative
	url := "/?json=1"
	//url := "http://fdb.serfr1.org/?json=1"
	// Propagate certain URL args to the JSON handler
	for _,key := range []string{"comp","fr24","fa","swim"} {
		if r.FormValue(key) != "" {
			url += fmt.Sprintf("&%s=%s", key, r.FormValue(key))
		}
	}

	var params = map[string]interface{}{
		"MapsAPIKey": config.Get("googlemaps.apikey"),
		"Center": sfo.KFixes["YADUT"],
		"Zoom": 9,
		"URLToPoll": url,
	}

	templates := hw.GetTemplates(ctx)
	if err := templates.ExecuteTemplate(w, "map-poller", params); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// }}}
// {{{ jsonOutputHandler

func jsonOutputHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	box := geo.FormValueLatlongBox(r, "box")
	src := r.FormValue("src")
	
	if box.IsNil() { box = sfo.KAirports["KSFO"].Box(250,250) }
	
	as := airspace.NewAirspace()
	var err error

	if src == "" || src == "fdb" {
		as,err = getAirspaceForDisplay(ctx, box)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

	} else if src == "aex" {
		addAExToAirspace(ctx, box, &as)

	} else {
		http.Error(w, fmt.Sprintf("airspace source '%s' not in {fdb,aex}"), http.StatusBadRequest)
		return
	}	

	if r.FormValue("comp") != "" {
		addAExToAirspace(ctx, box, &as)
		if r.FormValue("fr24") != "" { addFr24ToAirspace(ctx, &as) }
		if r.FormValue("swim") != "" { addSwimToAirspace(ctx, box, &as) }
		//if r.FormValue("fa") != "" { faToAirspace(ctx, &as) }

		// Weed out stale stuff (mostly from fa)
		for k,_ := range as.Aircraft {
			age := time.Since(as.Aircraft[k].Msg.GeneratedTimestampUTC)
			if age > kMaxStaleDuration * 2 {
				delete(as.Aircraft, k)
			}
		}
	}

	fmt.Printf("%s\n", Airspace2String(as))
	
	// Bodge, to let goapp serve'd things call the deployed version of this URL
	w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8080")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers",
		"Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token")
	w.Header().Set("Access-Control-Allow-Credentials", "true")

	data,err := json.MarshalIndent(as, "", "  ")
	if err != nil {
		http.Error(w, fmt.Sprintf("jOH/Marshal error: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// }}}

// {{{ Airspace2String

func Airspace2String(a airspace.Airspace) string {
	str := ""

	keys := []string{}
	for k,_ := range a.Aircraft { keys = append(keys, string(k)) }
	sort.Strings(keys)
	
	for _,k := range keys {
		ac := a.Aircraft[adsb.IcaoId(k)]
		str += fmt.Sprintf(" %8.8s/%-8.8s/%-6.6s (%s last:%6.1fs at %s/%-13.13s, %5d msgs) %5df, %3dk %s\n",
			ac.Msg.Callsign, ac.Msg.Icao24, ac.Registration,
			ac.Msg.DataSystem(),
			time.Since(ac.Msg.GeneratedTimestampUTC).Seconds(),
			ac.Source, ac.Msg.ReceiverName,
			ac.NumMessagesSeen,
			ac.Msg.Altitude, ac.Msg.GroundSpeed, ac.Msg.Position)
	}
	return str
}

// }}}


// {{{ backfillReferenceData

func backfillReferenceData(ctx context.Context, sp singleton.SingletonProvider, as *airspace.Airspace) {
	airframes,_ := ref.LoadAirframeCache(ctx, sp)
	schedules,_ := ref.LoadScheduleCache(ctx, sp)
	
	for k,aircraft := range as.Aircraft {
		// Need the bare Icao for lookups
		unprefixedK := strings.TrimPrefix(string(k), "QQ")
		unprefixedK = strings.TrimPrefix(unprefixedK, "EE")
		unprefixedK = strings.TrimPrefix(unprefixedK, "FF")

		if af := airframes.Get(unprefixedK); af != nil {
			// Update entry in map to include the airframe data we just found
			aircraft.Airframe = *af
			as.Aircraft[k] = aircraft
		}

		if schedules != nil && time.Since(schedules.LastUpdated) < kMaxStaleScheduleDuration {
			if fs := schedules.Get(unprefixedK); fs != nil {
				aircraft.Schedule = fs.Identity.Schedule
				as.Aircraft[k] = aircraft
			}
		}
	}
}

// }}}
// {{{ getAirspaceForDisplay

// We tart it up with airframe and schedule data, trim out stale entries, and trim to fit box
func getAirspaceForDisplay(ctx context.Context, bbox geo.LatlongBox) (airspace.Airspace, error) {
	a := airspace.NewAirspace()

	p,err := ds.NewCloudDSProvider(ctx, GoogleCloudProjectId)
	if err != nil {
		return a, fmt.Errorf("gAFD/NewCloudDSProvider error:%v", err)
	}
	sp := singleton.NewProvider(p)


	// Must wait until GAE can securely call into GCE within the same project
/*
	dialer := func(network, addr string, timeout time.Duration) (net.Conn, error) {
		return socket.DialTimeout(ctx, network, addr, timeout)
	}
	if err := a.JustAircraftFromMemcacheServer(ctx, dialer); err != nil {
		return a, fmt.Errorf("gAFD/FromMemcache error:%v", err)
	}
*/

	if err := sp.ReadSingleton(ctx, airspaceSingletonName, nil, &a); err != nil {
		return a, fmt.Errorf("gAFD/ReadSingleton error:%v", err)
	}
	//if err := a.JustAircraftFromMemcache(ctx); err != nil {
	//	return a, fmt.Errorf("gAFD/FromMemcache error:%v", err)
	//}
	
	for k,aircraft := range a.Aircraft {
		age := time.Since(a.Aircraft[k].Msg.GeneratedTimestampUTC)
		if age > kMaxStaleDuration {
			delete(a.Aircraft, k)
			continue
		}
		if !bbox.SW.IsNil() && !bbox.Contains(aircraft.Msg.Position) {
			delete(a.Aircraft, k)
			continue
		}
	}

	backfillReferenceData(ctx, sp, &a)
	
	return a,nil
}

// }}}

// {{{ addFr24ToAirspace

func addFr24ToAirspace(ctx context.Context, as *airspace.Airspace) {
	fr,_ := fr24.NewFr24(&http.Client{})

	if asFr24,err := fr.FetchAirspace(sfo.KAirports["KSFO"].Box(250,250)); err != nil {
		return
	} else {
		for k,ad := range asFr24.Aircraft {
			// FIXME: This whole thing is a crock. Track down usage of fr24/icaoids and rewrite all of it
			newk := string(k)
			newk = "EE" + strings.TrimPrefix(newk, "EE") // Remove (if present), then add
			ad.Airframe.Icao24 = newk
			ad.Source = "swim" // the airspace.MaybeUpdate() route doesn't have a way to specify this
			as.Aircraft[adsb.IcaoId(newk)] = ad
		}
	}
}

// }}}
// {{{ addFAToAirspace

func addFAToAirspace() {
/*	
var(
	TestAPIKey = "foo"
	TestAPIUsername = "bar"
)

// Overlays the flightaware 'Search' results into the airspace
func faToAirspace(c context.Context, as *airspace.Airspace) string {
	str := ""
	
	myFa := fa.Flightaware{APIKey:TestAPIKey, APIUsername:TestAPIUsername, Client:urlfetch.Client(c)}
	myFa.Init()
}
*/
}

// }}}
// {{{ addAExAirspace

func addAExToAirspace(ctx context.Context, box geo.LatlongBox, as *airspace.Airspace) {	
	if asAEx,err := aex.FetchAirspace(&http.Client{}, box); err != nil {
		return
	} else {

		p,err := ds.NewCloudDSProvider(ctx, GoogleCloudProjectId)
		if err != nil { return }
		sp := singleton.NewProvider(p)

		backfillReferenceData(ctx, sp, asAEx)

		for k,ad := range asAEx.Aircraft {
			newk := string(k)
			newk = "QQ" + strings.TrimPrefix(newk, "QQ") // Remove (if present), then re-add
			ad.Airframe.Icao24 = newk
			as.Aircraft[adsb.IcaoId(newk)] = ad
		}
	}
}

// }}}
// {{{ addSwimToAirspace

func addSwimToAirspace(ctx context.Context, box geo.LatlongBox, as *airspace.Airspace) {	
	asSwim := airspace.NewAirspace()
	
	p,err := ds.NewCloudDSProvider(ctx, GoogleCloudProjectId)
	if err != nil { return }
	sp := singleton.NewProvider(p)

	if err := sp.ReadSingleton(ctx, swimAirspaceSingletonName, nil, &asSwim); err != nil {
		log.Printf("gAFD/ReadSingleton error:%v", err)
		return
	}
	
	for k,_ := range asSwim.Aircraft {
		age := time.Since(asSwim.Aircraft[k].Msg.GeneratedTimestampUTC)
		if age > kMaxStaleDuration {
			delete(asSwim.Aircraft, k)
			continue
		}
		/*
		if !bbox.SW.IsNil() && !bbox.Contains(aircraft.Msg.Position) {
			delete(a.Aircraft, k)
			continue
		}
*/
	}

	// backfillReferenceData(ctx, sp, asAEx)

	for k,ad := range asSwim.Aircraft {
		newk := string(k)
		newk = "WW" + strings.TrimPrefix(newk, "WW") // Remove (if present), then re-add
		ad.Airframe.Icao24 = newk
		as.Aircraft[adsb.IcaoId(newk)] = ad
	}
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
