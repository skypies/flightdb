package main

import(
	"fmt"
	"time"

	_ "github.com/skypies/flightdb/app/backend"
	_ "github.com/skypies/flightdb/app/frontend"
	//_ "github.com/skypies/flightdb/ui" // panics if it can't find ./templates/foo.html

	"golang.org/x/net/context"
	mcprovider "github.com/skypies/util/singleton/memcache"
	"github.com/skypies/geo/sfo"

	"github.com/skypies/flightdb/ref"
	fdb "github.com/skypies/flightdb"
)

func main() {
	fmt.Printf("Hello!\n")

	if false {
		schedules()
		airframes()
	}
}

func schedules() {
	ctx := context.Background()
	sp := mcprovider.NewProvider("127.0.0.1:11211")

	sc,err := ref.LoadScheduleCache(ctx, sp)
	if err != nil {
		fmt.Printf("LoadScheduleCache err: %v\n", err)
		return
	}
	fmt.Printf("Loaded a schedcache!\n%s\n", sc)

	fs := fdb.FlightSnapshot{
		Flight: fdb.Flight{
			Airframe: fdb.Airframe{
				Registration: "N1234",
				EquipmentType: "B722",
			},
			Identity: fdb.Identity{
				IcaoId: "AE9988",
				Callsign: "SWA1234",
				Schedule: fdb.Schedule{
					Number: 1234,
					IATA: "SW",
					ICAO: "SWA",
					Origin: "SFO",
					Destination: "LAX",
				},
			},
		},
		Trackpoint: fdb.Trackpoint{
			DataSource:    "aiee",
			ReceiverName:  "MyRcvr",
			TimestampUTC:  time.Now().UTC(),
			Heading:       0.0,
			Latlong:       sfo.KLatlongSFO,
			GroundSpeed:   250.0,
			VerticalRate:  0.0,
			Altitude:      32000,
			Squawk:        "1122",
		},		
	}

	sc.Map[fs.IcaoId] = &fs

	if err := sc.SaveScheduleCache(ctx,sp); err != nil {
		fmt.Printf("SaveScheduleCache err: %v\n", err)
		return
	}

	fmt.Printf("Updated a schedcache!\n%s\n", sc)
}

func airframes() {
	ctx := context.Background()
	sp := mcprovider.NewProvider("127.0.0.1:11211")
	//sp.ErrIfNotFound = true

	airframes,err := ref.LoadAirframeCache(ctx, sp)
	if err != nil {
		fmt.Printf("LoadAirframeCache err: %v\n", err)
		return
	} 
	fmt.Printf("Cache loaded !\n%s\n", airframes)

	af := fdb.Airframe{"ABC124", "N21B", "B742", "AA"}
	airframes.Set(&af)
	fmt.Printf("\nAirfrmaes updated\n%s\n", airframes)

	if err := airframes.SaveAirframeCache(ctx,sp); err != nil {
		fmt.Printf("SaveAirframeCache err: %v\n", err)
	} else {
		fmt.Printf("Wahay, saved OK!\n")
	}
}
