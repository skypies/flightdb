package swim

// This all relates to the FAA's program to distribute data.
// SWIM: System Wide Information Management (the whole big program)
// SCDS: SWIM Cloud Distribution Service
// [S]FDPS: [SWIM] Flight Data Publication Service
// NAS: National Airspace System
// NSRR: NAS Service Registry and Repository (schema library)

// The only docs on this stuff are at nsrr.faa.gov; you'll need to get an account set up.
// https://nsrr.faa.gov/sites/default/files/SFDPS_ReferenceManual_v1.2.pdf

// To get the data itself, you'll need an account on SCDS. There are
// many flavors of data; you want the FDPS flavors.

// I subscribed to these families: En Route, Airspace AIXM, Status,
// and Flight FIXM. It isn't easy to tell which received messages
// relate to which family. But one of the drops the all important [TH]
// Track Information message. We get one such message every ~12
// seconds for each active track.

// Not everything in the JSON feed is an `ns5:MessageCollection`, but
// most of them are. You can tell which kind each message is by the
// `//message/flight/source` field. I've only looked at the `TH` ones.

// SWIM encodes the JSON in an annoying way. The field
// `/ns5:MessageCollection/message` is sometimes a simple map
// (containing one key, `flight`, that maps to a `Flight` object), but
// more often it is an array of `Flight` objects. This can't be
// expressed as a single golang struct, so we end up with two, and our
// parser needs to try both and see which succeeds.

type Ns5MessageCollectionSingleMessage struct {
	Ns5MessageCollection struct {
		Message struct {
			Flight Flight  `json:"flight"`
		} `json:"message"`
	} `json:"ns5:MessageCollection"`
}
type Ns5MessageCollectionMultiMessage struct {
	Ns5MessageCollection struct {
		Message []struct {
			Flight Flight  `json:"flight"`
		} `json:"message"`
	} `json:"ns5:MessageCollection"`
}

// We just parse out the very few fields we're interested in
type Flight struct {
	Source string `json:"source"`                                   // "TH" (SFDPS message type)
	Timestamp string `json:"timestamp"`                             // "2020-04-16T04:01:14Z"
	
	EnRoute struct {
		Position struct {
			PositionTime string `json:"positionTime"`                   // "2020-04-16T04:01:14Z"
			Position struct {
				Location struct {
					Pos string `json:"pos"`                                 // "41.525556 -122.520278"
				} `json:"location"`
			} `json:"position`

			TrackVelocity struct {
				X struct {
					Content float64 `json:"content"`
				} `json:"x"`
				Y struct {
					Content float64 `json:"content"`
				} `json:"y"`
			} `json:"trackVelocity"`

			ActualSpeed struct {
				Surveillance struct {
					Content float64 `json:"content"`
				} `json:"surveillance"`
			} `json:"actualSpeed"`

			Altitude struct {
				Content float64 `json:"content"`
			} `json:"altitude"`

		} `json:"position"`
	} `json:"enRoute"`

	FlightIdentification struct {
		ComputerId float64 `json:"computerId"`                        // Is three digit int
		AircraftIdentification string `json:"aircraftIdentification"` // "SWA1988"
	} `json:flightIdentification"`
}


/* Breakdown of the NasFlightType message types (//message/flight/source = "TH" etc)

  34292    source "TH"
    963    source "HP"
    867    source "OH"
    705    source "HX"
    522    source "AH"
    191    source "LH"
    160    source "CL"
    144    source "HZ"
    127    source "FH"
     86    source "HU"
     83    source "HV"
     72    source "RH"
     67    source "HT"
     46    source "DH"
     45    source "HF"
      5    source "NP"
      5    source "NL"
      2    source "PT"
      1    source "IH"

*/

/* Examples of the various messages as JSON

// [TH]
//
{
  "ns5:MessageCollection": {
    "xmlns:ns2": "http://www.fixm.aero/base/3.0",
    "xmlns:ns5": "http://www.faa.aero/nas/3.0",
    "xmlns:ns3": "http://www.fixm.aero/foundation/3.0",
    "xmlns:ns4": "http://www.fixm.aero/flight/3.0",
    "message": {
      "flight": {
        "gufi": {
          "codeSpace": "urn:uuid",
          "content": "aced6264-5eed-48e7-8bee-d9ea6e9bbb5c"
        },
        "enRoute": {
          "xsi:type": "ns5:NasEnRouteType",
          "position": {
            "targetPositionTime": "2020-04-16T03:51:55Z",
            "positionTime": "2020-04-16T03:51:54Z",
            "actualSpeed": {
              "surveillance": {
                "uom": "KNOTS",
                "content": 465
              }
            },
            "altitude": {
              "uom": "FEET",
              "content": 40000
            },
            "targetPosition": {
              "srsName": "urn:ogc:def:crs:EPSG::4326",
              "pos": "35.733889 -126.095556"
            },
            "xsi:type": "ns5:NasAircraftPositionType",
            "position": {
              "xsi:type": "ns2:LocationPointType",
              "location": {
                "srsName": "urn:ogc:def:crs:EPSG::4326",
                "pos": "35.733056 -126.096944"
              }
            },
            "trackVelocity": {
              "x": {
                "uom": "KNOTS",
                "content": 433
              },
              "y": {
                "uom": "KNOTS",
                "content": 172
              }
            },
            "reportSource": "SURVEILLANCE",
            "targetAltitude": {
              "uom": "FEET",
              "content": 40000
            }
          }
        },
        "controllingUnit": {
          "xsi:type": "ns2:IdentifiedUnitReferenceType",
          "sectorIdentifier": 35,
          "unitIdentifier": "ZOA"
        },
        "flightIdentification": {
          "computerId": 668,
          "aircraftIdentification": "SWA5313",
          "siteSpecificPlanId": 97,
          "xsi:type": "ns5:NasFlightIdentificationType"
        },
        "arrival": {
          "xsi:type": "ns5:NasArrivalType",
          "runwayPositionAndTime": {
            "runwayTime": {
              "estimated": {
                "time": "2020-04-16T04:28:00Z"
              }
            }
          },
          "arrivalPoint": "KOAK"
        },
        "flightPlan": {
          "identifier": "KO10948500"
        },
        "xsi:type": "ns5:NasFlightType",
        "centre": "ZOA",
        "flightStatus": {
          "xsi:type": "ns5:NasFlightStatusType",
          "fdpsFlightStatus": "ACTIVE"
        },
        "supplementalData": {
          "xsi:type": "ns5:NasSupplementalDataType",
          "additionalFlightInformation": {
            "nameValue": [
              {
                "name": "MSG_SEQ_NO",
                "value": 12638523
              },
              {
                "name": "FDPS_GUFI",
                "value": "us.fdps.2020-04-16T03:02:28Z.000/14/500"
              },
              {
                "name": "FLIGHT_PLAN_SEQ_NO",
                "value": 3
              }
            ]
          }
        },
        "source": "TH",
        "operator": {
          "operatingOrganization": {
            "organization": {
              "name": "SWA"
            }
          }
        },
        "system": "ATL",
        "assignedAltitude": {
          "simple": {
            "uom": "FEET",
            "content": 40000
          }
        },
        "departure": {
          "departurePoint": "PHNL",
          "xsi:type": "ns5:NasDepartureType"
        },
        "timestamp": "2020-04-16T03:52:05.208Z"
      },
      "xsi:type": "ns5:FlightMessageType",
      "xmlns:xsi": "http://www.w3.org/2001/XMLSchema-instance"
    }
  }
}

// [AH]
//
{
  "ns5:MessageCollection": {
    "xmlns:ns2": "http://www.fixm.aero/base/3.0",
    "xmlns:ns5": "http://www.faa.aero/nas/3.0",
    "xmlns:ns3": "http://www.fixm.aero/foundation/3.0",
    "xmlns:ns4": "http://www.fixm.aero/flight/3.0",
    "message": {
      "flight": {
        "gufi": {
          "codeSpace": "urn:uuid",
          "content": "93166757-6e43-4c4c-adad-317ec18ef881"
        },
        "enRoute": {
          "xsi:type": "ns5:NasEnRouteType",
          "beaconCodeAssignment": ""
        },
        "requestedAirspeed": {
          "nasAirspeed": {
            "uom": "KNOTS",
            "content": 250
          }
        },
        "flightIdentification": {
          "computerId": 864,
          "aircraftIdentification": "HGT3683",
          "siteSpecificPlanId": 248,
          "xsi:type": "ns5:NasFlightIdentificationType"
        },
        "aircraftDescription": {
          "aircraftType": {
            "icaoModelIdentifier": "E45X"
          },
          "capabilities": {
            "navigation": {
              "performanceBasedCode": "A1 C1 D1",
              "navigationCode": "D G I W"
            },
            "surveillance": {
              "surveillanceCode": "S B1"
            },
            "standardCapabilities": "STANDARD"
          },
          "xsi:type": "ns5:NasAircraftType",
          "accuracy": {
            "cmsFieldType": [
              {
                "phase": "ARRIVAL",
                "uom": "NAUTICAL_MILES",
                "type": "RNV",
                "content": 1
              },
              {
                "phase": "ENROUTE",
                "uom": "NAUTICAL_MILES",
                "type": "RNV",
                "content": 2
              },
              {
                "phase": "DEPARTURE",
                "uom": "NAUTICAL_MILES",
                "type": "RNV",
                "content": 1
              }
            ]
          },
          "equipmentQualifier": "L",
          "wakeTurbulence": "M"
        },
        "arrival": {
          "arrivalAerodromeAlternate": {
            "code": "KSMF",
            "xsi:type": "ns2:IcaoAerodromeReferenceType"
          },
          "xsi:type": "ns5:NasArrivalType",
          "runwayPositionAndTime": {
            "runwayTime": {
              "estimated": {
                "time": "2020-04-16T03:30:00Z"
              }
            }
          },
          "arrivalPoint": "KMHR"
        },
        "flightPlan": {
          "identifier": "KO85504400",
          "flightPlanRemarks": "|HIGHTECH"
        },
        "xsi:type": "ns5:NasFlightType",
        "centre": "ZOA",
        "flightStatus": {
          "xsi:type": "ns5:NasFlightStatusType",
          "fdpsFlightStatus": "PROPOSED"
        },
        "supplementalData": {
          "xsi:type": "ns5:NasSupplementalDataType",
          "additionalFlightInformation": {
            "nameValue": [
              {
                "name": "MSG_SEQ_NO",
                "value": 12604954
              },
              {
                "name": "FDPS_GUFI",
                "value": "us.fdps.2020-04-15T23:45:04Z.000/14/400"
              },
              {
                "name": "FLIGHT_PLAN_SEQ_NO",
                "value": 2
              }
            ]
          }
        },
        "requestedAltitude": {
          "simple": {
            "uom": "FEET",
            "content": 12000
          }
        },
        "flightType": "SCHEDULED",
        "source": "AH",
        "operator": {
          "operatingOrganization": {
            "organization": {
              "name": "HGT"
            }
          }
        },
        "coordination": {
          "coordinationTimeHandling": "P",
          "coordinationFix": {
            "fix": "KSJC",
            "xsi:type": "ns2:FixPointType"
          },
          "coordinationTime": "2020-04-16T02:45:00Z"
        },
        "system": "ATL",
        "departure": {
          "departurePoint": "KSJC",
          "xsi:type": "ns5:NasDepartureType",
          "runwayPositionAndTime": {
            "runwayTime": {
              "estimated": {
                "time": "2020-04-16T02:45:00Z"
              }
            }
          }
        },
        "agreed": {
          "route": {
            "adaptedArrivalDepartureRoute": {
              "nasRouteIdentifier": "BAM93"
            },
            "initialFlightRules": "IFR",
            "xsi:type": "ns5:NasRouteType",
            "nasRouteText": "KSJC.TECKY3.TECKY..SJC..BMRNG..ORRCA..KMHR/0045"
          }
        },
        "timestamp": "2020-04-16T01:59:00.104Z"
      },
      "xsi:type": "ns5:FlightMessageType",
      "xmlns:xsi": "http://www.w3.org/2001/XMLSchema-instance"
    }
  }
}

*/
