package ui

import(
	"html/template"
	"fmt"
	"net/http"

	"github.com/skypies/util/widget"
)

func init() {
	http.HandleFunc("/fdb/colorkey", ColorKeyHandler)
}


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

type ColorScheme struct {
	Strategy        ColoringStrategy
	DefaultOpacity  float64
	ExplicitColor   string // Color will be used if the Explicit strategy is selected
}

type ColoringStrategy int
const(
	ByData ColoringStrategy = iota
	ByAltitude
	ByAngleOfInclination
	ByComplaints
	ByTotalComplaints
	ByExplicitColor
	
	// Old ones, for trackpoints
	ByADSBReceiver
	ByCandyStripe
)

func (cs ColoringStrategy)String() string {
	switch cs {
	case ByData:               return "source"
	case ByAltitude:           return "altitude"
	case ByAngleOfInclination: return "angle"
	case ByComplaints:         return "complaints"
	case ByTotalComplaints:    return "totalcomplaints"
	case ByExplicitColor:      return "explicit"
	default:                   return ""
	}
}

func FormValueColoringStrategy(r *http.Request) ColoringStrategy {
	switch r.FormValue("colorby") {
	case "source":          return ByData
	case "altitude":        return ByAltitude
	case "angle":           return ByAngleOfInclination
	case "complaints":      return ByComplaints
	case "totalcomplaints": return ByTotalComplaints
	case "explicit":        return ByExplicitColor
	default:                return ByData
	}
}

// This doesn't "work", as the embedded ampersand ends up encoded.
func (cs ColorScheme)QuotedCGIArgs() template.JS {
	str := fmt.Sprintf("colorby=%s&maplineopacity=%.2f", cs.Strategy.String(), cs.DefaultOpacity)
	if cs.Strategy == ByExplicitColor {
		str += fmt.Sprintf("&explicitcolor=%s", cs.ExplicitColor)
	}
	return template.JS("\""+str+"\"")
}

func FormValueColorScheme(r *http.Request) ColorScheme {
	cs := ColorScheme{
		Strategy: FormValueColoringStrategy(r),
		DefaultOpacity: widget.FormValueFloat64EatErrs(r, "maplineopacity"),
		ExplicitColor: r.FormValue("explicitcolor"),
	}

	if cs.DefaultOpacity == 0.0 {
		cs.DefaultOpacity = 0.6
	}

	return cs
}

func ColorByTotalComplaintCount(n,scale int) string {
	switch {
	case n == 0: return "#404040"
	case n < 10*scale: return grad12[n/scale]
	default: return grad12[11]
	}
}
func ColorByComplaintCount(n int) string {
	n *= 2 // Get some dynamic range going
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


func ColorKeyHandler(w http.ResponseWriter, r *http.Request) {
	str := "<html><body>\n"

	str += "<h1>Altitude colors</h1>\n"
	str += "<table>\n"
	str += fmt.Sprintf("<tr><td> &lt;   500</td><td bgcolor='%s'>&nbsp;&nbsp;&nbsp;</td></tr>\n", grad12[11])
	str += fmt.Sprintf("<tr><td> &lt;  2000</td><td bgcolor='%s'>&nbsp;&nbsp;&nbsp;</td></tr>\n", grad12[10])
	str += fmt.Sprintf("<tr><td> &lt;  4000</td><td bgcolor='%s'>&nbsp;&nbsp;&nbsp;</td></tr>\n", grad12[9])
	str += fmt.Sprintf("<tr><td> &lt;  6000</td><td bgcolor='%s'>&nbsp;&nbsp;&nbsp;</td></tr>\n", grad12[8])
	str += fmt.Sprintf("<tr><td> &lt;  8000</td><td bgcolor='%s'>&nbsp;&nbsp;&nbsp;</td></tr>\n", grad12[7])
	str += fmt.Sprintf("<tr><td> &lt; 10000</td><td bgcolor='%s'>&nbsp;&nbsp;&nbsp;</td></tr>\n", grad12[6])
	str += fmt.Sprintf("<tr><td> &lt; 14000</td><td bgcolor='%s'>&nbsp;&nbsp;&nbsp;</td></tr>\n", grad12[5])
	str += fmt.Sprintf("<tr><td> &lt; 18000</td><td bgcolor='%s'>&nbsp;&nbsp;&nbsp;</td></tr>\n", grad12[4])
	str += fmt.Sprintf("<tr><td> &lt; 22000</td><td bgcolor='%s'>&nbsp;&nbsp;&nbsp;</td></tr>\n", grad12[3])
	str += fmt.Sprintf("<tr><td> &lt; 26000</td><td bgcolor='%s'>&nbsp;&nbsp;&nbsp;</td></tr>\n", grad12[2])
	str += fmt.Sprintf("<tr><td> &lt; 30000</td><td bgcolor='%s'>&nbsp;&nbsp;&nbsp;</td></tr>\n", grad12[1])
	str += fmt.Sprintf("<tr><td> &gt; 30000</td><td bgcolor='%s'>&nbsp;&nbsp;&nbsp;</td></tr>\n", grad12[0])
	str += "</table>\n"
	
	str += "</body></html>\n"
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, str)
}
