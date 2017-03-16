package analysis

import (
	"fmt"
	"math"

	fdb "github.com/skypies/flightdb"
	"github.com/skypies/flightdb/report"
)

func init() {
	//report.HandleReport("levelflight", LevelFlightReporter,
	//	"Level flight across whole {region} with angle <= {tol}")
	report.HandleReport("levelflight2", NewLevelFlightReporter,
		"Level flight within {region}: angle <= {tol} for {dist}")
}

func NewLevelFlightReporter(r *report.Report, f *fdb.Flight, tis []fdb.TrackIntersection) (report.FlightReportOutcome, error){
	ti,err := r.GetFirstAreaIntersection(tis)
	if err != nil {
		return report.RejectedByReport, err
	}

	r.I["[C] Flights passing through region"]++

	// Identify the longest run within the intersection
	longestLevelRunKM := -1.0
	iStart,iEnd := 0,0
	tName := ""
	noteLevelRun := func(trackName string, t *fdb.Track, i,j int) {
		if i==j { return }
		levelRunKM := (*t)[j].DistanceTravelledKM - (*t)[i].DistanceTravelledKM
		if levelRunKM > longestLevelRunKM {
			longestLevelRunKM = levelRunKM
			iStart,iEnd,tName = i,j,trackName
		}
		_=ti
		//r.Info(fmt.Sprintf("%s: %s[%d,%d] == %.2fKM\n", f.BestFlightNumber(), ti.TrackName, i, j,
		//	levelRunKM))
	}

	for _,ti := range tis {
		t := f.Tracks[ti.TrackName]
		t.PostProcess()

		iStart := -1
		for i:=ti.I; i<=ti.J; i++ {
			isLevel := math.Abs((*t)[i].AngleOfInclination) <= r.AltitudeTolerance
			if isLevel {
				if iStart < 0 { iStart = i }
			} else {
				if iStart >= 0 {
					// This point is where we're no longer level; examine from iStart to prev point
					noteLevelRun(ti.TrackName, t, iStart, i-1)
					iStart = -1
				}
			}
		}

		// We might still be level, as we exit the region.
		if iStart >= 0 && iStart < ti.J {
			noteLevelRun(ti.TrackName, t, iStart, ti.J-1)
		}
	}

	// r.I[fmt.Sprintf("[Z_] max was %.1f", longestLevelRunKM)]++

	if longestLevelRunKM < r.RefDistanceKM {
		r.I[fmt.Sprintf("[D] Flights without level flight (|angle| <= %.1f deg, for >= %.1f KM)</b>",
			r.AltitudeTolerance, r.RefDistanceKM)]++
		return report.RejectedByReport, nil
	}
	
	r.I[fmt.Sprintf("[D] <b>Flights with level flight (|angle| <= %.1f deg, for >= %.1f KM)</b>",
		r.AltitudeTolerance, r.RefDistanceKM)]++

	t := *f.Tracks[tName]

	// For this report, we blank out everything but the selected level flight span. The trackpoint
	// display ignores this, but the vector display will omit/downlight these bits.
	for i:=0; i<iStart; i++ {
		t[i].AnalysisDisplay = fdb.AnalysisDisplayOmit
	}
	for i:=iStart; i<=iEnd; i++ {
		t[i].AnalysisDisplay = fdb.AnalysisDisplayHighlight
		t[i].AnalysisAnnotation += fmt.Sprintf("* <b>Level flight for %.1f KM</b>\n", longestLevelRunKM)
	}
	for i:=iEnd+1; i<len(t); i++ {
		t[i].AnalysisDisplay = fdb.AnalysisDisplayOmit
	}
	
	row := []string{
		r.Links(f),
		"<code>" + f.IdentString() + "</code>",
		"<b>(LengthKM,Alt,I,J)</b>",
		fmt.Sprintf("%.2f", longestLevelRunKM),
		fmt.Sprintf("%.0f", t[iStart].Altitude),
		fmt.Sprintf("%d", iStart),
		fmt.Sprintf("%d", iEnd),
	}

	r.AddRow(&row, &row)
	
	return report.Accepted, nil
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
