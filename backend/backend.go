package backend

import(
	"html/template"
	"net/http"

	"github.com/skypies/flightdb/fgae"
	mytemplates "github.com/skypies/flightdb/templates"
)

var templates *template.Template

func init() {
	// NEW
	http.HandleFunc(fgae.RangeUrl,    fgae.BatchFlightDateRangeHandler)
	http.HandleFunc(fgae.DayUrl,      fgae.BatchFlightDayHandler)
	http.HandleFunc(fgae.InstanceUrl, fgae.BatchFlightHandler)

	templates = mytemplates.LoadTemplates("templates")
}
