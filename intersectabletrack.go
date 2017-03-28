package flightdb

import(
	"fmt"
	"time"

	"github.com/skypies/geo"
	quadgeo  "github.com/paulmach/go.geo"
	"github.com/paulmach/go.geo/quadtree"
)

// IntersectableTrack contains a track, and data structures for efficient intersections against
// geo.Restrictors
type IntersectableTrack struct {
	Track    // embed
	l      []geo.LatlongLine // an acceleration structure (coarse lines)
	qt      *quadtree.Quadtree
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
// {{{ t.ToIntersectableTrack[SampleEvery]

func (t Track)ToIntersectableTrackSampleEvery(d time.Duration) IntersectableTrack {
	it := IntersectableTrack{Track:t}
	it.l = t.AsLinesSampledEvery(d)
	it.qt = t.ToQuadtree()
	return it
}

func (t Track)ToIntersectableTrack() IntersectableTrack {
	return t.ToIntersectableTrackSampleEvery(time.Second * 2)
}

// }}}
// {{{ t.ToQuadtree

func ll2pt(ll geo.Latlong) *quadgeo.Point { return quadgeo.NewPoint(ll.Long, ll.Lat) }
func bbox2bound(b geo.LatlongBox) *quadgeo.Bound { 
	return quadgeo.NewBoundFromPoints(ll2pt(b.SW), ll2pt(b.NE))
}
type QTPoint struct {
	I int // index into the track (should move this into the trackpoint ?!)
	Trackpoint
}
func (qtp QTPoint)Point() *quadgeo.Point { return ll2pt(qtp.Trackpoint.Latlong) }

func (t Track)ToQuadtree() *quadtree.Quadtree {
	qt := quadtree.New(bbox2bound(t.FullBoundingBox()).Pad(0.05))

	for i,_ := range t {
		qt.Insert(QTPoint{i,t[i]})
	}
	
	return qt
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

// Area restrictions: the first and last contained points are returned.
// Edge cases:
// 1. No points actually inside area: if the line between two points intersects (and the far end
//    meets any altitude limits), then those two points are returned.
// 2. One point is inside: then that single point is returned
//
// Vertical planes: the first point on the 'other side' is returned, if (a) it meeets altitude
//  restrictions, and (b) the line to it from previous point intersects the bounded plane.
func (it IntersectableTrack)SatisfiesRestrictor(gr geo.Restrictor) RestrictorIntersectOutcome {
	outcome := it.satisfiesQuadtree(gr)

	outcome.Debug += gr.GetDebug()
	return outcome
}

// }}}

// {{{ it.findSubtrackWithQuadtree

// Returns a pruned track that may intersect the restrictor, and how many trackpoints were
// skipped to get there (or -1 if no subtrack found)
// Unhandled edge cases:
// * no points inside area. But pads box by 0.5KM, which is prob as much as we should trust ...
// * a track that goes in, then out, then in and out again; we include the 'out' bits
func (it IntersectableTrack)findSubtrackWithQuadtree(gr geo.Restrictor) (Track,int) {
	if it.qt == nil { return Track{},-1 }

	bounded := it.qt.InBound(bbox2bound(gr.BoundingBox()).GeoPad(500)) // GeoPad in meters
	if len(bounded) == 0 {
		gr.Debugf("** qt: no points found inside bounding box\n")
		return Track{},-1
	}

	gr.Debugf("** qt: have bounded %d points:-\n", len(bounded))

	if len(bounded) == 1 {
		qtp := bounded[0].(QTPoint)
		if gr.OverlapsAltitude(int64(qtp.Trackpoint.Altitude)).IsDisjoint() {
			gr.Debugf("** qt: sole point failed altitude\n")
			return Track{},-1
		}
		gr.Debugf("** qt: sole point; I,J=%d\n", qtp.I)
		return append([]Trackpoint{}, it.Track[qtp.I]), qtp.I
	}

	iStart,iEnd := 9999999,-9999999
	for _,ptr := range bounded {
		qtp := ptr.(QTPoint)
		//gr.Debugf("[%3d] %s\n", qtp.I, qtp.Trackpoint)

		if qtp.I < iStart { iStart = qtp.I }
		if qtp.I > iEnd   { iEnd = qtp.I }
	}

	// Now, extend to include the next points on either end of the track (i.e. the closest that
	// is also outside of the pad. This lets possibly very long lines be created between far
	// points, which help catch more intersections.
	if iStart > 0 { iStart-- }
	if iEnd < len(it.Track)-1 { iEnd++ }
	
	gr.Debugf("** qt: found range [%d,%d)\n", iStart, iEnd)
	if iEnd < 0 {
		return Track{},-1
	}

	return it.Track[iStart:iEnd+1], iStart
}

// }}}
// {{{ it.satisfiesQuadtree

// Unhandled edge cases:
// * no points inside area. (closest to ?)
// * a track that goes in, then out, then in and out again; we include the 'out' bits
func (it IntersectableTrack)satisfiesQuadtree(gr geo.Restrictor) RestrictorIntersectOutcome {
	if it.qt == nil {	return RestrictorIntersectOutcome{} }

	subTrack,offset := it.findSubtrackWithQuadtree(gr)
	outcome := IntersectableTrack{Track: subTrack}.satisfiesRefImpl(gr)

	// If we satisified a non-exluder, or failed to satisfy an excluder, then we had an intersection
	// we should fixup
	if outcome.Satisfies != gr.IsExclusion() {
		// outcome is relative to the subtrack; make it relative to the original track
		outcome.I += offset
		outcome.J += offset
		outcome.Start = it.Track[outcome.I]
		outcome.End = it.Track[outcome.J]

		// Decorate trackpoints
		it.Track[outcome.I].AnalysisDisplay = AnalysisDisplayHighlight
		it.Track[outcome.J].AnalysisDisplay = AnalysisDisplayHighlight
		for i:=outcome.I; i<=outcome.J; i++ {
			it.Track[i].AnalysisAnnotation += fmt.Sprintf("* Point satisfied georestriction %s\n", gr)
		}
		it.Track[outcome.I].AnalysisAnnotation += fmt.Sprintf("* First point to satisfy\n")
		it.Track[outcome.J].AnalysisAnnotation += fmt.Sprintf("* Last point to satisfy\n")
	}

	return outcome
}

// }}}

// {{{ it.satisfiesRefImpl

// A correct(?!) but inefficient reference implementation
func (it IntersectableTrack)satisfiesRefImpl(gr geo.Restrictor) RestrictorIntersectOutcome {
	gr.Debugf("** satisfiesRefImpl, on %d points\n", len(it.Track))
	if gr.CanContain() {
		return it.satisfiesAreaRefImpl(gr)
	}
	return it.satisfiesLineRefImpl(gr)
}

// }}}
// {{{ it.satisfiesAreaRefImpl

func (it IntersectableTrack)satisfiesAreaRefImpl(gr geo.Restrictor) RestrictorIntersectOutcome {
	out := RestrictorIntersectOutcome{}

	lookingForEntry := true
	foundExit := false
	gr.Debugf("** starting line-intersect crawl [%d points]\n", len(it.Track))

	// If the track is a singleton point - and it meets the gr - then return it direct. Singleton
	// tracks will happen more than you'd think because we trim the track via the quadtree.
	if len(it.Track) == 1 && gr.Contains(it.Track[0].Latlong) && !gr.OverlapsAltitude(int64(it.Track[0].Altitude)).IsDisjoint() {
		out.I = 0
		out.J = 0
		out.Start = it.Track[out.I]
		out.End = it.Track[out.J]
		out.Satisfies = true
	}
	
	for i:=1; i<len(it.Track); i++ {
		ln := it.Track[i-1].LineTo(it.Track[i].Latlong)
		hOverlap := gr.OverlapsLine(ln)
		vOverlap := gr.OverlapsAltitude(int64(it.Track[i].Altitude))
		
		if lookingForEntry {
			if hOverlap.IsDisjoint() { continue }
			gr.Debugf("** crawl: [%d,%d] is (first?) horizontal intersecting line (h:%v, v:%v)\n",
				i-1, i, hOverlap, vOverlap)				
				
			if vOverlap.IsDisjoint() {
				gr.Debugf("** crawl: no vertical overlap, so moving on (%v)\n", vOverlap)
				continue
			}
	
			lookingForEntry = false
			out.I = i

			if hOverlap == geo.OverlapR2Contains {
				gr.Debugf("** crawl: oh - line fully contains gr - picking uncontained points\n")
				out.I = i-1
				out.J = i
				out.End = it.Track[out.J]
				foundExit = true
			}
			out.Start = it.Track[out.I]
			out.Satisfies = true
		}

		// Note - ln might be both out entry & exit line. If so (e.g. out.I == i), then
		// treat a R2StraddlesStart as an R2IsContained - they both mean the endpoint is inside.
		if !lookingForEntry && !foundExit{

			if vOverlap.IsDisjoint() {
				gr.Debugf("** crawl: no vertical overlap, so we have our exit (%v)\n", vOverlap)
			} else {
				if out.I == i && hOverlap == geo.OverlapR2StraddlesStart { continue }
				if hOverlap == geo.OverlapR2IsContained { continue }
			}

			gr.Debugf("** crawl: [%d,%d] is first non-contained line (%v)\n", i-1,i,hOverlap)
			out.J = i-1
			out.End = it.Track[out.J]
			foundExit = true
			break // all done
		}
	}

	// check we didn't start inside the thing 
	if out.I == 1 && gr.Contains(it.Track[0].Latlong) {
		gr.Debugf("** crawl: start was contained, resetting I to 0\n")
		out.I = 0
		out.Start = it.Track[out.I]
	}
	
	// Fell off end; take last point
	if !lookingForEntry && !foundExit && len(it.Track) > 0 {
		gr.Debugf("** crawl: fell off end, setting J to end of track\n")
		out.J = len(it.Track) - 1
		out.End = it.Track[out.J]
	}
	
	return out
}

// }}}
// {{{ it.satisfiesLineRefImpl

// TODO: consider interpolation to get a better approximation of the altitude at the point
// of intersection, rather than just picking the point on the 'other side' of the vertical plane
func (it IntersectableTrack)satisfiesLineRefImpl(gr geo.Restrictor) RestrictorIntersectOutcome {
	out := RestrictorIntersectOutcome{}

	gr.Debugf("** starting dumb intersection checks for non-area\n")

	for i:=1; i<len(it.Track); i++ {
		ln := it.Track[i-1].LineTo(it.Track[i].Latlong)
		hOverlap := gr.OverlapsLine(ln)
		vOverlap := gr.OverlapsAltitude(int64(it.Track[i].Altitude))

		if ! hOverlap.IsDisjoint() {
			if vOverlap.IsDisjoint() {
				// Our desired point isn't at the right altitude; we're all done.
				gr.Debugf("** found a dumb intersection at [%d,%d], but no vert overlap (%.0fft == %v)\n",
					i-1,i, it.Track[i].Altitude, vOverlap)
				break
			}

			gr.Debugf("** found a dumb intersection at [%d,%d], good vert overlap (%v)\n", i-1,i,
				vOverlap)

			out.I = i
			out.Start = it.Track[out.I]
			out.Satisfies = true
			break
		}
	}
	
	return out
}

// }}}

// The old 'line accelerated' omni-imlementation. Altitude checks aren't fully implemented.
// I think the quadtree thing mostly trumps all this now.
// {{{ it.satisfiesWithLineApproximations

func (it IntersectableTrack)satisfiesWithLineApproximations(gr geo.Restrictor) RestrictorIntersectOutcome {	
	out := RestrictorIntersectOutcome{}

	// First: look for the first non-outside line segment that meets altitude restrictions
	iEntryLine := -1
	var overlap geo.OverlapOutcome
	for i,line := range it.l {
		if line.IsDegenerate() { continue }
		vOverlap1 := gr.OverlapsAltitude(int64(it.Track[line.I].Altitude))
		vOverlap2 := gr.OverlapsAltitude(int64(it.Track[line.J].Altitude))
		overlap = gr.OverlapsLine(line)

		// ARGH. If we only look at v2, and ensure the endpoint meets altitude restrictions,
		// we miss a vertical plane intersection where track is descending but just clips window
		// midway through a multi-point line.

		// But if we look at both, and see {intersects,disjoint}, then a test finds a match
		// that it shouldn't:
		//     line climbing high; l[1] intersects the area, but an interpolation to the point of
		//     intersection would show the altitude was too high (~300').
		// Maybe we should give up on this ? It's an "if you interpolate" test, after all.
		
		if vOverlap1.IsDisjoint() && vOverlap2.IsDisjoint() {
			//gr.Debugf("* %03d line[%3d,%3d] vertical == %v,%v (looking for !Disjoint)\n",
			//	i, line.I, line.J, vOverlap1, vOverlap2)
			continue
		}

		//gr.Debugf("* %03d line[%3d,%3d] == %v v(%v,%v) (looking for !Disjoint)\n",
		//	i, line.I, line.J, overlap, v1, v2)

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
		vOverlap := gr.OverlapsAltitude(int64(it.Track[out.TrackIntersection.I].Altitude))
		if vOverlap.IsDisjoint() {
			gr.Debugf("** Restriction can't contain, but fails altitude with %v\n", vOverlap)
			out.Satisfies = false

		} else {
			gr.Debugf("** Restriction can't contain, so we're all done\n")
			it.Track[out.TrackIntersection.I].AnalysisDisplay = AnalysisDisplayHighlight
			it.Track[out.TrackIntersection.I].AnalysisAnnotation +=
				fmt.Sprintf("* Point satisfied georestriction %s\n", gr)
		}
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
// {{{ it.findEntry

// We know the first contained point is somewhere in [ln.I,ln.J].
// Figure out which one.
func (it IntersectableTrack)findEntry(ln geo.LatlongLine, overlap geo.OverlapOutcome, gr geo.Restrictor) int {	
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
// {{{ it.findExit

// We know the final contained point is somewhere in [ln.I,ln.J].
// Figure out which one. (It feels like this should be a reverse, then call to .findEntry() ...)
func (it IntersectableTrack)findExit(ln geo.LatlongLine, overlap geo.OverlapOutcome, gr geo.Restrictor) int {	
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
