{{define "js-map-snapshotset"}} // Depends on: .Center (geo.Latlong), and .Zoom (int)

function localOverlay() {
    var legend = document.getElementById('legend');
    var div = document.createElement('div');
    div.innerHTML = {{.Legend}};
    legend.appendChild(div);

    // {pos:{lat:37.108600,lng:-122.014000}, heading:341, alt:10050, speed:290,
    //  flightnumber:"UA5400", icao24:"", callsign:"", reg:"", t:"HH:MM:SS PDT"}
    var flights = {{.FlightsJS}}
    var infowindow = new google.maps.InfoWindow({ content: "holding..." });

    for (var i in flights) {
        var f = flights[i]
        var infostring = '<div><b>'+f.flightnumber+'</b><br/>'+
            'Icao24: '+f.icao24+'<br/>'+
            'Callsign: '+f.callsign+'<br/>'+
            'Registration: '+f.reg+'<br/>'+
            'Altitude: '+f.alt+' feet<br/>'+
            'Speed: '+f.speed+' knots<br/>'+
            'Heading: '+f.heading+' degrees<br/>'+
            'Position: ('+f.pos.lat+','+f.pos.lng+')<br/>'+
            'Time of this trackpoint: '+f.t+'<br/>'+
            'DB Time marker: '+f.enterutc+'<br/>'+
            '<a target="_blank" href="'+f.url+'">Full trail map</a>'
            '</div>';

        var marker = new google.maps.Marker({
            title: f.flightnumber,
            html: infostring,
            position: f.pos,
            icon: {
                path: google.maps.SymbolPath.FORWARD_CLOSED_ARROW,
                scale: 3,
                strokeColor: '#0033ff',
                strokeWeight: 2,
                rotation: f.heading,
            },
            map: map
        });
        marker.addListener('click', function(){
            infowindow.setContent(this.html),
            infowindow.open(map, this);
        });
    }
}

{{end}}
