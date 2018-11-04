package report

import(
	"fmt"
	"strings"

	"golang.org/x/net/context"
	"google.golang.org/appengine/user"

	"github.com/skypies/flightdb/fgae"
	"github.com/skypies/flightdb/metar"
)

// TODO: just embed / extract this data from the regular context
type ReportingContext struct {
	context.Context
	metar.Archive
	UserEmail       string
}

var(
	// Lower case everything. Map email address to the FOIA sources they can access.
	// An empty list means access to all FOIA sources.
	// {"mtv-foia", "EB-FOIA"}
	ACLFOIA = map[string][]string{
		"adam@worrall.cc": []string{},
		"raymonde.guindon@gmail.com": []string{},
		"rguindon@alumni.stanford.edu": []string{},
		"meekgee@gmail.com": []string{},
		"nancyjordan650@gmail.com": []string{},
		"borg@smwlaw.com": []string{},
		"matt@classalameda.com": []string{},
		"jnelson@wiai.com": []string{},
		"robert.holbrook@gmail.com": []string{},
		"jtsunnyvaleair1@gmail.com": []string{},
		"mthurd2003@gmail.com": []string{},
		"fabrice.beretta@gmail.com": []string{},
		"kaneshirokimberly@gmail.com": []string{},
		"ssrhome@gmail.com": []string{},
		"randywaldeck@gmail.com": []string{"mtv-foia"},
		"clcmilton@gmail.com": []string{"mtv-foia"},
	}
)

func (r *Report)setupReportingContext(db fgae.FlightDB) error {
	ctx := db.Ctx()
	r.ReportingContext.Context = ctx
	
	metar,err := metar.LookupArchive(ctx, db.Backend, "KSFO",
		r.Options.Start.AddDate(0,0,-1), r.Options.End.AddDate(0,0,1))
	if err != nil {
		return err
	}
	r.ReportingContext.Archive = *metar
	
	user := user.Current(ctx)
	if user != nil {
		r.ReportingContext.UserEmail = user.Email
	}

	r.AddACLs()

	//airframes := ref.NewAirframeCache(ctx)
	
	return nil
}

func (r *Report)AddACLs() {
	email := strings.ToLower(r.ReportingContext.UserEmail)
	if srcs,exists := ACLFOIA[email]; exists {
		r.Options.CanSeeFOIA = true
		r.Options.CanSeeFOIASources = srcs
	}
}

func (r *Report)CanSeeThisFOIASource(src string) (bool, string) {
	if ! r.Options.CanSeeFOIA {
		return false, "[B] Eliminated: user can't access any FOIA"
	}

	if len(r.Options.CanSeeFOIASources) == 0 {
		return true, "" // User can see all sources
	}

	for _,allowed := range r.Options.CanSeeFOIASources {
		if src == allowed {
			return true, "" // User can see this source
		}
	}

	return false, fmt.Sprintf("[B] Eliminated: user can't see FOIA from %s", src)
}
