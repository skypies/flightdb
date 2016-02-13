package ui

import(
	"fmt"
	"html/template"
	"time"
	
	"github.com/skypies/geo"
	fdb "github.com/skypies/flightdb2"
	"github.com/skypies/util/date"
)

// MapShapes is a single thing that contains all the things we want to render on a map
type MapShapes struct {
	Circles []MapCircle
	Lines []MapLine
	Points []MapPoint
}
func NewMapShapes() *MapShapes {
	ms := MapShapes{
		Circles: []MapCircle{},
		Lines: []MapLine{},
		Points: []MapPoint{},
	}
	return &ms
}
func (ms1 *MapShapes)Add(ms2 *MapShapes) {
	ms1.Circles = append(ms1.Circles, ms2.Circles...)
	ms1.Lines   = append(ms1.Lines,   ms2.Lines...)
	ms1.Points  = append(ms1.Points,  ms2.Points...)
}
func (ms1 *MapShapes)AddLine(ml MapLine) { ms1.Lines = append(ms1.Lines, ml) }
func (ms1 *MapShapes)AddPoint(mp MapPoint) { ms1.Points = append(ms1.Points, mp) }
func (ms1 *MapShapes)AddCircle(mc MapCircle) { ms1.Circles = append(ms1.Circles, mc) }


type MapCircle struct {
	C *geo.LatlongCircle
	Color string
}
func (mc MapCircle)ToJSStr(text string) string {
	color := mc.Color
	if color == "" { color = "#000000" }
	return fmt.Sprintf("center :{lat:%f, lng:%f}, radiusmeters: %.0f, color:%q",
		mc.C.Lat, mc.C.Long, mc.C.RadiusKM*1000.0, color)
}
func MapCirclesToJSVar(circles []MapCircle) template.JS {
	str := "{\n"
	for i,mc := range circles {
		str += fmt.Sprintf("    %d: {%s},\n", i, mc.ToJSStr(""))
	}
	return template.JS(str + "  }\n")		
}

type MapPoint struct {
	ITP   *fdb.InterpolatedTrackpoint
	TP    *fdb.Trackpoint
	Pos   *geo.Latlong

	Icon   string  // The <foo> in /static/dot-<foo>.png
	Text   string	
}

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
		mp.Text = fmt.Sprintf("** %s\n* %s\n* DataSource: <b>%s</b>\n%s* %s",
			times, mp.TP, mp.TP.LongSource(), tp.AnalysisAnnotation, mp.Text)
		if tp.AnalysisMapIcon != "" {
			mp.Icon = tp.AnalysisMapIcon
			}
	} else {
		tp.Latlong = *mp.Pos
	}
	
	str := tp.ToJSString()
	str += fmt.Sprintf(", icon:%q, info:%q", mp.Icon, mp.Text)
	return str
}


type MapLine struct {
	Line        *geo.LatlongLine
	Start, End  *geo.Latlong

	Color  string  // A hex color value (e.g. "#ff8822")
}
func (ml MapLine)ToJSStr(text string) string {
	color := ml.Color
	if color == "" { color = "#000000" }

	if ml.Line != nil {
		return fmt.Sprintf("s:{lat:%f, lng:%f}, e:{lat:%f, lng:%f}, color:\"%s\"",
			ml.Line.From.Lat, ml.Line.From.Long, ml.Line.To.Lat, ml.Line.To.Long, color) 
	} else {
		return fmt.Sprintf("s:{lat:%f, lng:%f}, e:{lat:%f, lng:%f}, color:\"%s\"",
			ml.Start.Lat, ml.Start.Long, ml.End.Lat, ml.End.Long, color) 
	}
}

func MapPointsToJSVar(points []MapPoint) template.JS {
	str := "{\n"
	for i,mp := range points {
		str += fmt.Sprintf("    %d: {%s},\n", i, mp.ToJSStr(""))
	}
	return template.JS(str + "  }\n")		
}

func MapLinesToJSVar(lines []MapLine) template.JS {
	str := "{\n"
	for i,ml := range lines {
		str += fmt.Sprintf("    %d: {%s},\n", i, ml.ToJSStr(""))
	}
	return template.JS(str + "  }\n")		
}



func LatlongTimeBoxToMapLines(tb geo.LatlongTimeBox, color string) []MapLine {
	maplines := []MapLine{}
	for _,line := range tb.LatlongBox.ToLines() {
		x := line
		maplines = append(maplines, MapLine{Line: &x, Color:color})
	}
	return maplines

	/*
	SW,NE,SE,NW := tb.SW, tb.NE, tb.SE(), tb.NW()
	if color == "" { color = "#22aa33" }
	lines := []MapLine{
		MapLine{Start:&SE, End:&SW, Color:color},
		MapLine{Start:&SW, End:&NW, Color:color},
		MapLine{Start:&NW, End:&NE, Color:color},
		MapLine{Start:&NE, End:&SE, Color:color},
	}
	return lines*/
}

type ColoringStrategy int
const(
	ByADSBReceiver ColoringStrategy = iota
	ByDataSource
	ByInstance
	ByCandyStripe // a mix of ADSBreceiver & instance
)

func TrackToMapPoints(t *fdb.Track, icon, text string, coloring ColoringStrategy) []MapPoint {
	sourceColors := map[string]string{
		"ADSB":  "yellow",
		"fr24":  "green",
		"FA:TA": "blue",
		"FA:TZ": "blue",
	}
	receiverColors := map[string]string{
		"ScottsValley":  "yellow",
		"NorthPi":       "pink",
		"LosAltosHills": "blue",
		"Saratoga":      "red",
	}

	points := []MapPoint{}
	if len(*t) == 0 { return points }

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
			} else if coloring == ByDataSource {
				if c,exists := sourceColors[tp.DataSource]; exists { color = c }
			}
		}
		
		points = append(points, MapPoint{
			TP: &tp, Icon:color,
			Text:fmt.Sprintf("Point %d/%d\n%s", i, len(*t), text),
		})
	}
	return points
}
