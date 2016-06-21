// This package contains types and functions for storing / retrieving flight objects from
// datastore, memcache, and other AppEngine things.
package fgae

import(
	"golang.org/x/net/context"
	"google.golang.org/appengine/log"
)

const(
	kFlightKind = "flight"
)

var(
	Debug = false
)

type FlightDB struct {
	C context.Context
}

func (db *FlightDB)Debugf(format string, args ...interface{}) {
	if Debug {log.Debugf(db.C, format, args)}
}
func (db *FlightDB)Infof(format string, args ...interface{}) {log.Infof(db.C, format, args)}
func (db *FlightDB)Errorf(format string, args ...interface{}) {log.Errorf(db.C, format, args)}
func (db *FlightDB)Warningf(format string, args ...interface{}) {log.Warningf(db.C, format, args)}
func (db *FlightDB)Criticalf(format string, args ...interface{}) {log.Criticalf(db.C, format, args)}
