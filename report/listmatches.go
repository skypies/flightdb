package report

import(
	"fmt"
	"strings"
	fdb "github.com/skypies/flightdb2"
)

func init() {
	HandleReport(".list", ListReporter, "List flights meeting restrictions")
	TrackSpec(".list", []string{"fr24", "ADSB", "MLAT", "FA", "FOIA",})
}

var(
	ListReporterHeaders = []string{
		"ID", "FLIGHTNUMBER","ORIGIN","DESTINATION","TAGS",
		"YEAR(PST)", "MONTH(PST)","DAY(PST)","TIME(PST)",
		"ALTITUDE(FEET)","GROUNDSPEED(KNOTS)",
	}
)

func ListReporter(r *Report, f *fdb.Flight, intersections []fdb.TrackIntersection) (FlightReportOutcome, error) {	
	s,e := f.Times()
	dur := e.Sub(s)

	waypoints := strings.Join(f.WaypointList(), ",")
	if len(waypoints) > 35 {
		waypoints = waypoints[:35] + "..."
	}
	
	htmlrow := []string{
		r.Links(f),
		"<code>" + f.FullString() + "</code>",
		fmt.Sprintf("[<small>%s</small>]", strings.Join(f.TagList(), ",")),
		fmt.Sprintf("{<small>%s</small>}", waypoints),
		fmt.Sprintf("+%s", dur),
	}

	textrow := []string{
		f.IdentString(), f.IataFlight(), f.Origin, f.Destination, strings.Join(f.TagList(), " "),
	}

	// Generate market distribution for matches
	if f.Origin == "SFO" || f.Destination == "SFO" {
		r.I[fmt.Sprintf("[F] %s:%s", f.Origin, f.Destination)]++
	}
	
	bucketsAdded := false
	addTrackpointIntersection := func(tp fdb.Trackpoint) {
		textrow = append(textrow, []string{
			tp.TimestampUTC.Format("2006"),
			tp.TimestampUTC.Format("01"),
			tp.TimestampUTC.Format("02"),
			tp.TimestampUTC.Format("15:04:05"),
			fmt.Sprintf("%.0f", tp.Altitude),
			fmt.Sprintf("%.0f", tp.GroundSpeed),
		}...)

		if !bucketsAdded {
			r.I[fmt.Sprintf("[D] %s", alt2bkt(tp.Altitude))]++
			r.I[fmt.Sprintf("[E] %s", speed2bkt(tp.GroundSpeed))]++
			bucketsAdded = true // Only once per flight
		}
	}

	if len(intersections) > 0 {
		addTrackpointIntersection(intersections[0].Start)
	} else if len(r.HackWaypoints) > 0 {
		for _,wpName := range r.HackWaypoints {
			if trackName,i := f.AtWaypoint(wpName); trackName != "" {
				track := f.Tracks[trackName]
				// for interpolation, see DistAlongLine - or is it busted ?
				addTrackpointIntersection((*track)[i])
			}
		}
	}
	
	for _,ti := range intersections {
		htmlrow = append(htmlrow, ti.RowHTML()...)
	}

	r.AddRow(&htmlrow, &textrow)
	r.SetHeaders(ListReporterHeaders)
	
	return Accepted, nil
}

// Helpers
func alt2bkt(f float64) string {
	g := float64(int((f+500)/1000.0))  // Round to nearest thousand: 11499 -> 11, 11501 -> 12	
	return fmt.Sprintf("altband: %05.0f-%05.0f", g*1000-500, g*1000+499)
}

func speed2bkt(f float64) string {
	g := float64(int((f+10)/20.0))  // Round to nearest 20
	return fmt.Sprintf("speedband: %03.0f-%03.0f", g*20-10, g*20+9)
}
