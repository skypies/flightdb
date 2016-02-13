package report

import(
	"fmt"
	"strings"
	fdb "github.com/skypies/flightdb2"
)

func init() {
	HandleReport(".list", ListReporter, "List all matching flights")
}

func ListReporter(r *Report, f *fdb.Flight, intersections []fdb.TrackIntersection) (bool, error) {	
	row := []string{
		r.Links(f),
		"<code>" + f.IdentString() + "</code>",
		fmt.Sprintf("[%s]", strings.Join(f.TagList(), ",")),
	}

	for _,ti := range intersections {
		row = append(row, ti.RowHTML()...)
	}

	r.AddRow(&row, &row)
	
	return true, nil
}
