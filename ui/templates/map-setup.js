{{define "js-map-setup"}} // Depends on: .Center (geo.Latlong), and .Zoom (int)
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
    serfrOverlay()

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
        { center: {lat:  37.6188172 , lng:  -122.3754281 }, boundaryMeters:  12964 }, //  7NM
        { center: {lat:  37.6188172 , lng:  -122.3754281 }, boundaryMeters:  18520 }, // 10NM
        { center: {lat:  37.6188172 , lng:  -122.3754281 }, boundaryMeters:  27780 }, // 15NM
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

function serfrOverlay() {
    var fixes = {
        "SERFR": {pos:{lat: 36.0683056 , lng:  -121.3646639}},
        "NRRLI": {pos:{lat: 36.4956000 , lng:  -121.6994000}},
        "WWAVS": {pos:{lat: 36.7415306 , lng:  -121.8942333}},
        
        "EPICK": {pos:{lat: 36.9508222 , lng:  -121.9526722}},
        "EDDYY": {pos:{lat: 37.3264500 , lng:  -122.0997083}},
        "SWELS": {pos:{lat: 37.3681556 , lng:  -122.1160806}},
        "MENLO": {pos:{lat: 37.4636861 , lng:  -122.1536583}},

        // This is the rarely used 'bad weather' fork, WWAVS1 (uses the other runways)
        "WPOUT": {pos:{lat: 37.1194861 , lng:  -122.2927417}},
        "THEEZ": {pos:{lat: 37.5034694 , lng:  -122.4247528}},
        "WESLA": {pos:{lat: 37.6643722 , lng:  -122.4802917}},
        "MVRKK": {pos:{lat: 37.7369722 , lng:  -122.4544500}},

        // BRIXX
        //"CORKK": {pos:{lat: 37.7335889 , lng:  -122.4975500}},
        //"BRIXX": {pos:{lat: 37.6178444 , lng:  -122.3745278}},
        "LUYTA": {pos:{lat: 37.2948889 , lng:  -122.2045528}},
        "JILNA": {pos:{lat: 37.2488056 , lng:  -122.1495000}},
        "YADUT": {pos:{lat: 37.2039889 , lng:  -122.0232778}},
    }

    for (var fix in fixes) {
        var fixCircle = new google.maps.Circle({
            strokeWeight: 2,
            strokeColor: '#990099',
            //fillColor: '#990099',
            fillOpacity: 0.0,
            map: map,
            center: fixes[fix].pos,
            radius: 300
        });
        // Would be nice to render the name (fix) on the map somehow; see link below.
    }

    var rightFixes = ["SERFR", "NRRLI", "WWAVS", "EPICK", "EDDYY", "SWELS", "MENLO"];
    var rightLineCoords = []
    for (var fix in rightFixes) {
        rightLineCoords.push(fixes[rightFixes[fix]].pos);
    }
    var rightLine = new google.maps.Polyline({
        path: rightLineCoords,
        geodesic: true,
        strokeColor: '#990099',
        strokeOpacity: 0.8,
        strokeWeight: 1
    });
    rightLine.setMap(map)

    var wwavsFixes = ["WWAVS", "WPOUT", "THEEZ", "WESLA", "MVRKK"];
    var wwavsLineCoords = []
    for (var fix in wwavsFixes) {
        wwavsLineCoords.push(fixes[wwavsFixes[fix]].pos);
    }
    var wwavsLine = new google.maps.Polyline({
        path: wwavsLineCoords,
        geodesic: true,
        strokeColor: '#990099',
        strokeOpacity: 0.8,
        strokeWeight: 1
    });
    wwavsLine.setMap(map)

    var brixxFixes = ["LUYTA", "JILNA", "YADUT"];
    var brixxLineCoords = []
    for (var fix in brixxFixes) {
        brixxLineCoords.push(fixes[brixxFixes[fix]].pos);
    }
    var brixxLine = new google.maps.Polyline({
        path: brixxLineCoords,
        geodesic: true,
        strokeColor: '#990099',
        strokeOpacity: 0.8,
        strokeWeight: 1
    });
    brixxLine.setMap(map)
}

// http://stackoverflow.com/questions/3953922/is-it-possible-to-write-custom-text-on-google-maps-api-v3

{{end}}
