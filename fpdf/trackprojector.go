package fpdf

import(
	"github.com/skypies/geo"
	fdb "github.com/skypies/flightdb"
)

// A TrackProjector projects trackpoints into a 2D coordinate space (distNM by altitude)
type TrackProjector interface {
	Setup(fdb.Track, AnchorPoint) error
	Project(fdb.Trackpoint) (distNM float64, alt float64)
	Description() string
}

// {{{ ProjectAsCrowFlies

type ProjectAsCrowFlies struct {
	ap                      AnchorPoint
	distTravelledAtAnchorKM float64 // points with a shorter dist travelled come 'before' the anchor.
}
func (p *ProjectAsCrowFlies)Description() string {
	return "As Crow Flies (distance to anchor)"
}

func (p *ProjectAsCrowFlies)Setup(t fdb.Track, ap AnchorPoint) error {
	if i,distKM,err := ap.PointOfClosestApproach(t); err != nil {
		return err
	} else {
		p.ap = ap
		p.distTravelledAtAnchorKM = t[i].DistanceTravelledKM + distKM
	}
	return nil
}

func (p *ProjectAsCrowFlies)Project(tp fdb.Trackpoint) (float64,float64) {
	distNM := tp.DistNM(p.ap.Latlong) // simple linear distance to the anchor

	if tp.DistanceTravelledKM < p.distTravelledAtAnchorKM {
		distNM *= -1.0 // go -ve, as we've not reached the anchor
	}

	return distNM, tp.IndicatedAltitude
}

// }}}
// {{{ ProjectAlongPath

// ProjectAlongPath returns a projection function from trackpoints
// into a scalar range 'distance along path', which is -ve for
// trackpoints before the anchor, and +ve for those after. It computes
// distance flown, so flying in circles loops make values bigger.

type ProjectAlongPath struct {
	distTravelledAtAnchorKM float64 // points with a shorter dist travelled come 'before' the anchor.
}
func (p *ProjectAlongPath)Description() string { return "Along Path (i.e. distance travelled)" }

func (p *ProjectAlongPath)Setup(t fdb.Track, ap AnchorPoint) error {
	if i,distKM,err := ap.PointOfClosestApproach(t); err != nil {
		return err
	} else {
		p.distTravelledAtAnchorKM = t[i].DistanceTravelledKM + distKM
	}
	return nil
}

func (p *ProjectAlongPath)Project(tp fdb.Trackpoint) (float64,float64) {
	distFromAnchorKM := tp.DistanceTravelledKM - p.distTravelledAtAnchorKM
	return distFromAnchorKM*geo.KNauticalMilePerKM, tp.IndicatedAltitude
}

// }}}

// {{{ -------------------------={ E N D }=----------------------------------

// Local variables:
// folded-file: t
// end:

// }}}
