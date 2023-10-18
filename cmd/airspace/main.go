package main

import(
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/prototext"
	
	"github.com/skypies/flightdb/fr24"
	"github.com/skypies/geo/sfo"
)

var(
	host = "data-feed.flightradar24.com:443"// https://data-feed.flightradar24.com:443/fr24.feed.api.v1.Feed/LiveFeed

	defaultHeaders = metadata.New(map[string]string{
    "fr24-device-id": "web-10000000000000000000000000000000",

    //"User-Agent": "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/116.0",
    //"Accept": "*/*",
    //"Accept-Language": "en-US,en;q=0.5",
    //"Accept-Encoding": "gzip, deflate, br",
    //"X-User-Agent": "grpc-web-javascript/0.1",
    //"X-Grpc-Web": "1",
    // "Origin": "https://www.flightradar24.com", // triggers "HTTP 464 (); malformed header: missing HTTP content-type"
    //"DNT": "1",
    //"Connection": "keep-alive",
    //"Referer": "https://www.flightradar24.com/",
    //"Sec-Fetch-Dest": "empty",
    //"Sec-Fetch-Mode": "cors",
    //"Sec-Fetch-Site": "same-site",
    //"TE": "trailers",
    //"x-envoy-retry-grpc-on": "unavailable",
    //"Content-Type": "application/grpc-web+proto",
	})
)

func main() {
	// doit()

	as, err := fr24.GRPCFetchAirspace(sfo.KLatlongSFO.Box(320,320))
	if err != nil {
		log.Printf("fr24.GRPCFetchAirspace error: %v\n", err)
	}
	log.Printf("Airspace :-\n%s\n", as)
}

func doit() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	md := defaultHeaders
	ctx = metadata.NewOutgoingContext(ctx, md)

	tlscfg := cert2creds("/home/abw/sslkey-go-client")
	creds := credentials.NewTLS(tlscfg)

	//if err := grpc.SetHeader(ctx, md); err != nil {
	//	log.Fatal("wtf, %v\n", err)
	//}
	
	conn, err := grpc.Dial(host, grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	feed := fr24.NewFeedClient(conn)

	resp, err := feed.LiveFeed(ctx, newReq())
	if err != nil {
		log.Fatalf("client.Feed(_) = _, %v: ", err)
	}
	fmt.Printf("REPONSES!!!:\n%#v\n", resp)
}

/*
DEFAULT_REQUEST = LiveFeedRequest(
    bounds=LiveFeedRequest.Bounds(
        north=28.12,
        south=12.89,
        west=96.04,
        east=129.82
    ),
    settings=LiveFeedRequest.Settings(
        sources_list=list(range(10)),
        services_list=list(range(12)),
        traffic_type=LiveFeedRequest.Settings.TrafficType.ALL,
    ),
    field_mask=LiveFeedRequest.FieldMask(
        field_name=["flight", "reg", "route", "type", "schedule"]
        # auth required: squawk, vspeed, airspace
    ),
    stats=True,
    limit=1500,
    maxage=14400,
    # selected_flightid=[0x31da4a31, 0x31da4a31],
    selected_flightid=[ 836705876, 836711386, 836709426, 836686454 ],
)

*/

func newReq() *fr24.LiveFeedRequest {
	bounds := sfo.KLatlongSFO.Box(320,320)
	
	req := fr24.LiveFeedRequest{
		Bounds: &fr24.LiveFeedRequest_Bounds{
			North: float32(bounds.NE.Lat),
			South: float32(bounds.SW.Lat),
			West:  float32(bounds.SW.Long),
			East:  float32(bounds.NE.Long),
		},
		Settings: &fr24.LiveFeedRequest_Settings{
			SourcesList: []fr24.DataSource{
				fr24.DataSource_ADSB,
				fr24.DataSource_MLAT,
				fr24.DataSource_FLARM,
				fr24.DataSource_FAA,
				fr24.DataSource_ESTIMATED,	
			},
			ServicesList: []fr24.Service{
				fr24.Service_PASSENGER,
				fr24.Service_CARGO,
				fr24.Service_MILITARY_AND_GOVERNMENT,
				fr24.Service_MILITARY_AND_GOVERNMENT,
				fr24.Service_BUSINESS_JETS,
				fr24.Service_GENERAL_AVIATION,
			},
			TrafficType: fr24.LiveFeedRequest_Settings_ALL,
		},
		FieldMask: &fr24.LiveFeedRequest_FieldMask{
			FieldName: []string{"flight", "reg", "route", "type", "schedule"},
		},
		Stats: true,
		Limit: 100,
		Maxage: 600,
	}

	//req = fr24.LiveFeedRequest{}

	fmt.Printf("---- Outbound request:-\n%s----\n", prototext.Format(&req))
	
	return &req
}

// Only need this for debugging; else can use empty tls.Config{}
func cert2creds(keylog string) *tls.Config {
	// Get the http2 stack to write per-session keys into this file, for wireshark
	w, err := os.OpenFile(keylog, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatal(err)
	}
_=w
	
	return &tls.Config{
		// KeyLogWriter: w,
	}
}
