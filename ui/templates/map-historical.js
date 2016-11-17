{{define "js-map-historical"}}

{{template "js-overlays"}}
{{template "js-textboxes"}}
{{template "js-airspace"}}

var map;
function initMap() {
    map = new google.maps.Map(document.getElementById('map'), {
        center: {lat: {{.Center.Lat}}, lng: {{.Center.Long}}},
        mapTypeId: google.maps.MapTypeId.TERRAIN,
        zoom: {{.Zoom}}
    });

    map.controls[google.maps.ControlPosition.RIGHT_TOP].push(
        document.getElementById('legend'));
    map.controls[google.maps.ControlPosition.LEFT_BOTTOM].push(
        document.getElementById('details'));

    ClassBOverlay();
    PathsOverlay();
    localOverlay();
}

function localOverlay() {
    InitMapsAirspace();
    PaintLegend( {{.Legend}} );

    var aircraftjson = {{.AircraftJSON}}
    var aircraftmap = aircraftjson["Aircraft"]
    
    for (icaoid in aircraftmap) {
        PaintAircraft(aircraftmap[icaoid])
    }
}

{{end}}
