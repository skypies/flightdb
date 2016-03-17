package report

import(
	"fmt"
	"strings"
	"time"
	
	"github.com/skypies/util/widget"

	fdb "github.com/skypies/flightdb2"
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

	if k,exists := f.ForeignKeys["v1"]; exists {
		v1host := "https://stop.jetnoise.net"
		//v1url := v1host + "/fdb/lookup?map=1&rep=" + r.Name + "&id=" + k
		// addFrag("v1",   v1url)
		addFrag("map",  v1host + "/fdb/track2?idspec="   +k+"&"+reportArgs)
		addFrag("vec",  v1host + "/fdb/trackset2?idspec="+k+"&"+reportArgs)
		addFrag("side", v1host + "/fdb/approach2?idspec="+k+"&"+dateArgs)
	} else {
		addFrag("v2",            "/fdb/tracks?idspec="+f.IdSpecString()+"&"+reportArgs)
	}

	if f.HasTrack("ADSB") {
		fdbhost := "https://ui-dot-serfr0-fdb.appspot.com"
		addFrag("NewDB", fdbhost + f.TrackUrl()+"&"+reportArgs)
	}
	
	return "[" + strings.Join(frags, ",") + "]"
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
