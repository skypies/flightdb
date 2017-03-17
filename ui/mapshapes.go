package ui

import(
	"fmt"
	"html/template"
	"time"
	
	"github.com/skypies/geo"
	fdb "github.com/skypies/flightdb"
	"github.com/skypies/util/date"
)

// MapShapes is a single thing that contains all the things we want to render on a map
type MapShapes struct {
	Circles []MapCircle
	Lines []MapLine
	Points []MapPoint
	Icons []MapIcon
}

// {{{ NewMapShapes

func NewMapShapes() *MapShapes {
	ms := MapShapes{
		Circles: []MapCircle{},
		Lines: []MapLine{},
		Points: []MapPoint{},
		Icons: []MapIcon{},
	}
	return &ms
}

// }}}
// {{{ ms.Add [Line,Point,Circle,Icon]

func (ms1 *MapShapes)Add(ms2 *MapShapes) {
	ms1.Circles = append(ms1.Circles, ms2.Circles...)
	ms1.Lines   = append(ms1.Lines,   ms2.Lines...)
	ms1.Points  = append(ms1.Points,  ms2.Points...)
	ms1.Icons   = append(ms1.Icons,   ms2.Icons...)
}

func (ms1 *MapShapes)AddLine(ml MapLine) { ms1.Lines = append(ms1.Lines, ml) }
func (ms1 *MapShapes)AddPoint(mp MapPoint) { ms1.Points = append(ms1.Points, mp) }
func (ms1 *MapShapes)AddCircle(mc MapCircle) { ms1.Circles = append(ms1.Circles, mc) }
func (ms1 *MapShapes)AddIcon(mi MapIcon) { ms1.Icons = append(ms1.Icons, mi) }

// }}}

// {{{ MapCircle{}

type MapCircle struct {
	C *geo.LatlongCircle
	Color string
}

// }}}
// {{{ MapPoint{}

type MapPoint struct {
	ITP   *fdb.InterpolatedTrackpoint
	TP    *fdb.Trackpoint
	Pos   *geo.Latlong

	Icon   string  // The <foo> in /static/dot-<foo>.png
	Text   string	
}

// }}}
// {{{ MapLine{}

type MapLine struct {
	Start geo.Latlong `json:"s"`
	End   geo.Latlong `json:"e"`

	Color        string  `json:"color"`    // A hex color value (e.g. "#ff8822")
	Opacity      float64 `json:"opacity"`
}

// }}}
// {{{ MapIcon{}

type MapIcon struct {
	TP    *fdb.Trackpoint
	ZIndex int
	Color  string
	Text   string
}

// }}}

// {{{ mc.ToJSStr

func (mc MapCircle)ToJSStr(text string) string {
	color := mc.Color
	if color == "" { color = "#000000" }
	return fmt.Sprintf("center :{lat:%f, lng:%f}, radiusmeters: %.0f, color:%q",
		mc.C.Lat, mc.C.Long, mc.C.RadiusKM*1000.0, color)
}

// }}}
// {{{ mp.ToJSStr

func (mp MapPoint)ToJSStr(text string) string {
	if mp.Icon == "" { mp.Icon = "pink" }
	tp := fdb.Trackpoint{DataSource:"n/a"}
	
	if mp.ITP != nil {
		// Transform this into a tp with extra text
		tp = mp.ITP.Trackpoint
		tp.DataSource += "/interp"
		mp.Text = fmt.Sprintf("** Interpolated trackpoint\n"+
			" * Pre :%s\n * This:%s\n * Post:%s\n * Ratio: %.2f\n%s",
			mp.ITP.Pre, mp.ITP, mp.ITP.Post, mp.ITP.Ratio, mp.Text)
	} else if mp.TP != nil {
		tp = *mp.TP
		age := date.RoundDuration(time.Since(tp.TimestampUTC))
		times := fmt.Sprintf("%s (age:%s, epoch:%d)",
			date.InPdt(tp.TimestampUTC), age, tp.TimestampUTC.Unix())
		mp.Text = fmt.Sprintf("** %s \n* %s\n* DataSource: <b>%s</b>\n%s* %s",
			times, mp.TP, mp.TP.LongSource(), tp.AnalysisAnnotation, mp.Text)
		if tp.AnalysisDisplay == fdb.AnalysisDisplayHighlight {
			mp.Icon = "red-large"
		} else if tp.AnalysisDisplay == fdb.AnalysisDisplayOmit {
			mp.Text += "** OMIT\n"
		}
	} else {
		tp.Latlong = *mp.Pos
	}
	
	str := tp.ToJSString()
	str += fmt.Sprintf(", icon:%q, info:%q", mp.Icon, mp.Text)
	return str
}

// }}}
// {{{ ml.ToJSStr

func (ml MapLine)ToJSStr(text string) string {
	color,op := ml.Color, ml.Opacity
	if color == "" { color = "#000000" }
	if op == 0.0 { op = 1.0 }
	return fmt.Sprintf("s:{lat:%f, lng:%f}, e:{lat:%f, lng:%f}, color:\"%s\", opacity:%.2f",
		ml.Start.Lat, ml.Start.Long, ml.End.Lat, ml.End.Long, color, op) 
}

// }}}
// {{{ mi.ToJSStr

func (mi MapIcon)ToJSStr() string {
	color := mi.Color
	if color == "" { color = "#000000" }
	return fmt.Sprintf("center: {lat:%f, lng:%f}, rot: %.0f, color:%q, text:%q, zindex:%d",
		mi.TP.Lat, mi.TP.Long, mi.TP.Heading, color, mi.Text, mi.ZIndex)
}

// }}}

// These FooAsJSMap methods are invoked from the map-shapes.js template
// {{{ ms.CirclesAsJSMap

func (ms MapShapes)CirclesToJSMap() template.JS {
	str := "{\n"
	for i,mc := range ms.Circles {
		str += fmt.Sprintf("    %d: {%s},\n", i, mc.ToJSStr(""))
	}
	return template.JS(str + "  }\n")		
}

// }}}
// {{{ ms.PointsAsJSMap

func (ms MapShapes)PointsToJSMap() template.JS {
	str := "{\n"
	for i,mp := range ms.Points {
		str += fmt.Sprintf("    %d: {%s},\n", i, mp.ToJSStr(""))
	}
	return template.JS(str + "  }\n")		
}

// }}}
// {{{ ms.LinesAsJSMap

func (ms MapShapes)LinesToJSMap() template.JS {
	str := "{\n"
	for i,ml := range ms.Lines {
		str += fmt.Sprintf("    %d: {%s},\n", i, ml.ToJSStr(""))
	}
	return template.JS(str + "  }\n")		
}

// }}}
// {{{ ms.IconsAsJSMap

func (ms MapShapes)IconsToJSMap() template.JS {
	str := "{\n"
	for i,mi := range ms.Icons {
		str += fmt.Sprintf("    %d: {%s},\n", i, mi.ToJSStr())
	}
	return template.JS(str + "  }\n")		
}

// }}}

// {{{ LatlongTimeBoxToMapLines

func LatlongTimeBoxToMapLines(tb geo.LatlongTimeBox, color string) []MapLine {
	maplines := []MapLine{}
	for _,line := range tb.LatlongBox.ToLines() {
		mapline := MapLine{Start:line.From, End:line.To}
		mapline.Color = color
		maplines = append(maplines, mapline)
	}
	return maplines
}

// }}}
// {{{ TrackToMapPoints

// This is a kind of abuse of ColoringStrategy from colorscheme.go
func TrackToMapPoints(t *fdb.Track, icon, banner string, coloring ColoringStrategy) []MapPoint {
	sourceColors := map[string]string{
		"ADSB":  "yellow",
		"MLAT":  "red",
		"fr24":  "green",
		"FA:TA": "blue",
		"FA:TZ": "blue",
	}
	receiverColors := map[string]string{
		"NorthPi":       "green",
		"BlankPi":       "green",
		
		"ScottsValley":  "yellow",
		"ScottsValley3": "yellow",
		"ScottsValleyLite": "green",
		"Saratoga":      "red",
		"Saratoga2":     "blue",
	}

	points := []MapPoint{}
	if t==nil || len(*t) == 0 { return points }

	for i,_ := range *t {
		color := icon
		tp := (*t)[i]
		if coloring == ByCandyStripe {
			if tp.ReceiverName == "NorthPi" {
				if color == "blue" { color = "red" } else { color = "green" }
			} else if tp.ReceiverName == "ScottsValley" {
				if color == "blue" { color = "pink" }
			}

		} else if icon == "" {
			if coloring == ByADSBReceiver {
				if c,exists := receiverColors[tp.ReceiverName]; exists {
					color = c
				} else {
					color = receiverColors["default"]
				}
				// Offset receivers, slightly
				//if tp.ReceiverName == "ScottsValley" {
				//	tp.Lat  += 0.0006
				//	tp.Long += 0.0006
				//}
			} else if coloring == ByData {
				if c,exists := sourceColors[tp.DataSource]; exists { color = c }
			}
		}

		pointText := fmt.Sprintf("Point %d/%d\n%s", i, len(*t), banner)
		
		points = append(points, MapPoint{
			TP: &tp,
			Icon:color,
			Text: pointText,
		})
	}
	return points
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
