package analysis

// Class B Analysis of flight tracks, for SFO.

import (
	"fmt"
	"strings"
	
	"github.com/skypies/geo"
	"github.com/skypies/geo/altitude"
	"github.com/skypies/geo/sfo"

	fdb "github.com/skypies/flightdb2"
	"github.com/skypies/flightdb2/report"
)

func init() {
	report.HandleReport(SFOClassBReporter, "sfoclassb", "SFO Class B excursions on SERFR")
}

// {{{ SFOClassBReporter

func SFOClassBReporter(r *report.Report, f *fdb.Flight) (bool, error) {
	excursion := false
	trackName := "ADSB" // Could be in Report.Options ?

	// Now store our results
	r.I["[A] Total Flights Examined"]++

	if f == nil { return false,nil }
	if f.Tracks == nil { return false, fmt.Errorf("f.Tracks == nil !") }
	
	// if f.Destination != "SFO" { return excursion }

	track := f.AnyTrack()
	if trackName != "" {
		if f.HasTrack(trackName) {
			track = *f.Tracks[trackName]
		} else {
			trackNames := strings.Join(f.ListTracks(), ",")
			r.I["[B] Skipped, no ADSB track ("+trackNames+")"]++
			return false, nil //fmt.Errorf("track '%s' not found (have %s)", trackName, trackNames)
		}
	}

	deepest := geo.TPClassBAnalysis{}
	
	for i,tp := range track {
		if tp.Altitude < 50 { continue } // Skip datapoints with null/empty altitude data
		result := geo.TPClassBAnalysis{}

		lookup := r.Archive.Lookup(tp.TimestampUTC)
		track[i].AnalysisAnnotation += fmt.Sprintf("* inHg: %v\n", lookup)
		if lookup.Raw == "" {
			track[i].AnalysisAnnotation += fmt.Sprintf("* BAD Metar, aborting\n")
			return false, fmt.Errorf("No metar, aborting")
		}
		
		iAlt := altitude.PressureAltitudeToIndicatedAltitude(tp.Altitude, lookup.AltimeterSettingInHg)
		track[i].AnalysisAnnotation += fmt.Sprintf("* pAlt: %.0f, iAlt: %.0f\n",
			tp.Altitude, iAlt)

		sfo.SFOClassBMap.ClassBPointAnalysis(tp.Latlong, tp.GroundSpeed, iAlt, &result)

		kLimit := 15.5
		if result.DistNM < kLimit {
			// track[i].AnalysisAnnotation += fmt.Sprintf("** ClassB disabled, <%.1fNM\n", kLimit)

		} else if result.IsViolation() {
			excursion = true  // We have at least one violating point in this trail
			track[i].AnalysisMapIcon = "red-large"
			track[i].AnalysisAnnotation += result.Reasoning

			if result.BelowBy > deepest.BelowBy { deepest = result }
		}
	}


	if !excursion {
		r.I["[B] No excursion"]++
		return false, nil
	}

	r.I["[B] Excursion found"]++

	//h.Add(histogram.ScalarVal(deepest.BelowBy))

	row := []string{
		r.Links(f),
		"<code>" + f.IdentString() + "</code>",
		//fmt.Sprintf("[%s]", strings.Join(f.TagList(), ",")),
		fmt.Sprintf("%.1f NM", deepest.DistNM),
		fmt.Sprintf("%.0f ft", deepest.BelowBy),
	}
	r.AddRow(&row, &row)
	
	return true, nil
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
