package fpdf

import(
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/skypies/geo"
	fdb "github.com/skypies/flightdb"
)

type DescentPdf struct {
	ToShow          map[string]bool  // Which grids to render

	AltitudeMin     float64
	AltitudeMax     float64

	Anchor          geo.NamedLatlong
	AnchorAltitudeMin float64
	AnchorAltitudeMax float64
	// OriginPoint     geo.Latlong
	// OriginLabel     string

	LengthNM        float64
	AveragingWindow time.Duration
	ColorScheme     // embedded
	
	LineThickness   float64
	LineOpacity     float64 // 0.0==transparent, 1.0==opaque
	
	Grids        map[string]*BaseGrid
	*gofpdf.Fpdf // Embedded pointer

	Permalink    string
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

	if g.ToShow["altitude"] {
		ng := incompleteGrid()
		g.Grids["altitude"] = ng
		ng.LineColor = RedRGB
		ng.H = 100
		ng.MinY = g.AltitudeMin
		ng.MaxY = g.AltitudeMax
		ng.YMinorGridlineEvery = 1000
		ng.YGridlineEvery = 5000
		ng.XTickFmt = "%.0fNM"
		ng.YTickFmt = "%.0fft"
		ng.XTickOtherSide = true
		ng.YTickOtherSide = true
		
		v += 110
	}
	
	if g.ToShow["groundspeed"] {
		ng := incompleteGrid()
		g.Grids["groundspeed"] = ng
		ng.LineColor = RedRGB
		ng.H = 50
		ng.MinY = 0
		ng.MaxY = 500
		ng.YGridlineEvery = 100
		ng.YTickOtherSide = true
		ng.YTickFmt = "%.0f knots"

		// This is overlayed into the same grid as groundspeed
		if g.ToShow["groundacceleration"] {
			ng = incompleteGrid()
			g.Grids["groundacceleration"] = ng
			ng.LineColor = BlueRGB
			ng.H = 50
			ng.MinY = -6
			ng.MaxY = 6
			ng.YGridlineEvery = 3
			ng.YTickFmt = "%.0f knots/sec"
			ng.NoGridlines = true
		}

		v += 60
	}
	
	if g.ToShow["verticalspeed"] {
		ng := incompleteGrid()
		g.Grids["verticalspeed"] = ng
		ng.LineColor = RedRGB
		ng.H = 50
		ng.MinY = -2500
		ng.MaxY =  2500
		ng.YGridlineEvery = 1250
		ng.YTickOtherSide = true
		ng.YTickFmt = "%.0f feet/min"

		// This is overlayed into the same grid as verticalspeed
		if g.ToShow["angleofinclination"] {
			ng := incompleteGrid()
			g.Grids["angleofinclination"] = ng
			ng.LineColor = GreenRGB
			ng.H = 50
			ng.MinY = -8
			ng.MaxY = +8
			ng.YGridlineEvery = 4
			ng.YTickFmt = "%.0f deg"
		}
		
		// This is overlayed into the same grid as verticalspeed
		if g.ToShow["verticalacceleration"] {
			ng = incompleteGrid()
			g.Grids["verticalacceleration"] = ng
			ng.LineColor = BlueRGB
			ng.H = 50
			ng.MinY = -100
			ng.MaxY =  100
			ng.YGridlineEvery = 50
			ng.YTickFmt = "%.0f fpm/sec"
			ng.NoGridlines = true
		}

		v += 60
	}
}

// }}}
// {{{ dp.ReconfigureForDepartures

// If we are rendering departures, flip everything so the origin is on the Left Hand Side.
func (g *DescentPdf)ReconfigureForDepartures() {
	for name,_ := range g.Grids {
		g.Grids[name].InvertX = false
		g.Grids[name].YTickOtherSide = !g.Grids[name].YTickOtherSide
	}
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
	title := ""

	if g.Anchor.Name != "" {
		title += fmt.Sprintf("* 0NM anchor point: %s (altitude: %.0f - %.0f)\n", g.Anchor.Name,
			g.AnchorAltitudeMin, g.AnchorAltitudeMax)
	}
	
	if g.AveragingWindow > 0 {
		title += fmt.Sprintf("* Averaging window: %s\n", g.AveragingWindow)
	}

	title += g.Caption
	
	if g.ShowDebug {
		title += "--DEBUG--\n" + g.Debug
	}

	g.SetTextColor(0x50, 0x70, 0xc0)
	g.MoveTo(10, 10)
	g.MultiCell(0, 4, title, "", "", false)
	g.DrawPath("D")

	if g.Permalink != "" {
		g.SetFont("Arial", "B", 10)	
		g.MoveTo(190, 5)
		g.CellFormat(20, 4, "[Permalink]", "", 0, "", false, 0, g.Permalink)
		g.DrawPath("D")
		g.SetFont("Arial", "", 10)	
	}
}

// }}}
// {{{ dp.DrawColorSchemeKeys

func (g DescentPdf)DrawColorSchemeKeys() {
	for _,grid := range g.Grids {
		grid.DrawColorSchemeKey()
	}
}

// }}}

// {{{ dp.MaybeDrawSFOClassB

func (g DescentPdf)MaybeDrawSFOClassB() {
	grid,exists := g.Grids["altitude"]
	if !exists { return }

	grid.SetDrawColor(0x00, 0x00, 0x66)
	grid.SetLineWidth(0.45)

	// Should really parse this all out of geo/sfo.SFOClassBMap ...
	grid.MoveTo( 0.0, 10000.0)
	grid.LineTo(30.0, 10000.0)
	grid.LineTo(30.0,  8000.0)
	grid.LineTo(25.0,  8000.0)
	grid.LineTo(25.0,  6000.0)
	grid.LineTo(20.0,  6000.0)
	grid.LineTo(20.0,  4000.0)
	grid.LineTo(15.0,  4000.0)
	grid.LineTo(15.0,  3000.0)
	grid.LineTo(10.0,  3000.0)
	grid.LineTo(10.0,  1500.0)
	grid.LineTo( 7.0,  1500.0)
	grid.LineTo( 7.0,     0.0)

	grid.DrawPath("D")
}

// }}}
// {{{ dp.DrawReferencePoint

func (g DescentPdf)DrawReferencePoint(p geo.Latlong, label string) {	
	// This is DistanceFromOrigin; it'll be wrong if plotted into grids that use DistAlongPath
	nm := p.DistNM(g.Anchor.Latlong)

	//rgb := []int{0,250,250}

	for name,grid := range g.Grids {
		grid.SetDrawColor(20,220,20)
		grid.SetLineWidth(0.3)
		grid.MoveTo(nm, grid.MinY)
		grid.LineTo(nm, grid.MaxY)
		grid.DrawPath("D")
		
		if name == "altitude" && label != "" {
			grid.SetTextColor(20,220,20)
			grid.MoveTo(nm, grid.MinY)
			grid.MoveBy(-4, 2)  // Offset in MM
			grid.MultiCell(0, 4, label, "", "", false)
			grid.DrawPath("D")
		}
	}
}

// }}}

// {{{ dp.DrawTrackWithDistFunc

type DistanceFunc func(tp fdb.Trackpoint) (float64, float64, []int)

func (g *DescentPdf)DrawTrackWithDistFunc(t fdb.Track, f DistanceFunc, colorscheme ColorScheme) {
	g.SetDrawColor(0xff, 0x00, 0x00)
	g.SetAlpha(g.LineOpacity, "")

	if len(t) == 0 { return }
	
	for i,_ := range t[1:] {
		x1,alt1,_ := f(t[i])
		x2,alt2,rgb := f(t[i+1])
		
		g.Debug += fmt.Sprintf("[%3d] (%.2fNM, %.0fft)\n", i, x1, alt1)

		g.SetLineWidth(g.LineThickness)
		g.SetDrawColor(rgb[0], rgb[1], rgb[2])

		if grid,exists := g.Grids["altitude"]; exists {
			g.SetDrawColor(rgb[0], rgb[1], rgb[2])
			grid.Line(x1,alt1, x2,alt2)
		}

		tpA,tpB := t[i],t[i+1]
		
		if g.AveragingWindow > 0 {
			tpA = t.WindowedAverageAt(i, g.AveragingWindow)
			tpB = t.WindowedAverageAt(i+1, g.AveragingWindow)
		}

		// We can re-use the dist values (x1,x2), and plot other functions on the trackpoints
		if grid,exists := g.Grids["groundspeed"]; exists {
			grid.Line(x1,tpA.GroundSpeed, x2,tpB.GroundSpeed)
		}
		if grid,exists := g.Grids["groundacceleration"]; exists {
			grid.Line(x1,tpA.GroundAccelerationKPS, x2,tpB.GroundAccelerationKPS)
		}
		if grid,exists := g.Grids["verticalspeed"]; exists {
			grid.Line(x1,tpA.VerticalSpeedFPM, x2,tpB.VerticalSpeedFPM)
		}
		if grid,exists := g.Grids["verticalacceleration"]; exists {
			grid.Line(x1,tpA.VerticalAccelerationFPMPS, x2,tpB.VerticalAccelerationFPMPS)
		}
		if grid,exists := g.Grids["angleofinclination"]; exists {
			grid.Line(x1,tpA.AngleOfInclination, x2,tpB.AngleOfInclination)
		}
	}

	g.DrawPath("D")	
	g.SetAlpha(1.0, "")
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
		return tp.DistNM(g.Anchor.Latlong), tp.IndicatedAltitude, rgb
	}

	g.Debug += fmt.Sprintf("DrawTrackAsDistanceFromOrigin\n")
	
	g.DrawTrackWithDistFunc(t, trackpointToXY, g.ColorScheme)
}

// }}}
// {{{ dp.DrawTrackAsDistanceRemainingAlongPath

// Consider distance to be distance travelled along the path.
// (E.g. if the aircraft descends in a steady spiral, we'll plot the 'unrolled' version as a
// long steady line.)
// Also, plot as 'distance remaining until destination'
func (g *DescentPdf)DrawTrackAsDistanceRemainingAlongPath(t fdb.Track) {
	// We want to render this working backwards from the anchor, so descents can line up together.
	// That means we're interested in each point's distance travelled in relation to the anchor point.
	g.Debug += fmt.Sprintf("DrawTrackAsDistanceRemainingAlongPath\n")
	iClosest := t.ClosestTo(g.Anchor.Latlong, g.AnchorAltitudeMin, g.AnchorAltitudeMax)
	if iClosest < 0 { return }
	endpointKM := t[iClosest].DistanceTravelledKM

	// If the closest point isn't all that close, assume linear flight from it to the origin
	offsetKM := t[iClosest].DistKM(g.Anchor.Latlong)

	g.Debug += fmt.Sprintf("* endKM=%.2f, offsetKM=%.2f, index=%d\n", endpointKM, offsetKM, iClosest)

	if false {
		for i,tp := range t {
			g.Debug += fmt.Sprintf("%03d: trav=%.2fKM, dist=%.2fKM, alt=%.0f\n", i,
				tp.DistanceTravelledKM, tp.DistKM(g.Anchor.Latlong), tp.IndicatedAltitude)
		}
	}
	
	trackpointToXY := func(tp fdb.Trackpoint) (float64, float64, []int) {
		distToGoKM := endpointKM - tp.DistanceTravelledKM + offsetKM
		distToGoNM := distToGoKM * geo.KNauticalMilePerKM

		rgb := []int{0,0,0}
		if g.ColorScheme == ByPlotKind { rgb = []int{250,0,0} }

		//g.Debug += fmt.Sprintf("%s\n", tp)
		
		return distToGoNM, tp.IndicatedAltitude, rgb
	}

	g.DrawTrackWithDistFunc(t, trackpointToXY, g.ColorScheme)
}

// }}}
// {{{ dp.DrawTrackAsDistanceTravelledAlongPath

// As above, but plat as 'distance travelled from Origin' (i.e. suitable for departures)
func (g *DescentPdf)DrawTrackAsDistanceTravelledAlongPath(t fdb.Track) {
	// We want to render this working backwards from the anchor, so descents can line up together.
	// That means we're interested in each point's distance travelled in relation to the anchor point.
	g.Debug += fmt.Sprintf("DrawTrackAsDistanceTravelledAlongPath\n")

	// Assume that we depart from the OriginPoint. If we're missing early datapoints, then assume
	// linear travel
	offsetKM := t[0].DistKM(g.Anchor.Latlong)

	g.Debug += fmt.Sprintf("* offsetKM=%.2f\n", offsetKM)
	
	trackpointToXY := func(tp fdb.Trackpoint) (float64, float64, []int) {		
		distTravelledKM := tp.DistanceTravelledKM + offsetKM  // ignore endpointKM ??
		distTravelledNM := distTravelledKM * geo.KNauticalMilePerKM

		rgb := []int{0,0,0}
		if g.ColorScheme == ByPlotKind { rgb = []int{250,0,0} }
		
		return distTravelledNM, tp.IndicatedAltitude, rgb
	}

	g.DrawTrackWithDistFunc(t, trackpointToXY, g.ColorScheme)
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
