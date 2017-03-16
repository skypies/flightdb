package report

import(
	"fmt"
	"html/template"
	"sort"
	"time"

	"github.com/skypies/util/date"
	"github.com/skypies/util/histogram"
	fdb "github.com/skypies/flightdb"
)

type FlightReportOutcome int
const(
	RejectedByGeoRestriction FlightReportOutcome = iota
	RejectedByReport
	Accepted
	Undefined
)
type ReportFunc func(*Report, *fdb.Flight, []fdb.TrackIntersection)(FlightReportOutcome,error)
type SummarizeFunc func(*Report)

type ReportLogLevel int
const(
	DEBUG = iota
	INFO
)

type Report struct {
	Name              string
	ReportingContext  // embedded
	Options           // embedded
	Func              ReportFunc
	SummarizeFunc     // embedded, but just to avoid a more confusing name
	TrackSpec       []string

	// Private state a report might accumulate (be careful about RAM though!)
	Blobs map[string]interface{}
	
	// Output state
	RowsHTML  [][]template.HTML
	RowsText  [][]string
	
	HeadersText []string
	
	I         map[string]int
	F         map[string]float64
	S         map[string]string
	H         histogram.Histogram
	
	Stats histogram.Set // internal performance counters
	Log string
}

func BlankReport() Report {
	return Report{
		I: map[string]int{},
		F: map[string]float64{},
		S: map[string]string{},
		RowsHTML: [][]template.HTML{},
		RowsText: [][]string{},
		HeadersText: []string{},
		Blobs: map[string]interface{}{},
		Stats: histogram.NewSet(40000),  // maxval, in micros; 40ms == 40000us
	}
}

func (r *Report)Logger(level ReportLogLevel, s string) {
	if level < r.Options.ReportLogLevel { return }
	r.Log += s
}
func (r *Report)Infof(s string,args ...interface{}) { r.Logger(INFO, fmt.Sprintf(s,args...)) }
func (r *Report)Debugf(s string,args ...interface{}) { r.Logger(DEBUG, fmt.Sprintf(s,args...)) }
func (r *Report)Info(s string) { r.Infof(s) }
func (r *Report)Debug(s string) { r.Debugf(s) }

func (r *Report)SetHeaders(headers []string) {
	if len(r.HeadersText) == 0 { r.HeadersText = headers }
}
func (r *Report)AddRow(html *[]string, text *[]string) {
	htmlRow := []template.HTML{}
	for _,s  := range *html { htmlRow = append(htmlRow, template.HTML(s)) }
	if html != nil { r.RowsHTML = append(r.RowsHTML, htmlRow) }
	if text != nil { r.RowsText = append(r.RowsText, *text) }
}

func (r *Report)ListPreferredDataSources() []string {
	// Dumb logic for now ...
	if r.Options.TrackDataSource != "" {
		return []string{r.Options.TrackDataSource}
	}
	return r.TrackSpec
}

// Ensure the flight matches all the search restrictions
func (r *Report)PreProcess(f *fdb.Flight) (bool, []fdb.TrackIntersection) {
	r.I["[A] PreProcessed"]++

	for _,nottag := range r.NotTags {
		if f.HasTag(nottag) {
			r.I[fmt.Sprintf("[B] Eliminated: had not-tag '%s'", nottag)]++
			return false, []fdb.TrackIntersection{}
		}
	}

	for _,notwp := range r.NotWaypoints {
		if f.HasWaypoint(notwp) {
			r.I[fmt.Sprintf("[B] Eliminated: had not-waypoint '%s'", notwp)]++
			return false, []fdb.TrackIntersection{}
		}
	}

	if f.HasTag("FOIA") && ! r.Options.CanSeeFOIA {
		r.I[fmt.Sprintf("[B] Eliminated: user can't access FOIA")]++
		return false, []fdb.TrackIntersection{}
	}
	
	// If restrictions were specified, only match flights that satisfy them
	failed := false
	intersections := []fdb.TrackIntersection{}

	if !r.Options.GRS.IsNil() {
		tStart := time.Now()
		satisfied,outcomes := f.SatisfiesGeoRestrictorSet(r.Options.GRS)
		r.Debugf("---- %s\nSources: %v\n", f.IdentityString(), r.ListPreferredDataSources())
		r.Debugf("--{ GRS }--\n%s", r.Options.GRS)
		r.Debugf("--{ Outcome satisfies=%v }--\n", satisfied)
		r.Debugf("--{ Debug }--\n%s\n", outcomes.Debug())
		r.Stats.RecordValue("restrictions", (time.Since(tStart).Nanoseconds()/1000))
		
		if satisfied {
			for _,o := range outcomes.Outcomes {
				intersections = append(intersections, o.TrackIntersection)
			}
		} else {
			r.I["[B] Eliminated: did not satisfy "+outcomes.BlameString(r.Options.GRS)]++
			failed = true
		}
	}
	if failed { return false, intersections }

	r.I["[B] <b>Satisfied geo restrictions</b> "]++

	if r.TimeOfDay.IsInitialized() {

		//r.Info(fmt.Sprintf("**** ToD %s, %s\n", r.TimeOfDay, f))
		times := []time.Time{} // Accumulate interesting timestamps in one place
		
		if len(intersections) > 0 {
			for _,ti := range intersections {
				times = append(times, ti.Start.TimestampUTC)
				//r.Info(fmt.Sprintf("   * i.s %s\n", date.InPdt(ti.Start.TimestampUTC)))
				if !ti.IsPointIntersection() {
					times = append(times, ti.End.TimestampUTC)
					//r.Info(fmt.Sprintf("   * i.e %s\n", date.InPdt(ti.End.TimestampUTC)))
				}
			}
		} else if len(r.Waypoints) > 0 {
			for _,wpName := range r.Waypoints {
				if 	t,exists := f.Waypoints[wpName]; exists {
					// r.Info(fmt.Sprintf("   * wp  %s (%s)\n", date.InPdt(t), wpName))
					times = append(times, t)
				}
			}
		}

		meetsToD := false
		for _,t := range times {
			//tPdt := date.InPdt(t)
			//s,e := r.TimeOfDay.AnchorInsideDay(tPdt)
			//r.Info(fmt.Sprintf("  ** ToD %s {%s -- %s} %s : %v\n", r.TimeOfDay, s,e,
			//	tPdt, r.TimeOfDay.Contains(tPdt)))

			if r.TimeOfDay.Contains(date.InPdt(t)) {
				meetsToD = true
				break
			}
		}

		if meetsToD {
			r.I["[Bb] <b>Satisfied TimeOfDay restrictions</b> "]++
		} else {
			r.I["[Bb] Failed TimeOfDay restrictions "]++
			return false, intersections
		}
	}
	
	dataSrc := "non-ADSB"
	if f.HasTrack("ADSB") { dataSrc = "ADSB" }
	r.I["[Bz] track source: "+dataSrc]++
	
	return true, intersections
}

func (r *Report)Process(f *fdb.Flight) (FlightReportOutcome, error) {
	wasOK,intersections := r.PreProcess(f)
	if !wasOK { return RejectedByGeoRestriction,nil }
	return r.Func(r, f, intersections)
}

func (r *Report)FinishSummary() {
	r.Info("**** Stage: all done\n")
	r.Debug("* (DEBUG)\n")
	if r.SummarizeFunc != nil { r.SummarizeFunc(r) }
	r.Infof("Stats (in micros):-\n%s", r.Stats)
}

func (r *Report)MetadataTable()[][]template.HTML {
	all := map[string]string{}

	for k,v := range r.I { all[k] = fmt.Sprintf("%d", v) }
	for k,v := range r.F { all[k] = fmt.Sprintf("%.1f", v) }
	for k,v := range r.S { all[k] = v }

	if stats,valid := r.H.Stats(); valid {
		all["[Z] stats,  <b>N</b>"] = fmt.Sprintf("%d", stats.N)
		all["[Z] stats, Mean"] = fmt.Sprintf("%.0f", stats.Mean)
		all["[Z] stats, Stddev"] = fmt.Sprintf("%.0f", stats.Stddev)
		all["[Z] stats, 50%ile"] = fmt.Sprintf("%.0d", stats.Percentile50)
		all["[Z] stats, 90%ile"] = fmt.Sprintf("%.0d", stats.Percentile90)
	}
	
	keys := []string{}
	for k,_ := range all { keys = append(keys, k) }
	sort.Strings(keys)
	
	out := [][]template.HTML{}
	for _,k := range keys {
		out = append(out, []template.HTML{ template.HTML(k), template.HTML(all[k]) })
	}
	
	return out
}
