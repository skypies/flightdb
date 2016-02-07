package analysis

import(
	"github.com/skypies/flightdb2/metar"
)

// AnalysisContext provides data and helpers used throughout analysis
type AnalysisContext struct {
	Metar *metar.Archive
}
