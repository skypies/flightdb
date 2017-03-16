// This package contains all the types for the flight database. No AppEngine imports.
package flightdb

import "time"

const(
	// This is the 'quantization' value, used to index a flight based on
	// which timeslots it overlaps. Never change this value once you've
	// started populating a database, unless you're going to regenerate it
	TimeslotDuration = 30 * time.Minute
)
