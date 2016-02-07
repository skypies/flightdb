package report

import(
	"fmt"
	"html/template"
	"sort"
	
	fdb "github.com/skypies/flightdb2"
)

type ReportFunc func(*Report,*fdb.Flight)(bool,error)

type Report struct {
	Name              string
	ReportingContext  // embedded
	Options           // embedded
	Func              ReportFunc
	
	RowsHTML  [][]template.HTML
	RowsText  [][]string

	I         map[string]int
	F         map[string]float64
	S         map[string]string
}

func (r *Report)AddRow(html *[]string, text *[]string) {
	htmlRow := []template.HTML{}
	for _,s  := range *html { htmlRow = append(htmlRow, template.HTML(s)) }
	if html != nil { r.RowsHTML = append(r.RowsHTML, htmlRow) }
	if text != nil { r.RowsText = append(r.RowsText, *text) }
}

func (r *Report)Process(f *fdb.Flight) (bool, error) {
	return r.Func(r, f)
}

func (r *Report)MetadataTable()[][]template.HTML {
	all := map[string]string{}

	for k,v := range r.I { all[k] = fmt.Sprintf("%d", v) }
	for k,v := range r.F { all[k] = fmt.Sprintf("%.1f", v) }
	for k,v := range r.S { all[k] = v }

	keys := []string{}
	for k,_ := range all { keys = append(keys, k) }
	sort.Strings(keys)

	out := [][]template.HTML{}
	for _,k := range keys {
		out = append(out, []template.HTML{ template.HTML(k), template.HTML(all[k]) })
	}
	
	return out
}
