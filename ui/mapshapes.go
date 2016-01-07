package main

import(
	"fmt"
	"html/template"
	
	"github.com/skypies/geo"
	fdb "github.com/skypies/flightdb2"
)

type MapPoint struct {
//	ITP   *fdb.InterpolatedTrackPoint
	TP    *fdb.Trackpoint
	Pos   *geo.Latlong

	Icon   string  // The <foo> in /static/dot-<foo>.png
	Text   string	
}

func (mp MapPoint)ToJSStr(text string) string {
	if mp.Icon == "" { mp.Icon = "pink" }
	tp := fdb.Trackpoint{DataSource:"n/a"}
	
/*	if mp.ITP != nil {
		tp = mp.ITP.Trackpoint
		tp.Source += "/interp"
		mp.Text = fmt.Sprintf("** Interpolated trackpoint\n"+
			" * Pre :%s\n * This:%s\n * Post:%s\n * Ratio: %.2f\n%s",
			mp.ITP.Pre, mp.ITP, mp.ITP.Post, mp.ITP.Ratio, mp.Text)
	} else*/ if mp.TP != nil {
		tp = *mp.TP
		mp.Text = fmt.Sprintf("** Raw TP\n* %s\n* %s\n%s",
			mp.TP, mp.TP.LongSource(), mp.Text)
	} else {
		tp.Latlong = *mp.Pos
	}

	mp.Text += text

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
	SW,NE,SE,NW := tb.SW, tb.NE, tb.SE(), tb.NW()
	if color == "" { color = "#22aa33" }
	lines := []MapLine{
		MapLine{Start:&SE, End:&SW, Color:color},
		MapLine{Start:&SW, End:&NW, Color:color},
		MapLine{Start:&NW, End:&NE, Color:color},
		MapLine{Start:&NE, End:&SE, Color:color},
	}
	return lines
}

func TrackToMapPoints(t *fdb.Track, icon, text string) []MapPoint {
	points := []MapPoint{}
	for i,_ := range *t {
		color := icon
		tp := (*t)[i]
		if tp.ReceiverName == "SouthPi" {
			if color == "blue" { color = "red" } else { color = "green" }
		} else if tp.ReceiverName == "ScottsValley" {
			if color == "blue" { color = "pink" }
		}

		points = append(points, MapPoint{TP: &tp, Icon:color, Text:text})
	}
	return points
}
