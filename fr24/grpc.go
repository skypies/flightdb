package fr24

// Uses the 2023.10 gRPC API for fr24

import(
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"regexp"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	//"google.golang.org/protobuf/encoding/prototext"

	fdb "github.com/skypies/flightdb"
	"github.com/skypies/adsb"
	"github.com/skypies/geo"
	"github.com/skypies/pi/airspace"
)

var(
	grpcUrl = "data-feed.flightradar24.com:443"// https://data-feed.flightradar24.com:443/fr24.feed.api.v1.Feed/LiveFeed
)

func GRPCFetchAirspace(box geo.LatlongBox) (*airspace.Airspace, error) {
	as := airspace.NewAirspace()

	snapshots, err := GRPCGetFeed(box)
	if err != nil {
		return &as, err
	}

	counts := map[string]int{}
	for _, snap := range snapshots {
		if snap.Altitude < 10 { continue } // fr24 has lots of aircraft on the ground

		// An airspace usually uses IcaoID as a key. But we want to be able to support
		// an airspace containing skypi & fr24 data; so we prefix the fr24 data.
		// Plus, for fr24, we need to be able to support FAA data which has no IcaoID.
		// So, we use a distinct key, and pretend it is an IcaoID.
		key := snap.IcaoId
		if key == "" {
			counts[snap.Callsign]++
			key = fmt.Sprintf("X%s%02d", snap.Callsign, counts[snap.Callsign])
		}
		key = "EE"+key // Prefix, to avoid collisions
		
		as.Aircraft[adsb.IcaoId(key)] = SnapshotToAircraftData(snap)
	}

	return &as, nil
}

func GRPCGetFeed(box geo.LatlongBox) ([]fdb.FlightSnapshot, error) {
	ret := []fdb.FlightSnapshot{}

	req := &LiveFeedRequest{
		Bounds: &LiveFeedRequest_Bounds{
			North: float32(box.NE.Lat),
			South: float32(box.SW.Lat),
			West:  float32(box.SW.Long),
			East:  float32(box.NE.Long),
		},
		Settings: &LiveFeedRequest_Settings{
			SourcesList: []DataSource{
				DataSource_ADSB,
				DataSource_MLAT,
				DataSource_FLARM,
				DataSource_FAA,
				DataSource_ESTIMATED,	
			},
			ServicesList: []Service{
				Service_PASSENGER,
				Service_CARGO,
				Service_MILITARY_AND_GOVERNMENT,
				Service_MILITARY_AND_GOVERNMENT,
				Service_BUSINESS_JETS,
				Service_GENERAL_AVIATION,
			},
			TrafficType: LiveFeedRequest_Settings_ALL,
		},
		FieldMask: &LiveFeedRequest_FieldMask{
			// Too many fields requested for free user
			// FieldName: []string{"flight", "reg", "route", "type", "schedule"},
			FieldName: []string{"flight", "reg", "route", "type"},
		},
		Stats: true,
		Limit: 400,
		Maxage: 600,
	}

	extraHeaders := metadata.New(map[string]string{
    "fr24-device-id": "web-10000000000000000000000000000000",
	})
	ctx := metadata.NewOutgoingContext(context.Background(), extraHeaders)

	creds := credentials.NewTLS(&tls.Config{}) // Forces go-grpc to use TLS

	conn, err := grpc.Dial(grpcUrl, grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	feed := NewFeedClient(conn)
	resp, err := feed.LiveFeed(ctx, req)
	if err != nil {
		log.Fatalf("client.Feed(_) = _, %v: ", err)
	}

	for _, f := range resp.FlightsList {
		fs := flightData2snapshot(f)
		ret = append(ret, fs)
	}

	return ret, nil
}

func flightData2snapshot(in *LiveFeedResponse_FlightData) fdb.FlightSnapshot {
	out := fdb.FlightSnapshot{
		Trackpoint: fdb.Trackpoint{
			DataSource:    "fr24",
			ReceiverName:  in.GetSource().String(),
			TimestampUTC:  time.Unix(int64(in.Timestamp), 0).UTC(),
			Heading:       float64(in.Heading),
			Latlong:       geo.Latlong{float64(in.Latitude), float64(in.Longitude)},
			GroundSpeed:   float64(in.GroundSpeed),
			Altitude:      float64(in.Altitude),

			// Require extra entries in FieldMask, and also require auth
			//VerticalRate:  v[16].(float64),
			//Squawk:        v[7].(string),
		},
	}

	if in.ExtraInfo != nil {
		if in.ExtraInfo.Reg != nil {
			out.Flight.Registration = *(in.ExtraInfo.Reg)
		}
		if in.ExtraInfo.Type != nil {
			out.Flight.EquipmentType = *(in.ExtraInfo.Type)
		}
	}

	// The flightnumber, if present, takes precedence over any number we parse out of the
	// callsign.
	flightData2flightIdentity(in, &out.Flight.Identity)
	out.Flight.ParseCallsign()

	return out
}

func flightData2flightIdentity(in *LiveFeedResponse_FlightData, id *fdb.Identity) {
	if id.ForeignKeys == nil { id.ForeignKeys = map[string]string{} }
	id.ForeignKeys["fr24"] = fmt.Sprintf("%d", in.Flightid)

  //id.IcaoId = "654321" // field isn't anywhere in the new API :(
	id.Callsign = fdb.NewCallsign(in.Callsign).String()

	if in.ExtraInfo != nil {
		if in.ExtraInfo.Route != nil {
			id.Origin      = in.ExtraInfo.Route.From
			id.Destination = in.ExtraInfo.Route.To
		}

		if flightnumber := in.ExtraInfo.Flight; flightnumber != nil {	
			// FR24 copies callsigns of the form {[A-Z][0-9]+} into the flightnumber field. Undo that.
			if ! regexp.MustCompile("^[CN][0-9]+$").MatchString(*flightnumber) {
				id.ParseIata(*flightnumber) // Ignore errors
			}
		}
	}
}
