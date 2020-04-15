{{define "js-airspace"}}

// Routines to paint AircraftData objects as markers on a google map, and update them over time

//   InitMapsAirspace();
//   PaintAircraft(ad);
//   ExpireAircraft(liveIds);
//   icao = CurrentlySelectedIcao();
//   n = CountVisibleAircraft();

// Some global state
var gAircraft = {};         // Current marker objects for all live aircraft (keyed on icaoID)
var gExpiredAircraft = {};  // Expired (red) markers, kept for 60s or so (keyed on icaoID)
var gInfowindowIcao24 = ""; // The ID of the selected aircraft last rendered into the infowindow

var gInfowindow;

// Must call this before all the other junk. Can only call it once the google maps callbacks
// are being called, and libs have been loaded.
function InitMapsAirspace() {
    gInfowindow = new google.maps.InfoWindow({ content: "holding..." });
    gInfowindow.addListener('closeclick', function(){
        gInfowindowIcao24 = "";
    });

    paintAirspaceSearchBox();
}

function deleteExpiredMarker(icaoid) {
    gExpiredAircraft[icaoid].setMap(null);
    delete gExpiredAircraft[icaoid];
}

function CurrentlySelectedIcao() { return gInfowindowIcao24 }

// Re-color any aircraft that were dropped from the recent dataset in red. And for those
// that have been red too long, remove the marker entirely.
// Argument is a map whose keys are the currently live icaoids.
function ExpireAircraft(liveIds) {
    // Look for currently rendered aircraft that are not in the liveIds list
    for (k in gAircraft) {
        if (!k) continue;
        if (! liveIds.hasOwnProperty(k)) {
            marker = gAircraft[k];
            delete gAircraft[k];

            marker.expiredAt = new Date();
            marker.setIcon(arrowicon("#ff0000",marker.getIcon().rotation));
            marker.setZIndex(1000); // Drop expired things to the background

            gExpiredAircraft[k] = marker
        }
    }

    // Look for aircraft that are live, which were previously expired (delete old marker)
    for (k in liveIds) {
        if (gExpiredAircraft.hasOwnProperty(k)) {
            // No longer expired ! Delete the red marker
            deleteExpiredMarker(k);
        }
    }
 
    // Look for expired/red markers that have been around too long; just delete 'em
    var now = new Date();
    for (icaoid in gExpiredAircraft) {
        if (now - gExpiredAircraft[icaoid].expiredAt > 60000) {
            // Been hanging around too long; remove
            deleteExpiredMarker(icaoid);
        }
    }
}

// Adds (or updates) a marker in the relevant place, and links it to the single shared infowindow.=
function PaintAircraft(a) {
    var flightnumber = "";
    if (a.IATA && a.Number) { flightnumber = a.IATA+a.Number }
    var ident = flightnumber;
    if (!ident) { ident = a.Msg.Callsign }
    if (!ident) { ident = a.Registration }
    var infostring = aircraftToInfostring(a, ident, flightnumber)
    
    var zDepth = 3000;
    if (a.Source == "fr24") { zDepth = 2000 }

    var opacity = 0.8
    var color = "#0033ff"; // Default: SkyPi/ADSB color
    if (a.X_DataSystem == "MLAT") {
        color = "#508aff";
    }

    if (a.Source == "fr24") {
        color = "#00ff33";
        if (a.X_DataSystem == "T-F5M") {
            color = "#44ff22";
        }

    } else if (a.Source == "fa") {
        color = "#ff3300";

    } else if (a.Source == "AdsbExchange") {
        color = "#cc0066";
        if (a.X_DataSystem == "MLAT") {
            color = "#ff33ff";
        }
    }

    var newicon = arrowicon(color, a.Msg.Track);
    newicon.strokeOpacity = opacity
    var newpos = new google.maps.LatLng(a.Msg.Position.Lat, a.Msg.Position.Long);
    var oldmarker = gAircraft[a.Icao24]
    if (!oldmarker) {
        // New aircraft - create a fresh marker
        var marker = new google.maps.Marker({
            title: ident,
            callsign: a.Msg.Callsign,  // This is used for search
            html: infostring,
            position: newpos,
            zIndex: zDepth,
            icon: newicon,
            map: map
        });        
        marker.addListener('click', function(){
            gInfowindow.setContent(this.html),
            gInfowindow.open(map, this);
            gInfowindowIcao24 = a.Icao24;
        });
        gAircraft[a.Icao24] = marker

    } else {
        // Update existing marker
        oldmarker.setPosition(newpos);
        oldmarker.setIcon(newicon);
        oldmarker.html = infostring;
        // If the infowindow is currently displaying this aircraft, update it
        if (gInfowindowIcao24 == a.Icao24) {
            gInfowindow.setContent(infostring);
        }
    }
}

function aircraftToInfostring(a, ident, flightnumber) {
    var header = '<b>'+ident+'</b><br/>';
    if (a.X_UrlSkypi) {
        header = '<b><a target="_blank" href="'+a.X_UrlSkypi+'">'+ident+'</a></b> '+
            '[<a target="_blank" href="'+a.X_UrlFA+'">FA</a>,'+
            ' <a target="_blank" href="'+a.X_UrlFR24+'">FR24</a>,'+
            ' <a target="_blank" href="'+a.X_UrlDescent+'">Descent</a>'+
            ']<br/>'
    }

    var infostring = header +
        'FlightNumber: '+flightnumber+'<br/>'+
        'Schedule: '+a.Origin+" - "+a.Destination+'<br/>'+
        'Callsign: '+a.Msg.Callsign+'<br/>'+
        'Icao24: '+a.Icao24+'<br/>'+
        '  -- Registration: '+a.Registration+'<br/>'+
        '  -- IcaoPrefix: '+a.CallsignPrefix+'<br/>'+
        '  -- Equipment: '+a.EquipmentType+'<br/>'+
        'PressureAltitude: '+a.Msg.Altitude+' feet<br/>'+
        'GroundSpeed: '+a.Msg.GroundSpeed+' knots<br/>'+
        'Heading: '+a.Msg.Track+' degrees<br/>'+
        'Position: ('+a.Msg.Position.Lat+','+a.Msg.Position.Long+')<br/>'+
        // 'Last seen: ('+a.X_AgeSecs+'s ago) '+a.Msg.GeneratedTimestampUTC+'<br/>'+
        'Last seen: '+a.Msg.GeneratedTimestampUTC+'<br/>'+
        'Source: '+a.Source+'/'+a.Msg.ReceiverName+' ('+ a.X_DataSystem+')<br/>';

    infostring = '<div id="infowindow">'+infostring+'</div>'
    return infostring
}

// Input is an AircraftData JSON object, that looks like this:
//
//   {"Msg": {"Type":"MSG"    (or "MLAT")
//            "Icao24":"71BE21",
//            "GeneratedTimestampUTC":"2016-11-14T19:46:10.72Z",
//            "Callsign":"KAL018",
//            "Altitude":17100,
//            "GroundSpeed":427,
//            "Track":312,
//            "Position":
//            {"Lat":34.21724, "Long":-119.3715},
//            "VerticalRate":1600,
//            "Squawk":"1320",
//            "ReceiverName":"CulverCity"
//           },
//    "Icao24":"71BE21",
//    "Registration":"HL7621",
//    "EquipmentType":"A388",
//    "CallsignPrefix":"KAL",
//    "Number":0,
//    "IATA":"",
//    "ICAO":"",
//    "PlannedDepartureUTC":"0001-01-01T00:00:00Z",
//    "PlannedArrivalUTC":"0001-01-01T00:00:00Z",
//    "ArrivalLocationName":"",
//    "DepartureLocationName":"",
//    "Origin":"",
//    "Destination":"",
//    "NumMessagesSeen":278,
//    "Source": "SkyPi",
//    "X_UrlSkypi": "/fdb/tracks?idspec=406D78@1479338200",
//    "X_UrlDescent": "/fdb/descent?idspec=406D78@1479338200",
//    "X_UrlFA": "http://flightaware.com/live/flight/BAW279",
//    "X_UrlFR24": "http://www.flightradar24.com/BAW279",
//    "X_DataSystem": "ADSB",
//    "X_AgeSecs": "1"
//   }

function CountVisibleAircraft() {
    var visible = 0;
    for (k in gAircraft) {
        if (map.getBounds().contains(gAircraft[k].getPosition())) { visible++ }
    }
    return visible;
}

function arrowicon(color,rotation) {
    return {
        path: google.maps.SymbolPath.FORWARD_CLOSED_ARROW,
        scale: 3,
        strokeColor: color,
        strokeWeight: 2,
        rotation: rotation,
    };
}

// Crappy client-side searchbox. Interates over the markers, looking for one with the same callsign
function paintAirspaceSearchBox() {
    var html = '<div>'+
        '<form id="srchform">'+
        '<button type="submit">Search Callsign</button> '+
        '<input type="text" name="callsign" size="8"/>'+
        '</form>'+
        '</div>'

    PaintDetails(html);

    $(function() {
        $('#srchform').on("submit",function(e) {
            e.preventDefault(); // cancel the actual submit
            var callsign = document.getElementById('srchform').elements.callsign.value;
            document.getElementById('srchform').elements.callsign.value = '';
            searchAndMaybeHighlight(callsign.toUpperCase());
        });
    });
}

function searchAndMaybeHighlight(callsign) {
    console.log('Search for "' + callsign + '"');
    for (k in gAircraft) {
        var marker = gAircraft[k];
        if (marker.callsign == callsign) {
            if (gInfowindowIcao24 != "") {
                gInfowindow.close();
            }
            gInfowindow.setContent(marker.html);
            gInfowindow.open(map, marker);
            gInfowindowIcao24 = k
        }
    }
}


{{end}}
