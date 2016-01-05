package flightdb2

/* Some notes on identifiers for aircraft

# ADSB Mode-[E]S Identifiers (Icao24)

These are six digit hex codes, handed out to aircraft. Most aircraft
using ADS-B have this is a (semi?) permanent 'airframe' ID, but some
spoof it.


# ADSB Callsigns

1. Many airlines use the ICAO flight number: SWA3848
2. Many private aircraft use their registration: N839AL
3. Some (private ?) aircraft use just their equipment type:
4. Annoyingly, some airlines use a bare flight number:
		1106  - was FFT 1106 (F9 Frontier)
		4517  - was SWA 4517 (WN Southwest)
		 948  - was SWA 948  (WN Southwest)
     210  - was AAY 210  (G4 Allegiant Air)

5. Various kinds of null idenitifiers:
 00000000   - was A69C72 (a Cessna Citation)
 ????????   - was A85D50 (a Virgin America A320, flying as VX938 !!)

 */

import(
	"fmt"
	"time"
)

func (f Flight)IdentString() string {
	return fmt.Sprintf("%s (%s)", f.Callsign, string(f.IcaoId))
}

func (f Flight)TrackUrl() string {
	u := fmt.Sprintf("/fdb/tracks?icaoid=%s", string(f.IcaoId))
	times := f.Timeslots(time.Minute * 30)  // ARGH
	if len(times)>0 {
		u += fmt.Sprintf("&t=%d", times[0].Unix())
	}
	return u
}
