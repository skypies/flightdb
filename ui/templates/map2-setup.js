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
    {{if .AirspaceJS}}airspaceOverlay(){{end}}
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

var waypoints = {{.Waypoints}}


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
