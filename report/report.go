package report

import(
	"fmt"
	"html/template"
	"sort"

	"github.com/skypies/util/histogram"
	fdb "github.com/skypies/flightdb2"
)

type ReportFunc func(*Report, *fdb.Flight, []fdb.TrackIntersection)(bool,error)

type Report struct {
	Name              string
	ReportingContext  // embedded
	Options           // embedded
	Func              ReportFunc
	TrackSpec       []string
	
	RowsHTML  [][]template.HTML
	RowsText  [][]string

	I         map[string]int
	F         map[string]float64
	S         map[string]string
	H         histogram.Histogram
	
	DebugLog  string
}

func (r *Report)AddRow(html *[]string, text *[]string) {
	htmlRow := []template.HTML{}
	for _,s  := range *html { htmlRow = append(htmlRow, template.HTML(s)) }
	if html != nil { r.RowsHTML = append(r.RowsHTML, htmlRow) }
	if text != nil { r.RowsText = append(r.RowsText, *text) }
}

// Ensure the flight matches all the search restrictions
func (r *Report)PreProcess(f *fdb.Flight) (bool, []fdb.TrackIntersection) {
	r.I["[A] PreProcessed"]++

	// If restrictions were specified, only match flights that satisfy them
	failed := false
	intersections := []fdb.TrackIntersection{}
	for _,gr := range r.Options.ListGeoRestrictors() {
		satisfies,intersection,_ := f.SatisfiesGeoRestriction(gr, r.TrackSpec)
		if satisfies {
			intersections = append(intersections, intersection)
		} else {
			r.I["[B] Eliminated: did not satisfy "+gr.String()]++
			failed = true
			break
		}
	}
	if failed { return false, intersections }

	r.I["[B] <b>Satisfied geo restrictions</b> "]++
	return true, intersections
}

func (r *Report)Process(f *fdb.Flight) (bool, error) {
	wasOK,intersections := r.PreProcess(f)
	if !wasOK { return false,nil }
	return r.Func(r, f, intersections)
}

func (r *Report)MetadataTable()[][]template.HTML {
	all := map[string]string{}

	for k,v := range r.I { all[k] = fmt.Sprintf("%d", v) }
	for k,v := range r.F { all[k] = fmt.Sprintf("%.1f", v) }
	for k,v := range r.S { all[k] = v }

	if stats,valid := r.H.Stats(); valid {
		all["[Z] stats, <b>N</b>"] = fmt.Sprintf("%d", stats.N)
		all["[Z] stats, Mean"] = fmt.Sprintf("%.0f", stats.Mean)
		all["[Z] stats, Stddev"] = fmt.Sprintf("%.0f", stats.Stddev)
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
