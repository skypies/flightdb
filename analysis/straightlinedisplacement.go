package analysis

import (
	"fmt"

	"github.com/skypies/geo/sfo"
	"github.com/skypies/util/histogram"

	fdb "github.com/skypies/flightdb"
	"github.com/skypies/flightdb/report"
)

func init() {
	report.HandleReport("straightlinedisplacement", StraightLineDisplacementReporter,
		"Lateral displacement from the line {refpoint} to {refpoint2}")
}

func StraightLineDisplacementReporter(r *report.Report, f *fdb.Flight, tis []fdb.TrackIntersection) (report.FlightReportOutcome, error){
	if r.ReferencePoint.IsNil() {
		return report.RejectedByReport, fmt.Errorf("report option {refpoint} not defined")
	} else if r.ReferencePoint2.IsNil() {
		return report.RejectedByReport, fmt.Errorf("report option {refpoint2} not defined")
	}

	wp1,wp2 := r.ReferencePoint.Name, r.ReferencePoint2.Name
	line := sfo.KFixes[wp1].LineTo(sfo.KFixes[wp2])
	
	for _,wp := range []string{wp1,wp2} {
		if !f.HasWaypoint(wp) {
			r.I[fmt.Sprintf("[C] Flights without %s", wp)]++
			return report.RejectedByReport, nil
		}
	}

	typePicked,track := f.PreferredTrack([]string{"ADSB", "MLAT", "FOIA"})
	if typePicked == "" {
		r.I["[D] Skipped, no ADSB or FOIA track avail"]++
		return report.RejectedByReport,nil
	}
	r.I[fmt.Sprintf("[D] <b>Accepted for displacement analysis %s-%s</b>", wp1, wp2)]++
	r.I[fmt.Sprintf("[Y] <b>ALL VALUES IN METRES</b>")]++

	clipped := fdb.Track(track.ClipTo(f.Waypoints[wp1], f.Waypoints[wp2]))
	sampled := clipped.SampleEveryDist(1.0, false)

	// Uses integers; so displacement in metres
	hist := histogram.Histogram{ValMin:0, ValMax:1000, NumBuckets:20}
	for _,tp := range sampled {
		distKM := line.ClosestDistance(tp.Latlong)
		distM := int(distKM * 1000.0)
		
		hist.Add(histogram.ScalarVal(distM))
		r.H.Add(histogram.ScalarVal(distM))
	}
	
	row := []string{
		r.Links(f),
		"<code>" + f.IdentString() + "</code>",
		"<pre>" + hist.String() + "</pre>",
	}

	r.AddRow(&row, &row)
	
	return report.Accepted, nil
}
