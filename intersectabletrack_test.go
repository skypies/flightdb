package flightdb
// go test -v github.com/skypies/flightdb2

import(
	"fmt"
	"testing"
	"time"
	"github.com/skypies/geo"
)

// {{{ makeTrack

// Each float is {lat,long} or {lat,long,altitude}
func makeTrack(pts [][]float64, d time.Duration) Track {
	tpts := []Trackpoint{}
	tm := time.Now().UTC()
	for i,pt := range pts {
		tp := Trackpoint{
			Latlong: geo.Latlong{pt[0], pt[1]},
			TimestampUTC: tm.Add(d * time.Duration(i)),
			DataSource: "TEST",
			Altitude: float64(i * 100),
		}
		if len(pt) == 3 {
			tp.Altitude = pt[2]
		}
		tpts = append(tpts, tp)
	}
	return Track(tpts)
}

// }}}
// {{{ TestIntersectableTrackWithAreaRestriction

func TestIntersectableTrackWithAreaRestriction(t *testing.T) {
	sb := geo.SquareBoxRestriction{
		NamedLatlong: geo.NamedLatlong{"ORIGIN", geo.Latlong{0,0}},
		SideKM: 2000.0, // a rectangle about 8.8 / 11.1 per side (in latlong units)
		Debugger: new(geo.DebugLog),
	}

	tests := []struct {
		Expected    RestrictorIntersectOutcome
		P       [][]float64 // All test points relate to a box (-8.8,-11.1) to (8.8,11.1)
		Interval    time.Duration
	}{
		// Generate simple lines that have just start and end trackpoints
		//
		// {outside,outside,outside}
		{ RestrictorIntersectOutcome{TrackIntersection{}, false, ""},
			[][]float64{ {50,0}, {60,0}, {70,0}, {80,0} }, time.Minute,
		},
		// {outside,enter,contained,exit,outside}
		{ RestrictorIntersectOutcome{TrackIntersection{I:2,J:3}, true, ""},
			[][]float64{ {-20,0}, {-10,0}, {-5,0}, {5,0}, {10,0}, {20,0} }, time.Minute,
		},

		// {outside,enter,exit,outside}
		{ RestrictorIntersectOutcome{TrackIntersection{I:2,J:2}, true, ""},
			[][]float64{ {-20,0}, {-10,0}, {0,0}, {10,0}, {20,0} }, time.Minute,
		},

		// {enter,exit,outside}
		{ RestrictorIntersectOutcome{TrackIntersection{I:1,J:1}, true, ""},
			[][]float64{ {-10,0}, {0,0}, {10,0}, {20,0} }, time.Minute,
		},
		// {enter,exit}
		{ RestrictorIntersectOutcome{TrackIntersection{I:1,J:1}, true, ""},
			[][]float64{ {-10,0}, {0,0}, {10,0} }, time.Minute,
		},

		// {outside,enter,contained}
		{ RestrictorIntersectOutcome{TrackIntersection{I:2,J:3}, true, ""},
			[][]float64{ {-20,0}, {-10,0}, {-5,0}, {5,0} }, time.Minute,
		},
		// {contained,exit,outside}
		{ RestrictorIntersectOutcome{TrackIntersection{I:0,J:1}, true, ""},
			[][]float64{ {-5,0}, {5,0}, {10,0}, {20,0} }, time.Minute,
		},

		// {enter}
		{ RestrictorIntersectOutcome{TrackIntersection{I:1,J:1}, true, ""},
			[][]float64{ {-10,0}, {0,0} }, time.Minute,
		},
		// {exit}
		{ RestrictorIntersectOutcome{TrackIntersection{I:0,J:0}, true, ""},
			[][]float64{ {0,0}, {10,0} }, time.Minute,
		},

		// {contained}
		{ RestrictorIntersectOutcome{TrackIntersection{I:0,J:1}, true, ""},
			[][]float64{ {-5,0}, {5,0} }, time.Minute,
		},

		// single point, inside
		{ RestrictorIntersectOutcome{TrackIntersection{I:0,J:0}, true, ""},
			[][]float64{ {0,0} }, time.Minute,
		},
		// single point, outside
		{ RestrictorIntersectOutcome{TrackIntersection{}, false, ""},
			[][]float64{ {500,0} }, time.Minute,
		},

		// {outside,overlap,outside}  // no points inside area - return closest outside points
		{ RestrictorIntersectOutcome{TrackIntersection{I:1,J:2}, true, ""},
			[][]float64{ {-20,0}, {-10,0}, {10,0}, {20,0} }, time.Minute,
		},

		// {overlap}
		{ RestrictorIntersectOutcome{TrackIntersection{I:0,J:1}, true, ""},
			[][]float64{ {-10,0}, {10,0} }, time.Minute,
		},

		//// Lines that span multiple trackpoints, by messing with the interval (\n separates lines)
		//
		// {outside,enter,contained,exit,outside}
		{ RestrictorIntersectOutcome{TrackIntersection{I:6,J:14}, true, ""},
			[][]float64{
				{-20,0}, {-18,0}, {-16,0}, {-14,0},
				{-12,0}, {-10,0}, { -8,0}, { -6,0},
				{ -4,0}, { -2,0}, {  0,0}, {  2,0},
				{  4,0}, {  6,0}, {  8,0}, { 10,0},
				{ 12,0}, { 14,0}, { 16,0}, { 18,0},
			}, time.Second*3,
		},
		// {outside,enter} - only the very final point is inside
		{ RestrictorIntersectOutcome{TrackIntersection{I:8,J:8}, true, ""},
			[][]float64{
				{-20,0}, {-19,0}, {-18,0}, {-17,0},
				{-16,0}, {-14,0}, {-12,0}, {-10,0}, {-8,0},
			}, time.Second*3,
		},
		// {exit,outside} - only the very first point is inside
		{ RestrictorIntersectOutcome{TrackIntersection{I:0,J:0}, true, ""},
			[][]float64{
				{-8,0}, {-10,0}, {-12,0}, {-14,0},
				{-16,0}, {-18,0}, {-20,0}, {-22,0}, {-24,0},
			}, time.Second*3,
		},
		// {enter,exit}   // can walk in to find first contained; can walk back to find last contained
		{ RestrictorIntersectOutcome{TrackIntersection{I:2,J:6}, true, ""},
			[][]float64{
				{-12,0}, {-10,0}, { -8,0}, { -6,0},
				{  4,0}, {  6,0}, {  8,0}, { 10,0}, {12,0}, // Need trailing point because of sampling
			}, time.Second*3,
		},
		// {overlap}      // can do both even on same line
		{ RestrictorIntersectOutcome{TrackIntersection{I:1,J:3}, true, ""},
			[][]float64{
				{-12,0}, {-6,0}, { 0,0}, { 6,0}, { 12,0},
			}, time.Second*3,
		},

	}

	for i,test := range tests {
		track := makeTrack(test.P, test.Interval)
		it := track.ToIntersectableTrackSampleEvery(time.Second * 10)

		actual := it.SatisfiesRestrictor(sb)
		if test.Expected.Satisfies != actual.Satisfies ||
			test.Expected.I != actual.I || test.Expected.J != actual.J {

			fmt.Printf("%s\n%s\nLines:-\n", sb, track)
			for i,l := range it.l {
				fmt.Printf("* % 2d: %s\n", i, l)
			}
			fmt.Printf("Debug:-\n%s", actual.Debug)
			t.Errorf("test [%02d] : wanted{%v/%d,%d}, got{%v/%d,%d}\n", i,
				test.Expected.Satisfies, test.Expected.I, test.Expected.J,
				actual.Satisfies, actual.I, actual.J)
		}
	}
}

// }}}
// {{{ TestIntersectableTrackWithAreaRestrictionAndAltitude

func TestIntersectableTrackWithAreaRestrictionAndAltitude(t *testing.T) {
	sb := geo.SquareBoxRestriction{
		NamedLatlong: geo.NamedLatlong{"ORIGIN", geo.Latlong{0,0}},
		SideKM: 2000.0, // a rectangle about 8.8 / 11.1 per side (in latlong units)
		Debugger: new(geo.DebugLog),
	}

	tests := []struct {
		Expected      RestrictorIntersectOutcome
		P          [][]float64 // {lat,long,altitude}
		//P           []geo.Latlong // All test points relate to a box (-8.8,-11.1) to (8.8,11.1)
		Interval      time.Duration
		AltMin,AltMax int64 //
	}{
		// {outside,enter,contained,exit,outside} - but above altitude restriction
		{ RestrictorIntersectOutcome{TrackIntersection{}, false, ""},
			[][]float64{ {-20,0,100}, {-10,0,100}, {-5,0,100}, {5,0,100}, {10,0,100}, {20,0,100} },
			time.Minute,
			0, 99,
		},
		// line climbing high; l[1] intersects the area, but an interpolation to the point of
		// intersection would show the altitude was too high (~300').
		{ RestrictorIntersectOutcome{TrackIntersection{}, false, ""},
			[][]float64{ {-20,0,50}, {-10,0,99}, {-5,0,600}, {5,0,600}, {10,0,600}, {20,0,600} },
			time.Minute,
			1, 100,
		},
		// Only a subset of the contained lines meet the altitude restriction
		{ RestrictorIntersectOutcome{TrackIntersection{I:3,J:5}, true, ""},
			[][]float64{
				{-10,0, 4}, {-8,0, 4}, {-6,0, 4}, {-4,0,14}, {-2,0,14}, {0,0,14},
				{  2,0,24}, { 4,0,24}, { 6,0,24}, { 8,0,24}, {10,0,24},
			},
			time.Minute,
			10, 20,
		},

		// These two results are inconsistent. An accurate answer would require interpolation
		// No points inside the area; far end is too high.
		{ RestrictorIntersectOutcome{TrackIntersection{}, false, ""},
			[][]float64{ {-20,0,15}, {-10,0,15}, {10,0,25}, {20,0,25} },
			time.Minute,
			10, 20,
		},
		// No points inside the area; close end is too low
		{ RestrictorIntersectOutcome{TrackIntersection{I:1,J:2}, true, ""},
			[][]float64{ {-20,0, 5}, {-10,0, 5}, {10,0,15}, {20,0,15} },
			time.Minute,
			10, 20,
		},

		
		//// Lines that span multiple trackpoints, by messing with the interval (\n separates lines)
		//
		// {outside,enter,contained,exit,outside}, but only a subset of points meet altitude
		{ RestrictorIntersectOutcome{TrackIntersection{I:7,J:10}, true, ""},
			[][]float64{
				{-20,0,12}, {-18,0,12}, {-16,0,12}, {-14,0,12},
				{-12,0,12}, {-10,0,12}, { -8,0, 8}, { -6,0,10},
				{ -4,0,15}, { -2,0,19}, {  0,0,20}, {  2,0,21},
				{  4,0,25}, {  6,0,25}, {  8,0,25}, { 10,0,25},
				{ 12,0,25}, { 14,0,25}, { 16,0,25}, { 18,0,25},
			}, time.Second*3,
			10, 20,  // Allowed altitude range
		},
	}

	for i,test := range tests {
		track := makeTrack(test.P, test.Interval)
		it := track.ToIntersectableTrackSampleEvery(time.Second * 10)
		sb.AltitudeMin,sb.AltitudeMax = test.AltMin, test.AltMax		
		actual := it.SatisfiesRestrictor(sb)
		if test.Expected.Satisfies != actual.Satisfies ||
			test.Expected.I != actual.I || test.Expected.J != actual.J {
			fmt.Printf("%s\n%s\nLines:-\n", sb, track)
			for i,l := range it.l {
				fmt.Printf("* % 2d: %s (%.0fft,%.0fft)\n", i, l, track[l.I].Altitude, track[l.J].Altitude)
			}
			fmt.Printf("Debug:-\n%s", actual.Debug)
			t.Errorf("test [%02d] : wanted{%v/%d,%d}, got{%v/%d,%d}\n", i,
				test.Expected.Satisfies, test.Expected.I, test.Expected.J,
				actual.Satisfies, actual.I, actual.J)
		}
	}
}

// }}}
// {{{ TestIntersectableTrackWithVerticalPlaneAndAltitude

func TestIntersectableTrackWithVerticalPlaneAndAltitude(t *testing.T) {
	vp := geo.VerticalPlaneRestriction{ // plane is at lat==10
		Start: geo.NamedLatlong{"", geo.Latlong{10,-100}},
		End: geo.NamedLatlong{"", geo.Latlong{10, 100}},
		Debugger: new(geo.DebugLog),
	}

	tests := []struct {
		Expected      RestrictorIntersectOutcome
		P         [][]float64 // All test points relate to a box (-8.8,-11.1) to (8.8,11.1)
		Interval      time.Duration
		AltMin,AltMax int64
	}{
		// Generate simple lines that have just start and end trackpoints
		//
		// Simple intersection, and ignore altitudes
		{ RestrictorIntersectOutcome{TrackIntersection{I:2}, true, ""},
			[][]float64{ {2,0}, {8,0}, {16,0}, {20,0} }, time.Minute,
			0,0,
		},
		// Bypass plane
		{ RestrictorIntersectOutcome{TrackIntersection{}, false, ""},
			[][]float64{ {2,1000}, {8,1000}, {16,1000}, {20,1000} }, time.Minute,
			0,0,
		},
		// coincident with plane; need to work hard to make this be considered an intersection
		{ RestrictorIntersectOutcome{TrackIntersection{}, false, ""},
			[][]float64{ {10,20}, {10,30}, {10,40}, {10,50} }, time.Minute,
			0,0,
		},
		// apply altitude, undershoot
		{ RestrictorIntersectOutcome{TrackIntersection{}, false, ""},
			[][]float64{ {2,0,5}, {8,0,5}, {16,0,5}, {20,0,5} }, time.Minute,
			1000,0,
		},
		// apply altitude, descending; interpolation would yield match, but dumb algo misses
		/* { RestrictorIntersectOutcome{TrackIntersection{I:3}, true, ""},
			[][]float64{ {2,0,2000}, {8,0,1200}, {12,0,800}, {20,0,800} }, time.Minute,
			1000,0,
		}, */		
		//// Lines that span multiple trackpoints, by messing with the interval (\n separates lines)
		//
		// This is a miss. line[2] does indeed intersect the plane, and ends at the right altitude; but
		// the actual intersection at t[9] occurs at 22', which is too high.
		{ RestrictorIntersectOutcome{TrackIntersection{}, false, ""},
			[][]float64{
				{ -8,0, 5}, { -6,0, 5}, { -4,0, 5}, { -2,0, 5},
				{  0,0, 5}, {  2,0, 5}, {  4,0, 5}, {  6,0, 5},
				{  8,0, 6}, { 10,0, 8}, { 12,0,10}, { 14,0,12},
				{ 16,0,14}, { 18,0,16}, { 20,0,16}, { 22,0,16},
			}, time.Second*3,
			10, 20,  // Allowed altitude range
		},
		//// Lines that span multiple trackpoints, by messing with the interval (\n separates lines)
		//
		// line[2] intersects downwards, but is just within altitude range at t[9] intersection.
		{ RestrictorIntersectOutcome{TrackIntersection{I:9}, true, ""},
			[][]float64{
				{ -8,0,15}, { -6,0,15}, { -4,0,15}, { -2,0,15},
				{  0,0,15}, {  2,0,15}, {  4,0,15}, {  6,0,15},
				{  8,0,12}, { 10,0,11}, { 12,0,10}, { 14,0, 9},
				{ 16,0, 8}, { 18,0, 8}, { 20,0, 8}, { 22,0, 8},
			}, time.Second*3,
			10, 20,  // Allowed altitude range
		},
	}
		
	for i,test := range tests {
		track := makeTrack(test.P, test.Interval)
		it := track.ToIntersectableTrackSampleEvery(time.Second * 10)
		vp.AltitudeMin,vp.AltitudeMax = test.AltMin, test.AltMax
		actual := it.SatisfiesRestrictor(vp)
		if test.Expected.Satisfies != actual.Satisfies ||
			test.Expected.I != actual.I || test.Expected.J != actual.J {
			fmt.Printf("%s\n%s\nLines:-\n", vp, track)
			for i,l := range it.l {
				fmt.Printf("* % 2d: %s (%.0fft,%.0fft)\n", i, l, track[l.I].Altitude, track[l.J].Altitude)
			}
			fmt.Printf("Debug:-\n%s", actual.Debug)
			t.Errorf("test [%02d] : wanted{%v/%d,%d}, got{%v/%d,%d}\n", i,
				test.Expected.Satisfies, test.Expected.I, test.Expected.J,
				actual.Satisfies, actual.I, actual.J)
		}
	}
}

// }}}


// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
