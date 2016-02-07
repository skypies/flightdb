package report

import(
	"fmt"
	"strings"
	fdb "github.com/skypies/flightdb2"
)

func init() {
	HandleReport(SimpleReporter, ".simple", "List all matching flights")
}

func SimpleReporter(r *Report, f *fdb.Flight) (bool, error) {
	r.I["[A] Total"]++

	// If a region was specified, only match flights that pass through it
	regions := r.Options.ListRegions()
	if len(regions) > 0 {
		intersection,_ := f.AnyTrack().IntersectWith(regions[0], regions[0].String())
		if intersection == nil {
			tStr := strings.Join(f.ListTracks(),",")
			r.I["[B] Did not intersect region ("+tStr+")"]++
			return false,nil
		} else {
			r.I["[B] Intersected region"]++
		}
	}

	row := []string{
		r.Links(f),
		"<code>" + f.IdentString() + "</code>",
		fmt.Sprintf("[%s]", strings.Join(f.TagList(), ",")),
	}
	r.AddRow(&row, &row)
	
	return true, nil
}
