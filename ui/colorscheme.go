package ui

import "net/http"

var(
	// http://www.perbang.dk/rgbgradient/

	grad12 = []string{
		"#00BFA9",
		"#00C266",
		"#00C521",
		"#25C900",
		"#6FCC00",
		"#BBD000",
		"#D39D00",
		"#D75300",
		"#DA0600",
		"#DE0048",
		"#E10099",
		"#DB00E5",
	}

	gradRedToGreen7Rubbish = []string{
		"#E5002B",
		"#98001C",
		"#4C000E",
		"#000000",
		"#004C03",
		"#009807",
		"#00E50B",
	}

	// Goes via blue; easy to distinguish
	gradRedToGreen7 = []string{
		"#F62370",
		"#F62AE7",
		"#9732F7",
		"#3A44F7",
		"#42B1F8",
		"#4AF8DA",
		"#53F97F",
	}
)

type ColorScheme int
const(
	ByData ColorScheme = iota
	ByAltitude
	ByAngleOfInclination
	ByComplaints
	ByTotalComplaints
)

func (cs ColorScheme)String() string {
	switch cs {
	case ByData:               return "source"
	case ByAltitude:           return "altitude"
	case ByAngleOfInclination: return "angle"
	case ByComplaints:         return "complaints"
	case ByTotalComplaints:    return "totalcomplaints"
	default:                   return ""
	}
}

func FormValueColorScheme(r *http.Request) ColorScheme {
	switch r.FormValue("colorby") {
	case "source":          return ByData
	case "altitude":        return ByAltitude
	case "angle":           return ByAngleOfInclination
	case "complaints":      return ByComplaints
	case "totalcomplaints": return ByTotalComplaints
	default:                return ByData
	}
}

func ColorByTotalComplaintCount(n,scale int) string {
	switch {
	case n == 0: return "#404040"
	case n < 10*scale: return grad12[n/scale]
	default: return grad12[11]
	}
}
func ColorByComplaintCount(n int) string {
	switch {
	case n == 0: return "#404040"
	case n < 10: return grad12[n-1]
	default: return grad12[11]
	}
}

func ColorByAngle(a float64) string {
	switch {
	case a >  3.0: return gradRedToGreen7[6]
	case a >  1.5: return gradRedToGreen7[5]
	case a >  0.5: return gradRedToGreen7[4]

	case a < -3.0: return gradRedToGreen7[0]
	case a < -1.5: return gradRedToGreen7[1]
	case a < -0.5: return gradRedToGreen7[2]

	default:       return gradRedToGreen7[3] // "#5050ff"
	}
}

func ColorByAltitude(alt float64) string {
	switch {
	case alt <   500: return grad12[11]
	case alt <  2000: return grad12[10]
	case alt <  4000: return grad12[9]
	case alt <  6000: return grad12[8]
	case alt <  8000: return grad12[7]
	case alt < 10000: return grad12[6]
	case alt < 14000: return grad12[5]
	case alt < 18000: return grad12[4]
	case alt < 22000: return grad12[3]
	case alt < 26000: return grad12[2]
	case alt < 30000: return grad12[1]
	default:          return grad12[0]
	}
}
