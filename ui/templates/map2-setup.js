{{define "js-map2-setup"}} // Depends on: .Center (geo.Latlong), and .Zoom (int)

var map;
function initMap() {
    map = new google.maps.Map(document.getElementById('map'), {
        center: {lat: {{.Center.Lat}}, lng: {{.Center.Long}}},
        mapTypeId: google.maps.MapTypeId.TERRAIN,
        zoom: {{.Zoom}}
    });

    map.controls[google.maps.ControlPosition.RIGHT_TOP].push(
        document.getElementById('legend'));
    
    classBOverlay()
    {{if .WhiteOverlay}}
    var olay = new google.maps.Rectangle({
        strokeColor: '#ffffff',
        strokeOpacity: 0,
        strokeWeight: 0,
        fillColor: '#ffffff',
        fillOpacity: 0.4,
        map: map,
        bounds: new google.maps.LatLngBounds(
            new google.maps.LatLng(35,-125),
            new google.maps.LatLng(40,-117)),
    });
    {{end}}

    pathsOverlay()

    {{if .Legend}}
    var legend = document.getElementById('legend');
    var div = document.createElement('div');
    div.innerHTML = {{.Legend}};
    legend.appendChild(div);
    {{end}}
    
    {{if .Points}}pointsOverlay(){{end}}
}

function classBOverlay() {
    var classb = [
//      { center: {lat:  37.6188172 , lng:  -122.3754281 }, boundaryMeters:  12964 }, //  7NM
//      { center: {lat:  37.6188172 , lng:  -122.3754281 }, boundaryMeters:  18520 }, // 10NM
//      { center: {lat:  37.6188172 , lng:  -122.3754281 }, boundaryMeters:  27780 }, // 15NM
        { center: {lat:  37.6188172 , lng:  -122.3754281 }, boundaryMeters:  37040 }, // 20NM
        { center: {lat:  37.6188172 , lng:  -122.3754281 }, boundaryMeters:  46300 }, // 25NM
        { center: {lat:  37.6188172 , lng:  -122.3754281 }, boundaryMeters:  55560 }, // 30NM
    ]

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

// These should come from geo/sfo
var waypoints = {
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
    "BOLDR": {pos:{lat: 37.1708861, lng: -122.0761667}},       
}

function pathsOverlay() {
    for (var wp in waypoints) {
        var fixCircle = new google.maps.Circle({
            strokeWeight: 2,
            strokeColor: '#990099',
            //fillColor: '#990099',
            fillOpacity: 0.0,
            map: map,
            center: waypoints[wp].pos,
            radius: 300
        });
        // Would be nice to render the waypoint's name on the map somehow ...
        // http://stackoverflow.com/questions/3953922/is-it-possible-to-write-custom-text-on-google-maps-api-v3
    }

    // These should come from geo/sfo/procedures
    var SERFR2 = ["SERFR", "NRRLI", "WWAVS", "EPICK", "EDDYY", "SWELS", "MENLO"];
    var WWAVS1 = ["WWAVS", "WPOUT", "THEEZ", "WESLA", "MVRKK"];
    var BRIXX1 = ["LUYTA", "JILNA", "YADUT"];
    var BSR2   = ["CARME", "ANJEE", "SKUNK", "BOLDR", "MENLO"];

    drawPath(SERFR2, '#990099')
    drawPath(WWAVS1, '#990099')
    drawPath(BRIXX1, '#990099')
    drawPath(BSR2,   '#007788')
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
