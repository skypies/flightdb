package report

import(
	"fmt"
	"html/template"
	"net/http"
	"sort"

	"google.golang.org/appengine"
	// "google.golang.org/appengine/log"
)

// A simple registry of all known reports.
type ReportEntry struct {
	ReportFunc
	Name, Description string
}

var reportRegistry = map[string]ReportEntry{}

func HandleReport(f ReportFunc, name, description string) {
	reportRegistry[name] = ReportEntry{
		ReportFunc: f,
		Name: name,
		Description: description,
	}
}

func ListReports() []ReportEntry {
	out := []ReportEntry{}

	keys := []string{}
	for k,_ := range reportRegistry { keys = append(keys, k) }
	sort.Strings(keys)

	for _,k := range keys {
		out = append(out, reportRegistry[k])
	}
	return out
}

func SetupReport(r *http.Request) (Report, error) {
	opt,err := FormValueReportOptions(r)
	if err != nil { return Report{}, err }

	rep,err := InstantiateReport(opt.Name)
	if err != nil { return Report{}, err }

	rep.Options = opt

	//log.Errorf(appengine.NewContext(r), "Oho: %#v", opt)
	
	rep.setupReportingContext(appengine.NewContext(r))

	return rep, nil
}

func InstantiateReport(name string) (Report,error) {
	// Lookup in registry
	r := Report{
		Name: name,
		I: map[string]int{},
		F: map[string]float64{},
		S: map[string]string{},
		RowsHTML: [][]template.HTML{},
		RowsText: [][]string{},
	}
	if entry,exists := reportRegistry[name]; !exists {
		return r, fmt.Errorf("report '%s' not known", name)
	} else {
		r.Func = entry.ReportFunc
	}
	return r, nil
}
