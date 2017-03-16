package flightdb

import(
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"html/template"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/skypies/geo"
	"github.com/skypies/geo/altitude"
	"github.com/skypies/util/date"
	"github.com/skypies/flightdb/metar"
)

var(
	// These constants control how new ADSB fragments are glued onto existing tracks. This is a
	// fairly constrained problem, as we already know the fragments come from the same physical
	// aircraft; we're not needing to do a full space-time "does this make sense" analysis.
	
	// MaxGap is how large a gap of missing time can exist before we conclude it's a diff track
	kExtensionMaxGap = 10 * time.Minute
	// MaxOverlap is how much time overlap we tolerate before concluding it's a diff track
	kExtensionMaxOverlap = 1 * time.Minute
)

// A Track is a slice of Trackpoints. They are ordered in time, beginning to end.
type Track []Trackpoint

// {{{ lookups

type TrackByTimestampAscending Track
func (a TrackByTimestampAscending) Len() int           { return len(a) }
func (a TrackByTimestampAscending) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a TrackByTimestampAscending) Less(i, j int) bool {
	return a[i].TimestampUTC.Before(a[j].TimestampUTC)
}

func (t Track)Start() time.Time { return t[0].TimestampUTC }
func (t Track)End() time.Time { return t[len(t)-1].TimestampUTC }
func (t Track)Times() (s,e time.Time) { return t.Start(), t.End() }
func (t Track)Duration() time.Duration { return t.End().Sub(t.Start()) }
func (t Track)StartEndBoundingBox() geo.LatlongBox {
	if len(t) == 0 { return geo.LatlongBox{} }
	// This isn't the actual bounding box for the track; it assumes mostly linear flight.
	return t[0].BoxTo(t[len(t)-1].Latlong)
}
func (t Track)Notes() string {
	if len(t) == 0 { return "" }
	return t[0].Notes
}

// Full bounding box involves a full traversal of the track.
func (t Track)FullBoundingBox() geo.LatlongBox {
	if len(t) == 0 { return geo.LatlongBox{} }

	b := t[0].BoxTo(t[0].Latlong)
	for _,tp := range t[1:] {
		b.Enclose(tp.Latlong)
	}
	return b
}

// }}}

// {{{ t.String, t.{Short|Medium|Long}String

func (t Track)String() string {
	if len(t) == 0 { return "(empty track)" }
	str := fmt.Sprintf("Track: %4d points, start=%s", len(t),
		t[0].TimestampUTC.Format("2006.01.02 15:04:05"))
	if len(t) > 1 {
		s,e := t[0],t[len(t)-1]
		str += fmt.Sprintf(", %s, %.1fKM (%.0f deg)",
			date.RoundDuration(e.TimestampUTC.Sub(s.TimestampUTC)),
			s.Dist(e.Latlong), s.BearingTowards(e.Latlong))
		str += fmt.Sprintf(", src=%s", s.DataSource)
		if s.ReceiverName != "" { str += "/" + s.ReceiverName }
	}

	if t.Notes() != "" {
		str += " " + t.Notes()
	}

	return str
}

func (t Track)MediumString() string {
	if len(t) == 0 { return "(empty track)" }
	str := fmt.Sprintf("%4d pts, start=%s", len(t),
		t[0].TimestampUTC.Format("2006.01.02 15:04"))
	if len(t) > 1 {
		str += fmt.Sprintf(", src=%s", t[0].DataSource)
	}

	if t.Notes() != "" {
		str += " " + t.Notes()
	}

	return str
}

func (t Track)ShortString() string {
	if len(t) == 0 { return "[null track]" }

	s,e := t[0],t[len(t)-1]
	str := fmt.Sprintf("%s +%s (%.0fKM)",
		date.InPdt(s.TimestampUTC).Format("Jan02 15:04:05 MST"),
		date.RoundDuration(e.TimestampUTC.Sub(s.TimestampUTC)),
		s.Dist(e.Latlong));

	return str
}

func (t Track)LongString() string {
	str := t.MediumString() + "\n"
	for i,tp := range t {
		str += fmt.Sprintf("[%03d] %s\n", i, tp)
	}
	return str
}

// }}}
// {{{ t.ToJSVar

func (t Track)ToJSVar() template.JS {
	str := ""
	for i,tp := range t {
		str += fmt.Sprintf("    %d: {%s},\n", i, tp.ToJSString())
	}
	return template.JS("{\n"+str+"  }\n")
}

// }}}
// {{{ t.Base64{Encode,Decode}

func (t Track)Base64Encode() (string, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(t); err != nil {
		return "", err
	} else {
		return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
	}
}

func (t *Track)Base64Decode(str string) error {
	if data,err := base64.StdEncoding.DecodeString(str); err != nil {
		return err
	} else {
		buf := bytes.NewBuffer(data)
		err := gob.NewDecoder(buf).Decode(&t)
		return err
	}
}

// }}}
// {{{ t.LongSource

func (t Track)LongSource() string {
	if len(t) == 0 { return "(no trackpoints)" }
	return t[0].LongSource()
}

// }}}

// {{{ t.PostProcess

// TODO: make this idempotent (perhaps finally create a sentinel object for Track ?)

// Derive a bunch of data fields from the raw data.
// NOTE - the vertical data gets too jerky with ADSB, because altitude change appears more like
// an occasional step function when the datapoints are too close. You should use t.SampleEvery()
// to space things out a bit before using those fields.
func (t Track)PostProcess() {
	// Skip the first point
	for i:=1; i<len(t); i++ {
		// No heading info in FlightAware tracks
		//if t[0].DataSource == "FA" {
		//	t[i-1].Heading = t[i-1].BearingTowards(t[i].Latlong)
		//}

		// VerticalRate data exists, but only for locally received data
		
		dur  := t[i].TimestampUTC.Sub(t[i-1].TimestampUTC)
		distKM := t[i].DistKM(t[i-1].Latlong)

		if t[0].DataSource == "RG-FOIA" || t[0].DataSource == "EB-FOIA" {
			// FOIA data has no groundspeed data. Compute it.
			// 1 knot == 1 NM/hour == 1.852 KM/hour
			t[i].GroundSpeed = (distKM/dur.Hours()) / 1.852
			if i == 1 { t[0].Notes += fmt.Sprintf("(groundspeeds derived from position data)") }
		}
			
		// Compute elapsed distance along path, and acceleration rates
		t[i].DistanceTravelledKM = t[i-1].DistanceTravelledKM + distKM
		t[i].GroundAccelerationKPS = (t[i].GroundSpeed - t[i-1].GroundSpeed) / dur.Seconds()
		t[i].VerticalSpeedFPM = (t[i].Altitude - t[i-1].Altitude) / dur.Minutes()		
		t[i].VerticalAccelerationFPMPS = (
			t[i].VerticalSpeedFPM - t[i-1].VerticalSpeedFPM) / dur.Seconds()		

		// AngleOfInclination; distKM=adjacent, vertDistKM=opposite; ang=arctan(opp/adj)
		vertDistKM := (t[i].Altitude - t[i-1].Altitude) / geo.KFeetPerKM
		t[i].AngleOfInclination = math.Atan2(vertDistKM,distKM) * 180.0/math.Pi
	}
}

// }}}
// {{{ t.AdjustAltitudes

func (t Track)AdjustAltitudes(metars *metar.Archive) {
	nAdjusted := 0
	totAdjustment := 0.0
	
	for i,tp := range t {
		if metars != nil {
			if lookup := metars.Lookup(tp.TimestampUTC); lookup != nil && lookup.Raw != "" {
				t[i].IndicatedAltitude = altitude.PressureAltitudeToIndicatedAltitude(
					tp.Altitude, lookup.AltimeterSettingInHg)
				adjustment := t[i].IndicatedAltitude - t[i].Altitude
				t[i].AnalysisAnnotation += fmt.Sprintf("* altitude correction: inHg %v (%+.0f ft)\n",
					lookup, adjustment)
				totAdjustment += adjustment
				nAdjusted++
			} else {
				// Hack, because we don't have historic METAR yet
				t[i].AnalysisAnnotation += fmt.Sprintf("* altitude correction: no historic METAR\n")
				t[i].IndicatedAltitude = tp.Altitude
			}
		} else {
			t[i].AnalysisAnnotation += fmt.Sprintf("* altitude correction: not reqeusted (no METAR)\n")
			t[i].IndicatedAltitude = tp.Altitude
		}
	}

	if nAdjusted>0 {
		t[0].Notes += fmt.Sprintf("(%d altitude corrections, avg=%+.0f ft)", nAdjusted,
			totAdjustment / float64(nAdjusted))
	}
}

// }}}

// {{{ t.Merge

func (t1 *Track)Merge(t2 *Track) {
	for _,tp := range *t2 {
		*t1 = append(*t1, tp)
	}
	sort.Sort(TrackByTimestampAscending(*t1))
}

// }}}
// {{{ t.[Padded]TrimToTimes

// Returns a (possibly empty) subtrack of points within [s,e] (inclusive).
// If padding is non-zero, we include that many additional points just to
// either side of the [s,e] (i.e neighboring points that don't quite lie in the range)
func (t *Track)TrimToTimes(s,e time.Time) *Track { return t.PaddedTrimToTimes(s,e,0) }
func (t *Track)PaddedTrimToTimes(s,e time.Time, n int) *Track {
	ret := Track{}
	for i,tp := range *t {
		if !tp.TimestampUTC.Before(s) && !tp.TimestampUTC.After(e) {
			if len(ret)==0 && n>0 && i>=n {
				// We're just about to add the first legit point; add padding !
				// note - slice syntax, [x:y] includes x..(y-1)
				ret = append(ret, (*t)[i-n:i]...)
			}
			ret = append(ret, tp)
		} else {
			if len(ret)>0 && n>0 && i<len(*t)-n {
				// We've just passed the final point; add padding if we need, then bail
				ret = append(ret, (*t)[i+1:i+n+1]...)
				return &ret
			}
		}
	}
	return &ret
}

// }}}
// {{{ t.Compare

type CompareOutcome struct {
	TimeDisposition  geo.OverlapOutcome // how the tracks compare in terms of time overlap
	OverlapStart     time.Time          // If there *is* a time overlap, this is when it starts
	                 time.Duration      // embedded; Duration of any overlap, or gap between non-overlapping tracks

	// If times overlap, figure out space overlap
	OverlapA        *Track
	OverlapB        *Track
	SpaceDisposition geo.OverlapOutcome
	SpaceOverlap     float64

	// Debugging junk
	Log              string
}
func (o CompareOutcome)String() string {
	return fmt.Sprintf("-- Outcome=%d[%d]\n%s", o.TimeDisposition, o.SpaceDisposition, o.Log)
}

func (t1 *Track)Compare(t2 *Track) CompareOutcome {
	o := CompareOutcome{}
	o.Log += fmt.Sprintf("t1: %s\nt2: %s\nt1:  %s  ->  %s\nt2:  %s  ->  %s\n",
		t1, t2, t1.Start(), t1.End(), t2.Start(), t2.End())

	// Compute the four deltas between the start/ends of both tracks
	t1s2e := t2.End().Sub(t1.Start())  // duration from t1's start to t2's end
	t1e2s := t2.Start().Sub(t1.End())
	
	r1 := geo.TimeRange{t1.Start(),t1.End()}
	r2 := geo.TimeRange{t2.Start(),t2.End()}
	o.TimeDisposition = geo.RangeOverlap(r1,r2)

	switch o.TimeDisposition{
	case geo.DisjointR2ComesBefore:
		o.Duration = -1*t1s2e  // Flip the sign to get a positive duration
		o.Log += fmt.Sprintf("t2 comes entirely before t1, by %s", o.Duration)
	case geo.DisjointR2ComesAfter:
		o.Duration = t1e2s
		o.Log += fmt.Sprintf("t2 comes entirely after t1, by %s\n", o.Duration)
	case geo.OverlapR2StraddlesEnd:
		o.Duration = -1*t1e2s
		o.OverlapStart = t2.Start()
		o.Log += "t2 extends into the future, straddling end of t1\n"
	case geo.OverlapR2StraddlesStart:
		o.Duration = t1s2e
		o.OverlapStart = t1.Start()
		o.Log += "t2 extends into the past, straddling the start of t1\n"
	case geo.OverlapR2IsContained:
		o.Duration = t2.Duration()
		o.OverlapStart = t2.Start()
		o.Log += "t2 is entirely contained within t1\n"
	case geo.OverlapR2Contains:
		o.Duration = t1.Duration()
		o.OverlapStart = t1.Start()
		o.Log += "t2 contains t1 entirely\n"
	}

	if !o.TimeDisposition.IsDisjoint() {
		o.OverlapA = t1.PaddedTrimToTimes(o.OverlapStart, o.OverlapStart.Add(o.Duration), 1)
		o.OverlapB = t2.PaddedTrimToTimes(o.OverlapStart, o.OverlapStart.Add(o.Duration), 1)		
		o.Log += fmt.Sprintf("Overlap: from %s, for %s\n* OverlapA: %s\n* OverlapB: %s\n",
			o.OverlapStart, o.Duration, o.OverlapA, o.OverlapB)
		o.SpaceDisposition, o.SpaceOverlap = o.OverlapA.CompareInSpace(o.OverlapB)
		o.Log += fmt.Sprintf("* space comparison: [%v], %f\n", o.SpaceDisposition, o.SpaceOverlap)
	}

	return o
}

// }}}
// {{{ t.CompareInSpace

// OverlapOutcome is abused here; only use the return value for .IsDisjoint() checks.
func (t1 *Track)CompareInSpace(t2 *Track) (geo.OverlapOutcome,float64) {
	if len(*t1) == 0 || len(*t2) == 0 {
		return geo.Undefined, 0.0
	}

	if len(*t1) == 1 && len(*t2) == 1 {
		if (*t1)[0].Latlong.Equal((*t2)[0].Latlong) {
			return geo.OverlapR2IsContained, 1.0
		} else {
			return geo.DisjointR2ComesAfter, 0.0
		}
	}

	// Bounding boxes involve full traversals, so compute once.
	b1,b2 := t1.FullBoundingBox(),t2.FullBoundingBox()

	if len(*t1) == 1 {
		if b2.Contains((*t1)[0].Latlong) { return geo.OverlapR2Contains, 1.0 }
	} else if len(*t2) == 1 {
		if b1.Contains((*t2)[0].Latlong) { return geo.OverlapR2IsContained, 0.0 }
	} else {
		// At last, the non-degenerate case: two tracks with >1 points
		return b1.OverlapsWith(b2)
	}

	return geo.DisjointR2ComesAfter, 0.0
}

// }}}
// {{{ t.PlausibleExtension

// Does t2 more or less continue where t1 left off ?
func (t1 *Track)PlausibleExtension(t2 *Track) (bool, string) {
	o := t1.Compare(t2)

	if o.TimeDisposition == geo.DisjointR2ComesBefore {
		return false, o.String()

	} else if o.TimeDisposition == geo.DisjointR2ComesAfter {	
		if o.Duration <= kExtensionMaxGap {
			return true, o.String() + "looks good disjoint, plausible is YES\n"
		} else {
			return false, o.String() + fmt.Sprintf("gap is too long (>%s)", kExtensionMaxGap)
		}

	} else {
		// They overlapped in time. Do they overlap in space/altitude/etc ?
		if o.SpaceDisposition.IsDisjoint() {
			return false, o.String() + "No space overlap, despite time overlap\n"
		}

		return true, o.String() + "Time and space overlap, plausible is YES\n"
/*		
		if o.TimeDisposition == geo.OverlapR2StraddlesEnd {
			if o.Duration <= kExtensionMaxOverlap {
				return true, o.String() + "looks good overlap, plausible is YES\n"
			} else {
				return false, o.String() + fmt.Sprintf("overlap is too great (>%s)", kExtensionMaxOverlap)
			}
		}
*/
	}
}

// }}}
// {{{ t.PlausibleContribution

// Does t2 seem like it could be glued onto/into t1 ?
func (t1 *Track)PlausibleContribution(t2 *Track) (bool, string) {
	o := t1.Compare(t2)

	if o.TimeDisposition==geo.DisjointR2ComesAfter {
		if o.Duration <= kExtensionMaxGap {
			return true, o.String() + "looks good disjoint suffix, plausible is YES\n"
		} else {
			return false, o.String() + fmt.Sprintf("suffix gap is too long (>%s)", kExtensionMaxGap)
		}

	} else if o.TimeDisposition==geo.DisjointR2ComesBefore {
		// Out-of-order delivery; t2 are trackpoints that precede
		// trackpoints we already have in t1. Almost certainly part of the
		// same flight, so relax the duration constraint.
		if o.Duration <= 3 * kExtensionMaxGap {
			return true, o.String() + "looks good disjoint prefix, plausible is YES\n"
		} else {
			return false, o.String() + fmt.Sprintf("prefix gap is too long (>%s)", kExtensionMaxGap * 3)
		}

	} else {
		// They overlapped in time. Do they overlap in space/altitude/etc ?
		if o.SpaceDisposition.IsDisjoint() {
			// Weird. Let's plot them. Our one instance was an error from over-approximate bboxes.
			str := fmt.Sprintf("* view at fdb.serfr1.org/fdb/map?boxes=b1,b2&%s&%s\n",
				t1.StartEndBoundingBox().ToCGIArgs("b1"),
				t2.StartEndBoundingBox().ToCGIArgs("b2"))
			return false, o.String() + str + "No space overlap, despite time overlap\n"
		}

		return true, o.String() + "Time and space overlap, plausible is YES\n"
	}
}

// }}}
// {{{ t.IndexAtTime

// Returns -1 if not found
// Finds the largest index whose trackpoint does not come before the specified time
func (t Track)IndexAtTime(tm time.Time) int {
	if tm.Before(t.Start()) || tm.After(t.End()) { return -1 }

	// TODO: rewrite with something hierarchical
	// Start loop on second point
	for i:=1; i<len(t); i++ {
		// If this point comes after our time, then the preceding point is our winner
		if t[i].TimestampUTC.After(tm) { return i-1 }
	}

	return -1 // "Should not happen"
}

// }}}
// {{{ t.IndicesAtDistKMsFrom

// This is not a very smart algorithm.
// It assumes that dist to refpt is monotonic and decreasing.
func (t Track)IndicesAtDistKMsFrom(refpt geo.Latlong, distsKM []float64) []int {
	results := []int{}

	if len(distsKM) == 0 { return results }
	targetDist, distsKM := distsKM[0], distsKM[1:]

	// Walk the track, picking out trackpoints that precede the given distances.
	for i,tp := range t {
		dist := tp.DistKM(refpt)

		// If we've just moved past the target dist, then we have a match for this target
		if dist < targetDist {
			if i > 0 { results = append(results, i-1) }

			// Finish, or reset for a new target.
			if len(distsKM) == 0 { return results }
			targetDist, distsKM = distsKM[0], distsKM[1:]
		}
	}

	return results
}

// }}}
// {{{ t.WindowedAverageAt

// Returns a synthetic trackpoint with values for {ground|vertical}{speed|acceleration} computed
// from a sliding window, based on position/altitude/time data. The time window is centered on
// trackpoint i.
func (t Track)WindowedAverageAt(i int, window time.Duration) Trackpoint {
	// Find the trackpoints closest to the edges of the window
	u,v := i,i
	for u>0 {
		if t[i].TimestampUTC.Sub(t[u].TimestampUTC) > window/2 {
			// u is now outside the window; move back in, and bail
			if (u<i) { u++ }
			break
		}
		u--
	}

	for v+1<len(t) {
		if t[v].TimestampUTC.Sub(t[i].TimestampUTC) > window/2 {
			// v is now outside the window; move back in, and bail
			if (v>i) { v-- }
			break
		}
		v++
	}

	// So we want to generate an average value based on the trackpoints [u,v], inclusive of u and v.
	out := TrackpointFromAverage(t[u:v+1])
	out.Notes = fmt.Sprintf("(WindowedAverageAt(i=%d,w=%s) u,v=[%d,%d])",i, window, u, v)

	return out

/* This approach doesn't actually do any averaging - if data is up/down spikey, then it
   will end up picking up/up, up/down or down/down pairs, making output just as spikey.

	// Work out how far inbetween U and V we should consider I to be at ...
	dUV := t[v].TimestampUTC.Sub(t[u].TimestampUTC)
	dUI := t[i].TimestampUTC.Sub(t[u].TimestampUTC)
	ratio := float64(dUI) / float64(dUV)

	if u == v { ratio = 0.0 } // So much nicer than NaN (but not naan)
	
	itp := t[u].InterpolateTo(t[v], ratio)

	itp.Notes = fmt.Sprintf("WindowedAverageAt(i=%d,w=%s) u,v=[%d,%d], ratio=%.3f",
		i, window, u, v, ratio)

	return &itp
*/	
}

// }}}

// {{{ t.ClipTo

// Returns copies of the trackpoints within the timerange; or empty track if none were found.
func (t Track)ClipTo(s,e time.Time) []Trackpoint {
	if e.Before(s) { e,s = s,e }

	ret := []Trackpoint{}
	if e.Before(t.Start()) || s.After(t.End()) { return ret }

	if iStart := t.IndexAtTime(s); iStart >= 0 {
		for i:=iStart; i<len(t); i++ {
			if t[i].TimestampUTC.After(e) { break }
			if t[i].TimestampUTC.Before(s) { continue }
			ret = append(ret, t[i])
		}
	}

	return ret
}

// }}}
// {{{ t.SampleEvery

// Returns a track that has (more or less) one point per time.Duration.
// If interpolate is true, then we interpolate through gaps that are too long.
// The returned track contains copies of the trackpoints
func (t Track)SampleEvery(d time.Duration, interpolate bool) Track {
	if len(t) == 0 { return []Trackpoint{} }

	new := []Trackpoint{t[0]}

	iLast := 0
	for i:=1; i<len(t); i++ {
		// i is the point we're looking at; iLast is the point at the end of the previous box.
		tDelta := t[i].TimestampUTC.Sub(t[iLast].TimestampUTC)

		if tDelta > d {
			if interpolate && tDelta > 2*d {
				// IMPLEMENT ME
			}
			new = append(new, t[i])
			iLast = i

		} else {
			// Do nothing, skip to next
		}
	}

	if len(new)>0 {
		new[0].Notes = fmt.Sprintf("(sampled every %s from %d points)", d, len(t))
	}
	
	return new
}

// }}}
// {{{ t.SampleEveryDist

// Returns a track that has (more or less) one point per distance unit (as flown along the path).
// If interpolate is true, then we interpolate through gaps that are too long; else they remain.
// The returned track contains copies of the trackpoints.
func (t Track)SampleEveryDist(distKM float64, interpolate bool) Track {
	if len(t) == 0 { return []Trackpoint{} }

	new := []Trackpoint{t[0]}

	iLast := 0
	for i:=1; i<len(t); i++ {
		// i is the point we're looking at; iLast is the point at the end of the previous box.
		distDelta := t[i].DistKM(t[iLast].Latlong)

		if distDelta > distKM {
			if interpolate && distDelta > 2*distKM {
				// IMPLEMENT ME
			}
			new = append(new, t[i])
			iLast = i

		} else {
			// Do nothing, skip to next
		}
	}

	if len(new)>0 {
		new[0].Notes = fmt.Sprintf("(sampled every %.2fKM from %d points)", distKM, len(t))
	}
	
	return new
}

// }}}
// {{{ t.AsContiguousBoxes

func (from Trackpoint)LatlongTimeBoxTo(to Trackpoint) geo.LatlongTimeBox {
	return geo.LatlongTimeBox{
		LatlongBox: from.Latlong.BoxTo(to.Latlong),
		Start: from.TimestampUTC,
		End: to.TimestampUTC,
		HeadingDelta: geo.HeadingDelta(from.Heading, to.Heading),
		Source: from.DataSource,
	}
}

// If there are gaps in the track, this will interpolate between them.
// Will also fatten up the boxes, if they're too flat or too tall
func (t Track)AsContiguousBoxes() []geo.LatlongTimeBox {
	minSize := 0.05  // In 'latlong' units; comes out something like ~3NM (~5 vertical)
	maxSize := 0.10  // Boxes bigger than this get chopped into smaller bits
	minWidth := 0.01 // Boxes are stretched until at least this wide/tall

	boxes := []geo.LatlongTimeBox{}
	iLast := 0
	for i:=1; i<len(t); i++ {
		// i is the point we're looking at; iLast is the point at the end of the previous box.
		// should we create a box from i back to iLast ? multiple boxes ? Or skip to i+1 ?
		dist := t[iLast].Latlong.LatlongDist(t[i].Latlong)
		if dist > maxSize {
			// Need to interpolate some boxes into this gap
			nNeeded := int(dist/maxSize) + 1  // num boxes to create. int() rounds down
/*
			// If the aircraft is changing direction, then we should have fewer (and thus
			// fatter) boxes, to better approximate what the aircraft might be doing.
			headingDelta := math.Abs(geo.HeadingDelta(t[iLast].Heading, t[i].Heading))
			if headingDelta > 0 {
				nHeadingMax := int(65.0/headingDelta) + 1
				if nHeadingMax < nNeeded { nNeeded = nHeadingMax }
			}
*/
			len := 1.0 / float64(nNeeded)     // fraction of dist - size of each box 
			sTP, eTP := t[iLast], t[i]
			for j:=0; j<nNeeded; j++ {
				startFrac := len * float64(j)
				endFrac := startFrac + len
				sITP := sTP.InterpolateTo(eTP, startFrac)
				eITP := sTP.InterpolateTo(eTP, endFrac)
				box := sITP.Trackpoint.LatlongTimeBoxTo(eITP.Trackpoint)
				box.I,box.J = iLast,i
				box.Interpolated = true
				box.RunLength = nNeeded
				
				centroidHeading := sITP.BearingTowards(box.Center())
				box.CentroidHeadingDelta = geo.HeadingDelta(sITP.Heading, centroidHeading)
				
				box.Debug = fmt.Sprintf(
					" - src: %s\n"+
					" - sTP: %s\n - eTP: %s\n - span: %.2f-%.2f\n"+
					" - centroid: %.2f; sITP: %.2f; delta: %.2f\n"+
					" - interp: %d points\n"+
					" - sITP: %s\n - eITP: %s\n",
					t[0].DataSource, sTP, eTP, startFrac, endFrac,
					centroidHeading, sITP.Heading, box.CentroidHeadingDelta,
					nNeeded,
					sITP, eITP)

				boxes = append(boxes, box)
			}
			iLast = i

		} else if dist > minSize {
			// Grow an initial box with all the succeeding trackpoints
			box := t[iLast].LatlongTimeBoxTo(t[iLast+1])
			for j:=iLast+2; j<=i; j++ {
				box.Enclose(t[j].Latlong, t[j].TimestampUTC)
			}
			box.I,box.J = iLast,i

			centroidHeading := t[iLast].BearingTowards(box.Center())
			box.CentroidHeadingDelta = geo.HeadingDelta(t[iLast].Heading, centroidHeading)
			
			boxes = append(boxes, box)
			iLast = i

		} else {
			// This point is too close to the prev; let the loop iterate to the next one
		}
	}	

	// We don't want boxes that are too skinny, so we pad them out here.
	for i,_ := range boxes {
		boxes[i].EnsureMinSide(minWidth)
	}

	return boxes
}

// }}}
// {{{ t.OverlapsWith

// Given two tracks, do they overlap in time and space well enough to be the same thing ?
// NOTE: should precede this with a boundingbox test; tracks that can plausibly glue together
// but which don't actually overlap in time will return 'false' from this.

// overlaps: if we should consider them the same thing
// conf: how confident we are
// debug: some debug text about it.
func (t1 Track)OverlapsWith(t2 Track) (overlaps bool, conf float64, debug string) {
	b1 := t1.AsContiguousBoxes()
	b2 := t2.AsContiguousBoxes()
	return geo.CompareBoxSlices(&b1,&b2)
}

// }}}

// {{{ t.AsLinesSampledEvery

// Consider caching this in an ephemeral field ?
func (t Track)AsLinesSampledEvery(d time.Duration) []geo.LatlongLine {
	lines := []geo.LatlongLine{}

	if len(t)<2 { return lines }

	iLast := 0
	for i:=1; i<len(t); i++ {
		// i is the point we're looking at; iLast is the point at the end of the previous line.
		if d < t[i].TimestampUTC.Sub(t[iLast].TimestampUTC) {
			// Time to flush a line segment
			line := t[iLast].BuildLine(t[i].Latlong)
			line.I,line.J = iLast,i
			lines = append(lines, line)
			iLast = i
		}
	}

	return lines
}

// }}}
// {{{ t.AsSanityFilteredTrack

// Strip out any trackpoints that look really bogus.
// Leaves original track entirely untouched, and returns a completely separate new copy.
func (in Track)AsSanityFilteredTrack() Track {
	rejectedPoints := []int{}

	// Skip the first point
	for i:=1; i<len(in); i++ {
		// Compute the implied groundspeed, to see if this position is crazy given the previous pos
		durSincePrevPoint     := in[i].TimestampUTC.Sub(in[i-1].TimestampUTC)
		distFromPrevPointKM   := in[i].DistanceTravelledKM - in[i-1].DistanceTravelledKM
		impliedGroundSpeedKPH := distFromPrevPointKM / durSincePrevPoint.Hours()

		if impliedGroundSpeedKPH > 1000 {
			rejectedPoints = append(rejectedPoints, i)
		}
	}

	// Go in reverse order, or the index values will be invalid
	sort.Sort(sort.Reverse(sort.IntSlice(rejectedPoints)))

  tmpTrack := make([]Trackpoint, len(in))
  copy(tmpTrack, in)
	out := Track(tmpTrack)
	
	if len(rejectedPoints) > 0 {
		for _,index := range rejectedPoints {
			out = append(out[:index], out[index+1:]...)
		}
		out.PostProcess()  // Recompute stuff, now we've removed bogus points
	}

	return out
}

// }}}

// {{{ t.ClosestTo

// returns the index of the trackpoint that was closest to the
// reference point and within the altitude range, or -1 if track has
// no points within the altitude range. Set maxAltitude to zero to disable
// altitude checks.
func (t Track)ClosestTo(ref geo.Latlong, minAltitude, maxAltitude float64) (int) {
	if len(t) == 0 { return -1 }

	iMin,sqDistMin := 0,math.MaxFloat64

	for i,tp := range t {
		if maxAltitude > 0 && (tp.Altitude > maxAltitude || tp.Altitude < minAltitude) { continue }
		dist := ref.LatlongDistSq(tp.Latlong)
		if dist < sqDistMin {
			iMin,sqDistMin = i,dist
		}
	}

	return iMin
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
