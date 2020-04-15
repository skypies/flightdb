{{define "js-overlays"}} // Depends on: .Center (geo.Latlong), and .Zoom (int)

// Library of some routines that add furniture to the map.
//   ClassBOverlay();
//   PathsOverlay();  Depends on .Waypoints
var classBpoints = {
    "N37W122": {pos:{lat: 37.1716667 , lng: -122.3763889}}, // 37-10-18.000N / 122-21-95.000W
    "VPAGY": {pos:{lat: 38.0000000 , lng: -122.4167000}}, // 38-00-00.000N / 122-25-00.120W
    "VPAKQ": {pos:{lat: 37.9806139 , lng: -122.0957333}}, // 37-58-50.210N / 122-05-44.640W
    "VPASK": {pos:{lat: 37.9017139 , lng: -121.9867167}}, // 37-54-06.170N / 121-59-12.180W
    "VPBAB": {pos:{lat: 37.8568611 , lng: -121.9327972}}, // 37-51-24.700N / 121-55-58.070W
    "VPBAD": {pos:{lat: 37.7005000 , lng: -121.8548000}}, // 37-42-01.800N / 121-51-17.280W
    "VPBAG": {pos:{lat: 37.5838889 , lng: -121.6291667}}, // 37-35-02.000N / 121-37-45.000W
    "VPBAK": {pos:{lat: 37.5172222 , lng: -121.6197222}}, // 37-31-02.000N / 121-37-11.000W
    "VPBAL": {pos:{lat: 37.3922000 , lng: -121.7119000}}, // 37-23-31.920N / 121-42-42.840W
    "VPBAM": {pos:{lat: 37.3752778 , lng: -121.8680556}}, // 37-22-31.000N / 121-52-05.000W
    "VPBAN": {pos:{lat: 37.4422222 , lng: -121.7638889}}, // 37-26-32.000N / 121-45-50.000W
    "VPBAP": {pos:{lat: 37.4861111 , lng: -121.7511111}}, // 37-29-10.000N / 121-45-04.000W
    "VPBAS": {pos:{lat: 37.5648222 , lng: -121.7802833}}, // 37-33-53.360N / 121-46-49.020W
    "VPBAT": {pos:{lat: 37.5482556 , lng: -121.8524389}}, // 37-32-53.720N / 121-51-08.780W
    "VPBAX": {pos:{lat: 37.4032111 , lng: -121.9321333}}, // 37-24-11.560N / 121-55-55.680W
    "VPBBB": {pos:{lat: 37.3177750 , lng: -122.0635056}}, // 37-19-03.990N / 122-03-48.620W
    "VPBBD": {pos:{lat: 37.1765778 , lng: -122.0083194}}, // 37-10-35.680N / 122-00-29.950W
    "VPBBJ": {pos:{lat: 37.0972000 , lng: -121.9771000}}, // 37-05-49.920N / 121-58-37.560W
    "VPBBN": {pos:{lat: 37.2561000 , lng: -122.8380000}}, // 37-15-21.960N / 122-50-16.800W
    "VPBBQ": {pos:{lat: 37.2510778 , lng: -122.4152889}}, // 37-15-03.880N / 122-24-55.040W
    "VPBBS": {pos:{lat: 37.6588000 , lng: -122.8548000}}, // 37-39-31.680N / 122-51-17.280W
    "VPBBV": {pos:{lat: 37.4935472 , lng: -122.4548167}}, // 37-29-36.770N / 122-27-17.340W
    "VPBBW": {pos:{lat: 37.3826889 , lng: -122.3266889}}, // 37-22-57.680N / 122-19-36.080W
    "VPBBX": {pos:{lat: 37.3343333 , lng: -122.1297528}}, // 37-20-03.600N / 122-07-47.110W
    "VPBBZ": {pos:{lat: 37.4718417 , lng: -121.9637222}}, // 37-28-18.630N / 121-57-49.400W
    "VPBCB": {pos:{lat: 37.5425361 , lng: -121.9326917}}, // 37-32-33.130N / 121-55-57.690W
    "VPBCC": {pos:{lat: 37.6972444 , lng: -121.9607333}}, // 37-41-50.080N / 121-57-38.640W
    "VPBCD": {pos:{lat: 37.8864861 , lng: -122.1576889}}, // 37-53-11.350N / 122-09-27.680W
    "VPBDD": {pos:{lat: 37.7778083 , lng: -122.7870111}}, // 37-46-40.110N / 122-47-13.240W
    "VPBDF": {pos:{lat: 37.7343000 , lng: -122.8582000}}, // 37-44-03.480N / 122-51-29.520W
    "VPBDI": {pos:{lat: 37.8395000 , lng: -122.6852000}}, // 37-50-22.200N / 122-41-06.720W
    "VPBEV": {pos:{lat: 37.8546306 , lng: -121.9807639}}  // 37-51-16.670N / 121-58-50.750W
}

function drawClassBPoly(names, color) {
    var polyCoords = []
    for (var name in names) {
        polyCoords.push(classBpoints[names[name]].pos);
    }
    var poly = new google.maps.Polygon({
        paths: polyCoords,
        strokeColor: color,
        strokeOpacity: 0.8,
        strokeWeight: 0.3,
        fillColor: color,
        fillOpacity: 0.08
    });
    poly.setMap(map)
}

function ClassBOverlay() {

    // Vertices of the polygon that is the perimeter of the 100/80 area (no more layer cake!)
    var perimeter_100_80 = [
        "VPBBD", "VPBBJ", "N37W122",
        "VPBBN", "VPBBS", "VPBDF", "VPBDD", "VPBDI", "VPAGY", "VPAKQ", "VPASK", "VPBAB",
        "VPBAD", "VPBAG", "VPBAK", "VPBAL", "VPBAM", "VPBAX", "VPBBB"];

    // many vertices overlap with 100_80. We include 100/70 alongside 100/60.
    var perimeter_100_60 = [
        "VPBBD", "VPBBQ", "N37W122",
        "VPBBN", "VPBBS", "VPBDF", "VPBDD", "VPBDI", "VPAGY", "VPAKQ", "VPASK", "VPBAB",
        "VPBAD",
        "VPBCC", "VPBCB", "VPBAT", "VPBAS", "VPBAP", "VPBAN",
        "VPBAM", "VPBAX", "VPBBB"];

    // includes the 100/50 area too
    var perimeter_100_40 = [
        "VPBBX", "VPBBW", "VPBBV",
        "VPBBS", "VPBDF", "VPBDD", "VPBDI", "VPAGY", "VPAKQ",
        "VPBCD", "VPBEV", "VPBCC", "VPBCB", "VPBBZ"
    ]

    drawClassBPoly(perimeter_100_80, "#0000FF")
    drawClassBPoly(perimeter_100_60, "#0000FF")
    drawClassBPoly(perimeter_100_40, "#0000FF")

    // Draw ghost of the old one
    var oldClassb = [
        { center: {lat:  37.6188172 , lng:  -122.3754281 }, boundaryMeters:  37040 }, // 20NM
        { center: {lat:  37.6188172 , lng:  -122.3754281 }, boundaryMeters:  46300 }, // 25NM
        { center: {lat:  37.6188172 , lng:  -122.3754281 }, boundaryMeters:  55560 }, // 30NM
    ];
    for (var i=0; i< oldClassb.length; i++) {
        // Add the circle for this city to the map.
        var cityCircle = new google.maps.Circle({
            strokeColor: '#0000FF',
            strokeOpacity: 0.8,
            strokeWeight: 0.3,
            fillOpacity: 0,
            map: map,
            center: oldClassb[i].center,
            radius: oldClassb[i].boundaryMeters
        });
    }

}


function OldClassBOverlay() {
    var classb = [
        { center: {lat:  37.6188172 , lng:  -122.3754281 }, boundaryMeters:  37040 }, // 20NM
        { center: {lat:  37.6188172 , lng:  -122.3754281 }, boundaryMeters:  46300 }, // 25NM
        { center: {lat:  37.6188172 , lng:  -122.3754281 }, boundaryMeters:  55560 }, // 30NM
    ];
    for (var i=0; i< classb.length; i++) {
        // Add the circle for this city to the map.
        var cityCircle = new google.maps.Circle({
            strokeColor: '#0000FF',
            strokeOpacity: 0.8,
            strokeWeight: 0.3,
            fillColor: '#0000FF',
            fillOpacity: 0.08,
            map: map,
            center: classb[i].center,
            radius: classb[i].boundaryMeters
        });
    }
}

function OldClassBOverlay() {
    var classb = [
        { center: {lat:  37.6188172 , lng:  -122.3754281 }, boundaryMeters:  37040 }, // 20NM
        { center: {lat:  37.6188172 , lng:  -122.3754281 }, boundaryMeters:  46300 }, // 25NM
        { center: {lat:  37.6188172 , lng:  -122.3754281 }, boundaryMeters:  55560 }, // 30NM
    ];
    for (var i=0; i< classb.length; i++) {
        // Add the circle for this city to the map.
        var cityCircle = new google.maps.Circle({
            strokeColor: '#0000FF',
            strokeOpacity: 0.8,
            strokeWeight: 0.3,
            fillColor: '#0000FF',
            fillOpacity: 0.08,
            map: map,
            center: classb[i].center,
            radius: classb[i].boundaryMeters
        });
    }
}

// Small subset; use these unless we're presented with a .Waypoint var.
var waypoints = {
    // SERFR3 (provisional changes)
    "NARWL-SERFR3": {pos:{lat: 37.2747806 , lng:  -122.0792944}},
    "EDDYY-SERFR3": {pos:{lat: 37.3749028 , lng:  -122.1187500}},

    // SERFR2
    "SERFR": {pos:{lat: 36.0683056 , lng:  -121.3646639}},
    "NRRLI": {pos:{lat: 36.4956000 , lng:  -121.6994000}},
    "WWAVS": {pos:{lat: 36.7415306 , lng:  -121.8942333}},        
    "EPICK": {pos:{lat: 36.9508222 , lng:  -121.9526722}},
    "EDDYY": {pos:{lat: 37.3264500 , lng:  -122.0997083}},
    "SWELS": {pos:{lat: 37.3681556 , lng:  -122.1160806}},
    "MENLO": {pos:{lat: 37.4636861 , lng:  -122.1536583}},

    // WWAVS1 This is the rarely used 'bad weather' fork (uses the other runways)
    "WPOUT": {pos:{lat: 37.1194861 , lng:  -122.2927417}},
    "THEEZ": {pos:{lat: 37.5034694 , lng:  -122.4247528}},
    "WESLA": {pos:{lat: 37.6643722 , lng:  -122.4802917}},
    "MVRKK": {pos:{lat: 37.7369722 , lng:  -122.4544500}},

    // BRIXX1 (skip the first two, nothing flies along them anyway and they make a mess)
    //"CORKK": {pos:{lat: 37.7335889 , lng:  -122.4975500}},
    //"BRIXX": {pos:{lat: 37.6178444 , lng:  -122.3745278}},
    "LUYTA": {pos:{lat: 37.2948889 , lng:  -122.2045528}},
    "JILNA": {pos:{lat: 37.2488056 , lng:  -122.1495000}},
    "YADUT": {pos:{lat: 37.2039889 , lng:  -122.0232778}},

    // http://flightaware.com/resources/airport/SFO/STAR/BIG+SUR+TWO/pdf
    "CARME": {pos:{lat: 36.4551833, lng: -121.8797139}},
    "ANJEE": {pos:{lat: 36.7462861, lng: -121.9648917}},
    "SKUNK": {pos:{lat: 37.0075944, lng: -122.0332278}},
    "BOLDR": {pos:{lat: 37.1708861, lng: -122.0761667}}
}

function PathsOverlay() {
    {{if .Waypoints}}waypoints = {{.Waypoints}}{{end}}

    var infowindow = new google.maps.InfoWindow({});
    var marker = new google.maps.Marker({ map: map });

    for (var wp in waypoints) {
        var fixCircle = new google.maps.Circle({
            title: wp, // this attribute is for the mouse events below
            strokeWeight: 2,
            strokeColor: '#990099',
            //fillColor: '#990099',
            fillOpacity: 0.0,
            map: map,
            zIndex: 20,
            center: waypoints[wp].pos,
            radius: 300
        });

        // Add a tooltip thingy
        google.maps.event.addListener(fixCircle, 'mouseover', function () {
            if (typeof this.title !== "undefined") {
                marker.setPosition(this.getCenter()); // get circle's center
                infowindow.setContent("<b>" + this.title + "</b>"); // set content
                infowindow.open(map, marker); // open at marker's location
                marker.setVisible(false); // hide the marker
            }
        });
        google.maps.event.addListener(fixCircle, 'mouseout', function () {
            infowindow.close();
        });

        // Would be nice to render the waypoint's name on the map somehow ...
        // http://stackoverflow.com/questions/3953922/is-it-possible-to-write-custom-text-on-google-maps-api-v3
    }

    // These should come from geo/sfo/procedures
    var SERFR2 = ["SERFR", "NRRLI", "WWAVS", "EPICK", "EDDYY", "SWELS", "MENLO"];
    var WWAVS1 = ["WWAVS", "WPOUT", "THEEZ", "WESLA", "MVRKK"];
    var BRIXX1 = ["LUYTA", "JILNA", "YADUT"];
    var BSR2   = ["CARME", "ANJEE", "SKUNK", "BOLDR", "MENLO"];
    //var SERFR3 = ["SERFR", "NRRLI", "WWAVS", "EPICK", "NARWL-SERFR3", "EDDYY-SERFR3"];
    drawPath(SERFR2, '#990099')
    drawPath(WWAVS1, '#990099')
    drawPath(BRIXX1, '#990099')
    drawPath(BSR2,   '#007788')
    //drawPath(SERFR3, '#44bb22') // overlays SERFR too closely
}

function drawPath(fixes, color) {
    var pathLineCoords = []
    for (var fix in fixes) {
        pathLineCoords.push(waypoints[fixes[fix]].pos);
    }
    var pathLine = new google.maps.Polyline({
        path: pathLineCoords,
        geodesic: true,
        strokeColor: color,
        strokeOpacity: 0.8,
        strokeWeight: 1
    });
    pathLine.setMap(map)
}

{{end}}
