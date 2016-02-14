package analysis

import (
	"fmt"

	"github.com/skypies/util/histogram"

	fdb "github.com/skypies/flightdb2"
	"github.com/skypies/flightdb2/report"
)

func init() {
	// Stacking report: for flights that pass into area of interest, count them in altitude bands
	report.HandleReport("closestpoint", ClosestApproachReporter, "Closest point to {ref-point}")
}

func ClosestApproachReporter(r *report.Report, f *fdb.Flight, tis []fdb.TrackIntersection) (bool, error) {

	var t *fdb.Track
	for _,tName := range []string{"ADSB", "FA", "fr24"} {
		if f.HasTrack(tName) {
			t = f.Tracks[tName]
			break
		}
	}

	if t == nil { return false, nil } // Flight had no track data !
	if r.ReferencePoint.IsNil() { return false, nil } // No ref pt
	
	iClosest := t.ClosestTo(r.ReferencePoint)
	if iClosest < 0 { return false, nil } // track was in fact empty ?

	dist := (*t)[iClosest].DistKM(r.ReferencePoint)
	summaryStr := fmt.Sprintf("* Closest to %s\n* <b>%.2f</b> KM away\n", r.ReferencePoint, dist)
	(*t)[iClosest].AnalysisMapIcon = "red-large"
	(*t)[iClosest].AnalysisAnnotation += summaryStr

	r.I[fmt.Sprintf("[C] <b>Flights compared against ref pt %s </b>", r.ReferencePoint)]++
	r.S["[Z] Stats: <b>distance from ref pt in meters</b>"] = ""
	r.H.Add(histogram.ScalarVal(dist * 1000.0))
	
	row := []string{
		r.Links(f),
		"<code>" + f.IdentString() + "</code>",
		"<b>TrackIndex</b>", fmt.Sprintf("%d", iClosest),
		"<b>Dist(KM)</b>", fmt.Sprintf("%.2f", dist),
	}

	r.AddRow(&row, &row)

	return true, nil
}
