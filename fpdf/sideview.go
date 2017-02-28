package fpdf

import(
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/skypies/geo"
	fdb "github.com/skypies/flightdb2"
)

var (
	BlackRGB = []int{0, 0, 0}
	RedRGB   = []int{0xff, 0, 0}
	GreenRGB = []int{0, 0xff, 0}
	BlueRGB  = []int{0, 0, 0xff}
)

type AnchorPoint struct {
	geo.NamedLatlong
	AltitudeMin      float64
	AltitudeMax      float64
	DistMaxKM        float64 // Must be at least this close, or we skip flight
}
func (ap AnchorPoint)String() string {
	return fmt.Sprintf("%s, <=%.1fKM, [%.0f-%.0f]ft", ap.NamedLatlong, ap.DistMaxKM,
		ap.AltitudeMin, ap.AltitudeMax)
}

// Given an anchor and a trackpoint, generate a distance and altitude (and render hint)
type ProjectionFunc func(tp fdb.Trackpoint, ap AnchorPoint) (distNM float64, alt float64, rgb []int)

type SideviewPdf struct {
	ToShow          map[string]bool       // Which grids to render
	Grids           map[string]*BaseGrid

	AltitudeMin     float64  // Min/max for the altitude and distance axes
	AltitudeMax     float64
	AnchorDistMinNM float64
	AnchorDistMaxNM float64

	AnchorPoint     // embedded

	AveragingWindow time.Duration

	ColorScheme     // embedded	
	LineThickness   float64
	LineOpacity     float64 // 0.0==transparent, 1.0==opaque (>1 is a thickness)

	*gofpdf.Fpdf    // embedded

	Permalink       string
	Caption         string
	Debug           string
	ShowDebug       bool
}

// {{{ svp.Init

func (g *SideviewPdf)Init() {
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
			MinX: g.AnchorDistMinNM,
			MaxX: g.AnchorDistMaxNM,
			XGridlineEvery: 10,
			Clip: true,    // set to false for debugging datasets
			InvertX: false,  // Descend to origin, on the right  // IGNORE THIS for now; redundant?
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
		if g.AnchorPoint.Name != "" {
			ng.XOriginTickFmt = "%.0fNM("+g.AnchorPoint.Name+")"
		}
		
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
			ng.YTickOtherSide = true
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
			ng.YTickOtherSide = true
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
			ng.YTickOtherSide = true
			ng.NoGridlines = true
		}

		v += 60
	}
}

// }}}
// {{{ svp.ReconfigureForDepartures

// If we are rendering departures, flip everything so the origin is on the Left Hand Side.
func (g *SideviewPdf)ReconfigureForDepartures() {
	for name,_ := range g.Grids {
		g.Grids[name].InvertX = false
		g.Grids[name].YTickOtherSide = !g.Grids[name].YTickOtherSide
	}
}

// }}}

// {{{ svp.DrawFrames

func (g SideviewPdf)DrawFrames() {
	for _,grid := range g.Grids {
		grid.DrawGridlines()
	}
}

// }}}
// {{{ svp.DrawCaption

func (g SideviewPdf)DrawCaption() {
	title := ""

	if g.AnchorPoint.Name != "" {
		title += fmt.Sprintf("* 0NM anchor: %s\n", g.AnchorPoint)
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
// {{{ svp.DrawColorSchemeKeys

func (g SideviewPdf)DrawColorSchemeKeys() {
	for _,grid := range g.Grids {
		grid.DrawColorSchemeKey()
	}
}

// }}}

// {{{ svp.MaybeDrawSFOClassB

func (g SideviewPdf)MaybeDrawSFOClassB() {
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
// {{{ svp.DrawReferencePoint

func (g SideviewPdf)DrawReferencePoint(p geo.Latlong, label string) {	
	// This is DistanceFromOrigin; it'll be wrong if plotted into grids that use DistAlongPath
	nm := p.DistNM(g.AnchorPoint.Latlong)

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

// {{{ svp.trackIndexAtAnchor

// trackIndexAtAnchor finds the index of the trackpoint closest to the
// anchor, that is within the altitude range, and also the max dist.
// If no trackpoint matches, returns -1. The float is the closest
// distance, in KM.
func (g *SideviewPdf)trackIndexAtAnchor(t fdb.Track) (int, float64) {
	i := t.ClosestTo(g.AnchorPoint.Latlong, g.AnchorPoint.AltitudeMin, g.AnchorPoint.AltitudeMax)
	if i < 0 {
		g.Debug += fmt.Sprintf("TrackIndexAtAnchor: nothing in alt range (%s)\n", g.AnchorPoint)
		return -1,0
	}

	closestDist := t[i].DistKM(g.AnchorPoint.Latlong)
	if closestDist > g.AnchorPoint.DistMaxKM {
		g.Debug += fmt.Sprintf("TrackIndexAtanchor: closest[%d] too far (%f > %f) from anchor %s\n",
			i, closestDist, g.AnchorPoint.DistMaxKM,g.AnchorPoint)
		return -1,0
	}

	return i, closestDist
}

// }}}

// {{{ svp.BuildAsCrowFliesFunc

// Consider distance as being simply the distance from the origin, and plot against altitude.
// The less the aircraft flies in a straight line to the origin, the less useful this will be.
// (E.g. if an aircraft descends in a spiral, it will plot as a zig-zag, getting closer then
// further away as it descends.)

func (g *SideviewPdf)BuildAsCrowFliesFunc(t fdb.Track) ProjectionFunc {	
	g.Debug += fmt.Sprintf("Built AsCrowFliesFunc\n")

	i,closestDist := g.trackIndexAtAnchor(t)
	if i < 0 { return nil }

	// trackpoints with a shorter dist travelled are 'before' the anchor.
	distTravelledAtAnchorKM := t[i].DistanceTravelledKM + closestDist

	g.Debug += fmt.Sprintf("* endKM=%.2f, offsetKM=%.2f, index=%d\n",
		distTravelledAtAnchorKM, closestDist, i)
	
	projectionFunc := func(tp fdb.Trackpoint, ap AnchorPoint) (float64, float64, []int) {
		distNM := tp.DistNM(ap.Latlong)
		if tp.DistanceTravelledKM < distTravelledAtAnchorKM {
			distNM *= -1.0 // go -ve, as we've not reached the anchor
		}
		return distNM, tp.IndicatedAltitude, BlackRGB
	}

	return projectionFunc
}

// }}}
// {{{ svp.BuildAlongPathFunc

// BuildAlongPathFunc returns a projection function from trackpoints
// into a scalar range 'distance along path', which is -ve for
// trackpoints before the anchor, and +ve for those after. It computes
// distance flown, so flying in circles loops make values bigger.
func (g *SideviewPdf)BuildAlongPathFunc(t fdb.Track) ProjectionFunc {
	g.Debug += fmt.Sprintf("Built AlongPathFunc\n")

	i,closestDistKM := g.trackIndexAtAnchor(t)
	if i < 0 { return nil }

	// If the closest point isn't all that close, assume linear flight
	// from it to the origin. When aircraft pass very close to the
	// anchor, this has no effect; this is mostly a trick to handle
	// flights landing at an airport when our data peters out before
	// actual touchdown.
	distTravelledAtAnchorKM := t[i].DistanceTravelledKM + closestDistKM

	g.Debug += fmt.Sprintf("* endKM=%.2f, closestDistKM=%.2f, index=%d\n",
		distTravelledAtAnchorKM, closestDistKM, i)

	projectionFunc := func(tp fdb.Trackpoint, ap AnchorPoint) (float64, float64, []int) {
		distFromAnchorKM := tp.DistanceTravelledKM - distTravelledAtAnchorKM
		distFromAnchorNM := distFromAnchorKM * geo.KNauticalMilePerKM
		return distFromAnchorNM, tp.IndicatedAltitude, BlackRGB
	}

	return projectionFunc
}

// }}}

// {{{ svp.DrawTrackProjection

func (g *SideviewPdf)DrawTrackProjection(t fdb.Track, f ProjectionFunc, colorscheme ColorScheme) {
	g.SetDrawColor(0xff, 0x00, 0x00)
	g.SetAlpha(g.LineOpacity, "")
	
	if f == nil {
		g.Debug += "**** NO FUNC\n"
		return
	}

	if len(t) == 0 { return }

	for i,_ := range t[1:] {
		x1,alt1,_ := f(t[i], g.AnchorPoint)
		x2,alt2,rgb := f(t[i+1], g.AnchorPoint)

		//g.Debug += fmt.Sprintf("[%3d] (%.2fNM, %.0fft) <%.2fKM>\n", i, x1, alt1,
		//	t[i].DistKM(g.AnchorPoint.Latlong))

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

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
