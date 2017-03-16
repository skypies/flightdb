package flightdb

import "fmt"

type TrackIntersection struct {
	Start,End Trackpoint
	TrackName string
	I,J int
}

func (ev TrackIntersection)String() string {
	itp := ev.Start.InterpolateTo(ev.End, 0.5)
	return fmt.Sprintf("%.1fNM(%s), Alt(avg=%.0f,delta=%.0f), groundspeed=(avg=%.0f,delta=%.0f)",
		ev.Start.Latlong.DistNM(ev.End.Latlong),
		ev.End.TimestampUTC.Sub(ev.Start.TimestampUTC),
		itp.Altitude, (ev.End.Altitude - ev.Start.Altitude),
		itp.GroundSpeed, (ev.End.GroundSpeed - ev.Start.GroundSpeed))
}

func (ti TrackIntersection)IsPointIntersection() bool { return ti.J == 0 }

func (ev TrackIntersection)RowHTML() []string {
	if ev.IsPointIntersection() {
		return []string{
			"<b>Alt</b>",
			fmt.Sprintf("%.0f", ev.Start.Altitude),
			"<b>GroundSpeed</b>",
			fmt.Sprintf("%.0f", ev.Start.GroundSpeed),
			"<b>Vert(fpm)</b>",
			fmt.Sprintf("%.0f", ev.Start.VerticalRate),
		}
	}

	itp := ev.Start.InterpolateTo(ev.End, 0.5)
	s := []string{
		//fmt.Sprintf("[I=%d,J=%d]", ev.I,ev.J),
		"<b>Duration</b>",
		fmt.Sprintf("%.0fs", ev.End.TimestampUTC.Sub(ev.Start.TimestampUTC).Seconds()),

		"<b>Alt(start,end,delta,avg)</b>",
		fmt.Sprintf("%.0f", ev.Start.Altitude),
		fmt.Sprintf("%.0f", ev.End.Altitude),
		fmt.Sprintf("%.0f", (ev.End.Altitude - ev.Start.Altitude)),
		fmt.Sprintf("%.0f", itp.Altitude),
/*
		"<b>GroundSpeed</b>",
		fmt.Sprintf("%.0f", ev.Start.GroundSpeed),
		fmt.Sprintf("%.0f", ev.End.GroundSpeed),
		fmt.Sprintf("%.0f", (ev.End.GroundSpeed - ev.Start.GroundSpeed)),
		fmt.Sprintf("%.0f", itp.GroundSpeed),
*/
	}
	return s
}





/* This is fully obseleted by the SatisfiesGeoRestriction stuff, which is faster

// Will annotate the trackpoints in-place.
// Note that an intersection may have only one point inside it.
// Note this is rubbish; it should really build lines between trackpoints and intersect those, or
// something.

func (track Track)IntersectWith(reg geo.Region, name string) (*TrackIntersection, string) {
	str := fmt.Sprintf("** Intersecting %s[%s] against track %s\n", name, reg, track)
	iStart,iEnd := 0,0
	for i,tp := range track {
		if iStart == 0 {
			if reg.ContainsPoint(tp.Latlong) {
				str += fmt.Sprintf("* [%4d] contained ! (%s, %s)\n", i, reg, tp.Latlong)
				iStart = i
			}
		} else {
			str += fmt.Sprintf("* [%4d] contained !\n", i)
			if !reg.ContainsPoint(tp.Latlong) {
				str += fmt.Sprintf("* [%4d] not contained\n", i)
				iEnd = i-1
				break
			}
		}
	}

	if iStart == 0 {
		str += fmt.Sprintf("** %d,%d; return nil\n", iStart, iEnd)
		return nil, str
	}

	if iEnd == 0 {
		str += fmt.Sprintf("* track ended inside region; take last datapoint as end\n")
		iEnd = len(track)-1
	}
	
	ti := TrackIntersection{
		Start: track[iStart],
		End:   track[iEnd],
		I: iStart,
		J: iEnd,
	}

	for i:=iStart; i<=iEnd; i++ {
		track[i].AnalysisAnnotation += fmt.Sprintf("* Intersected with %s[%s]\n", name, reg)
	}

	return &ti, str
}
*/
