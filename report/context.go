package report

import(
	"fmt"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/appengine/user"
	"google.golang.org/appengine/urlfetch"

	//fdb "github.com/skypies/flightdb2"
	"github.com/skypies/flightdb2/metar"
)

type ReportingContext struct {
	context.Context
	metar.Archive
	UserEmail       string
}

var(
	ACLFOIA = []string{
		"adam@worrall.cc",
		"raymonde.guindon@gmail.com",
		"rguindon@alumni.stanford.edu",
		"meekGee@gmail.com",
		"nancyjordan650@gmail.com",
	}
)

func (r *Report)setupReportingContext(ctx context.Context) error {
	r.ReportingContext.Context = ctx
	
	metar,err := metar.FetchFromNOAA(urlfetch.Client(ctx), "KSFO",
		r.Options.Start.AddDate(0,0,-1), r.Options.End.AddDate(0,0,1))
	if err != nil {
		return err
	}
	r.ReportingContext.Archive = *metar
	
	if user := user.Current(ctx); user != nil {
		r.ReportingContext.UserEmail = user.Email
	}
	
	r.AddACLs()
	if err := r.EnforceACLs(); err != nil {
		return err
	}
	
	//airframes := ref.NewAirframeCache(ctx)
	
	return nil
}

func (r *Report)AddACLs() {
	email := r.ReportingContext.UserEmail
	if userInList(email, ACLFOIA) {
		r.Options.CanSeeFOIA = true
	}
}

// This is pretty junky. A better solution depends on a way to include/exclude track types.
func (r *Report)EnforceACLs() error {
	cutoff,_ := time.Parse("2006.01.02", "2015.10.01")
	if false && r.Start.Before(cutoff) && !r.Options.CanSeeFOIA {
		return fmt.Errorf(fmt.Sprintf("User '%s' not in FOIA ACL", r.UserEmail))
	}
	return nil
}

func userInList(user string, acl []string) bool {
	for _,e := range acl {
		if user == e { return true }
	}
	return false
}
