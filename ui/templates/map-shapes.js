{{define "js-map-shapes"}} // Depends on: .Center (geo.Latlong), and .Zoom (int)

function ShapesOverlay() {
    var infowindow = new google.maps.InfoWindow({ content: "holding..." });

    points = {{.Points}}
    for (var i in points) {
        var icon = points[i].icon
        if (!icon) { icon = "pink" }
        var imgurl = '/static/dot-' + icon + '.png';
        var infostring = '<div><pre>' + points[i].info + '</pre></div>';
        var marker = new google.maps.Marker({
            position: points[i].pos,
            zIndex: 100,
            map: map,
            title: points[i].id,
            icon: imgurl,
            html: infostring,
        });
        marker.addListener('click', function(){
            infowindow.setContent(this.html),
            infowindow.open(map, this);
        });
    }

    lines = {{.Lines}}
    for (var i in lines) {
        var color = lines[i].color
        if (!color) { color = "#dd5508" }
        var opacity = lines[i].opacity
        if (!opacity) { opacity = 1.0 }
        var coords = []
        coords.push(lines[i].s)
        coords.push(lines[i].e)
        var line = new google.maps.Polyline({
            path: coords,
            geodesic: true,
            strokeColor: color,
            strokeOpacity: opacity,
            zIndex: 100,
            strokeWeight: 1
        });
        line.setMap(map)
    }

    circles = {{.Circles}}
    for (var i in circles) {
        var color = circles[i].color
        var circle = new google.maps.Circle({
            strokeColor: color,
            strokeOpacity: 1,
            strokeWeight: 1,
            //fillColor: '#0000FF',
            fillOpacity: 0,
            map: map,
            zIndex: 100,
            center: circles[i].center,
            radius: circles[i].radiusmeters
        });
    }
   
}
{{end}}
