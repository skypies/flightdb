package analysis

import (
	"fmt"
	"math"

	fdb "github.com/skypies/flightdb2"
	"github.com/skypies/flightdb2/report"
)

func init() {
	report.HandleReport("levelflight", LevelFlightReporter, "Level flight across {region}")
	report.TrackSpec("levelflight", []string{"FA", "fr24"}) // *Not* ADSB; need <6000' data
}

func LevelFlightReporter(r *report.Report, f *fdb.Flight, tis []fdb.TrackIntersection) (report.FlightReportOutcome, error){
	ti,err := r.GetFirstAreaIntersection(tis)
	if err != nil { return report.RejectedByReport, err }
	
	if ti.Start.Altitude > 8000.0 {
		r.I["[C] Flights passed through, but too high (>8000 ft)"]++
		return report.RejectedByReport,nil
	}

	r.I["[C] Flights passing through region, below 8000 ft"]++

	altDelta := ti.End.Altitude - ti.Start.Altitude
	if math.Abs(altDelta) > r.Options.AltitudeTolerance {
		r.I[fmt.Sprintf("[D] Flights whose altitude changed by >%.0f", r.AltitudeTolerance)]++
		return report.RejectedByReport,nil
	}

	r.I[fmt.Sprintf("[D] <b>Flights with level flight (delta<%.0f)</b>", r.AltitudeTolerance)]++
	
	row := []string{
		r.Links(f),
		"<code>" + f.IdentString() + "</code>",
	}
	row = append(row, ti.RowHTML()...)

	r.AddRow(&row, &row)
	
	return report.Accepted, nil
}
