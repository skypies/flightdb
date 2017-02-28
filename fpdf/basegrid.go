package fpdf

import (
	"fmt"
	"github.com/jung-kurt/gofpdf"
)

// Describes a grid we're going to plot ov`er, and the location of its top-left corner in PDF space
type BaseGrid struct {
	*gofpdf.Fpdf        // Embed the thing we're writing to

	// Describe the portion of PDF page space the grid will be drawn over (labels go outside of this)
	OffsetU     float64 // where the origin (top-right) should be, in PDF coords (top-right)
	OffsetV     float64 // where the origin (top-right) should be, in PDF coords (top-right)
	W,H         float64 // width and height of the grid, in PDF units (should be mm)

	// Control how (x,y) vals are mapped into (u,v) vals
	InvertX,InvertY     bool    // A grid's origin defaults to bottom-left; these bools flip that
	MinX,MinY,MaxX,MaxY float64 // the range of values that should be scaled onto the grid.
	Clip                bool    // whether to clip lines to fit inside grid	
	
	// How to draw gridlines
	NoGridlines                    bool    // No lines at all for this graph
	XGridlineEvery, YGridlineEvery float64 // From Min[XY] to Max[XY]
	XMinorGridlineEvery, YMinorGridlineEvery float64 // From Min[XY] to Max[XY]
	XTickFmt,       YTickFmt       string  // Will be passed a float64 via fmt.Sprintf; blank==none
	XTickOtherSide, YTickOtherSide bool    // Note that InvertX,Y also affect where ticks go

	XOriginTickFmt,   YOriginTickFmt string  // Tick formats for the zero origin label
	
	// Other formatting
	LineColor []int // rgb, each [0,255] - axis labels
}

// {{{ bg.U, V, UV

// the bools are whether the coords are out-of-bounds for the grid.
func (bg BaseGrid)U(x float64) (float64, bool) {
	// Scale the X value to [0.0, 1.0], then map into PDF coords
	xRatio := (x - bg.MinX) / (bg.MaxX - bg.MinX)
	if bg.InvertX { xRatio = 1.0 - xRatio }

	u := bg.OffsetU + (xRatio * bg.W)
	outOfBounds := xRatio<0 || xRatio>1
	
	return u,outOfBounds
}

// the bool is whether the coords are out-of-bounds for the grid.
func (bg BaseGrid)V(y float64) (float64, bool) {
	yRatio := (y - bg.MinY) / (bg.MaxY - bg.MinY)
	if bg.InvertY { yRatio = 1.0 - yRatio }

	v := bg.OffsetV + (bg.H - (yRatio * bg.H))
	outOfBounds := yRatio<0 || yRatio>1
	
	return v,outOfBounds
}

// the bool is whether the coords are out-of-bounds for the grid.
func (bg BaseGrid)UV(x,y float64) (float64, float64, bool) {
	u,oobU := bg.U(x)
	v,oobV := bg.V(y)
	
	return u, v, (oobU || oobV)
}

// }}}
// {{{ bg.MoveBy, LineBy

func (bg BaseGrid)MoveBy(x,y float64) {
	currX,currY := bg.GetXY()
	bg.Fpdf.MoveTo(currX+x, currY+y)
}
func (bg BaseGrid)LineBy(x,y float64) {
	currX,currY := bg.GetXY()
	bg.Fpdf.LineTo(currX+x, currY+y)
}

// }}}
// {{{ bg.MaybeSet{Draw|Text}Color

func (bg BaseGrid)MaybeSetDrawColor() {
	if len(bg.LineColor) == 3 {
		bg.SetDrawColor(bg.LineColor[0], bg.LineColor[1], bg.LineColor[2])
	}
}

func (bg BaseGrid)MaybeSetTextColor() {
	if len(bg.LineColor) == 3 {
		bg.SetTextColor(bg.LineColor[0], bg.LineColor[1], bg.LineColor[2])
	}
}

// }}}

// {{{ bg.MoveTo, LineTo, Line

// We submit coords in gridspace (e.g. x,y), and the grid transforms them into PDFspace.
func (bg BaseGrid)MoveTo(x,y float64) bool {
	u,v,oob := bg.UV(x,y)
	bg.Fpdf.MoveTo(u,v)
	return oob
}

func (bg BaseGrid)LineTo(x,y float64) bool {
	u,v,oob := bg.UV(x,y)
	bg.Fpdf.LineTo(u,v)
	return oob
}

// Only draw the line if both points are inside bounds
func (bg BaseGrid)Line(x1,y1,x2,y2 float64) {
	u1,v1,oob1 := bg.UV(x1,y1)
	u2,v2,oob2 := bg.UV(x2,y2)

	if !bg.Clip || (!oob1 && !oob2) {
		bg.MaybeSetDrawColor()
		bg.Fpdf.MoveTo(u1,v1)
		bg.Fpdf.LineTo(u2,v2)
	}

	bg.DrawPath("D")
}

// }}}

// {{{ bg.DrawGridlines

func (bg BaseGrid)DrawGridlines() {
	bg.SetFont("Arial", "", 8)

	dashPattern := []float64{2,2}

	bg.SetLineWidth(0.03)
	bg.SetDrawColor(0xe0, 0xe0, 0xe0)
	for x := bg.MinX; x <= bg.MaxX; x += bg.XGridlineEvery {
		if !bg.NoGridlines {
			bg.MoveTo(x, bg.MinY)
			bg.LineTo(x, bg.MaxY)
		}
		
		if bg.XTickFmt != "" {
			if bg.XTickOtherSide {
				bg.MoveTo(x,bg.MaxY)
				bg.MoveBy(-4, -5)  // Offset in MM
			} else {
				bg.MoveTo(x,bg.MinY)
				bg.MoveBy(-4, 2)  // Offset in MM
			}
			bg.SetTextColor(0,0,0) // Should maybe be a bit more configurable
			tickfmt := bg.XTickFmt
			if x == 0 && bg.XOriginTickFmt != "" {
				tickfmt = bg.XOriginTickFmt
			}
			bg.Cell(30, float64(4), fmt.Sprintf(tickfmt, x))
			bg.DrawPath("D")
		}
	}

	if !bg.NoGridlines && bg.XMinorGridlineEvery > 0 {
		bg.SetLineWidth(0.01)
		bg.SetDashPattern(dashPattern, 0.0)
		for x := bg.MinX; x <= bg.MaxX; x += bg.XMinorGridlineEvery {
			bg.MoveTo(x, bg.MinY)
			bg.LineTo(x, bg.MaxY)
		}
		bg.DrawPath("D")
		bg.SetDashPattern([]float64{}, 0.0)
	}
	
	bg.SetLineWidth(0.03)
	bg.SetDrawColor(0xe0, 0xe0, 0xe0)
	for y := bg.MinY; y <= bg.MaxY; y += bg.YGridlineEvery {
		if !bg.NoGridlines {
			bg.MoveTo(bg.MinX, y)
			bg.LineTo(bg.MaxX, y)
		}

		align := "L"		
		if bg.YTickFmt != "" {
			if bg.YTickOtherSide {
				// By default the 'not other' side is on the right
				bg.MoveTo(bg.MinX, y)
				if bg.InvertX {
					bg.MoveBy(0.5, -2)
				} else {
					bg.MoveBy(-19, -2)
					align = "R"
				}
			} else {
				bg.MoveTo(bg.MaxX, y)
				if bg.InvertX {
					bg.MoveBy(-19, -2)
					align = "R"
				} else {
					bg.MoveBy(0.5, -2)
				}
			}

			bg.MaybeSetTextColor()
			bg.CellFormat(18, 4, fmt.Sprintf(bg.YTickFmt, y), "", 0, align, false, 0, "")
			bg.DrawPath("D")
		}
	}
	bg.DrawPath("D")

	if !bg.NoGridlines && bg.YMinorGridlineEvery > 0 {
		bg.SetLineWidth(0.01)
		bg.SetDashPattern(dashPattern, 0.0)
		for y := bg.MinY; y <= bg.MaxY; y += bg.YMinorGridlineEvery {
			bg.MoveTo(bg.MinX, y)
			bg.LineTo(bg.MaxX, y)
		}
		bg.DrawPath("D")
		bg.SetDashPattern([]float64{}, 0.0)
	}
}

// }}}
// {{{ bg.DrawColorSchemeKey

func (bg BaseGrid)DrawColorSchemeKey() {
	/*
	if g.ColorScheme == ByPlotKind {
		g.SetDrawColor(0,250,0)
		g.MoveToNMAlt(g.LengthNM,0)
		g.MoveBy(4, -2)
		g.LineBy(8, -1)
		g.MoveBy(0.5, -5)
		g.Cell(30, 10, "Distance from destination")
		g.DrawPath("D")		

		g.SetDrawColor(250,0,0)
		g.MoveToNMAlt(g.LengthNM,0)
		g.MoveBy(4, -8)
		g.LineBy(8, -1)
		g.MoveBy(0.5, -5)
		g.Cell(30, 10, "Distance along path")
		g.DrawPath("D")		
	}
*/
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
