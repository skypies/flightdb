package flightdb2
// go test -v github.com/skypies/flightdb2

import(
	"fmt"
	"testing"
	"time"
	"github.com/skypies/geo"
)

func makeTrack(pts []geo.Latlong, d time.Duration) Track {
	tpts := []Trackpoint{}
	tm := time.Now().UTC()
	for i,pt := range pts {
		tpts = append(tpts, Trackpoint{
			Latlong: pt,
			TimestampUTC: tm.Add(d * time.Duration(i)),
			DataSource: "TEST",
			Altitude: float64(i * 100),
		})
	}
	return Track(tpts)
}

func TestIntersectableTrackWithAreaRestriction(t *testing.T) {
	sb := geo.SquareBoxRestriction{
		NamedLatlong: geo.NamedLatlong{"ORIGIN", geo.Latlong{0,0}},
		SideKM: 2000.0, // a rectangle about 8.8 / 11.1 per side (in latlong units)
		Debugger: new(geo.DebugLog),
	}

	tests := []struct {
		Expected    RestrictorIntersectOutcome
		P         []geo.Latlong // All test points relate to a box (-8.8,-11.1) to (8.8,11.1)
		Interval    time.Duration
	}{
		// Generate simple lines that have just start and end trackpoints
		//
		// {outside,outside,outside}
		{ RestrictorIntersectOutcome{TrackIntersection{}, false, ""},
			[]geo.Latlong{ {50,0}, {60,0}, {70,0}, {80,0} }, time.Minute,
		},
		// {outside,enter,contained,exit,outside}
		{ RestrictorIntersectOutcome{TrackIntersection{I:2,J:3}, true, ""},
			[]geo.Latlong{ {-20,0}, {-10,0}, {-5,0}, {5,0}, {10,0}, {20,0} }, time.Minute,
		},
		// {outside,enter,exit,outside}
		{ RestrictorIntersectOutcome{TrackIntersection{I:2,J:2}, true, ""},
			[]geo.Latlong{ {-20,0}, {-10,0}, {0,0}, {10,0}, {20,0} }, time.Minute,
		},
		// {enter,exit,outside}
		{ RestrictorIntersectOutcome{TrackIntersection{I:1,J:1}, true, ""},
			[]geo.Latlong{ {-10,0}, {0,0}, {10,0}, {20,0} }, time.Minute,
		},
		// {enter,exit}
		{ RestrictorIntersectOutcome{TrackIntersection{I:1,J:1}, true, ""},
			[]geo.Latlong{ {-10,0}, {0,0}, {10,0} }, time.Minute,
		},
		// {outside,enter,contained}
		{ RestrictorIntersectOutcome{TrackIntersection{I:2,J:3}, true, ""},
			[]geo.Latlong{ {-20,0}, {-10,0}, {-5,0}, {5,0} }, time.Minute,
		},
		// {contained,exit,outside}
		{ RestrictorIntersectOutcome{TrackIntersection{I:0,J:1}, true, ""},
			[]geo.Latlong{ {-5,0}, {5,0}, {10,0}, {20,0} }, time.Minute,
		},
		// {enter}
		{ RestrictorIntersectOutcome{TrackIntersection{I:1,J:1}, true, ""},
			[]geo.Latlong{ {-10,0}, {0,0} }, time.Minute,
		},
		// {exit}
		{ RestrictorIntersectOutcome{TrackIntersection{I:0,J:0}, true, ""},
			[]geo.Latlong{ {0,0}, {10,0} }, time.Minute,
		},
		// {contained}
		{ RestrictorIntersectOutcome{TrackIntersection{I:0,J:1}, true, ""},
			[]geo.Latlong{ {-5,0}, {5,0} }, time.Minute,
		},
		// {outside,overlap,outside}  // no points inside area - return closest outside points
		{ RestrictorIntersectOutcome{TrackIntersection{I:1,J:2}, true, ""},
			[]geo.Latlong{ {-20,0}, {-10,0}, {10,0}, {20,0} }, time.Minute,
		},
		// {overlap}
		{ RestrictorIntersectOutcome{TrackIntersection{I:0,J:1}, true, ""},
			[]geo.Latlong{ {-10,0}, {10,0} }, time.Minute,
		},

		//// Lines that span multiple trackpoints, by messing with the interval (\n separates lines)
		//
		// {outside,enter,contained,exit,outside}
		{ RestrictorIntersectOutcome{TrackIntersection{I:6,J:14}, true, ""},
			[]geo.Latlong{
				{-20,0}, {-18,0}, {-16,0}, {-14,0},
				{-12,0}, {-10,0}, { -8,0}, { -6,0},
				{ -4,0}, { -2,0}, {  0,0}, {  2,0},
				{  4,0}, {  6,0}, {  8,0}, { 10,0},
				{ 12,0}, { 14,0}, { 16,0}, { 18,0},
			}, time.Second*3,
		},
		// {outside,enter} - only the very final point is inside
		{ RestrictorIntersectOutcome{TrackIntersection{I:8,J:8}, true, ""},
			[]geo.Latlong{
				{-20,0}, {-19,0}, {-18,0}, {-17,0},
				{-16,0}, {-14,0}, {-12,0}, {-10,0}, {-8,0},
			}, time.Second*3,
		},
		// {exit,outside} - only the very first point is inside
		{ RestrictorIntersectOutcome{TrackIntersection{I:0,J:0}, true, ""},
			[]geo.Latlong{
				{-8,0}, {-10,0}, {-12,0}, {-14,0},
				{-16,0}, {-18,0}, {-20,0}, {-22,0}, {-24,0},
			}, time.Second*3,
		},
		// {enter,exit}   // can walk in to find first contained; can walk back to find last contained
		{ RestrictorIntersectOutcome{TrackIntersection{I:2,J:6}, true, ""},
			[]geo.Latlong{
				{-12,0}, {-10,0}, { -8,0}, { -6,0},
				{  4,0}, {  6,0}, {  8,0}, { 10,0}, {12,0}, // Need trailing point because of sampling
			}, time.Second*3,
		},
		// {overlap}      // can do both even on same line
		{ RestrictorIntersectOutcome{TrackIntersection{I:1,J:3}, true, ""},
			[]geo.Latlong{
				{-12,0}, {-6,0}, { 0,0}, { 6,0}, { 12,0},
			}, time.Second*3,
		},
	}

	for i,test := range tests {
		track := makeTrack(test.P, test.Interval)
		it := IntersectableTrack{
			Track: track,
			l:     track.AsLinesSampledEvery(time.Second * 10),
		}

		actual := it.SatisfiesRestrictor(sb)
		if test.Expected.Satisfies != actual.Satisfies ||
			test.Expected.I != actual.I || test.Expected.J != actual.J {

			fmt.Printf("%s\nLines:-\n", track)
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
