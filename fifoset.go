package flightdb

import (
	"time"
)
var (
	KMemcacheFIFOSetKey = "singleton-gob:fifoset"
)

type FIFOItem struct {
	Created  time.Time
	Item     FlightSnapshot 
}

type FIFOSet map[string]FIFOItem

func (s FIFOSet)String() string {
	str := "{"
	for k,_ := range s {
		str += " " + k
	}
	return str + " }"
}

func fsUniqueId(fs FlightSnapshot) string {
	return fs.Airframe.Registration + ":" + fs.Identity.Callsign
}

func (s FIFOSet)Exists(fs FlightSnapshot) bool {
	_,exists := s[fsUniqueId(fs)]
	return exists
}

func (s FIFOSet)AddIfNew(fs FlightSnapshot) (addedOk bool) {
	if s.Exists(fs) {
		return false
	}
	s[fsUniqueId(fs)] = FIFOItem{ time.Now().UTC(), fs }
	return true
}

func (s FIFOSet)AgeOut(d time.Duration) {
	for k,v := range s {
		if time.Since(v.Created) > d {
			delete (s, k)
		}
	}
}

func (s FIFOSet)Remove(fs FlightSnapshot) {
	delete (s, fsUniqueId(fs))
}

// Updates the FIFOSet, adding newly flights; returns just those flights that were previousy unknown
func (f FIFOSet)FindNew(input []FlightSnapshot) []FlightSnapshot {
	output := []FlightSnapshot{}
	for _,fs := range input {
		if f.AddIfNew(fs) {
			output = append(output, fs)
		}
	}
	return output
}
