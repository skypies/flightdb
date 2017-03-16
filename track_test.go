package flightdb

// go test -v github.com/skypies/flightdb2

import (
	"encoding/json"
	"fmt"
	//"math"
	"testing"
	//"time"
)

var(
	// t1: a contiguous real track, broken in half into t1a and t1b.
	t1a = []byte(`[
{"TimestampUTC":"2016-01-01T21:36:08.217Z","Lat":37.23262,"Long":-122.06646,"Altitude":19025,"GroundSpeed":433,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:08.767Z","Lat":37.23178,"Long":-122.06539,"Altitude":19050,"GroundSpeed":433,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:11.176Z","Lat":37.22815,"Long":-122.06073,"Altitude":19125,"GroundSpeed":434,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:11.646Z","Lat":37.22731,"Long":-122.05968,"Altitude":19150,"GroundSpeed":434,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:12.566Z","Lat":37.22617,"Long":-122.05822,"Altitude":19175,"GroundSpeed":434,"Heading":134}]`)
	t1b = []byte(`[
{"TimestampUTC":"2016-01-01T21:36:12.987Z","Lat":37.22562,"Long":-122.05752,"Altitude":19200,"GroundSpeed":434,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:14.416Z","Lat":37.22363,"Long":-122.055,"Altitude":19250,"GroundSpeed":434,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:14.977Z","Lat":37.2225,"Long":-122.05355,"Altitude":19275,"GroundSpeed":434,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:16.517Z","Lat":37.22026,"Long":-122.05068,"Altitude":19325,"GroundSpeed":434,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:17.026Z","Lat":37.21967,"Long":-122.04992,"Altitude":19350,"GroundSpeed":434,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:17.526Z","Lat":37.21884,"Long":-122.04885,"Altitude":19375,"GroundSpeed":435,"Heading":134}]`)

	// t2: different days; t2b is from a different day in the future
	t2a = []byte(`[
{"TimestampUTC":"2016-01-04T21:36:08.217Z","Lat":37.23262,"Long":-122.06646,"Altitude":19025,"GroundSpeed":433,"Heading":134},
{"TimestampUTC":"2016-01-04T21:36:08.767Z","Lat":37.23178,"Long":-122.06539,"Altitude":19050,"GroundSpeed":433,"Heading":134},
{"TimestampUTC":"2016-01-04T21:36:11.176Z","Lat":37.22815,"Long":-122.06073,"Altitude":19125,"GroundSpeed":434,"Heading":134}]`)
	t2b = []byte(`[
{"TimestampUTC":"2016-01-06T21:36:12.987Z","Lat":37.22562,"Long":-122.05752,"Altitude":19200,"GroundSpeed":434,"Heading":134},
{"TimestampUTC":"2016-01-06T21:36:14.416Z","Lat":37.22363,"Long":-122.055,"Altitude":19250,"GroundSpeed":434,"Heading":134},
{"TimestampUTC":"2016-01-06T21:36:14.977Z","Lat":37.2225,"Long":-122.05355,"Altitude":19275,"GroundSpeed":434,"Heading":134}]`)

	// t3: proposed extention precedes the one we have
	t3a = []byte(`[
{"TimestampUTC":"2016-01-04T21:36:08.217Z","Lat":37.23262,"Long":-122.06646,"Altitude":19025,"GroundSpeed":433,"Heading":134},
{"TimestampUTC":"2016-01-04T21:36:08.767Z","Lat":37.23178,"Long":-122.06539,"Altitude":19050,"GroundSpeed":433,"Heading":134},
{"TimestampUTC":"2016-01-04T21:36:11.176Z","Lat":37.22815,"Long":-122.06073,"Altitude":19125,"GroundSpeed":434,"Heading":134}]`)
	t3b = []byte(`[
{"TimestampUTC":"2016-01-04T21:31:12.987Z","Lat":37.22562,"Long":-122.05752,"Altitude":19200,"GroundSpeed":434,"Heading":134},
{"TimestampUTC":"2016-01-04T21:31:14.416Z","Lat":37.22363,"Long":-122.055,"Altitude":19250,"GroundSpeed":434,"Heading":134},
{"TimestampUTC":"2016-01-04T21:31:14.977Z","Lat":37.2225,"Long":-122.05355,"Altitude":19275,"GroundSpeed":434,"Heading":134}]`)
	
	// t6: contains some misordering points; t6a overlaps with t6b (last two of 6a were [2,3] of 6b)
	t6a = []byte(`[
{"TimestampUTC":"2016-01-01T21:36:08.217Z","Lat":37.23262,"Long":-122.06646,"Altitude":19025,"GroundSpeed":433,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:08.767Z","Lat":37.23178,"Long":-122.06539,"Altitude":19050,"GroundSpeed":433,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:11.176Z","Lat":37.22815,"Long":-122.06073,"Altitude":19125,"GroundSpeed":434,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:11.646Z","Lat":37.22731,"Long":-122.05968,"Altitude":19150,"GroundSpeed":434,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:12.987Z","Lat":37.22562,"Long":-122.05752,"Altitude":19200,"GroundSpeed":434,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:14.416Z","Lat":37.22363,"Long":-122.055,"Altitude":19250,"GroundSpeed":434,"Heading":134}]`)
	t6b = []byte(`[
{"TimestampUTC":"2016-01-01T21:36:12.566Z","Lat":37.22617,"Long":-122.05822,"Altitude":19175,"GroundSpeed":434,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:14.977Z","Lat":37.2225,"Long":-122.05355,"Altitude":19275,"GroundSpeed":434,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:16.517Z","Lat":37.22026,"Long":-122.05068,"Altitude":19325,"GroundSpeed":434,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:17.026Z","Lat":37.21967,"Long":-122.04992,"Altitude":19350,"GroundSpeed":434,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:17.526Z","Lat":37.21884,"Long":-122.04885,"Altitude":19375,"GroundSpeed":435,"Heading":134}]`)

	// t7: given a distance of X, but a time gap of Y, conclude it took too long
	t7a = []byte(`[
{"TimestampUTC":"2016-01-01T20:36:08.217Z","Lat":37.23262,"Long":-122.06646,"Altitude":19025,"GroundSpeed":433,"Heading":134},
{"TimestampUTC":"2016-01-01T20:36:08.767Z","Lat":37.23178,"Long":-122.06539,"Altitude":19050,"GroundSpeed":433,"Heading":134},
{"TimestampUTC":"2016-01-01T20:36:38.296Z","Lat":37.18956,"Long":-122.01159,"Altitude":20000,"GroundSpeed":441,"Heading":134}]`)
	t7b = []byte(`[
{"TimestampUTC":"2016-01-01T21:36:08.217Z","Lat":37.10262,"Long":-121.96646,"Altitude":19025,"GroundSpeed":433,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:08.767Z","Lat":37.10178,"Long":-121.96539,"Altitude":19050,"GroundSpeed":433,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:38.296Z","Lat":37.08956,"Long":-121.91159,"Altitude":20000,"GroundSpeed":441,"Heading":134}]`)


	
	// This is a real track, for reference 
	tN = []byte(`[
{"TimestampUTC":"2016-01-01T21:36:08.217Z","Lat":37.23262,"Long":-122.06646,"Altitude":19025,"GroundSpeed":433,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:08.767Z","Lat":37.23178,"Long":-122.06539,"Altitude":19050,"GroundSpeed":433,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:11.176Z","Lat":37.22815,"Long":-122.06073,"Altitude":19125,"GroundSpeed":434,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:11.646Z","Lat":37.22731,"Long":-122.05968,"Altitude":19150,"GroundSpeed":434,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:12.566Z","Lat":37.22617,"Long":-122.05822,"Altitude":19175,"GroundSpeed":434,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:12.987Z","Lat":37.22562,"Long":-122.05752,"Altitude":19200,"GroundSpeed":434,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:14.416Z","Lat":37.22363,"Long":-122.055,"Altitude":19250,"GroundSpeed":434,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:14.977Z","Lat":37.2225,"Long":-122.05355,"Altitude":19275,"GroundSpeed":434,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:16.517Z","Lat":37.22026,"Long":-122.05068,"Altitude":19325,"GroundSpeed":434,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:17.026Z","Lat":37.21967,"Long":-122.04992,"Altitude":19350,"GroundSpeed":434,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:17.526Z","Lat":37.21884,"Long":-122.04885,"Altitude":19375,"GroundSpeed":435,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:18.527Z","Lat":37.21715,"Long":-122.04671,"Altitude":19400,"GroundSpeed":435,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:19.637Z","Lat":37.21545,"Long":-122.04455,"Altitude":19450,"GroundSpeed":435,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:20.617Z","Lat":37.21431,"Long":-122.04309,"Altitude":19475,"GroundSpeed":435,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:21.726Z","Lat":37.21262,"Long":-122.04092,"Altitude":19525,"GroundSpeed":435,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:22.296Z","Lat":37.21176,"Long":-122.03989,"Altitude":19550,"GroundSpeed":435,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:23.176Z","Lat":37.21065,"Long":-122.0384,"Altitude":19575,"GroundSpeed":436,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:23.587Z","Lat":37.21004,"Long":-122.03769,"Altitude":19600,"GroundSpeed":436,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:24.556Z","Lat":37.20836,"Long":-122.03555,"Altitude":19625,"GroundSpeed":437,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:25.636Z","Lat":37.20667,"Long":-122.03339,"Altitude":19650,"GroundSpeed":437,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:26.107Z","Lat":37.20612,"Long":-122.03263,"Altitude":19675,"GroundSpeed":437,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:26.946Z","Lat":37.20497,"Long":-122.03122,"Altitude":19700,"GroundSpeed":437,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:27.906Z","Lat":37.20352,"Long":-122.02939,"Altitude":19725,"GroundSpeed":437,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:28.416Z","Lat":37.20297,"Long":-122.02867,"Altitude":19750,"GroundSpeed":437,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:29.256Z","Lat":37.20268,"Long":-122.0283,"Altitude":19775,"GroundSpeed":437,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:29.677Z","Lat":37.20213,"Long":-122.0276,"Altitude":19800,"GroundSpeed":437,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:30.096Z","Lat":37.20154,"Long":-122.02684,"Altitude":19800,"GroundSpeed":437,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:31.116Z","Lat":37.20013,"Long":-122.02509,"Altitude":19825,"GroundSpeed":437,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:31.517Z","Lat":37.19957,"Long":-122.02431,"Altitude":19850,"GroundSpeed":438,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:33.477Z","Lat":37.19673,"Long":-122.02073,"Altitude":19900,"GroundSpeed":439,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:33.906Z","Lat":37.19584,"Long":-122.01959,"Altitude":19900,"GroundSpeed":439,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:34.357Z","Lat":37.19528,"Long":-122.01888,"Altitude":19925,"GroundSpeed":439,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:35.376Z","Lat":37.19385,"Long":-122.01708,"Altitude":19950,"GroundSpeed":439,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:35.806Z","Lat":37.1933,"Long":-122.01632,"Altitude":19950,"GroundSpeed":439,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:36.837Z","Lat":37.19156,"Long":-122.01416,"Altitude":19975,"GroundSpeed":440,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:37.877Z","Lat":37.19012,"Long":-122.01237,"Altitude":20000,"GroundSpeed":440,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:38.296Z","Lat":37.18956,"Long":-122.01159,"Altitude":20000,"GroundSpeed":441,"Heading":134},
{"TimestampUTC":"2016-01-01T21:36:39.276Z","Lat":37.18785,"Long":-122.00943,"Altitude":20025,"GroundSpeed":441,"Heading":134}]`)
)

func loadTrack(b []byte) Track {
	t := Track{}
	if err := json.Unmarshal(b, &t); err != nil {
		fmt.Printf("BAD TRACK: %v\n", err)
	}
	return t
}

func testTracks(t *testing.T, b1, b2 []byte, expected bool, descrip string) {
	tA, tB := loadTrack(b1), loadTrack(b2)
	actual,debug := tA.PlausibleExtension(&tB)
	if actual != expected {
		fmt.Printf("tA: %s\ntB: %s\n* debug:-\n%s\n", tA, tB, debug)
		t.Errorf("Track test '%s' - expected %v, got %v", descrip, expected, actual)
	}
}

func TestPlausibleExtension(t *testing.T) {
	testTracks(t, t1a,t1b, true,  "Contiguous tracks")
	testTracks(t, t2a,t2b, false, "On different days")
	testTracks(t, t3a,t3b, false, "From the past")
	testTracks(t, t6a,t6b, true,  "Misordered, overlapping") // Should have a space overlap ?
	testTracks(t, t7a,t7b, false,  "Took too long to cover gap")
}


/*

type WinAvgTest struct {
	I   int
	Dur string
	GroundSpeed float64
}

func TestWindowedAverageAt(t *testing.T) {
	tr := loadTrack(tN)

	testcases := []WinAvgTest{
		{ 0, "0.0001s", 433},  // Too small; u==v==0
		{12, "0.0001s", 435},  // Too small; u==v==6

		{ 1, "6s",      433.1603}, // u=0,v=3 - doesn't fall off beginning
		{36, "6s",      440.4974}, // u=32,v=37 - doesn't fall off end

		{18, "20s",     436.4713}, // u=7,v=31 - big avg

		{34, "2.07s",   440.0}, // u=33,v=34 (i==v)
		{35, "1.90s",   440.0}, // u=35,v=37 (i==u)
	}

	for i,testcase := range testcases {
		dur,_ := time.ParseDuration(testcase.Dur)
		itp := tr.WindowedAverageAt(testcase.I, dur)
		if math.Abs(itp.GroundSpeed-testcase.GroundSpeed) > 0.0001 {
			//fmt.Printf("----\n%s\n%s\n--\n%s\n%s\n%s\n", tr,tr.LongString(),tr[testcase.I],itp,itp.Notes)
			t.Errorf("WinAvg test %d: got %f, wanted %f\n", i, itp.GroundSpeed, testcase.GroundSpeed)
		}
	}
}
*/
