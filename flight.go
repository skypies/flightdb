package flightdb2

import(
	"fmt"
	"time"
	"github.com/skypies/date"
)

type Flight struct {
	Identity // embedded
	EquipmentType string
	Tracks map[string]*Track
	Tags map[string]int

	
	// Internal fields
	datastoreKey string
	DebugLog     string
}

func (f Flight)String() string {
	str := f.IdentString() + " "
	for k,t := range f.Tracks {
		str += fmt.Sprintf(" %s %s", k, t)
	}
	return str
}

func (f Flight)TagList() []string {
	ret := []string{}
	for tag,_ := range f.Tags {
		ret = append(ret,tag)
	}
	return ret
}

func (f Flight)Times() (s,e time.Time) {
	if len(f.Tracks) == 0 { return }
	s,_ = time.Parse("2006.01.02", "2199.01.01")
	e,_ = time.Parse("2006.01.02", "1972.01.01")
	for _,t := range f.Tracks {
		ts,te := t.Times()
		if ts.Before(s) { s = ts }
		if te.After(e)  { e = te }
	}
	return
}

// This is to help save RAM when compiling lists of flights
func (f *Flight)PruneTrackContents() {
	for k,track := range f.Tracks {
		if len(*track)>2 {
			t := Track{(*track)[0], (*track)[len(*track)-1]}
			f.Tracks[k] = &t
		}
	}
}

//// The functions below are just to support indexing & retrieval in the DB
//

func (f Flight)GetDatastoreKey() string { return f.datastoreKey }
func (f *Flight)SetDatastoreKey(k string) { f.datastoreKey = k }

func (f Flight)Timeslots(d time.Duration) []time.Time {
	s,e := f.Times()
	return date.Timeslots(s,e,d)
}

func NewFlightFromADSBTrackFragment(frag *ADSBTrackFragment) *Flight {
	f := Flight{
		Identity: Identity{
			IcaoId: string(frag.IcaoId),
			Callsign: frag.Callsign,
		},
		Tracks: map[string]*Track{},
		Tags: map[string]int{},
	}
	f.Tracks["ADSB"] = &frag.Track
	
	return &f
}

func (f Flight)ADSBTrackFragmentIsPlausible(t Track) bool {
	//currS,currE
	return true
}
