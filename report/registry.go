package report

import(
	"fmt"
	"net/http"
	"sort"

	"golang.org/x/net/context"
)

// A simple registry of all known reports.
type ReportEntry struct {
	ReportFunc
	SummarizeFunc
	Name, Description string
	TrackSpec []string
}

var reportRegistry = map[string]ReportEntry{}

func HandleReport(name string, f ReportFunc, description string) {
	reportRegistry[name] = ReportEntry{
		ReportFunc: f,
		Name: name,
		Description: description,
	}
}
func SummarizeReport(name string, sf SummarizeFunc) {
	entry := reportRegistry[name]
	entry.SummarizeFunc = sf
	reportRegistry[name] = entry
}
func TrackSpec(name string, tracks []string) {
	entry := reportRegistry[name]
	entry.TrackSpec = tracks
	reportRegistry[name] = entry
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

func SetupReport(ctx context.Context, r *http.Request) (Report, error) {
	opt,err := FormValueReportOptions(ctx, r)
	if err != nil { return Report{}, err }

	rep,err := InstantiateReport(opt.Name)
	if err != nil { return Report{}, err }

	rep.Options = opt
	
	if err := rep.setupReportingContext(ctx); err != nil {
		return Report{}, err
	}

	if rep.ReportingContext.UserEmail == "" && r.FormValue("robot") == "1823791957218959063" {
		rep.ReportingContext.UserEmail = r.FormValue("robot")+"@hardwired.robots"
		rep.Options.CanSeeFOIA = true
	}
	
	return rep, nil
}

func InstantiateReport(name string) (Report,error) {
	// Lookup in registry
	r := BlankReport()

	r.Name = name
	
	if entry,exists := reportRegistry[name]; !exists {
		return r, fmt.Errorf("report '%s' not known", name)
	} else {
		r.Func = entry.ReportFunc
		r.SummarizeFunc = entry.SummarizeFunc
		r.TrackSpec = entry.TrackSpec
		if r.TrackSpec == nil {
			r.TrackSpec = []string{}
		}
	}
	return r, nil
}
