package backend

import(
	"html/template"
	"net/http"

	"github.com/skypies/flightdb/fgae"
	"github.com/skypies/flightdb/ui"
	mytemplates "github.com/skypies/flightdb/templates"
)

var AppTemplates *template.Template

func init() {
	AppTemplates = mytemplates.LoadTemplates("templates")

	http.HandleFunc(fgae.RangeUrl,    fgae.BatchFlightDateRangeHandler)
	http.HandleFunc(fgae.DayUrl,      fgae.BatchFlightDayHandler)
	http.HandleFunc(fgae.InstanceUrl, fgae.BatchFlightHandler)

	http.HandleFunc("/report", ui.WithCtxOptTmpl(AppTemplates, ui.ReportHandler))
}
