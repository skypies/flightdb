package flightdb2

import(
	"fmt"
	"time"

	"github.com/skypies/geo"
)

// IntersectableTrack contains a track, and data structures for efficient intersections against
// geo.Restrictors
type IntersectableTrack struct {
	Track    // embed
	l      []geo.LatlongLine // an acceleration structure (coarse lines)
}

// RestrictorIntersectOutcome represesnts the findings from comparing against a geo.Restrictor
type RestrictorIntersectOutcome struct {
	TrackIntersection  // a line segment, relating to a specific track.
	Satisfies          bool
	Debug              string
}

// RestrictorSetIntersectOutcome gathers together the outcomes from the individual restrictors
type RestrictorSetIntersectOutcome struct {
	Outcomes     []RestrictorIntersectOutcome 
}
// {{{ o.Satisfies

// Satsifies applies the combination logic to the individual outcomes to get a final outcome.
func (o RestrictorSetIntersectOutcome)Satisfies(logic RestrictorCombinationLogic) bool {
	switch logic {
	case CombinationLogicAll:
		for _,outcome := range o.Outcomes {
			if !outcome.Satisfies {
				return false
			}
		}
		return true

	case CombinationLogicAny:
		for _,outcome := range o.Outcomes {
			if outcome.Satisfies {
				return true
			}
		}
		return false
	}

	return false // "should never happen"
}

// }}}
// {{{ o.BlameString

func (o RestrictorSetIntersectOutcome)BlameString(grs GeoRestrictorSet) string {
	if len(grs.R) == 1 {
		return "Did not satisfy "+grs.R[0].String()
	}

	if grs.Logic == CombinationLogicAll {
		for i,outcome := range o.Outcomes {
			if ! outcome.Satisfies {
				return "Did not satisfy "+grs.R[i].String()
			}
		}
		return "cannot assign blame!?"

	} else {
		return  "Did not satisfy any restriction in "+grs.Name
	}
}

// }}}
// {{{ o.Debug

func (o RestrictorSetIntersectOutcome)Debug() string {
	str := ""
	for i,outcome := range o.Outcomes {
		str += fmt.Sprintf("----/ outcome %02d /----\n%s\n", i, outcome.Debug)
	}
	return str
}

// }}}

// {{{ f.GetIntersectableTrack

// GetIntersectableTrack contains the logic that decides which particular track(s) within a flight
// is best suited for georestriction analysis. Those tracks are potentially mutated, and the
// output is returned as a track with pre-computed junk.
func (f *Flight)GetIntersectableTrack() IntersectableTrack {
	tName, t := f.PreferredTrack([]string{"FOIA", "ADSB", "MLAT", "fr24"})
	if tName == "" {
		return IntersectableTrack{}
	}

	t.PostProcess()
	return t.ToIntersectableTrack()
}

// }}}
// {{{ t.ToIntersectableTrack

func (t Track)ToIntersectableTrack() IntersectableTrack {
	it := IntersectableTrack{Track:t}
	it.l = t.AsLinesSampledEvery(time.Second * 2)

	return it
}

// }}}

// {{{ it.SatisfiesRestrictorSet

func (it IntersectableTrack)SatisfiesRestrictorSet(grs GeoRestrictorSet) RestrictorSetIntersectOutcome {
	out := RestrictorSetIntersectOutcome{}

	for _,gr := range grs.R {
		out.Outcomes = append(out.Outcomes, it.SatisfiesRestrictor(gr))
	}

	return out
}

// }}}
// {{{ it.SatisfiesRestrictor

func (it IntersectableTrack)SatisfiesRestrictor(gr geo.NewRestrictor) RestrictorIntersectOutcome {
	out := RestrictorIntersectOutcome{}

	// First: look for the first non-outside line segment
	iEntryLine := -1
	var overlap geo.OverlapOutcome
	for i,line := range it.l {
		if line.IsDegenerate() { continue }
		overlap = gr.OverlapsLine(line)
		//gr.Debugf("* %03d line[%3d,%3d] == %v (looking for !Disjoint)\n", i, line.I, line.J, overlap)

		if ! overlap.IsDisjoint() {
			iEntryLine = i
			break
		}
	}

	if iEntryLine < 0 {
		gr.Debugf("** No entry point found\n")
		out.Debug = gr.GetDebug()
		return out
	}

	gr.Debugf("  [entry point found in line [%d,%d]]\n", it.l[iEntryLine].I, it.l[iEntryLine].J, )
	out.Satisfies = true
	out.TrackIntersection.I = it.findEntry(it.l[iEntryLine], overlap, gr)
	out.TrackIntersection.Start = it.Track[out.TrackIntersection.I]

	if !gr.CanContain() {
		// TODO: Altitude Checks ??
		gr.Debugf("** Restriction can't contain, so we're all done\n")
		it.Track[out.TrackIntersection.I].AnalysisDisplay = AnalysisDisplayHighlight
		it.Track[out.TrackIntersection.I].AnalysisAnnotation +=
			fmt.Sprintf("* Point satisfied georestriction %s\n", gr)
		out.Debug = gr.GetDebug()
		return out
	}

	iStartLine := iEntryLine+1
	// If this line contains the gr, then it will also contain our exit point, so include it
	if overlap == geo.OverlapR2Contains || overlap == geo.OverlapR2StraddlesEnd {
		gr.Debugf("** entryline is also exit line; start with it\n")
		iStartLine--
	}
	
	// Discard all lines that precede our entry line (keep that one, as it might also be exit line).
	// Now look for the first non-fully-contained line
	iExitLine := -1
	for i,line := range it.l[iStartLine:] {
		if line.IsDegenerate() { continue }
		overlap = gr.OverlapsLine(line)
		//gr.Debugf("* %03d line[%3d,%3d] == %v (looking for !Contained)\n",
		//	i+iStartLine, line.I, line.J, overlap)

		if overlap != geo.OverlapR2IsContained {
			iExitLine = i
			break
		}
	}

	if iExitLine < 0 {
		gr.Debugf("** No exit point found; assume final point of track\n")
		out.TrackIntersection.J = len(it.Track)-1
		out.TrackIntersection.End = it.Track[out.TrackIntersection.J]

	} else {
		gr.Debugf("  [exit point found in line [%d,%d]]\n", it.l[iExitLine].I, it.l[iExitLine].J, )
		out.TrackIntersection.J = it.findExit(it.l[iStartLine+iExitLine], overlap, gr)
		out.TrackIntersection.End = it.Track[out.TrackIntersection.J]
	}

	// TODO: Altitude checks ?
	
	// Decorate trackpoints
	it.Track[out.TrackIntersection.I].AnalysisDisplay = AnalysisDisplayHighlight
	it.Track[out.TrackIntersection.J].AnalysisDisplay = AnalysisDisplayHighlight
	for i:=out.TrackIntersection.I; i<=out.TrackIntersection.J; i++ {
		it.Track[i].AnalysisAnnotation += fmt.Sprintf("* Point satisfied georestriction %s\n", gr)
	}
	it.Track[out.TrackIntersection.I].AnalysisAnnotation += fmt.Sprintf("* First point to satisfy\n")
	it.Track[out.TrackIntersection.J].AnalysisAnnotation += fmt.Sprintf("* Last point to satisfy\n")
	
	if gr.IsExclusion() {
		out.Satisfies = !out.Satisfies  // ... unless this gr is supposed to exclude tracks
	}

	gr.Debugf("**** Final outcome generated: %v=[%d,%d]\n", out.Satisfies, out.I, out.J)
	out.Debug = gr.GetDebug()
	return out
}

// }}}

// {{{ findEntry

// We know the first contained point is somewhere in [ln.I,ln.J].
// Figure out which one.
func (it IntersectableTrack)findEntry(ln geo.LatlongLine, overlap geo.OverlapOutcome, gr geo.NewRestrictor) int {	
	if overlap == geo.OverlapR2StraddlesEnd || overlap == geo.OverlapR2IsContained {
		gr.Debugf("** start point is contained, pick I over J for entry\n")
		return ln.I
	}


	// A nicer algorithm would be a binary chop using gr.Contains(), but we need to do it this
	// ugly way to handle windows (which have no .Contains), so may as well use it for everything.
	
	// If this line has multiple points, find out more precisely which is the first contained point
	if ln.J-ln.I > 1 {
		// Create sublines between each pair of points, walking in until we find the straddler
		for i:=ln.I+1; i<=ln.J; i++ {
			subLine := it.Track[i-1].LineTo(it.Track[i].Latlong)
			subLine.I, subLine.J = i-1, i

			if subLine.IsDegenerate() { continue }
			overlap = gr.OverlapsLine(subLine)
			gr.Debugf("- %03d subLine[%3d,%3d] == %v (looking for straddler)\n", i,
				subLine.I, subLine.J, overlap)

			// OverlapR2IsContained should not crop up; the preceding line should have been found
			if overlap == geo.OverlapR2StraddlesStart {
				gr.Debugf("  [straddling pair found! returning .J=%d]\n",subLine.J)
				return subLine.J
			}
		}
		gr.Debugf("-- NO SUBLINE FOUND ?\n")

	} else if overlap == geo.OverlapR2Contains {
		// This is a real corner case; the line intersects the gr, but no trackpoints lie within it.
		// So we end up identfying the trackpoints immediately outside the area. It would be nice to
		// somehow indicate we're doing this as a flag in the TrackIntersection.
		gr.Debugf("** line fully contains gr, with no sublines; pick I over J for entry\n")
		return ln.I
	}
	
	return ln.J // Default; assume the line straddles with I outside, J inside
}

// }}}
// {{{ findExit

// We know the final contained point is somewhere in [ln.I,ln.J].
// Figure out which one. (It feels like this should be a reverse, then call to .findEntry() ...)
func (it IntersectableTrack)findExit(ln geo.LatlongLine, overlap geo.OverlapOutcome, gr geo.NewRestrictor) int {	
	if overlap == geo.OverlapR2StraddlesStart || overlap == geo.OverlapR2IsContained {
		gr.Debugf("** end point is contained, pick J over I for exit\n")
		return ln.J
	}

	// If this line has multiple points, find out more precisely which is the first contained point
	if ln.J-ln.I > 1 {
		// Create sublines between each pair of points, walking back until we find the straddler
		for i:=ln.J; i>ln.I; i-- {

			subLine := it.Track[i-1].LineTo(it.Track[i].Latlong)
			subLine.I, subLine.J = i-1, i

			if subLine.IsDegenerate() { continue }
			overlap = gr.OverlapsLine(subLine)
			gr.Debugf("- %03d subLine[%3d,%3d] == %v (looking for straddler)\n", i,
				subLine.I, subLine.J, overlap)

			// OverlapR2IsContained should not crop up; the preceding line should have been found
			if overlap == geo.OverlapR2StraddlesEnd {
				gr.Debugf("  [straddling pair found! returning .I=%d]\n", subLine.I)
				return subLine.I
			}
		}
		gr.Debugf("-- NO SUBLINE FOUND ?\n")

	} else if overlap == geo.OverlapR2Contains {
		// This is a real corner case; the line intersects the gr, but no trackpoints lie within it.
		// So we end up identfying the trackpoints immediately outside the area. It would be nice to
		// somehow indicate we're doing this as a flag in the TrackIntersection.
		gr.Debugf("** line fully contains gr, with no sublines; pick J over I for exit\n")
		return ln.J
	}
	
	return ln.I // Assume the line straddles with I inside, J outside
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
