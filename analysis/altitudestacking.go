package analysis

import (
	"fmt"

	fdb "github.com/skypies/flightdb2"
	"github.com/skypies/flightdb2/report"
)

func init() {
	// Stacking report: for flights that pass into area of interest, count them in altitude bands
	report.HandleReport(AltitudeBandsReporter, "altitudebands", "Altitude Bands across {region}")
}

func alt2bkt(f float64) string {
	g := float64(int((f+500)/1000.0))  // Round to nearest thousand: 11499 -> 11, 11501 -> 12	
	return fmt.Sprintf("%05.0f-%05.0f", g*1000-500, g*1000+500)
}

func AltitudeBandsReporter(r *report.Report, f *fdb.Flight) (bool, error) {
	r.I["[A] Total Flights Examined"]++
	if f == nil { return false,nil }
	if f.Tracks == nil { return false, fmt.Errorf("f.Tracks == nil !") }

	regions := r.Options.ListRegions()
	if len(regions) == 0 {
		r.I["_baddata"]++
		return false,nil
	}
	reg := regions[0]
	
	if false && !f.IsScheduled() {
		// We only want scheduled flights
		r.I["_unscheduled"]++
		return false,nil
	} 

	t := f.AnyTrack()
	if r.Options.TrackDataSource == "FA" { // Ignore requests for ADSB; its not comprehensive
		if f.HasTrack("FA") {
			t = *f.Tracks["FA"]
		} else {
			r.I["_notrack"]++
			return false,nil // No track data
		}
	}
	
	// Brittle: if a flight passes through twice, we only see the first.
	ti,_ := t.IntersectWith(reg, reg.String())
	if ti == nil {
		r.I["[B] Flights did not pass through region"]++
		return false, nil
	}

	avgAlt := ti.Start.Altitude + (ti.End.Altitude - ti.Start.Altitude) / 2.0
	if avgAlt < 8000.0 { return false, nil } // Too low to be interesting
	bkt := alt2bkt(avgAlt)
		
	r.I["[B] Flights passing through region, above 8000'"]++
	r.I[fmt.Sprintf("[C] %s ",bkt)]++
	
	row := []string{
		r.Links(f),
		"<code>" + f.IdentString() + "</code>",
	}
	row = append(row, ti.RowHTML()...)

	r.AddRow(&row, &row)
	
	return true, nil
}
