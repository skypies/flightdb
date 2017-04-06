package backend

import(
	"html/template"
	"net/http"

	"github.com/skypies/flightdb/fgae"
	"github.com/skypies/flightdb/ui"
	"github.com/skypies/util/widget"
)

var AppTemplates *template.Template

func init() {
	AppTemplates = widget.ParseRecursive(template.New("").Funcs(ui.TemplateFuncMap()), "templates")

	http.HandleFunc(fgae.RangeUrl,    fgae.BatchFlightDateRangeHandler)
	http.HandleFunc(fgae.DayUrl,      fgae.BatchFlightDayHandler)
	http.HandleFunc(fgae.InstanceUrl, fgae.BatchFlightHandler)

	http.HandleFunc("/report", ui.WithCtxOptTmpl(AppTemplates, ui.ReportHandler))
}
