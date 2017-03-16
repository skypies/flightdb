package analysis

import (
	"fmt"

	"github.com/skypies/util/date"

	fdb "github.com/skypies/flightdb"
	"github.com/skypies/flightdb/report"
)

func init() {
	report.HandleReport("flowndist", FlownDist, "Flown dist from {refpoint} to {refpoint2}")
}

func FlownDist(r *report.Report, f *fdb.Flight, tis []fdb.TrackIntersection) (report.FlightReportOutcome, error){
 	r.I["[C] Flights considered"]++

	wp1,wp2 := r.Options.ReferencePoint.Name, r.Options.ReferencePoint2.Name
	
	if wp1 == "" {
		r.I["[C] <b>{refpoint} NOT DEFINED</b>"]++
		return report.RejectedByReport, nil
	} else if wp2 == "" {
		r.I["[C] <b>{refpoint2} NOT DEFINED</b>"]++
		return report.RejectedByReport, nil
	}

	if !f.HasWaypoint(wp1) {
		r.I["[D] flights without "+wp1]++
		return report.RejectedByReport, nil
	} else if !f.HasWaypoint(wp2) {
		r.I["[D] flights without "+wp2]++
		return report.RejectedByReport, nil
	}

	iTrack,i := f.AtWaypoint(wp1)
	jTrack,j := f.AtWaypoint(wp2)
	if i < 0 {
		r.I["[D] flight failed to match "+wp1]++
		return report.RejectedByReport, nil
	} else if j < 0 {
		r.I["[D] flight failed to match "+wp2]++
		return report.RejectedByReport, nil
	} else if iTrack != jTrack {
		r.I["[D] flight mixed tracks "+iTrack+","+jTrack]++
		return report.RejectedByReport, nil
	}
	r.I["[D] <b>flight had "+wp1+"-"+wp2+"</b>"]++

	track := *f.Tracks[iTrack]
	track.PostProcess()
	track[i].AnalysisDisplay = fdb.AnalysisDisplayHighlight
	track[j].AnalysisDisplay = fdb.AnalysisDisplayHighlight
	track[i].AnalysisAnnotation += "* <b>Start of flown dist measurement</b>\n"
	track[j].AnalysisAnnotation += "* <b>End of flown dist measurement</b>\n"

	flownDist := track[j].DistanceTravelledKM - track[i].DistanceTravelledKM
	
	htmlRow := []string{
		r.Links(f),
		"<code>" + f.IdentString() + "</code>",
		"<code>" + f.EquipmentType + "</code>",
		iTrack,
		fmt.Sprintf("%s-%s in KM", wp1, wp2),
		fmt.Sprintf("%.2f", flownDist),
		"time@"+wp1,
		date.InPdt(track[i].TimestampUTC).Format("15:04:05"),
		"time@"+wp2,
		date.InPdt(track[j].TimestampUTC).Format("15:04:05"),
	}
	
	r.AddRow(&htmlRow, &htmlRow)
	
	return report.Accepted, nil
}
