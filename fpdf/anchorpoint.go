package fpdf

import(
	"fmt"

	"github.com/skypies/geo"
	fdb "github.com/skypies/flightdb"
)

type AnchorPoint struct {
	geo.NamedLatlong
	AltitudeMin      float64
	AltitudeMax      float64 // if max uninitialized, no checks performed
	DistMaxKM        float64 // if non-zero, must be at least this close, or we skip flight
}

func (ap AnchorPoint)String() string {
	str := ap.NamedLatlong.String()
	if ap.DistMaxKM > 0 {
		str = fmt.Sprintf("Within %.1fKM of %s", ap.DistMaxKM, str)
	}
	if ap.AltitudeMax > 0 {
		str += fmt.Sprintf(", altitude[%.0f-%.0f]ft", ap.AltitudeMin, ap.AltitudeMax)
	}
	return str
}

func (ap AnchorPoint)PointOfClosestApproach(t fdb.Track) (int,float64,error) {
	i := t.ClosestTo(ap.Latlong, ap.AltitudeMin, ap.AltitudeMax)
	if i < 0 {
		return 0,0,fmt.Errorf("AnchorPoint.POCA: nothing in alt range")
	}

	closestDistKM := t[i].DistKM(ap.Latlong)
	if ap.DistMaxKM > 0 && closestDistKM > ap.DistMaxKM {
		return 0,0,fmt.Errorf("AnchorPoint.POCA: closest[%d] too far (%f > %f) from anchor %s\n",
			i, closestDistKM, ap.DistMaxKM, ap)
	}

	return i, closestDistKM, nil
}
