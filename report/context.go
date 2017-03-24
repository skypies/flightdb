package report

import(
	"strings"

	"golang.org/x/net/context"
	"google.golang.org/appengine/user"

	"github.com/skypies/flightdb/metar"
)

// TODO: just embed / extract this data from the regular context
type ReportingContext struct {
	context.Context
	metar.Archive
	UserEmail       string
}

var(
	// Lower case everything
	ACLFOIA = []string{
		"adam@worrall.cc",
		"raymonde.guindon@gmail.com",
		"rguindon@alumni.stanford.edu",
		"meekgee@gmail.com",
		"nancyjordan650@gmail.com",
		"borg@smwlaw.com",
		"matt@classalameda.com",
		"jnelson@wiai.com",
		"robert.holbrook@gmail.com",
		"jtsunnyvaleair1@gmail.com",
		"mthurd2003@gmail.com",
		"fabrice.beretta@gmail.com",
	}
)

func (r *Report)setupReportingContext(ctx context.Context) error {
	r.ReportingContext.Context = ctx
	
	metar,err := metar.LookupArchive(ctx, "KSFO",
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
	email := r.ReportingContext.UserEmail
	if userInList(email, ACLFOIA) {
		r.Options.CanSeeFOIA = true
	}
}

func userInList(user string, acl []string) bool {
	for _,e := range acl {
		if strings.ToLower(user) == strings.ToLower(e) { return true }
	}
	return false
}
