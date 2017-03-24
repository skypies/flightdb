package faadata

// TODO: move the foia handlers over to this.

import(
	"fmt"
	"io"
	"sort"
	"time"

	"golang.org/x/net/context"

	fdb "github.com/skypies/flightdb"
)

// {{{ makeFlight

func makeFlight(rows []Row, tStart time.Time, genesisStr string) (*fdb.Flight, error) {
	if len(rows) == 0 { return nil, fmt.Errorf("No rows!") }

	t := fdb.Track{}
	for _,row := range rows {
		t = append(t, row.ToTrackpoint())
	}

	sort.Sort(fdb.TrackByTimestampAscending(t))
	
	f := rows[0].ToFlightSkeleton()
	f.Tracks["FOIA"] = &t
	f.SetTag("FOIA")

	tStartAnalyse := time.Now()
	f.Analyse()

	f.DebugLog += fmt.Sprintf("%s\n** full load+parse: %dus (analyse: %dus)\n",
		genesisStr,
		time.Since(tStart).Nanoseconds() / 1000,
		time.Since(tStartAnalyse).Nanoseconds() / 1000)
	
	return f,nil
}

// }}}

// {{{ ReadFrom

type NewFlightCallback func(context.Context, *fdb.Flight) (bool, string, error)

func ReadFrom(ctx context.Context, name string, rdr io.Reader, cb NewFlightCallback) (int, string,error) {

	str := fmt.Sprintf("---- Flights loaded from %s\n", name)
	i := 1
	nFlightsAdded := 0
	nPossibleFlights := 0
	tStart := time.Now()

	rows := []Row{}
	rowReader := NewRowReader(rdr)
	
	for {
		row,err := rowReader.Read()
		if err == io.EOF { break }
		if err != nil { return nFlightsAdded,str,err }

		//if nPossibleFlights > 100 {
		//	rows = nil
		//	break
		//}

		// If this row appears to be a different flight than the one we're accumulating, flush
		if len(rows)>0 && !row.FromSameFlightAs(rows[0]) {
			logPrefix := fmt.Sprintf("%s:%d-%d", name, i-len(rows), i-1)
			
			if f,err := makeFlight(rows, tStart, "Genesis: "+logPrefix+"\n"); err != nil {
				return nFlightsAdded,str,err
			} else if added,subStr,err := cb(ctx,f); err != nil {
				return nFlightsAdded,str, err
			} else {
				nPossibleFlights++
				if added { nFlightsAdded++ }
				if subStr != "" {
					str += logPrefix + ": " + subStr
				}
			}
			tStart = time.Now()
			rows = nil // reset slice
		}

		rows = append(rows, row)
		i++
	}

	if len(rows)>0 {
		logPrefix := fmt.Sprintf("%s:%d-%d", name, i-len(rows), i-1)

		if f,err := makeFlight(rows, tStart, logPrefix); err != nil {
			return nFlightsAdded,str,err
		} else if added,subStr,err := cb(ctx,f); err != nil {
			return nFlightsAdded,str,err
		} else {
			if added { nFlightsAdded++ }
			if subStr != "" {
				str += logPrefix + ": " + subStr
			}
		}
	}

	str = fmt.Sprintf("---- File read, %d rows, %d flights added (out of %d proposed)\n", i,
		nFlightsAdded, nPossibleFlights) + str

	return nFlightsAdded,str,nil
}

// }}}


// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
