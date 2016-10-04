// Package Metar provides tools for retrieving, storing and parsing Metar weather records.
package metar

import(
	"fmt"
	"sort"
	"time"
)

const StandardPressureInHg = 29.9213

// Metar: https://en.wikipedia.org/wiki/METAR, http://meteocentre.com/doc/metar.html

// {{{ Report{}

type Report struct {
	Raw                  string // METAR KTTN 051853Z 04011KT 1/2SM A3006 RMK AO2 P0002 T10171017=
	Source               string // "NOAA", maybe "wunderground" someday

	IcaoAirport          string
	Time                 time.Time
	AltimeterSettingInHg float64    // In inches of mercury, e.g. 30.06 (often only 4 sig figs)
}
func (mr Report)String() string {
	return fmt.Sprintf("%s@%s: %.4f inHg [%s]", mr.IcaoAirport,
		mr.Time.Format("2006.01.02-15:04:05-0700"), mr.AltimeterSettingInHg, mr.Source)
}

// }}}
// {{{ Archive{}

type ByTimeAsc []time.Time
func (a ByTimeAsc) Len() int           { return len(a) }
func (a ByTimeAsc) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByTimeAsc) Less(i, j int) bool { return a[i].Before(a[j]) }

type Archive struct {
	Reports      map[time.Time]*[24]Report  // key=UTC midnight; value = [24]Report
	IcaoAirport  string
}

func NewArchive() *Archive {
	a := Archive{ Reports: map[time.Time]*[24]Report{} }
	return &a
}

func (a Archive)allReports() []Report {
	keys := []time.Time{}
	for k,_ := range a.Reports { keys = append(keys, k) }
	sort.Sort(ByTimeAsc(keys))

	out := []Report{}
	for _,k := range keys {
		for _,r := range a.Reports[k] {
			if r.Raw == "" { continue }
			out = append(out, r)
		}
	}
	return out
}

func (a Archive)String() string {
	str := fmt.Sprintf("METAR for %s (%d):-\n", a.IcaoAirport, len(a.Reports))

	keys := []time.Time{}
	for k,_ := range a.Reports { keys = append(keys, k) }
	sort.Sort(ByTimeAsc(keys))

	for _,k := range keys {
		str += fmt.Sprintf("%s:-\n", k.Format("2006.01.02"))
		for i,r := range a.Reports[k] {
			if r.Raw == "" { continue }
			str += fmt.Sprintf (" %02d: %s\n", i, r)
		}
	}
	return str
}

// }}}

// {{{ a.AtUTCMidnight

func (a Archive)AtUTCMidnight(in time.Time) time.Time {
	in = in.UTC()
	return time.Date(in.Year(), in.Month(), in.Day(), 0, 0, 0, 0, in.Location())
}

// }}}

// {{{ a.Lookup

// The report for this 'hour' might be ahead of t (they're generally
// at 56m past); so rewind an hour if so.
func (a Archive)Lookup(t time.Time) *Report {
	r := a.DirectLookup(t)
	if r == nil || r.Time.After(t) {
		return a.DirectLookup(t.Add(time.Hour * -1))
	}
	return r
}

// }}}
// {{{ a.DirectLookup

// Return the report we have stored for this 'hour'
func (a Archive)DirectLookup(t time.Time) *Report {
	t = t.UTC()  // Just in case
	tMidnight := a.AtUTCMidnight(t)

	dayReports,exists := a.Reports[tMidnight]
	if !exists { return nil }

	if len(dayReports) < t.Hour() { return nil }
	r := &dayReports[t.Hour()]

	if len(r.Raw) == 0 { return nil }  // This means it was never populated

	return r
}

// }}}
// {{{ a.Add

func (a Archive)Add(r Report) {
	// Get the UTC day (00:00:00) for this report
	tMidnight := time.Date(r.Time.Year(), r.Time.Month(), r.Time.Day(), 0, 0, 0, 0, r.Time.Location())

	if _,exists := a.Reports[tMidnight]; !exists {
		reps := [24]Report{}
		a.Reports[tMidnight] = &reps
	}

	// The 'normal' ones are at 56m past the hour, but results appear in
	// descending time; don't overwrite results with a later timestamp.
	if orig := a.Reports[tMidnight][r.Time.Hour()]; orig.Raw != "" {
		if orig.Time.After(r.Time) { return }
	}

	a.Reports[tMidnight][r.Time.Hour()] = r
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
