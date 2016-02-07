// Provides routines to render flights as PDFs in various ways
package fpdf

import(
	"fmt"
	"io"
	"math"
	"time"
	"github.com/jung-kurt/gofpdf"
	"github.com/skypies/geo/sfo"
	fdb "github.com/skypies/flightdb2"
)

// https://godoc.org/github.com/jung-kurt/gofpdf

type ColorScheme int
const(
	ByGroundspeed ColorScheme = iota
	ByDeltaGroundspeed
)

// {{{ var()

// The ApproachBox is from NW(10,10) to SE(270,110)
var(
	ApproachBoxWidth = 260.0
	ApproachBoxHeight = 100.0
	ApproachBoxOffsetX = 10.0
	ApproachBoxOffsetY = 10.0

	ApproachWidthNM = 80.0   // How many NM out the box starts
	ApproachHeightFeet = 20000.0 // How many feet up the box starts

	SpeedGradientMin = 200.0
	SpeedGradientMax = 400.0

	DeltaGradientMax = 20.0
	
	// http://www.perbang.dk/rgbgradient/
	SpeedGradientColors = [][]int{
/*
		{0x3F, 0xFD, 0x2B}, // 3FFD2B
		{0x52, 0xE4, 0x28}, // 52E428
		{0x65, 0xCC, 0x25}, // 65CC25
		{0x78, 0xB4, 0x22}, // 78B422
		{0x8B, 0x9C, 0x1F}, // 8B9C1F
		{0x9E, 0x84, 0x1C}, // 9E841C
		{0xB1, 0x6B, 0x19}, // B16B19
		{0xC4, 0x53, 0x16}, // C45316
		{0xD7, 0x3B, 0x13}, // D73B13
		{0xEA, 0x23, 0x10}, // EA2310

		//{0x00, 0xBF, 0x21}, // 00BF21
		{0x0D, 0xC2, 0x00}, // 0DC200
		{0x3E, 0xC6, 0x00}, // 3EC600
		{0x70, 0xCA, 0x00}, // 70CA00
		{0xA5, 0xCE, 0x00}, // A5CE00
		{0xD2, 0xC7, 0x00}, // D2C700
		{0xD5, 0x96, 0x00}, // D59600
		{0xD9, 0x64, 0x00}, // D96400
		{0xDD, 0x2F, 0x00}, // DD2F00
		{0xE1, 0x00, 0x06}, // E10006
		{0xE5, 0x00, 0x3F}, // E5003F
*/

		{0x00, 0xBF, 0xA9}, // 00BFA9
		{0x00, 0xC2, 0x66}, // 00C266
		{0x00, 0xC5, 0x21}, // 00C521
		{0x25, 0xC9, 0x00}, // 25C900
		{0x6F, 0xCC, 0x00}, // 6FCC00
		{0xBB, 0xD0, 0x00}, // BBD000
		{0xD3, 0x9D, 0x00}, // D39D00
		{0xD7, 0x53, 0x00}, // D75300
		{0xDA, 0x06, 0x00}, // DA0600
		{0xDE, 0x00, 0x48}, // DE0048
		{0xE1, 0x00, 0x99}, // E10099
		{0xDB, 0x00, 0xE5}, // DB00E5
	}

	
	DeltaGradientColors = [][]int{
		{0xF5, 0x00, 0x2B}, // E5002B
		{0xA8, 0x00, 0x1C}, // 98001C
		{0x7C, 0x00, 0x0E}, // 4C000E
		{0x70, 0x70, 0x70}, // 000000
		{0x00, 0x6C, 0x03}, // 004C03
		{0x00, 0x98, 0x07}, // 009807
		{0x00, 0xE5, 0x0B}, // 00E50B
	}
)

// }}}

// {{{ groundspeedToRGB, groundspeedDeltaToRGB

func groundspeedToRGB(speed float64) []int {
	if speed >= SpeedGradientMax { return SpeedGradientColors[len(SpeedGradientColors)-1] }
	if speed <= SpeedGradientMin { return SpeedGradientColors[0] }

	f := (speed-SpeedGradientMin) / (SpeedGradientMax-SpeedGradientMin)
	i := int (f * float64(len(SpeedGradientColors) - 2))
	return SpeedGradientColors[i+1]
}

func groundspeedDeltaToRGB(delta float64) []int {
	f := delta / 4.0  // How many 5knot increments this delta is
	f += 3.0          // [0,1,2] are braking, [3] is nochange, [4,5,6] are accelerating
	i := int(f)

	if i<0 { i = 0 }
	if i>6 { i = 6 }

	rgbw := DeltaGradientColors[i]

	fAbs := math.Abs(delta/4.0)
	widthPercent := int (fAbs * 0.33 * 100)
	if widthPercent < 10 { widthPercent = 10 }
	rgbw = append(rgbw, widthPercent)
	
	return rgbw
}

// }}}
// {{{ altitudeToY, distNMToX

func altitudeToY(alt float64) float64 {
	distY := (alt/ApproachHeightFeet) * ApproachBoxHeight
	y := ApproachBoxHeight - distY // In PDF, the Y scale goes down the page
	return y + ApproachBoxOffsetY
}
func distNMToX(distNM float64) float64 {
	distX := (distNM/ApproachWidthNM) * ApproachBoxWidth // How many X units away from SFO
	x := ApproachBoxWidth - distX  // SFO is on the right of the box
	return x + ApproachBoxOffsetX
}

// }}}

// {{{ DrawSpeedGradientKey, DrawDeltaGradientKey

func DrawSpeedGradientKey(pdf *gofpdf.Fpdf) {
	width,height := 8,4
	// Allow for the underflow & overflow colors at either end of the gradient
	speedPerBox := (SpeedGradientMax-SpeedGradientMin) / float64(len(SpeedGradientColors)-2)

	for i,rgb := range SpeedGradientColors {
		x,y := ApproachBoxOffsetX, ApproachBoxHeight-float64((i-1)*height)
		pdf.SetFillColor(rgb[0], rgb[1], rgb[2])
		pdf.Rect(x+2.0, y, float64(width), float64(height), "F")
		min := SpeedGradientMin + float64(i)*speedPerBox
		pdf.MoveTo(x+float64(width)+2.0, y)
		text := fmt.Sprintf(">=%.0f knots GS", min)
		if i==0 { text = fmt.Sprintf("<%.0f knots GS", min) }
		pdf.Cell(30, float64(height), text)
	}
}

func DrawDeltaGradientKey(pdf *gofpdf.Fpdf) {
	width,height := 8,4

	labels := []string{
		"braking: by >8 knots within 5s",
		"braking: by 4-8 knots within 5s",
		"braking: by 0-4 knots within 5s",
		"no change",
		"accelerating: by 0-4 knots within 5s",
		"accelerating: by 4-8 knots within 5s",
		"accelerating: by >8 knots within 5s",
	}		

	for i,rgb := range DeltaGradientColors {
		x,y := ApproachBoxOffsetX, ApproachBoxHeight-float64((i-1)*height)
		pdf.SetFillColor(rgb[0], rgb[1], rgb[2])
		pdf.Rect(x+2.0, y, float64(width), float64(height), "F")
		pdf.MoveTo(x+float64(width)+2.0, y)
		pdf.Cell(30, float64(height), labels[i])
	}
}

// }}}
// {{{ DrawTitle

func DrawTitle(pdf *gofpdf.Fpdf, title string) {
	pdf.MoveTo(10, ApproachBoxHeight + ApproachBoxOffsetY)
	pdf.Cell(40, 10, title)
}

// }}}
// {{{ DrawApproachFrame

func DrawApproachFrame(pdf *gofpdf.Fpdf) {
	pdf.SetDrawColor(0x0, 0x00, 0x00)
	pdf.SetLineWidth(0.5)
	pdf.MoveTo(ApproachBoxOffsetX,                  ApproachBoxOffsetY)
	pdf.LineTo(ApproachBoxOffsetX+ApproachBoxWidth, ApproachBoxOffsetY)
	pdf.LineTo(ApproachBoxOffsetX+ApproachBoxWidth, ApproachBoxOffsetY+ApproachBoxHeight)
	pdf.LineTo(ApproachBoxOffsetX,                  ApproachBoxOffsetY+ApproachBoxHeight)
	pdf.LineTo(ApproachBoxOffsetX,                  ApproachBoxOffsetY)
	pdf.DrawPath("D")
}

// }}}
// {{{ DrawSFOClassB

func DrawSFOClassB(pdf *gofpdf.Fpdf) {
	pdf.SetDrawColor(0x00, 0x00, 0x66)
	pdf.SetLineWidth(0.25)		
	pdf.MoveTo(ApproachBoxOffsetX+ApproachBoxWidth, altitudeToY(10000.0))

/*
					{ 7,  0, 100},   // from origin to  7NM : 100/00 (no floor)
					{10, 15, 100},   // from   7NM  to 10NM : 100/15
					{15, 30, 100},   // from  10NM  to 15NM : 100/30
					{20, 40, 100},   // from  15NM  to 20NM : 100/40
					{25, 60, 100},   // from  20NM  to 25NM : 100/60
					{30, 80, 100},   // from  25NM  to 30NM : 100/80
*/

	// Should really parse this all out of the constants in geo/sfo ...
	pdf.LineTo(distNMToX(30.0), altitudeToY(10000.0))
	pdf.LineTo(distNMToX(30.0), altitudeToY( 8000.0))
	pdf.LineTo(distNMToX(25.0), altitudeToY( 8000.0))
	pdf.LineTo(distNMToX(25.0), altitudeToY( 6000.0))
	pdf.LineTo(distNMToX(20.0), altitudeToY( 6000.0))
	pdf.LineTo(distNMToX(20.0), altitudeToY( 4000.0))
	pdf.LineTo(distNMToX(15.0), altitudeToY( 4000.0))
	pdf.LineTo(distNMToX(15.0), altitudeToY( 3000.0))
	pdf.LineTo(distNMToX(10.0), altitudeToY( 3000.0))
	pdf.LineTo(distNMToX(10.0), altitudeToY( 1500.0))
	pdf.LineTo(distNMToX( 7.0), altitudeToY( 1500.0))
	pdf.LineTo(distNMToX( 7.0), altitudeToY(    0.0))

	pdf.DrawPath("D")
}

// }}}
// {{{ DrawTrack

func trackpointToApproachXY(tp fdb.Trackpoint) (float64, float64) {
	return distNMToX(tp.DistNM(sfo.KLatlongSFO)), altitudeToY(tp.IndicatedAltitude)
}
// {{{ DrawTrackOld

func DrawTrackOld(pdf *gofpdf.Fpdf, t fdb.Track) {
	pdf.SetDrawColor(0xff, 0x00, 0x00)
	pdf.SetLineWidth(1)
	for i,_ := range t[1:] {
		x1,y1 := trackpointToApproachXY(t[i])
		x2,y2 := trackpointToApproachXY(t[i+1])
		if x2 < ApproachBoxOffsetX { continue }
		if y2 < ApproachBoxOffsetY { continue }

		rgb := groundspeedToRGB(t[i].GroundSpeed)
		pdf.SetDrawColor(rgb[0], rgb[1], rgb[2])
		pdf.Line(x1,y1,x2,y2)
	}
	pdf.DrawPath("D")	
}

// }}}

func DrawTrack(pdf *gofpdf.Fpdf, tInput fdb.Track, colorscheme ColorScheme) {
	pdf.SetDrawColor(0xff, 0x00, 0x00)
	pdf.SetLineWidth(0.25)

	// We don't need trackpoints every 200ms
	sampleRate := time.Second * 5
	t := tInput.SampleEvery(sampleRate, false)

	if len(t) == 0 { return }
	
	for i,_ := range t[1:] {
		x1,y1 := trackpointToApproachXY(t[i])
		x2,y2 := trackpointToApproachXY(t[i+1])
		// ... or compare against x2/y2 and clip against frame ...
		if x1 < ApproachBoxOffsetX { continue }
		if y1 < ApproachBoxOffsetY { continue }

		rgb := []int{}
		switch colorscheme {
		case ByGroundspeed: rgb = groundspeedToRGB(t[i].GroundSpeed)
		case ByDeltaGroundspeed: rgb = groundspeedDeltaToRGB(t[i+1].GroundSpeed - t[i].GroundSpeed)
		}

		pdf.SetLineWidth(0.25)
		if len(rgb)>3 {
			pdf.SetLineWidth(float64(rgb[3]) / 100.0)
		}
		
		pdf.SetDrawColor(rgb[0], rgb[1], rgb[2])
		pdf.Line(x1,y1,x2,y2)
	}
	pdf.DrawPath("D")	
}

// }}}

// {{{ NewApproachPdf

func NewApproachPdf(colorscheme ColorScheme) *gofpdf.Fpdf {
	pdf := gofpdf.New("L", "mm", "Letter", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "", 10)	
	DrawApproachFrame(pdf)
	DrawSFOClassB(pdf)

	if colorscheme == ByDeltaGroundspeed {
		DrawDeltaGradientKey(pdf)
	} else {
		DrawSpeedGradientKey(pdf)
	}
	return pdf
}

// }}}

// {{{ WriteTrack

func WriteTrack(output io.Writer, t fdb.Track) error {
	pdf := NewApproachPdf(ByGroundspeed)
	DrawTrack(pdf, t, ByGroundspeed)
	return pdf.Output(output)
}

// }}}
// {{{ WriteFlight

func WriteFlight(output io.Writer, f fdb.Flight) error {
	pdf := NewApproachPdf(ByGroundspeed)

	pdf.MoveTo(10, ApproachBoxHeight + ApproachBoxOffsetY)
	pdf.Cell(40, 10, fmt.Sprintf("%s", f))

	DrawTrack(pdf, f.AnyTrack(), ByGroundspeed)
	return pdf.Output(output)
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
