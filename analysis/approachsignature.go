package analysis

import (
	"fmt"
	"strings"

	"github.com/skypies/geo"
	"github.com/skypies/geo/sfo"
	"github.com/skypies/util/date"

	fdb "github.com/skypies/flightdb"
	"github.com/skypies/flightdb/report"
)

func init() {
	report.HandleReport("approachsignature", ApproachSignature,
		"Signature for SFO approaches, only when equip has prefix {str}")
}

func ApproachSignature(r *report.Report, f *fdb.Flight, tis []fdb.TrackIntersection) (report.FlightReportOutcome, error){
	r.I["[C] Flights considered for signatures"]++
	
	// Maybe make this configurable later
	equipPrefix := r.TextString
	if equipPrefix == "" { equipPrefix = "B73" } // Default
	dest := "SFO"
	reqWaypoints := []string{"EPICK","EDDYY","SWELS"}
	sigDistNMs := []float64{41.1, 37.5, 34.5, 33.5}

	if f.Destination != dest {
		r.I[fmt.Sprintf("[D] dest not %s", dest)]++
		return report.RejectedByReport, nil
	}
	if ! strings.HasPrefix(f.EquipmentType, equipPrefix) {
		//if ! regexp.MustCompile("^").MatchString(f.EquipmentType) {
		r.I[fmt.Sprintf("[D] equip didn't have prefix %s", equipPrefix)]++
		return report.RejectedByReport, nil
	}
	for _,wpName := range reqWaypoints {
		if ! f.HasWaypoint(wpName) {
			r.I[fmt.Sprintf("[D] didn't hit waypoint %s", wpName)]++
			return report.RejectedByReport, nil
		}
	}

	trackName,track := f.PreferredTrack([]string{"ADSB", "MLAT", "FOIA"})
	if trackName == "" {
		r.I["[D] Skipped, no ADSB, MLAT or FOIA track avail"]++
		return report.RejectedByReport,nil
	}

	r.I["[D] <b>flight accepted</b>"]++

	track.PostProcess()
	track.AdjustAltitudes(&r.Archive)
	
	sigDistKMs := []float64{}
	for _,v := range sigDistNMs { sigDistKMs = append(sigDistKMs, geo.NM2KM(v)) }
	results := track.IndicesAtDistKMsFrom(sfo.KAirports["KSFO"], sigDistKMs)

	tEpick := date.InPdt(f.Waypoints["EPICK"])	
	htmlRow := []string{
		r.Links(f),
		"<code>" + f.IdentString() + "</code>",
		"<code>" + f.EquipmentType + "</code>",
		trackName,
		tEpick.Format("15:04:05"),
//		fmt.Sprintf("%v", results),
	}	
	textRow := []string{
		f.IdentString(),
		f.EquipmentType,
		trackName,
		tEpick.Format("2006/01/02"),
		tEpick.Format("15:04:05"),
	}
	
	for i,result := range results {
		tp := track[result]

		dist := tp.DistNM(sfo.KAirports["KSFO"])

		track[result].AnalysisDisplay = fdb.AnalysisDisplayHighlight
		track[result].AnalysisAnnotation += fmt.Sprintf("* <b>Signature at %.1fNM</b>, from %v\n",
			dist, sigDistNMs)

		tpVals := []string{
			fmt.Sprintf("{time,alt,pressurealt,angle,accel}@%.1fNM", sigDistNMs[i]),
			date.InPdt(tp.TimestampUTC).Format("15:04:05"),
			fmt.Sprintf("%.0f", tp.IndicatedAltitude),
			fmt.Sprintf("%.0f", tp.Altitude),
			fmt.Sprintf("%.2f", tp.AngleOfInclination),
			fmt.Sprintf("%.2f", tp.GroundAccelerationKPS),
		}
		htmlRow = append(htmlRow, tpVals...)
		textRow = append(textRow, tpVals...)
	}
	
	r.AddRow(&htmlRow, &textRow)
	
	return report.Accepted, nil
}
