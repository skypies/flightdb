package analysis

import (
	"fmt"

	fdb "github.com/skypies/flightdb2"
	"github.com/skypies/flightdb2/report"
)

func init() {
	// Stacking report: for flights that pass into area of interest, count them in altitude bands
	report.HandleReport("altitudebands", AltitudeBandsReporter, "Altitude Bands across {region}")
}

func alt2bkt(f float64) string {
	g := float64(int((f+500)/1000.0))  // Round to nearest thousand: 11499 -> 11, 11501 -> 12	
	return fmt.Sprintf("%05.0f-%05.0f", g*1000-500, g*1000+500)
}

func AltitudeBandsReporter(r *report.Report, f *fdb.Flight, tis []fdb.TrackIntersection) (bool, error) {
	ti,err := r.GetFirstAreaIntersection(tis)
	if err != nil { return false, err }

	avgAlt := ti.Start.Altitude + (ti.End.Altitude - ti.Start.Altitude) / 2.0
	if avgAlt < 8000.0 {
		r.I["[C] Flights skipped, below 8000'"]++
		return false, nil
	} // Too low to be interesting

	bkt := alt2bkt(avgAlt)
		
	r.I["[C] <b>Flights passing through region, above 8000'</b>"]++
	r.I[fmt.Sprintf("[D] %s",bkt)]++
	
	row := []string{
		r.Links(f),
		"<code>" + f.IdentString() + "</code>",
	}
	row = append(row, ti.RowHTML()...)

	r.AddRow(&row, &row)
	
	return true, nil
}
