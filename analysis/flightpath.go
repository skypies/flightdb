package analysis

import(
	"fmt"
	
	"github.com/skypies/geo"
	"github.com/skypies/geo/sfo"
	fdb "github.com/skypies/flightdb2"
)

// Routines that take a track, and try to figure out which waypoints & procedures it might be
	
func MatchProcedure(t fdb.Track) (*geo.Procedure, string, error) {
	procedures := []geo.Procedure{ sfo.Serfr1 }
	str := ""

	boxes := t.AsContiguousBoxes()
	
	for _,proc := range procedures {
		proc.Populate(sfo.KFixes)
		lines := proc.ComparisonLines()

		for _,l := range lines {
			str += fmt.Sprintf("* I was looking at %s\n", l)
		}
		
		return &proc, str, nil
	}
	_=boxes

	return nil, str, nil
}
