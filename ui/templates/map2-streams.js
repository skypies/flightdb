{{define "js-map2-streams"}}

function streamVectors() {
    var idspecs = {{.IdSpecs}}
    for (var i in idspecs) {
        var idspec = idspecs[i].idspec
        var url = {{.VectorURLPath}}+'?idspec='+idspec+'&json=1&trackspec='+{{.TrackSpec}}+
            '&'+{{.ColorScheme.QuotedCGIArgs}}
        
        $.getJSON( url, function( data ) {
            $.each( data, function( key, val ) {

                var color = val.color
                if (!color) { color = "#0022ff" }
                var opacity = val.opacity
                var coords = []
                coords.push({lat:val.s.Lat, lng:val.s.Long})
                coords.push({lat:val.e.Lat, lng:val.e.Long})

                var weight = 1
                if (opacity > 1) { weight = opacity }
                
                var line = new google.maps.Polyline({
                    path: coords,
                    geodesic: true,
                    strokeColor: color,
                    strokeOpacity: opacity,
                    strokeWeight: weight,
                    zIndex: 100
                });

                line.setMap(map)                
            });
        });
    }
}

{{end}}
