package flightdb2

import(
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"html/template"
	"fmt"
	"sort"
	"time"

	"github.com/skypies/geo"
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

type byTimestampAscending Track
func (a byTimestampAscending) Len() int           { return len(a) }
func (a byTimestampAscending) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byTimestampAscending) Less(i, j int) bool {
	return a[i].TimestampUTC.Before(a[j].TimestampUTC)
}

func (t Track)Start() time.Time { return t[0].TimestampUTC }
func (t Track)End() time.Time { return t[len(t)-1].TimestampUTC }
func (t Track)Times() (s,e time.Time) { return t.Start(), t.End() }
func (t Track)Duration() time.Duration { return t.End().Sub(t.Start()) }
func (t Track)StartEndBoundingBox() geo.LatlongBox {
	// This isn't the actual bounding box for the track; it assumes mostly linear flight.
	// 
	return t[0].BoxTo(t[len(t)-1].Latlong)
}

func (t Track)String() string {
	str := fmt.Sprintf("Track: %d points, start=%s", len(t),
		t[0].TimestampUTC.Format("2006.01.02 15:04:05"))
	if len(t) > 1 {
		s,e := t[0],t[len(t)-1]
		str += fmt.Sprintf(", %s, %.1fKM (%.0f deg)",
			e.TimestampUTC.Sub(s.TimestampUTC), s.Dist(e.Latlong), s.BearingTowards(e.Latlong))
		str += fmt.Sprintf(", src=%s/%s", s.DataSource, s.ReceiverName)
	}
/*	str += "\n"
	for i,tp := range t {
		str += fmt.Sprintf("  [%2d] %s\n", i, tp)
	}*/
	return str
}

func (t Track)ToJSVar() template.JS {
	str := ""
	for i,tp := range t {
		str += fmt.Sprintf("    %d: {%s},\n", i, tp.ToJSString())
	}
	return template.JS("{\n"+str+"  }\n")
}

//func (t Track)ToJSON() string {
//	b,_ := json.Marshal(t)
//	return string(b)
//}

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

func (t Track)LongSource() string {
	if len(t) == 0 { return "(no trackpoints)" }
	return t[0].LongSource()
}


func (t1 *Track)Merge(t2 *Track) {
	for _,tp := range *t2 {
		*t1 = append(*t1, tp)
	}
	sort.Sort(byTimestampAscending(*t1))
}

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

// OverlapOutcome isn't purely accurate; just use it for .IsDisjoint, etc
func (t1 *Track)CompareInSpace(t2 *Track) (geo.OverlapOutcome,float64) {
	if len(*t1) == 0 || len(*t2) == 0 {
		return geo.Undefined, 0.0
	} else if len(*t1) == 1 && len(*t2) == 1 {
		if (*t1)[0].Latlong.Equal((*t2)[0].Latlong) { return geo.OverlapR2IsContained, 1.0 }
	} else if len(*t1) == 1 {
		if t2.StartEndBoundingBox().Contains((*t1)[0].Latlong) { return geo.OverlapR2Contains, 1.0 }
	} else if len(*t2) == 1 {
		if t1.StartEndBoundingBox().Contains((*t2)[0].Latlong) { return geo.OverlapR2IsContained, 0.0 }
	} else {
		// Non-degenerate case: two tracks with >1 points
		return t1.StartEndBoundingBox().OverlapsWith(t2.StartEndBoundingBox())
	}

	return geo.DisjointR2ComesAfter, 0.0
}

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




// PostProcess does some tidyup on the data
/*
func (t Track)PostProcess() Track {
	if len(t) == 0 { return t}
	if t[0].DataSource == "FA:TZ" || t[0].DataSource == "FA:TA" {
		// FLightAware tracks have no heading; compute a rough one based on point-by-point comparison.
		for i:=0; i<len(t)-1; i++ {
			t[i].Heading = t[i].BearingTowards(t[i+1].Latlong)
		}
	}
	return t
}
*/

/*
func (t Track)DurationAloft() (time.Duration, error) {
	var s time.Time
	started := false
	for _,tp := range t {
		if !started {
			if tp.Altitude>0 && tp.GroundSpeed>0 { s = tp.TimestampUTC; started=true; }
		} else {
			if tp.Altitude==0 || tp.GroundSpeed==0 {
				return tp.TimestampUTC.Sub(s), nil
			}
		}
	}
	if started {
		// Was still aloft at the end of the track ...
		return t[len(t)-1].TimestampUTC.Sub(s), nil
	}

	return 0, fmt.Errorf("DurationAloft: too dumb for this track")
}
*/
/*
// This is not a robust function.
func (t Track)TouchdownPDT() time.Time {
	var s time.Time
	// Start halfway through, and see where that gets us
	for i:=int(len(t)/2); i<len(t); i++ {
		s = t[i].TimestampUTC
		if t[i].Altitude == 0 {
			return date.InPdt(s)
		}
	}
	return date.InPdt(s)
}
*/
/*
func (t Track)TimesInBox(b geo.LatlongBox) (s,e time.Time) {
	inside := false

	for _,tp := range t {
		if tp.Altitude==0 || tp.GroundSpeed==0 { continue }
		if !inside && b.Contains(tp.Latlong) {
			s = tp.TimestampUTC
			inside=true

		} else if inside {
			e = tp.TimestampUTC  // keep overwriting e until we're outside (or we've landed)
			if !b.Contains(tp.Latlong) { break }
		}
	}
	return
}
//func (t Track)IsFromADSB() bool {
//	return (len(t)>0 && t[0].DataSource == "FA:TA")
//}

*/
