{{define "js-map"}} // Depends on: .Center (geo.Latlong), and .Zoom (int)

{{template "js-overlays"}}
{{template "js-textboxes"}}

{{template "js-map-shapes" . }}
{{template "js-map-ajax" . }}

var map;

function initMap() {
    map = new google.maps.Map(document.getElementById('map'), {
        center: {lat: {{.Center.Lat}}, lng: {{.Center.Long}}},
        mapTypeId: {{.MapType}},
        scaleControl: true,
        zoom: {{.Zoom}}
    });

    map.controls[google.maps.ControlPosition.RIGHT_TOP].push(
        document.getElementById('legend'));
    map.controls[google.maps.ControlPosition.RIGHT_CENTER].push(
        document.getElementById('details'));
    
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

    {{if .ClassBOverlay}}ClassBOverlay();{{end}}
    PathsOverlay();
    PaintLegend( {{.Legend}} );
    {{if .Points}}ShapesOverlay();{{end}}
    {{if .IdSpecs}}StreamVectors();{{end}}
}

{{end}}
