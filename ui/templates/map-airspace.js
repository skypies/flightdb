{{define "js-map-airspace"}} // Cloned from pi/frontend/templates/map-airspace.js

function airspaceOverlay() {
    var aircraft = {{.AirspaceJS}}
    var infowindow = new google.maps.InfoWindow({ content: "holding..." });

    for (var i in aircraft) {
        var a = aircraft[i]

        var ident = a.flightnumber
        if (!ident) {
            ident = a.callsign
        }
        if (!ident) {
            ident = a.reg
        }

        var header = '<div><b>'+ident+'</b><br/>'
        
        if (a.fdburl) {
            header = '<div><b><a target="_blank" href="'+a.fdburl+'">'+ident+'</a></b><br/>'+
            '[<a target="_blank" href="'+a.faurl+'">FA</a>,'+
            ' <a target="_blank" href="'+a.approachurl+'">DescentGraph</a>'+
                ']<br/>'
        } else {
            header += '[<a target="_blank" href="'+a.faurl+'">FlightAware</a>]<br/>'
        }

        var infostring = header +
            'FlightNumber: '+a.flightnumber+'<br/>'+
            'Schedule: '+a.orig+" - "+a.dest+'<br/>'+
            'Callsign: '+a.callsign+'<br/>'+
            'Icao24: '+a.icao24+'<br/>'+
            '  -- Registration: '+a.reg+'<br/>'+
            '  -- IcaoPrefix: '+a.icao+'<br/>'+
            '  -- Equipment: '+a.equip+'<br/>'+
            'PressureAltitude: '+a.alt+' feet<br/>'+
            'GroundSpeed: '+a.speed+' knots<br/>'+
            'Heading: '+a.heading+' degrees<br/>'+
            'Position: ('+a.pos.lat+','+a.pos.lng+')<br/>'+
            //'Last seen: '+a.age+' seconds ago<br/>'+
            'Source: '+a.source+' / '+a.receiver+' ('+ a.system+')<br/>'+
            '</div>';

        var zDepth = 3000;
        if (a.source == "fr24") {
            zDepth = 2000;
        }
        
        var marker = new google.maps.Marker({
            title: ident,
            html: infostring,
            position: a.pos,
            zIndex: zDepth,
            icon: {
                path: google.maps.SymbolPath.FORWARD_CLOSED_ARROW,
                scale: 3,
                strokeColor: a.color,
                strokeWeight: 2,
                rotation: a.heading,
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
