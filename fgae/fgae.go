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
)

const(
	kFlightKind = "flight"
)

var(
	Debug = false
)

type FlightDB struct {
	C context.Context
	StartTime time.Time
}

func NewDB(r *http.Request) FlightDB {
	ctx,_ := context.WithTimeout(appengine.NewContext(r), 10 * time.Minute)
	return FlightDB{C:ctx, StartTime:time.Now()}
}
func (db *FlightDB)HTTPClient() *http.Client {
	return urlfetch.Client(db.C)
}
func (db *FlightDB)Ctx() context.Context { return db.C }

func (db *FlightDB)Debugf(format string, args ...interface{}) {
	if Debug {log.Debugf(db.C,format,args...)}
}
func (db *FlightDB)Infof(format string,args ...interface{}) {log.Infof(db.C,format,args...)}
func (db *FlightDB)Errorf(format string,args ...interface{}) {log.Errorf(db.C,format,args...)}
func (db *FlightDB)Warningf(format string,args ...interface{}) {log.Warningf(db.C,format,args...)}
func (db *FlightDB)Criticalf(format string,args ...interface{}) {log.Criticalf(db.C,format,args...)}

// Perff is a debugf with a 'step' arg, and adds its own latency timings
func (db *FlightDB)Perff(step string, format string, args ...interface{}) {
	payload := fmt.Sprintf(format, args...)
	log.Debugf(db.C, "[%s] %9.6f %s", step, time.Since(db.StartTime).Seconds(), payload)
}
