// This package contains types and functions for storing / retrieving flight objects from
// datastore, memcache, and other AppEngine things.
package fgae

import(
	"fmt"
	"net/http"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"

	"github.com/skypies/util/dsprovider"
)

var Debug = false

type FlightDB struct {
	ctx        context.Context
	StartTime  time.Time
	Backend    dsprovider.DatastoreProvider
}

func NewDB(ctx context.Context) FlightDB {
	return FlightDB{
		ctx:ctx,
		StartTime:time.Now(),
		Backend: dsprovider.AppengineDSProvider{},
	}
}
func NewDBFromReq(r *http.Request) FlightDB {
	ctx,_ := context.WithTimeout(appengine.NewContext(r), 10 * time.Minute)
	return NewDB(ctx)
}

func (db *FlightDB)NewQuery() *FQuery {
	return NewFlightQuery()
}

func (db *FlightDB)NewIterator(fq *FQuery) *FlightIterator {
	return NewFlightIterator(db.Ctx(), db.Backend, fq)
}

func (db *FlightDB)Ctx() context.Context { return db.ctx }
func (db *FlightDB)HTTPClient() *http.Client { return urlfetch.Client(db.Ctx()) }

func (db *FlightDB)Debugf(format string, args ...interface{}) {
	if Debug {log.Debugf(db.Ctx(),format,args...)}
}
func (db *FlightDB)Infof(format string,args ...interface{}) {log.Infof(db.Ctx(),format,args...)}
func (db *FlightDB)Errorf(format string,args ...interface{}) {log.Errorf(db.Ctx(),format,args...)}
func (db *FlightDB)Warningf(format string,args ...interface{}) {log.Warningf(db.Ctx(),format,args...)}
func (db *FlightDB)Criticalf(format string,args ...interface{}) {log.Criticalf(db.Ctx(),format,args...)}

// Perff is a debugf with a 'step' arg, and adds its own latency timings
func (db *FlightDB)Perff(step string, format string, args ...interface{}) {
	payload := fmt.Sprintf(format, args...)
	log.Debugf(db.Ctx(), "[%s] %9.6f %s", step, time.Since(db.StartTime).Seconds(), payload)
}
