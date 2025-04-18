package metar

// Routines to implement an on-demand data source for NOAA metar data, stored in DS

// All 'Report' objects for the same UTC day stored into a singleton DS object, called a DayReport.
// We store one such object per airport per day.

// LookupOrFetch (via cron+ui:lookupHandler) will fetch new data from NOAA and store it, if needed.
// Lookup will simply try to look it up, and fail if not found.
// The main consumer of this stuff is prefetching by report context, which is used when
//  rendering resultsets, and also in various ui/ routines that pull the metar out of the context.
// Given reporting load, I think the memcaching can be skipped.

import(
	"fmt"
	"time"

	"context"

	"github.com/skypies/util/date"
	"github.com/skypies/util/gcp/ds"
	sprovider "github.com/skypies/util/gcp/singleton"
)

const DateFormat = "2006-01-02"

var(
	DefaultStation = "KSFO"

	ErrDayReportUninitialized = fmt.Errorf("DayReport was uninitialized")
	ErrTimeNotInDayReport = fmt.Errorf("The time was not within the DayReport's UTC day")
	ErrNotFound = fmt.Errorf("No Metar record found")
)

type DayReport struct {
	IcaoAirport     string    // E.g. "KSFO"
	Time            time.Time // A time within the UTC day for this report
	Reports       [24]Report  // One report per UTC hour; must always have exactly 24 slots!
}
func NewDayReport() *DayReport { return &DayReport{} }
func (dr *DayReport)IsInitialized() bool { 	return !dr.Time.IsZero() }

// {{{ dr.String

func (dr DayReport)String() string {
	str := "[" + dr.IcaoAirport + "] "
	if dr.Time.IsZero() {
		str += "t=0 {"
	} else {
		str += dr.Time.Format(DateFormat) + dr.Time.Format("MST")+ " {"
	}
	for _,r := range dr.Reports {
		if r.Raw == "" {
			str += " "
		} else {
			delta := StandardPressureInHg - r.AltimeterSettingInHg
			str += fmt.Sprintf("%c", pressureDeltaToRune(delta))
		}
	}
	str += "}"
	return str
}

var downRunes = []rune{
	'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j',
	'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't'}
var upRunes = []rune{
	'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J',
	'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T'}
func pressureDeltaToRune(delta float64) rune {
	n := int(50.0 * delta) // typical range of delta: [-30,30]

	if n == 0 { return '.' }

	runeset := upRunes
	if n < 0 {
		runeset = downRunes
		n *= -1
	}

	if n >= len(runeset) { n = len(runeset) -1 }
	return runeset[n]
}

// }}}
// {{{ dr.Insert

func (dr *DayReport)Insert(mr Report) error {
	if dr.Time.IsZero() {
		return ErrDayReportUninitialized
	} else if dr.Time.After(mr.Time) || dr.Time.AddDate(0,0,1).Before(mr.Time) {
		return ErrTimeNotInDayReport
	}

	dr.Reports[mr.Time.UTC().Hour()] = mr

	return nil
}

// }}}
// {{{ dr.Lookup

// Nil if nothing found
func (dr *DayReport)Lookup(t time.Time) (*Report,error) {
	if dr.Time.IsZero() {
		return nil, ErrDayReportUninitialized
	}
	if dr.Time.After(t) || dr.Time.AddDate(0,0,1).Before(t) {
		return nil, ErrTimeNotInDayReport
	}

	h := t.UTC().Hour()
	if dr.Reports[h].Raw == "" {
		return nil, nil
	}
	return &dr.Reports[h],nil
}

// }}}

// {{{ toMetarSingletonKey

func toMetarSingletonKey(loc string, t time.Time) string {
	tstamp := date.TruncateToUTCDay(t).Format("2006-01-02")
	return fmt.Sprintf("metar:%s:%s", loc, tstamp)
}

// }}}

// {{{ LookupDayReport

// Pull an entire UTC day's worth of reports.
func LookupDayReport(ctx context.Context, p ds.DatastoreProvider, loc string, t time.Time) (*DayReport, error) {
	sp := sprovider.NewProvider(p)
	dr := NewDayReport()
	t = t.UTC()
	key := toMetarSingletonKey(loc, t)

	err := sp.ReadSingleton(ctx, key, nil, dr)
	if err != nil {
		return nil,err

	} else if ! dr.IsInitialized() {
		return nil,ErrNotFound

	} else {
		return dr,nil
	}
}

// }}}
// {{{ directLookup

// This just looks up the relevant slot. No smarts about 56m past the hour. Use with care.

func directLookup(ctx context.Context, p ds.DatastoreProvider, loc string, t time.Time) (*Report, error) {

	if dr,err := LookupDayReport(ctx, p, loc, t); err != nil {
		return nil, err

	} else if mr,err := dr.Lookup(t); err != nil {
		return nil,err

	} else if mr == nil {
		return nil,ErrNotFound

	} else {
		return mr,nil
	}
}

// }}}
// {{{ LookupOrFetch

func LookupOrFetch(ctx context.Context, p ds.DatastoreProvider, loc string, t time.Time) (*Report, error, string) {
	sp := sprovider.NewProvider(p)
	dr := NewDayReport()
	prevDr := NewDayReport()  // when we're called for 00:00, we need to update 23:56 for prev day

	t = t.UTC()
	key := toMetarSingletonKey(loc, t)
	str := fmt.Sprintf("[LookupOrFetch] key=%s\n", key)
	
	err := sp.ReadSingleton(ctx, key, nil, dr)
	str += fmt.Sprintf("*** ReadSingleton\n* err : %v\n* dr  : %s\n", err, dr)

	// Try to fetch previous day; ignore errors
	prevKey := toMetarSingletonKey(loc, t.AddDate(0,0,-1))
	sp.ReadSingleton(ctx, prevKey, nil, prevDr)
	if prevDr.IsInitialized() {
		str += fmt.Sprintf("* prev: %s\n", prevDr)
	}

	shouldPersistChanges := false
	shouldPersistChangesToPrevDay := false
	if err != nil {
		str += fmt.Sprintf("*** DS lookup fail\n* err: %v\n", err)
		return nil,err,str

	} else if ! dr.IsInitialized() {
		str += fmt.Sprintf("*** DS lookup OK, but no day found\n")

		dr = NewDayReport()
		dr.IcaoAirport = loc
		dr.Time = date.TruncateToUTCDay(t)

		str += fmt.Sprintf("* fresh dr: %s\n", dr)
		shouldPersistChanges = true

	} else {
		str += fmt.Sprintf("*** DS lookup OK !\n* fetched dr: %s\n", dr)
	}

	str += fmt.Sprintf("*** dr.Lookup\n dr.Start= %s\n dr.End  = %s\n t       = %s\n",
		dr.Time, dr.Time.AddDate(0,0,1).Add(-1*time.Second), t.UTC())
	mr,err := dr.Lookup(t)
	if err != nil {
		str += fmt.Sprintf("* err: %v\n", err)
		return nil,err,str
	} else if mr == nil {
		str += fmt.Sprintf("* dr.Lookup came up empty; going to NOAA\n")

		reps,err := fetchReportsFromNOAA(p.HTTPClient(ctx), loc,
			t.Add(-1*time.Hour), t.Add(time.Hour))
		str += fmt.Sprintf("[fetchReportsFromNOAA]\n -- err=%v\n -- ar: %v\n", err, reps)

		for _,mr := range reps {
			if err := dr.Insert(mr); err == ErrTimeNotInDayReport {
				if prevDr.IsInitialized() {
					err2 := prevDr.Insert(mr)
					str += fmt.Sprintf(" -! %s {%s} %v\n", mr, prevDr, err2)
					shouldPersistChangesToPrevDay = true
				}
			}

			str += fmt.Sprintf(" -- %s {%s} %v\n", mr, dr, err)
		}

		shouldPersistChanges = true
	}

	str += fmt.Sprintf("*** final dr: %s\n", dr)
	str += fmt.Sprintf("* final mr: %s\n* shouldPersist: %v\n", mr, shouldPersistChanges)

	if shouldPersistChanges {
		if err := sp.WriteSingleton(ctx, key, nil, dr); err != nil {
			return nil,err,str
		}
	}
	if shouldPersistChangesToPrevDay {
		if err := sp.WriteSingleton(ctx, prevKey, nil, prevDr); err != nil {
			return nil,err,str
		}
	}

	return mr,nil,str
}

// }}}
// {{{ LookupReport

// This is the main API entrypoint. Does not fetch new data; only looks up from datastore.

// Metar entries are published at 56m past the hour. Appengine hourly
// cron entries can only run on the hour. The net result is that we
// will *never* have a Metar entry for the current hour; we must
// always lookup the previous hour.
func LookupReport(ctx context.Context, p ds.DatastoreProvider, loc string, t time.Time) (*Report, error) {
	return directLookup(ctx, p, loc, t.Add(time.Hour * -1)) // See comment above, about the -1h
}

// }}}
// {{{ LookupArchive

// The actually used API entrypoint. Generates a metar archive for a timespan.

func LookupArchive(ctx context.Context, p ds.DatastoreProvider, loc string, s,e time.Time) (*Archive, error) {
	ar := NewArchive()

	for _,t := range date.Timeslots(s.UTC(), e.UTC(), time.Hour) {
		mr,err := directLookup(ctx, p, loc, t)
		if err == ErrNotFound {
			continue

		} else if err != nil {
			return nil,err

		} else {
			ar.Add(*mr)
		}
	}

	return ar,nil
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
