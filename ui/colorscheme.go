package ui

import "net/http"

var(
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
)

type ColorScheme int
const(
	ByData ColorScheme = iota
	ByAltitude
	ByComplaints
	ByTotalComplaints
)

func (cs ColorScheme)String() string {
	switch cs {
	case ByData:            return "source"
	case ByAltitude:        return "altitude"
	case ByComplaints:      return "complaints"
	case ByTotalComplaints: return "totalcomplaints"
	default:                return ""
	}
}

func FormValueColorScheme(r *http.Request) ColorScheme {
	switch r.FormValue("colorby") {
	case "source":          return ByData
	case "altitude":        return ByAltitude
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
