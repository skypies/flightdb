package report

import(
	// "encoding/json"
	"fmt"
	"strings"

	"github.com/skypies/util/date"
	

	fdb "github.com/skypies/flightdb"
)

func init() {
	HandleReport(".list", ListReporter, "List flights meeting restrictions")
	TrackSpec(".list", []string{"fr24", "ADSB", "MLAT", "FA", "FOIA",})
}

var(
	ListReporterHeaders = []string{
		"ID", "FLIGHTNUMBER","EQUIP","ORIGIN","DESTINATION","TAGS",
		"DATETIME(PST)", "YEAR(PST)", "MONTH(PST)","DAY(PST)","TIME(PST)",
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
		f.IdentString(), f.IataFlight(), f.EquipmentType, f.Origin, f.Destination,
		strings.Join(f.TagList(), " "),
	}

	// Generate market distribution for matches
	if f.Origin == "SFO" || f.Destination == "SFO" {
		r.I[fmt.Sprintf("[F] %s:%s", f.Origin, f.Destination)]++
	}
	
	bucketsAdded := false
	addTrackpointIntersection := func(tp fdb.Trackpoint) {
		tpInPT := date.InPdt(tp.TimestampUTC)
		textrow = append(textrow, []string{
			tpInPT.Format("01/02/2006 15:04"),
			tpInPT.Format("2006"),
			tpInPT.Format("01"),
			tpInPT.Format("02"),
			tpInPT.Format("15:04:05"),
			fmt.Sprintf("%.0f", tp.Altitude),
			fmt.Sprintf("%.0f", tp.GroundSpeed),
		}...)

		if !bucketsAdded {
			r.I[fmt.Sprintf("[D] %s", alt2bkt(tp.Altitude))]++
			r.I[fmt.Sprintf("[E] %s", speed2bkt(tp.GroundSpeed))]++
			bucketsAdded = true // Only once per flight

			// Add first to HTML output, too
			htmlrow = append(htmlrow, []string{
				tpInPT.Format("15:04"),
			}...)
		}
	}

	if len(intersections) > 0 {
		addTrackpointIntersection(intersections[0].Start)
	} else if len(r.Waypoints) > 0 {
		for _,wpName := range r.Waypoints {
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
/*	
	dstr := fmt.Sprintf("*** %s\n", f.IdSpec())
	dstr += fmt.Sprintf("    %s\n", f.IdentityString())
	dstr += fmt.Sprintf("    %s\n", f.FullString())
	dstr += fmt.Sprintf("    airframe: %s\n", f.Airframe.String())
	dstr += fmt.Sprintf("    index tags: %v\n", f.IndexTagList())	
	// s,e := f.Times()
	dstr += fmt.Sprintf("    s: %s [%s] %d\n", s, date.InPdt(s), s.Unix())
	dstr += fmt.Sprintf("    e: %s [%s] %d\n", e, date.InPdt(e), e.Unix())

	s2,e2 := f.Tracks["ADSB"].Times()
	dstr += fmt.Sprintf("    s2: %s [%s] %d\n", s2, date.InPdt(s2), s2.Unix())
	dstr += fmt.Sprintf("    e2: %s [%s] %d\n", e2, date.InPdt(e2), e2.Unix())

	for i,t := range f.Timeslots() {
		dstr += fmt.Sprintf("    %d: %s [%s] %d\n", i, t, date.InPdt(t), t.Unix())
	}
	for i,t := range date.Timeslots(s,e,fdb.TimeslotDuration) {
		dstr += fmt.Sprintf("   [%d] %s [%s] %d\n", i, t, date.InPdt(t), t.Unix())
	}

	r.Info(dstr+"\n\n")
*/	
/*
	f.PruneTrackContents()
	blob,_ := f.ToBlob()
	blob.Blob = []byte{}
	// jsonBytes,_ := json.MarshalIndent(blob, "", "  ")
	// r.Info(string(jsonBytes) + "\n\n")
	r.Info(fmt.Sprintf(" * %s\n", f))
	for i,t := range blob.Timeslots {
		r.Info(fmt.Sprintf(" %03d] %s, %d\n", i, t, t.Unix()))
	}
*/	

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
