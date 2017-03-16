package analysis

// Report disabled; needs to move to the new restrictions model

/*
import (
	"fmt"
	"sort"
	"time"

	fdb "github.com/skypies/flightdb"
	"github.com/skypies/flightdb/report"
)

func init() {
//	report.HandleReport("trains", TrainsReporter, "Flight Trains; within {duration}, when within {dist} of {refpoint}")
//	report.SummarizeReport("trains", TrainsSummarizer)
//	report.TrackSpec("trains", []string{"FA", "fr24"}) // *Not* ADSB; need data over ocean
}

// The few things we store about a flight
type TrainsFlight struct {
	Links           string
	FlightString    string
	time.Time       // embedded; time at which flight passed through reference window
	fdb.Trackpoint  // embedded; the trackpoint for the flight
}
type TrainsBlob struct { Flights []TrainsFlight }

type ByTimeAsc []TrainsFlight
func (a ByTimeAsc) Len() int           { return len(a) }
func (a ByTimeAsc) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByTimeAsc) Less(i, j int) bool { return a[i].Time.Before(a[j].Time) }

func TrainsReporter(r *report.Report, f *fdb.Flight, tis []fdb.TrackIntersection) (report.FlightReportOutcome, error) {
	blob := TrainsBlob{Flights: []TrainsFlight{}}
	if r.Blobs["trains"] != nil { blob = r.Blobs["trains"].(TrainsBlob) }

	r.I["[C] blob preprocessing"]++

	intersects,ti,_ := f.SatisfiesGeoRestriction(r.GetRefpointRestrictor(), r.TrackSpec)
	if !intersects {
		r.I["[D] flight missed entrainment refpoint"]++
		return report.RejectedByReport, nil
	}
	
	// We actually do very little here; just cache the time of the window intersection,
	// and a few details about the flight.

	tp := r.GetFirstIntersectingTrackpoint([]fdb.TrackIntersection{ti})
	if tp == nil { return report.RejectedByReport, nil }

	r.I["[D] <b>flight intersected entrainment refpoint</b>"]++
	
	tf := TrainsFlight{
		Links: r.Links(f),
		FlightString: f.IdentityString(),
		Trackpoint: *tp,
		Time: tp.TimestampUTC,
	}
	blob.Flights = append(blob.Flights, tf)

	r.Blobs["trains"] = blob	
	return report.Accepted, nil
}

func TrainsSummarizer(r *report.Report) {
	genericBlob,exists := r.Blobs["trains"]
	if !exists { return }
	blob := genericBlob.(TrainsBlob)

	// OK, now do all the hard work.
	sort.Sort(ByTimeAsc(blob.Flights))

	currTrain := []int{}  // Store the indices of the current train
	tLast := blob.Flights[0].Time.Add(time.Hour*-24)
	for i,tf := range blob.Flights {
		if tf.Sub(tLast) > r.Options.Duration {
			// This flight is not part of any existing train; so flush what we have
			if len(currTrain) > 0 {
				r.I[fmt.Sprintf("[E] trains of length=%02d", len(currTrain))]++
				currTrain = []int{}

				spacer := []string{"<hr/>"}
				r.AddRow(&spacer,&spacer)
			}
		}

		currTrain = append(currTrain, i)
		tLast = tf.Time
		
		row := []string{
			tf.Links,
			tf.Trackpoint.String(),
			tf.FlightString,
		}
		r.AddRow(&row, &row)
	}

	// Flush out the final train
	if len(currTrain) > 0 {
		r.I[fmt.Sprintf("[E] trains of length=%02d", len(currTrain))]++
	}
	
	r.S[fmt.Sprintf("[Z] Max duration for entrainment")] = r.Options.Duration.String()
}
*/
