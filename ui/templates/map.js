{{define "js-map"}} // Depends on: .Center (geo.Latlong), and .Zoom (int)

{{template "js-overlays" . }}
{{template "js-textboxes"}}

{{if .Points}}  {{template "js-map-shapes" . }} {{end}}
{{if .IdSpecs}} {{template "js-map-ajax" . }}   {{end}}
{{if .Heatmap}} {{template "js-heatmap"}}       {{end}}

var map;

function initMap() {
    {{template "js-map-styles"}}

    map = new google.maps.Map(document.getElementById('map'), {
        center: {lat: {{.Center.Lat}}, lng: {{.Center.Long}}},
        scaleControl: true,
        zoom: {{.Zoom}},
        mapTypeControlOptions: {
            mapTypeIds: ['roadmap', 'satellite', 'hybrid', 'terrain',
                         'Silver']
        }
    });

    map.mapTypes.set('Silver', styledMapSilver);
    map.setMapTypeId({{.MapType}});

    
    map.controls[google.maps.ControlPosition.RIGHT_TOP].push(
        document.getElementById('legend'));
    map.controls[google.maps.ControlPosition.RIGHT_CENTER].push(
        document.getElementById('notes'));
    
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
    {{if .Heatmap}}FetchAndPaintHeatmap({{.Heatmap}});{{end}}
}

{{end}}
