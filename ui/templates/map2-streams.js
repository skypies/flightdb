{{define "js-map2-streams"}} // Depends on: .MapLineOpacity

function streamVectors() {
    var idspecs = {{.IdSpecs}}
    for (var i in idspecs) {
        var idspec = idspecs[i].idspec
        var url = '/fdb/vector?idspec='+idspec+'&json=1&trackspec='+{{.TrackSpec}}
        
        $.getJSON( url, function( data ) {
            $.each( data, function( key, val ) {

                var color = val.color
                if (!color) { color = "#0022ff" }
                var opacity = val.opacity
                if (!opacity) { opacity = 1.0 }
                {{if .MapLineOpacity}}opacity = {{.MapLineOpacity}}{{end}}
                var coords = []
                coords.push(val.s)
                coords.push(val.e)

                var line = new google.maps.Polyline({
                    path: coords,
                    geodesic: true,
                    strokeColor: color,
                    strokeOpacity: opacity,
                    strokeWeight: 1,
                    zIndex: 100
                });

                line.setMap(map)                
            });
        });
    }
}


{{end}}
