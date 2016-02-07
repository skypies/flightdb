package analysis

import(
	//"github.com/skypies/geo"
	//"github.com/skypies/geo/sfo"
	fdb "github.com/skypies/flightdb2"
)

func Analyse(f *fdb.Flight) (error, string) {
	proc,str,err := MatchProcedure(f.AnyTrack())
	if err != nil {
		return err, str
	} else if proc != nil {
		f.SetTag(proc.Name)
	}

	return nil, str
}
