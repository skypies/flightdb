package flightdb2

import(
	"fmt"
	"github.com/skypies/geo"
)

type FlightSnapshot struct {
	Flight
	Trackpoint
	
	PrevPos Trackpoint     // Where the aircraft is at this point in time (contains a timestamp)
	NextPos Trackpoint     // For historic results, the trackpoint that follows the time

	// If we have a reference point, figure out where this flight is in relation to it
	Reference          geo.Latlong
	DistToReferenceKM  float64  // 2D distance, between latlongs
	Dist3ToReferenceKM float64  // 3D distance, accounting for altitude (not yet elevation though)
	BearingToReference float64  // [0,360)
}

func (fs FlightSnapshot)String() string {
	return fmt.Sprintf("%s:%-21.21s @ %s", fs.IcaoId, fs.FullString(), fs.Trackpoint) //date.InPdt(fs.Pos.TimestampUTC), fs.Pos.Latlong)
}

func (fs *FlightSnapshot)LocalizeTo(refpt geo.Latlong, elevation float64) {
	altitudeDelta := fs.Trackpoint.Altitude - elevation
	fs.Reference = refpt
	fs.DistToReferenceKM = fs.Reference.DistKM(fs.Trackpoint.Latlong)
	fs.Dist3ToReferenceKM = fs.Reference.Dist3(fs.Trackpoint.Latlong, altitudeDelta)
	fs.BearingToReference = fs.Trackpoint.Latlong.BearingTowards(fs.Reference)
}


type FlightSnapshotsByDist []FlightSnapshot
func (s FlightSnapshotsByDist) Len() int      { return len(s) }
func (s FlightSnapshotsByDist) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s FlightSnapshotsByDist) Less(i, j int) bool {
	return s[i].DistToReferenceKM < s[j].DistToReferenceKM
}

type FlightSnapshotsByDist3 []FlightSnapshot
func (s FlightSnapshotsByDist3) Len() int      { return len(s) }
func (s FlightSnapshotsByDist3) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s FlightSnapshotsByDist3) Less(i, j int) bool {
	return s[i].Dist3ToReferenceKM < s[j].Dist3ToReferenceKM
}

func DebugFlightSnapshotList(snaps []FlightSnapshot) string {
	debug := "3Dist   2Dist   Brng   Hdng    Alt      Speed Equp Flight   Orig  Dest  Latlong\n"
	for _,fs := range snaps {
		debug += fmt.Sprintf(
			"%5.1fKM %5.1fKM %3.0fdeg %3.0fdeg %6.0fft %4.0fkt %4s %-8.8s %-5.5s %-5.5s (% 8.4f,%8.4f)\n",
			fs.Dist3ToReferenceKM,
			fs.DistToReferenceKM, fs.BearingToReference,
			fs.Heading, fs.Altitude, fs.GroundSpeed,
			fs.EquipmentType, fs.BestFlightNumber(), fs.Origin, fs.Destination, fs.Lat, fs.Long)
	}
	return debug
}
