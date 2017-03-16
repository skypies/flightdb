package fpdf

import(
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/skypies/geo"
	fdb "github.com/skypies/flightdb"
)

var (
	BlackRGB = []int{0, 0, 0}
	RedRGB   = []int{0xff, 0, 0}
	GreenRGB = []int{0, 0xff, 0}
	BlueRGB  = []int{0, 0, 0xff}
)

type SideviewPdf struct {
	ToShow          map[string]bool       // Which grids to render
	Grids           map[string]*BaseGrid

	AltitudeMin     float64  // Min/max for the altitude and distance axes
	AltitudeMax     float64
	AnchorDistMinNM float64
	AnchorDistMaxNM float64

	AnchorPoint     // embedded
	TrackProjector  // embedded

	AveragingWindow time.Duration

	ColorScheme     // embedded	
	LineThickness   float64
	LineOpacity     float64 // 0.0==transparent, 1.0==opaque (>1 is a thickness)

	*gofpdf.Fpdf    // embedded

	Permalink       string
	MapPermalink    string
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
		title += fmt.Sprintf("* Projection:%s. 0NM anchor: %s.\n",
			g.TrackProjector.Description(), g.AnchorPoint)
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
		g.MoveTo(180, 5)
		g.CellFormat(20, 4, "[Permalink]", "", 0, "", false, 0, g.Permalink)

		g.MoveTo(200, 5)
		g.CellFormat(20, 4, "[Map]", "", 0, "", false, 0, g.MapPermalink)
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

// Only valid if the grid is anchored to SFO.
func (g SideviewPdf)MaybeDrawSFOClassB() {
	grid,exists := g.Grids["altitude"]
	if !exists { return }

	grid.SetDrawColor(0x00, 0x00, 0x66)
	grid.SetLineWidth(0.45)

	// Should really parse this all out of geo/sfo.SFOClassBMap ...
	grid.MoveTo(  0.0, 10000.0)
	grid.LineTo(-30.0, 10000.0)
	grid.LineTo(-30.0,  8000.0)
	grid.LineTo(-25.0,  8000.0)
	grid.LineTo(-25.0,  6000.0)
	grid.LineTo(-20.0,  6000.0)
	grid.LineTo(-20.0,  4000.0)
	grid.LineTo(-15.0,  4000.0)
	grid.LineTo(-15.0,  3000.0)
	grid.LineTo(-10.0,  3000.0)
	grid.LineTo(-10.0,  1500.0)
	grid.LineTo( -7.0,  1500.0)
	grid.LineTo( -7.0,     0.0)

	grid.LineTo(  0.0,     0.0)
	grid.LineTo(  0.0, 10000.0)

	grid.DrawPath("D")
}

// }}}
// {{{ svp.DrawPointProjectedIntoTrack

// This is trickier than it looks. We find the trackpoint closest to the refpt, then
// project it via whatever projector we're using in this sideview.
func (g *SideviewPdf)DrawPointProjectedIntoTrack(t fdb.Track, p geo.Latlong, label string) {	
	ap := AnchorPoint{NamedLatlong:geo.NamedLatlong{Latlong:p}}	
	i,_,err := ap.PointOfClosestApproach(t)
	if err != nil { return }

	nm,_ := g.TrackProjector.Project(t[i])

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

// {{{ svp.DrawProjectedTrack

func (g *SideviewPdf)DrawProjectedTrack(t fdb.Track, colorscheme ColorScheme) error {	
	if len(t) == 0 {
		return nil
	}
	if g.TrackProjector == nil {
		return fmt.Errorf("DrawTrackProjection: no projector in ")
	}
	if err := g.TrackProjector.Setup(t, g.AnchorPoint); err != nil {
		return err
	}

	g.SetDrawColor(0xff, 0x00, 0x00)
	g.SetAlpha(g.LineOpacity, "")
		
	for i,_ := range t[1:] {
		x1,alt1 := g.TrackProjector.Project(t[i])
		x2,alt2 := g.TrackProjector.Project(t[i+1])

		g.SetLineWidth(g.LineThickness)

		if grid,exists := g.Grids["altitude"]; exists {
			grid.Line(x1,alt1, x2,alt2)
		}

		// Smooth out the speeds & accelerations using an averaging window
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

	return nil
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
