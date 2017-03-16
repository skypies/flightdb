package report

import(
	"fmt"
	"strings"
	"time"
	
	"github.com/skypies/util/widget"

	fdb "github.com/skypies/flightdb"
)

// A few helper functions to make writing report routines a bit less cut-n-pasto

func (r *Report)Links(f *fdb.Flight) string {
	frags := []string{}
	
	addFrag := func(k,u string){
		frags = append(frags, fmt.Sprintf("<a target=\"_blank\" href=\"%s\">%s</a>", u, k))
	}
	
	s,e := f.Times()
	dateArgs := widget.DateRangeToCGIArgs(s.Add(-24*time.Hour),e.Add(24*time.Hour))
	reportArgs := r.ToCGIArgs()

	bestIdSpec := f.IdSpecString()
	
	if k,exists := f.ForeignKeys["v1"]; exists {
		bestIdSpec = k
		v1host := "https://stop.jetnoise.net"
		//v1url := v1host + "/fdb/lookup?map=1&rep=" + r.Name + "&id=" + k
		// addFrag("v1",   v1url)
		addFrag("map",  v1host + "/fdb/track2?idspec="   +k+"&"+reportArgs)
		addFrag("vec",  v1host + "/fdb/trackset2?idspec="+k+"&"+reportArgs)
		addFrag("side", v1host + "/fdb/descent2?idspec="+k+"&"+dateArgs+"&length=100")

		if f.HasTrack("ADSB") || f.HasTrack("MLAT") {
			fdbhost := "https://ui-dot-serfr0-fdb.appspot.com"
			addFrag("DBv2", fdbhost + f.TrackUrl()+"&"+reportArgs)
		}

	} else {
		addFrag("map",     "/fdb/tracks?idspec="+f.IdSpecString()+"&"+reportArgs)
		addFrag("vec",     "/fdb/trackset?idspec="+f.IdSpecString()+"&"+reportArgs)


		
		sideUrl := "/fdb/sideview?idspec="+f.IdSpecString()
		if f.HasTag("NORCAL:") {
			addFrag("dep", sideUrl + "&departing=K"+f.Origin)
		} else if f.HasTag(":NORCAL") {
			addFrag("arr", sideUrl + "&arriving=K"+f.Destination)
		}
	}

	tickbox := "<input type=\"checkbox\" name=\"idspec\" checked=\"yes\" value=\""+bestIdSpec+"\"/>"
	
	return tickbox + " [" + strings.Join(frags, ",") + "]"
}

func (r *Report)GetFirstIntersectingTrackpoint(t []fdb.TrackIntersection) *fdb.Trackpoint {
	for _,intersection := range t {
		return &intersection.Start
	}
	return nil
}

func (r *Report)GetFirstAreaIntersection(t []fdb.TrackIntersection) (*fdb.TrackIntersection,error) {
	if len(t) == 0 {
		return nil, fmt.Errorf("no area intersection (list empty)")
	}

	ti := fdb.TrackIntersection{}
	for _,intersection := range t {
		if ! intersection.IsPointIntersection() {
			ti = intersection
		}
	}
	if ti.I == 0 {
		return nil, fmt.Errorf("no area intersection")
	}

	return &ti, nil
}
