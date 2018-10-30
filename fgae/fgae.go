// This package contains types and functions for storing / retrieving flight objects from
// datastore, memcache, and other AppEngine things.
package fgae

import(
	"fmt"
	"net/http"
	"time"

	"golang.org/x/net/context"

	"github.com/skypies/util/gcp/ds"
)

type FlightDB struct {
	ctx        context.Context
	StartTime  time.Time
	Backend    ds.DatastoreProvider
}

func New(ctx context.Context, p ds.DatastoreProvider) FlightDB {
	// TODO: should this be the place that calls ds.GetProviderOrPanic ?
	return FlightDB{
		ctx:ctx,
		StartTime:time.Now(),
		Backend: p,
	}
}

func (db *FlightDB)NewQuery() *FQuery {
	return NewFlightQuery()
}

func (db *FlightDB)NewIterator(fq *FQuery) *FlightIterator {
	return NewFlightIterator(db.Ctx(), db.Backend, fq)
}

func (db *FlightDB)Ctx() context.Context { return db.ctx }
func (db *FlightDB)HTTPClient() *http.Client { return db.Backend.HTTPClient(db.Ctx()) }

func (db *FlightDB)Debugf(format string, args ...interface{}) {
	db.Backend.Debugf(db.Ctx(), format, args...)
}
func (db *FlightDB)Infof(format string,args ...interface{}) {
	db.Backend.Infof(db.Ctx(), format, args...)
}
func (db *FlightDB)Errorf(format string,args ...interface{}) {
	db.Backend.Errorf(db.Ctx(), format, args...)
}
func (db *FlightDB)Warningf(format string,args ...interface{}) {
	db.Backend.Warningf(db.Ctx(), format, args...)
}
func (db *FlightDB)Criticalf(format string,args ...interface{}) {
	db.Backend.Criticalf(db.Ctx(), format, args...)
}

// Perff is a debugf with a 'step' arg, and adds its own latency timings
func (db *FlightDB)Perff(step string, format string, args ...interface{}) {
	payload := fmt.Sprintf(format, args...)
	db.Debugf("[%s] %9.6f %s", step, time.Since(db.StartTime).Seconds(), payload)
}
