package analysis

import (
	"fmt"
	"math"

	fdb "github.com/skypies/flightdb2"
	"github.com/skypies/flightdb2/report"
)

func init() {
	report.HandleReport("levelflight", LevelFlightReporter,
		"Level flight across {region} with angle <= {tol}")
}

func LevelFlightReporter(r *report.Report, f *fdb.Flight, tis []fdb.TrackIntersection) (report.FlightReportOutcome, error){
	ti,err := r.GetFirstAreaIntersection(tis)
	if err != nil {
		return report.RejectedByReport, err
	}

	r.I["[C] Flights passing through region"]++

	// See if any trackpoints inside the intersection lie outside the tolerance
	for _,ti := range tis {
		t := f.Tracks[ti.TrackName]
		t.PostProcess()

		for i:=ti.I; i<=ti.J; i++ {
			if math.Abs((*t)[i].AngleOfInclination) > r.AltitudeTolerance {
				r.I[fmt.Sprintf("[D] Flights not level (|angle| > %.1f deg)</b>", r.AltitudeTolerance)]++
				return report.RejectedByReport,nil
			}
		}
	}
	
	r.I[fmt.Sprintf("[D] <b>Flights with level flight (|angle| <= %.1f deg)</b>",
		r.AltitudeTolerance)]++
	
	row := []string{
		r.Links(f),
		"<code>" + f.IdentString() + "</code>",
	}
	row = append(row, ti.RowHTML()...)

	r.AddRow(&row, &row)
	
	return report.Accepted, nil
}
