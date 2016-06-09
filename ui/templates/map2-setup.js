{{define "js-map2-setup"}} // Depends on: .Center (geo.Latlong), and .Zoom (int)

var map;
function initMap() {
    map = new google.maps.Map(document.getElementById('map'), {
        center: {lat: {{.Center.Lat}}, lng: {{.Center.Long}}},
        mapTypeId: google.maps.MapTypeId.TERRAIN,
        scaleControl: true,
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
        fillOpacity: 0.6,
        zIndex: 0,
        map: map,
        bounds: new google.maps.LatLngBounds(
            new google.maps.LatLng(30,-130),
            new google.maps.LatLng(45,-112)),
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
    {{if .IdSpecs}}streamVectors(){{end}}
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
            zIndex: 10,
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
    "BRIXX": {pos:{lat: 37.6178444 , lng:  -122.3745278}},
    "LUYTA": {pos:{lat: 37.2948889 , lng:  -122.2045528}},
    "JILNA": {pos:{lat: 37.2488056 , lng:  -122.1495000}},
    "YADUT": {pos:{lat: 37.2039889 , lng:  -122.0232778}},

    // http://flightaware.com/resources/airport/SFO/STAR/BIG+SUR+TWO/pdf
    "CARME": {pos:{lat: 36.4551833, lng: -121.8797139}},
    "ANJEE": {pos:{lat: 36.7462861, lng: -121.9648917}},
    "SKUNK": {pos:{lat: 37.0075944, lng: -122.0332278}},
    "BOLDR": {pos:{lat: 37.1708861, lng: -122.0761667}},       

    // Things for SFO arrivals
    "HEMAN": {pos:{lat: 37.5338500, lng: -122.1733333}},
    "DUYET": {pos:{lat: 37.5674000, lng: -122.2529278}},
    "NEPIC": {pos:{lat: 37.5858944, lng: -122.2968833}},

    // Things for SFO departures
    "PORTE": {pos:{lat: 37.4897861, lng: -122.4745778}},
    "SSTIK": {pos:{lat: 37.6783444, lng: -122.3616583}},

    // Things for Oceanic
    "PPEGS": {pos:{lat: 37.3920722, lng: -122.2817222}},
    "ALLBE": {pos:{lat: 37.5063889, lng: -127.0000000}},
    "ALCOA": {pos:{lat: 37.8332528, lng: -125.8345250}},
    "CINNY": {pos:{lat: 36.1816667, lng: -124.7600000}},
    "PAINT": {pos:{lat: 38.0000000, lng: -125.5000000}},
    "OSI"  : {pos:{lat: 37.3925000, lng: -122.2813000}},
    "PIRAT": {pos:{lat: 37.2576500, lng: -122.8633528}},
    "PYE"  : {pos:{lat: 38.0797567, lng: -122.8678275}},
    "STINS": {pos:{lat: 37.8236111, lng: -122.7566667}},
    "HADLY": {pos:{lat: 37.4022222, lng: -122.5755556}},

    "PONKE": {pos:{lat: 37.4588167, lng: -121.9960528}},
    "WETOR": {pos:{lat: 37.4847194, lng: -122.0571417}},

    // Things for SILCN3
    "VLLEY": {pos:{lat: 36.5091667, lng:-121.4402778}},
    "GUUYY": {pos:{lat: 36.7394444, lng:-121.5411111}},
    "SSEBB": {pos:{lat: 36.9788889, lng:-121.6425000}},
    "GSTEE": {pos:{lat: 37.0708333, lng:-121.6716667}},
    "KLIDE": {pos:{lat: 37.1641667, lng:-121.7130556}},
    "BAXBE": {pos:{lat: 36.7730556, lng:-121.6263889}},
    "APLLE": {pos:{lat: 37.0338889, lng:-121.8050000}},

    // Randoms
    "PARIY": {pos:{lat: 37.3560056, lng:-121.9231222}},  // SJC ?
    "ZORSA": {pos:{lat: 37.3627583, lng:-122.0500306}},

    // Things for East Bay
    "HOPTA": {pos:{lat: 37.78501944, lng: -122.154}},
    "BOYSS": {pos:{lat: 38.02001944, lng: -122.3778639}},
    "WNDSR": {pos:{lat: 38.681808, lng: -122.478747}},
    "WEBRR": {pos:{lat: 38.243881, lng: -122.412142}},
    "SPAMY": {pos:{lat: 39.200661, lng: -122.591042}},
    "HUBRT": {pos:{lat: 39.040228, lng: -122.568314}},
    "DRAXE": {pos:{lat: 38.759, lng: -122.389047}},
    "BMBOO": {pos:{lat: 38.892972, lng: -122.233019}},
    "RBUCL": {pos:{lat: 39.070053, lng: -122.02615}},
    "GRTFL": {pos:{lat: 38.35216944, lng: -122.2314694}},
    "TRUKN": {pos:{lat: 37.71755833, lng: -122.2145889}},
    "DEDHD": {pos:{lat: 38.33551666, lng: -122.1128083}},
    "HYPEE": {pos:{lat: 37.88024444, lng: -122.0674833}},
    "COSMC": {pos:{lat: 37.82606111, lng: -122.0049}},
    "TYDYE": {pos:{lat: 37.689319, lng: -122.268944}},
    "ORRCA": {pos:{lat: 38.610325, lng: -121.551622}},
    "MOGEE": {pos:{lat: 38.336111, lng: -121.389722}},
    "TIPRE": {pos:{lat: 38.205833, lng: -121.035833}},
    "SYRAH": {pos:{lat: 37.99105, lng: -121.103089}},
    "RAIDR": {pos:{lat: 38.0325, lng: -122.5575}},
    "CRESN": {pos:{lat: 37.697475, lng: -122.012019}},
    "AAAME": {pos:{lat: 37.770908, lng: -122.082811}},
    "ALLXX": {pos:{lat: 37.729606, lng: -122.064283}},
    "HIRMO": {pos:{lat: 37.92765, lng: -122.14835}},
    "CEXUR": {pos:{lat: 37.934161, lng: -122.252928}},
    "WOULD": {pos:{lat: 37.774508, lng: -122.058064}},
    "FINSH": {pos:{lat: 37.651203, lng: -122.257161}},
    "HUSHH": {pos:{lat: 37.7495, lng: -122.338592}},
    "AANET": {pos:{lat: 38.530769, lng: -122.497194}},

    // Personal entries
    "X_RSH": {pos:{lat: 36.868582,  lng:-121.691934}},
    "X_BLH": {pos:{lat: 37.2199471, lng:-122.0425108}},
    "X_HBR": {pos:{lat: 37.309564,  lng:-122.112378}}
}

function pathsOverlay() {
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
        zIndex: 20,
        strokeWeight: 1
    });
    pathLine.setMap(map)
}

{{end}}
