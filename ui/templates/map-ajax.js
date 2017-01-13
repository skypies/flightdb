{{define "js-map-ajax"}}

// depends on "js-textboxes"

// StreamVectors()   - Looks for .IdSpecs

function generateDetailsClickClosure(detailsText) {
    //console.log('closure on "'+detailsText+'" generated')
    return function() {
        setTextBox('details', detailsText);
    };
}

function addLineFrag( key, val, detailsText ) {
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

    line.setMap(map);
    line.addListener('click', function(){ AddNote(detailsText) })
}

function generateKeyValConsumingFunction(detailsText) {
    return function( key, val ) {
        addLineFrag(key, val, detailsText);
    }
}

function generateUrlConsumingFunction(detailsText) {
    return function( data ) {
        $.each( data, generateKeyValConsumingFunction(detailsText) );
    }
}

function StreamVectors() {
    var idspecs = {{.IdSpecs}};
    for (var i in idspecs) {
        var idspec = idspecs[i].idspec;
        var url = {{.VectorURLPath}}+'?idspec='+idspec+'&json=1&trackspec='+{{.TrackSpec}}+
            '&'+{{.ColorScheme.QuotedCGIArgs}};
        {{if .Report}}
        url += '&'+{{.Report.QuotedCGIArgs}};
        {{end}}
        var detailsText = '<a target="_blank" href="/fdb/tracks?idspec='+idspec+'">['+i+'] '+
            idspec+'</a>';
        $.getJSON( url, generateUrlConsumingFunction(detailsText) );
    }
}

{{end}}
