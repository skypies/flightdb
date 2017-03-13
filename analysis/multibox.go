package analysis

import (
	//"encoding/json"
	"fmt"

	"github.com/skypies/geo"
	fdb "github.com/skypies/flightdb2"
	"github.com/skypies/flightdb2/report"
)

func init() {
	report.HandleReport("multibox", MultiBoxReporter, "Intersections against a long list of boxes")
	report.TrackSpec("multibox", []string{"fr24", "ADSB", "MLAT", "FA", "FOIA",})
}

type Box struct {
	Name        string
	geo.Latlong
 	SideKM      float64
}
func (b Box)Description() string { return fmt.Sprintf("%s@%.1fKM", b.Name, b.SideKM) }

/*
Waypoints = {'MVCPA':  ('37.390200','-122.082397'),   # 3km, 2km, 1km, 0.5km
             'SJCRF1': ('37.390455', '-121.959121'),  # 4km gets reverse flow FOIA - Chestnut & Cheeney
             'ZILED': ('37.495625', '-121.958211'),   # 29km gets reverse flow for MLAT
             'HITIR':('37.323567', '-122.007392'),    # 3km
             'ZORSA':('37.3627583', '-122.0500306'),  # 4km, 1km, 0.5km
             'JESEN':('37.294831','-121.975569'),     # 3km, 0.5km
             'PUCKK':('37.363500','-122.009667'),     # Future use: data before 2013
             'PUAKO':('37.421719','-122.028425'),      # 0.4km plus window of last four coords
                      '37.401811','-122.061139','37.399838','-122.056579'), # RNP path over Whisman 
             'MVSRA':('37.401806','-122.09327')     # 3km gets MV - Central @ Sierra Vista. For SFO.
            }
 */
var(
	boxDefinitions = []Box{
		{"MVCPA", geo.Latlong{37.390200,-122.082397},   3},   // 3km, 2km, 1km, 0.5km
		{"MVCPA", geo.Latlong{37.390200,-122.082397},   2},
		{"MVCPA", geo.Latlong{37.390200,-122.082397},   1},
		{"MVCPA", geo.Latlong{37.390200,-122.082397},   0.5},
		{"SJCRF1",geo.Latlong{37.390455,-121.959121},   4},   // 4km gets reverse flow FOIA - Chestnut & Cheeney
		{"ZILED", geo.Latlong{37.495625,-121.958211},  29},   // 29km gets reverse flow for MLAT
		{"HITIR", geo.Latlong{37.323567,-122.007392},   3},   // 3km
		{"ZORSA", geo.Latlong{37.3627583,-122.0500306}, 4},   // 4km, 1km, 0.5km
		{"ZORSA", geo.Latlong{37.3627583,-122.0500306}, 1},
		{"ZORSA", geo.Latlong{37.3627583,-122.0500306}, 0.5},
		{"JESEN", geo.Latlong{37.294831,-121.975569},   3},   // 3km, 0.5km
		{"JESEN", geo.Latlong{37.294831,-121.975569},   0.5},
		{"PUCKK", geo.Latlong{37.363500,-122.009667},   3},   // Future use: data before 2013
		{"PUAKO", geo.Latlong{37.421719,-122.028425},   0.4}, // 0.4km plus window of last four coords
		{"MVSRA", geo.Latlong{37.401806,-122.09327},    3},   // 3km gets MV - Central @ Sierra Vista. For SFO.
	}
)

func MultiBoxReporter(r *report.Report, f *fdb.Flight, tis []fdb.TrackIntersection) (report.FlightReportOutcome, error) {
	//json,_ := json.MarshalIndent(boxDefinitions,"","  ")
	//r.Info(fmt.Sprintf("%s", json))

	rstrcs := map[string]geo.Restrictor{}
	for _,defn := range boxDefinitions {
		k := defn.Description()
		rstrcs[k] = geo.LatlongBoxRestrictor{LatlongBox: defn.Latlong.Box(defn.SideKM,defn.SideKM)}
	}

	for k,gr := range rstrcs {
		satisfies,intersection,deb := f.SatisfiesGeoRestriction(gr, r.ListPreferredDataSources())
		_=intersection
		_=deb
		if satisfies {
			r.I[fmt.Sprintf("[D] intersects %s", k)]++
		}
	}

	row := []string{
		r.Links(f),
		"<code>" + f.IdentString() + "</code>",
	}

	r.AddRow(&row, &row)

	return report.Accepted, nil
}
