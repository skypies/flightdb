package report

import(
	"golang.org/x/net/context"
	"google.golang.org/appengine/urlfetch"

	//fdb "github.com/skypies/flightdb2"
	"github.com/skypies/flightdb2/metar"
)

type ReportingContext struct {
	metar.Archive
	context.Context
}

func (r *Report)setupReportingContext(ctx context.Context) error {
	r.ReportingContext.Context = ctx
	
	metar,err := metar.FetchFromNOAA(urlfetch.Client(ctx), "KSFO",
		r.Options.Start.AddDate(0,0,-1), r.Options.End.AddDate(0,0,1))
	if err != nil {
		return err
	}

	r.ReportingContext.Archive = *metar

	//airframes := ref.NewAirframeCache(ctx)
	
	return nil
}
