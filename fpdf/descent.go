package fpdf

import(
	"fmt"

	"github.com/jung-kurt/gofpdf"
	"github.com/skypies/geo"
	fdb "github.com/skypies/flightdb2"
)

var (
	RedRGB   = []int{0xff, 0, 0}
	GreenRGB = []int{0, 0xff, 0}
	BlueRGB  = []int{0, 0, 0xff}
)

type DescentPdf struct {
	Grids        map[string]*BaseGrid

	AltitudeMin  float64
	AltitudeMax  float64
	OriginPoint  geo.Latlong
	OriginLabel  string
	LengthNM     float64
	ColorScheme  // embedded

	LineThickness  float64
	LineOpacity    float64 // 0.0==transparent, 1.0==opaque
	
	*gofpdf.Fpdf // Embedded pointer

	Caption      string
	Debug        string
	ShowDebug    bool
}

// {{{ dp.Init

func (g *DescentPdf)Init() {
	g.Fpdf = gofpdf.New("P", "mm", "Letter", "")
	g.AddPage()
	g.SetFont("Arial", "", 10)	

	if g.LineThickness == 0.0 { g.LineThickness = 0.25 }
	if g.LineOpacity == 0.0   { g.LineOpacity = 1.0 }
	
	g.Grids = map[string]*BaseGrid{}

	u,v := 22.0,35.0 // The top-left origin, in PDF space; increment as we go down the page
	
	incompleteGrid := func() *BaseGrid {
		return &BaseGrid{
			Fpdf: g.Fpdf,
			OffsetU: u,
			OffsetV: v,
			W: 170,
			MinX: 0,
			MaxX: g.LengthNM,
			XGridlineEvery: 10,
			Clip: true, // set to false for debugging datasets
			InvertX: true,  // Descend to origin, on the right
		}
	}
	
	ng := incompleteGrid()
	g.Grids["altitude"] = ng
	ng.LineColor = RedRGB
	ng.H = 100
	ng.MinY = g.AltitudeMin
	ng.MaxY = g.AltitudeMax
	ng.YGridlineEvery = 5000
	ng.XTickFmt = "%.0fNM"
	ng.YTickFmt = "%.0fft"
	ng.XTickOtherSide = true
	ng.YTickOtherSide = true

	v += 110
	
	ng = incompleteGrid()
	g.Grids["groundspeed"] = ng
	ng.LineColor = RedRGB
	ng.H = 50
	ng.MinY = 0
	ng.MaxY = 400
	ng.YGridlineEvery = 100
	ng.YTickFmt = "%.0f knots"

	// This is overlayed into the same grid as groundspeed
	ng = incompleteGrid()
	g.Grids["groundacceleration"] = ng
	ng.LineColor = BlueRGB
	ng.H = 50
	ng.MinY = -6
	ng.MaxY = 6
	ng.YGridlineEvery = 3
	ng.YTickFmt = "%.0f knots/sec"
	ng.YTickOtherSide = true
	ng.NoGridlines = true
	
	v += 60
	
	ng = incompleteGrid()
	g.Grids["verticalspeed"] = ng
	ng.LineColor = RedRGB
	ng.H = 50
	ng.MinY = -2500
	ng.MaxY =  2500
	ng.YGridlineEvery = 1250
	ng.YTickFmt = "%.0f feet/min"

	// This is overlayed into the same grid as verticalspeed
	ng = incompleteGrid()
	g.Grids["verticalacceleration"] = ng
	ng.LineColor = BlueRGB
	ng.H = 50
	ng.MinY = -100
	ng.MaxY =  100
	ng.YGridlineEvery = 50
	ng.YTickFmt = "%.0f fpm/sec"
	ng.YTickOtherSide = true
	ng.NoGridlines = true
	
/*
	g.AltitudeGrid = BaseGrid{
			Fpdf: g.Fpdf,
			OffsetU: 10,
			OffsetV: 10,
			W: 245,
			H: 100,
			MinX: 0,
			MaxX: g.LengthNM,
			MinY: g.AltitudeMin,
			MaxY: g.AltitudeMax,
			XGridlineEvery: 10,
			YGridlineEvery: 5000,
			XTickFmt: "%.0fNM",
			YTickFmt: "%.0fft",
			InvertX: true,  // Descend to origin, on the right
		}
*/
}

// }}}

// {{{ dp.DrawFrames

func (g DescentPdf)DrawFrames() {
	for _,grid := range g.Grids {
		grid.DrawGridlines()
	}
}

// }}}
// {{{ dp.DrawCaption

func (g DescentPdf)DrawCaption() {
	title := g.Caption
	
	if g.ShowDebug {
		title += "--DEBUG--\n" + g.Debug
	}

	g.SetTextColor(0xb0, 0xb0, 0xf0)
	g.MoveTo(10, 10)
	g.MultiCell(0, 4, title, "", "", false)
	g.DrawPath("D")
}

// }}}
// {{{ dp.DrawColorSchemeKeys

func (g DescentPdf)DrawColorSchemeKeys() {
	for _,grid := range g.Grids {
		grid.DrawColorSchemeKey()
	}
}

// }}}

// {{{ dp.DrawTrackWithDistFunc

type DistanceFunc func(tp fdb.Trackpoint) (float64, float64, []int)

func (g *DescentPdf)DrawTrackWithDistFunc(t fdb.Track, f DistanceFunc, colorscheme ColorScheme) {
	g.SetDrawColor(0xff, 0x00, 0x00)
	
	if len(t) == 0 { return }
	
	for i,_ := range t[1:] {
		x1,alt1,_ := f(t[i])
		x2,alt2,rgb := f(t[i+1])
		
		g.SetLineWidth(g.LineThickness)
		g.SetAlpha(g.LineOpacity, "")
		g.SetDrawColor(rgb[0], rgb[1], rgb[2])

		if grid,exists := g.Grids["altitude"]; exists {
			g.SetDrawColor(rgb[0], rgb[1], rgb[2])
			grid.Line(x1,alt1, x2,alt2)
		}

		// We can re-use the dist values (x1,x2), and plot other functions on the trackpoints
		if grid,exists := g.Grids["groundspeed"]; exists {
			grid.Line(x1,t[i].GroundSpeed, x2,t[i+1].GroundSpeed)
		}
		if grid,exists := g.Grids["groundacceleration"]; exists {
			grid.Line(x1,t[i].GroundAccelerationKPS, x2,t[i+1].GroundAccelerationKPS)
		}
		if grid,exists := g.Grids["verticalspeed"]; exists {
			grid.Line(x1,t[i].VerticalSpeedFPM, x2,t[i+1].VerticalSpeedFPM)
		}
		if grid,exists := g.Grids["verticalacceleration"]; exists {
			grid.Line(x1,t[i].VerticalAccelerationFPMPS, x2,t[i+1].VerticalAccelerationFPMPS)
		}

		g.Debug += fmt.Sprintf("%3d: %.1f, %.1f\n", i, t[i].VerticalSpeedFPM,
			t[i].VerticalAccelerationFPMPS)
	}

	g.DrawPath("D")	
}

// }}}
// {{{ dp.DrawTrackAsDistanceFromOrigin

// Consider distance as being simply the distance from the origin, and plot against altitude.
// The less the aircraft flies in a straight line to the origin, the less useful this will be.
// (E.g. if an aircraft descends in a spiral, it will plot as a zig-zag, getting closer then
// further away as it descends.)

func (g *DescentPdf)DrawTrackAsDistanceFromOrigin(t fdb.Track) {	
	trackpointToXY := func(tp fdb.Trackpoint) (float64, float64, []int) {
		rgb := []int{0,0,0}
		if g.ColorScheme == ByPlotKind { rgb = []int{0,250,0} }
		return tp.DistNM(g.OriginPoint), tp.IndicatedAltitude, rgb
	}

	g.Debug += fmt.Sprintf("DrawTrackAsDistanceFromOrigin\n")
	
	g.DrawTrackWithDistFunc(t, trackpointToXY, g.ColorScheme)
}

// }}}
// {{{ dp.DrawTrackAsDistanceAlongPath

// Consider distance to be distance travelled along the path.
// (E.g. if the aircraft descends in a steady spiral, we'll plot the 'unrolled' version as a
// long steady line.)
func (g *DescentPdf)DrawTrackAsDistanceAlongPath(t fdb.Track) {
	// We want to render this working backwards from the end, so descents can line up together.
	// That means we're interested in each point's distance travelled in relation to the end point.
	g.Debug += fmt.Sprintf("DrawTrackAsDistanceAlongPath\n")
	iClosest := t.ClosestTo(g.OriginPoint)
	if iClosest < 0 { return }
	endpointKM := t[iClosest].DistanceTravelledKM

	g.Debug += fmt.Sprintf("* endKM=%.2f\n", endpointKM)
	
	trackpointToXY := func(tp fdb.Trackpoint) (float64, float64, []int) {
		distToGoKM := endpointKM - tp.DistanceTravelledKM
		distToGoNM := distToGoKM * geo.KNauticalMilePerKM

		rgb := []int{0,0,0}
		if g.ColorScheme == ByPlotKind { rgb = []int{250,0,0} }

		//g.Debug += fmt.Sprintf("%s\n", tp)
		
		return distToGoNM, tp.IndicatedAltitude, rgb
	}

	g.DrawTrackWithDistFunc(t, trackpointToXY, g.ColorScheme)
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
