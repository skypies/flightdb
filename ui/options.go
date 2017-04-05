package ui

import(
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"time"
	"golang.org/x/net/context"

	"github.com/skypies/util/widget"
	fdb "github.com/skypies/flightdb"
	"github.com/skypies/flightdb/fpdf"
	"github.com/skypies/flightdb/report"
)


// Common parameters for UI rendering, as parsed from CGI params
type UIOptions struct {
	UserEmail       string // if nil, user not logged in
	LoginUrl        string
	
	Permalink       string
	ResultsetID     string
	IdSpecStrings []string
	Report         *report.Report  // nil if none defined; may trigger calls to datastore

	Start,End       time.Time      // From the report daterange, or guessed from idspecs
	
	ColorScheme     ColorScheme
	PDFColorScheme  fpdf.ColorScheme
}

// *shame*
func (opt UIOptions)PermalinkWithViewtype(view string) string {
	re := regexp.MustCompile("viewtype=[a-z]*")
	return re.ReplaceAllLiteralString(opt.Permalink, "viewtype="+view)
}

// Parse a full set of UI Options
//  &idspec=...,...    OR    &resultset=asdasdasdasdasd
func FormValueUIOptions(ctx context.Context, r *http.Request) (UIOptions, error) {
	opt := UIOptions{
		IdSpecStrings: formValueIdSpecStrings(r),
		ResultsetID: r.FormValue("resultset"),
		ColorScheme: FormValueColorScheme(r),
		PDFColorScheme: formValuePDFColorScheme(r),
	}
	
	// Try and guess some start/end times for the dataset in question; add paranoid buffers
	if r.FormValue("rep") != "" {
		// This may trigger DS reads, to pull up RestrictorSets
		if rep,err := report.SetupReport(ctx, r); err != nil {
			return opt, fmt.Errorf("report parse error: %v", err)
		} else {
			opt.Report = &rep
			opt.Start = rep.Start.AddDate(0,0,-1)
			opt.End = rep.End.AddDate(0,0,1)
		}

	} else {
		// Guess the time range, based on nothing more than the timestamps embedded in the idspecs,
		// and a three-hour buffer either side.
		if len(opt.IdSpecStrings) > 0 {
			idspec,_ := fdb.NewIdSpec(opt.IdSpecStrings[0])
			min,max := idspec.Time,idspec.Time
			for _,idspecstring := range(opt.IdSpecStrings[1:]) {
				idspec,_ := fdb.NewIdSpec(idspecstring)
				if idspec.Before(min) { min = idspec.Time}
				if idspec.After(max) { max = idspec.Time}
			}
			opt.Start = min.Add(-3*time.Hour)
			opt.End = max.Add(3*time.Hour)
		}
	}
	
	return opt,nil
}


func (opt UIOptions)IdSpecs() ([]fdb.IdSpec, error) {
	ret := []fdb.IdSpec{}

	for _,str := range opt.IdSpecStrings {
		id,err := fdb.NewIdSpec(str)
		//if err != nil { continue } // FIXME - why does this happen ? e.g. ACA564@1389250800
		if err != nil { return nil, err }
		ret = append(ret, id)
	}
	
	return ret, nil
}

func formValuePDFColorScheme(r *http.Request) fpdf.ColorScheme {
	switch r.FormValue("colorby") {
	case "delta": return fpdf.ByDeltaGroundspeed
	case "plot": return fpdf.ByPlotKind
	default: return fpdf.ByGroundspeed
	}
}

// Presumes a form field 'idspec', as per identity.go, and also maxflights (as a cap)
// Supports &resultset=asdasdasdasdasdasda (a key that comes from results page)
// Note that the magic UIOptionsHandler thing will transparently save/load idpsecs into resultsets
func formValueIdSpecStrings(r *http.Request) ([]string) {
	idspecs := widget.FormValueCommaSepStrings(r, "idspec")
	
	// If asked for a random subset, go get 'em
	maxFlights := widget.FormValueInt64(r, "maxflights")	
	if maxFlights > 0 && len(idspecs) > int(maxFlights) {
		randomSubset := map[string]int{}

		for i:=0; i<int(maxFlights * 10); i++ {
			if len(randomSubset) >= int(maxFlights) { break }
			randomSubset[idspecs[rand.Intn(len(idspecs))]]++
		}
		
		idspecs = []string{}
		for id,_ := range randomSubset {
			idspecs = append(idspecs, id)
		}
	}

	return idspecs
}
