package analysis

// Class B Analysis of flight tracks, for SFO.

import (
	"fmt"
	
	"github.com/skypies/geo"
	"github.com/skypies/geo/altitude"
	"github.com/skypies/geo/sfo"
	"github.com/skypies/util/date"
	"github.com/skypies/util/histogram"

	fdb "github.com/skypies/flightdb"
	"github.com/skypies/flightdb/metar"
	"github.com/skypies/flightdb/report"
)

func init() {
	report.HandleReport("sfoclassb", SFOClassBReporter, "SFO Class B excursions (use EDDYY tag)")
	report.TrackSpec("sfoclassb", []string{"ADSB","FA", "FOIA"}) // That's all we'll accept
}


// {{{ ClassBForTrack

// If there is an excursion, return the deepest point of it; else return nil
func ClassBForTrack(r *report.Report, track fdb.Track) (*geo.TPClassBAnalysis,error) {
	deepest := geo.TPClassBAnalysis{}
	
	for i,tp := range track {
		if tp.Altitude < 50 { continue } // Skip datapoints with null/empty altitude data

		iAlt := tp.Altitude
		inchesHg := 29.9213
		
		if tp.DataSource == "RG-FOIA" {
			// iAlt is already pressure corrected
			track[i].AnalysisAnnotation += fmt.Sprintf("* FOIA data, no altitude correction performed\n")
			
		} else {
			lookup := r.Archive.Lookup(tp.TimestampUTC)
			if lookup == nil {
				track[i].AnalysisAnnotation += fmt.Sprintf("* No Metar, using fake data (assume 29.9213)\n")
				lookup = &metar.Report{"asd", "fake", "KSFO", tp.TimestampUTC, 29.9213}
			}
			if lookup == nil || lookup.Raw == "" {
				track[i].AnalysisAnnotation += fmt.Sprintf("* No Metar, skipping\n")
				return nil, fmt.Errorf("No metar, aborting")
			}
			track[i].AnalysisAnnotation += fmt.Sprintf("* METAR: %v\n", lookup)
		
			inchesHg = lookup.AltimeterSettingInHg
			iAlt = altitude.PressureAltitudeToIndicatedAltitude(tp.Altitude, inchesHg)
			
			track[i].AnalysisAnnotation += fmt.Sprintf("* PressureAltitude: %.0f, "+
				"IndicatedAltitude: %.0f\n", tp.Altitude, iAlt)
		}
	
		result := geo.TPClassBAnalysis{
			I:i,
			InchesHg: inchesHg,
			IndicatedAltitude: iAlt,
		}

		sfo.SFOClassBMap.ClassBPointAnalysis(tp.Latlong, tp.GroundSpeed, iAlt,
			r.Options.AltitudeTolerance, &result)

		kLimit := 15.5
		if result.DistNM < kLimit {
			// track[i].AnalysisAnnotation += fmt.Sprintf("** ClassB disabled, <%.1fNM\n", kLimit)

		} else if result.IsViolation() {
			track[i].AnalysisDisplay = fdb.AnalysisDisplayHighlight
			track[i].AnalysisAnnotation += result.Reasoning

			if result.BelowBy > deepest.BelowBy { deepest = result }
		}
	}

	if deepest.BelowBy > 0.0 { return &deepest,nil }

	return nil,nil
}

// }}}

// {{{ SFOClassBReporter

func SFOClassBReporter(r *report.Report, f *fdb.Flight, tis []fdb.TrackIntersection) (report.FlightReportOutcome, error) {
	// Now store our results
	r.I["[C] Total Flights Examined"]++

	if f == nil { return report.RejectedByReport,nil }
	if f.Tracks == nil { return report.RejectedByReport, fmt.Errorf("f.Tracks == nil !") }
	
	if f.Destination != "SFO" {
		r.I["[D] dest != SFO"]++
		return report.RejectedByReport,nil
	}

	// For Class B, we're very picky about data sources.
	typePicked,track := f.PreferredTrack([]string{"ADSB", "FA", "FOIA"})
	if typePicked == "" {
		r.I["[D] Skipped, no ADSB, FA or FOIA track avail"]++
		return report.RejectedByReport,nil
	}

	r.I["[D] <b>Accepted for Class B Analysis</b>"]++

	deepest,err := ClassBForTrack(r, track)
	if err != nil {
		r.I["_classb_err_"+err.Error()]++
		return report.RejectedByReport, err
	}

	if deepest == nil {
		r.I["[E] No excursion found"]++
		return report.RejectedByReport, nil
	}

	if typePicked != "ADSB" && typePicked != "FOIA" {
		r.I["[E] Excursions skipped, datasource ("+track.LongSource()+")"]++
		return report.RejectedByReport,nil
	}
	//r.Hist.Add(histogram.ScalarVal(deepest.BelowBy))

// Also check for SKUNK+BOLDR
//	if !f.HasWaypoint("EDDYY") && !f.HasWaypoint("EPICK") {
//		r.I["[E] Excurions skipped, not on EPICK-EDDYY"]++
//		return report.RejectedByReport,nil
//	}

	r.I["[E] <b>Excursion found</b> via "+track.LongSource()]++

	r.H.Add(histogram.ScalarVal(deepest.BelowBy))

	tp := track[deepest.I] // The trackpoint we're using to highlight the excursion
	
	row := []string{
		r.Links(f),
		f.IcaoId,
		"<code>" + f.IdentString() + "</code>",
		fmt.Sprintf("%s", date.InPdt(tp.TimestampUTC).Format("01/02")),
		fmt.Sprintf("%s", date.InPdt(tp.TimestampUTC).Format("15:04:05 MST")),
		fmt.Sprintf("%f", tp.Latlong.Lat),
		fmt.Sprintf("%f", tp.Latlong.Long),
		fmt.Sprintf("%.1f NM", deepest.DistNM),
		fmt.Sprintf("%.0f", tp.Altitude),
		fmt.Sprintf("%.0f", deepest.IndicatedAltitude),
		fmt.Sprintf("(%.2f inHg)", deepest.InchesHg),
		fmt.Sprintf("%.0f", deepest.BelowBy),
	}
	r.AddRow(&row, &row)
	
	return report.Accepted, nil
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
