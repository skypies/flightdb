package flightdb2

import(
	"fmt"
	"time"
	"github.com/skypies/geo"
)

// {{{ f.SatisfiesGeoRestriction

// Also return trackpoint info ?
func (f *Flight)SatisfiesGeoRestriction(gr geo.GeoRestrictor, tracks []string) (bool, TrackIntersection, string) {
	//if f.Callsign != "CCA985" { return false, fmt.Sprintf("hackety hack %s\n", f) }

	if len(tracks) == 0 {
		return f.AnyTrack().SatisfiesGeoRestriction(gr)
	} else {
		for _,tName := range tracks {
			if f.HasTrack(tName) {
				return f.Tracks[tName].SatisfiesGeoRestriction(gr)
			}
		}
		str := fmt.Sprintf("* wanted tracks %v, only had %v", tracks, f.ListTracks())
		return false, TrackIntersection{}, str
	}
}

// }}}

// {{{ t.findEntry

func (t Track)findEntry(lines []geo.LatlongLine, start int, gr geo.GeoRestrictor) (int,int,string) {
	str := ""
	for i:=start; i<len(lines); i++ {
		line := lines[i]
		if line.IsDegenerate() { continue }
//		str += fmt.Sprintf("* %03d line[%3d,%3d] == %v (look for entry)\n", i, line.I, line.J,
//			gr.IntersectsLine(line))
		if gr.IntersectsLine(line) == true {
						// The state change happens in this line segment - track down the point at which happens
			if line.J-line.I > 1 {
				for i:=line.I+1; i<line.J; i++ {
					subLine := t[i-1].LineTo(t[i].Latlong)
					subLine.I, subLine.J = i-1, i
					if gr.IntersectsLine(subLine) == true {
						line = subLine
						str += fmt.Sprintf("* zoomed into [%3d,%3d]\n", line.I, line.J)
						break
					}
				}
			}

			// line.I and line.J are adjacent points, at which we stop/start intersecting. Return
			// the index of the contained point.
			str += fmt.Sprintf("* returning [%d] for lookFor=entry\n", line.J)
			return line.J,i,str // We were looking to start intersecting - so second point is inside
		}
	}

	// We never found the point we were looking for.
	str += fmt.Sprintf("* nomatch for entry :( start=%d)\n", start)
	return -1,0,str
}

// }}}
// {{{ t.findExit

func (t Track)findExit(lines []geo.LatlongLine, start int, gr geo.GeoRestrictor) (int,int,string) {
	str := ""
	for i:=start; i<len(lines); i++ {
		line := lines[i]
		if line.IsDegenerate() { continue }
		//str += fmt.Sprintf("* %03d line[%3d,%3d] == %v (look for exit)\n", i, line.I, line.J,
		//	gr.IntersectsLine(line))

		if gr.IntersectsLine(line) == false {
			_,deb := gr.IntersectsLineDeb(line)
			str += fmt.Sprintf("* Big Bingo\n* line=%s\n* gr=%s\n%s", line, gr,deb)
			// This is (presumably?) the first line that does not satisfy; it lies outside.
			// So we the previous line, the final line that did somehow intersect, is the
			// interesting one; we should zoom into that.
			if i == 0 || lines[i-1].IsDegenerate() {
				str += "* bad data, i==0 (or degenerate), but looking for exit\n"
				return -1, -1, str
			}
			line = lines[i-1]
			
			// The state change happens in this line segment - track down the point at which happens
			if line.J-line.I > 1 {
				// Walk backwards (from not intersecting) until we intersect
				for i:=line.J; i>line.I; i-- {
					subLine := t[i-1].LineTo(t[i].Latlong)
					subLine.I, subLine.J = i-1, i
					if gr.IntersectsLine(subLine) == true {
						line = subLine
						str += fmt.Sprintf("* zoomed into [%3d,%3d]\n", line.I, line.J)
						break
					}
				}
			}

			// line.I and line.J are adjacent points, at which we stop/start intersecting. Return
			// the index of the contained point.
			str += fmt.Sprintf("* returning [%d] for lookFor=exit\n", line.I)
			return line.I,i,str // We were looking to stop intersecting - so first point is inside
		}
	}

	// We never found the point we were looking for.
	str += fmt.Sprintf("* nomatch :( start=%d)\n", start)
	return -1,0,str
}

// }}}
// {{{ t.SatisfiesGeoRestriction

func (t Track)SatisfiesGeoRestriction(gr geo.GeoRestrictor) (bool, TrackIntersection, string) {
	lines := t.AsLinesSampledEvery(time.Second * 5)
	str := fmt.Sprintf("** %s\n** Geo   %s\n", t, gr)

	iEntry,iLine,deb := t.findEntry(lines, 0, gr)
	str += deb
	
	if iEntry < 0 {
		str = "* No entry point found\n"
		return false, TrackIntersection{}, str
	}

	if ! gr.LookForExit() {
		str += "* Not looking for exit\n"
		t[iEntry].AnalysisAnnotation += fmt.Sprintf("* Sole point to satisfy\n")
		return true, TrackIntersection{Start:t[iEntry], I:iEntry}, str
	}

	iExit,_,deb := t.findExit(lines, iLine, gr)
	str += deb

	if iExit < 0 {
		str += "* No exit found\n"
		iExit = len(t)-1
	}  // track ran out before we exited; pick the end.

	for i:=iEntry; i<=iExit; i++ {
		t[i].AnalysisAnnotation += fmt.Sprintf("* Point satisfied georestriction %s\n", gr)
	}
	t[iEntry].AnalysisAnnotation += fmt.Sprintf("* First point to satisfy\n")
	t[iExit].AnalysisAnnotation += fmt.Sprintf("* Last point to satisfy\n")
	
	return true, TrackIntersection{Start:t[iEntry], End:t[iExit], I:iEntry, J:iExit}, str
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
