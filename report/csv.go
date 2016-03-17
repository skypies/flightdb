package report

import(
	"fmt"
	"encoding/csv"
	"net/http"
)

func (r *Report)OutputAsCSV(w http.ResponseWriter) {
	filename := fmt.Sprintf("report-%s-%s:%s.csv", r.Name, r.Start.Format("20060102"),
		r.End.Format("20060102"))
	_=filename
	w.Header().Set("Content-Type", "text/plain")
	//w.Header().Set("Content-Type", "application/csv")
//	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	csvWriter := csv.NewWriter(w)
	csvWriter.Write(r.HeadersText)
	for _,row := range r.RowsText {
		csvWriter.Write(row)
	}
	csvWriter.Flush()
}
