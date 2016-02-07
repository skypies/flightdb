package analysis

import (
	"fmt"
	"math"

	fdb "github.com/skypies/flightdb2"
	"github.com/skypies/flightdb2/report"
)

func init() {
	report.HandleReport(LevelFlightReporter, "levelflight", "Level flight across {region}")
}

func LevelFlightReporter(r *report.Report, f *fdb.Flight) (bool, error) {
	r.I["[A] Total Flights Examined"]++
	if f == nil { return false,nil }
	if f.Tracks == nil { return false, fmt.Errorf("f.Tracks == nil !") }

	regions := r.Options.ListRegions()
	if len(regions) == 0 {
		r.I["_baddata"]++
		return false,nil
	}
	reg := regions[0]

	t := f.AnyTrack()
	if r.Options.TrackDataSource == "FA" { // Ignore requests for ADSB; its not comprehensive
		if f.HasTrack("FA") {
			t = *f.Tracks["FA"]
		} else {
			r.I["_notrack"]++
			return false,nil // No track data
		}
	}

	ti,_ := t.IntersectWith(reg, reg.String())
	if ti == nil {
		r.I["[B] Flights did not pass through region"]++
		return false, nil
	}

	if ti.Start.Altitude > 8000.0 {
		r.I["[B] Flights passed through, but too high (>8000 ft)"]++
		return false,nil
	}

	r.I["[B] Flights passing through region, below 8000 ft"]++

	altDelta := ti.End.Altitude - ti.Start.Altitude
	if math.Abs(altDelta) > r.Options.AltitudeTolerance {
		r.I[fmt.Sprintf("[B] Flights whose altitude changed by >%.0f", r.AltitudeTolerance)]++
		return false,nil
	}

	r.I[fmt.Sprintf("[B] Flights with level flight (delta<%.0f)", r.AltitudeTolerance)]++
	
	row := []string{
		r.Links(f),
		"<code>" + f.IdentString() + "</code>",
	}
	row = append(row, ti.RowHTML()...)

	r.AddRow(&row, &row)
	
	return true, nil
}
