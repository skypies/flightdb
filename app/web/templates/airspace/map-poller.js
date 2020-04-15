{{define "js-map-poller"}}

{{template "js-polling"}}
{{template "js-overlays"}}
{{template "js-textboxes"}}
{{template "js-airspace"}}

{{template "js-heatmap"}}

var map;
function initMap() {
    {{template "js-map-styles"}}
    
    map = new google.maps.Map(document.getElementById('map'), {
        center: {lat: {{.Center.Lat}}, lng: {{.Center.Long}}},
        zoom: {{.Zoom}},
        scaleControl: true,
        mapTypeControlOptions: {
            mapTypeIds: ['roadmap', 'satellite', 'hybrid', 'terrain',
                         'Silver']
        }
    });

    map.mapTypes.set('Silver', styledMapSilver);
    map.setMapTypeId('Silver');
    
    map.controls[google.maps.ControlPosition.RIGHT_TOP].push(
        document.getElementById('legend'));
    map.controls[google.maps.ControlPosition.LEFT_BOTTOM].push(
        document.getElementById('details'));
    map.controls[google.maps.ControlPosition.LEFT_BOTTOM].push(
        document.getElementById('notes'));

    ClassBOverlay();
    PathsOverlay();
    
    InitMapsAirspace();

    StartPolling( function(){
        pollAndPaintAircraft( {{.URLToPoll}} );
    }, gPollingIntervalMillis, "aircraft");

    InitHeatmap();
    PollForHeatmap("45s", 2000, 1000*3);
}

function PaintPollingLegend() {
    PaintLegend( generateLegend() );
    attachOnClicksToLegendText();  // Kinda hacky
}

// Any/all onClick events tagged to hyperlinks in the legend should be set here.
function attachOnClicksToLegendText() {
    $(function() {
        $("#toggle").click(function(e) {
            e.preventDefault();
            togglePolling()
        });
    });
}

function generateLegend() {
    var legend = ''
    if (gPollingPaused) { legend += '[<a href="#" id="toggle">Resume polling</a>]' }
    else                { legend += "" }

    var now = new Date();
    legend = legend + " " + CountVisibleAircraft() + " aircraft, " + now.toTimeString();

    return legend
}

var gPollingIntervalMillis = 1000;
var gMaxPollsRemain = (60000/gPollingIntervalMillis)*30;  // 30m
var gPollsRemain = gMaxPollsRemain;

var gPollingPaused = false;
function togglePolling() {
    if (gPollingPaused) {
        gPollingPaused = false;
        gPollsRemain = gMaxPollsRemain;
    } else {
        gPollingPaused = true;
    }
    PaintPollingLegend();
}

function pollAndPaintAircraft(url) {
    if (gPollingPaused) { return }
    PaintPollingLegend();

    gPollsRemain--;
    if (gPollsRemain <= 0) {
        togglePolling();
        return;
    }
    
    var liveAircraft = {};
    $.getJSON( url, function( data ) {
        $.each( data["Aircraft"], function( icaoid, aircraft ) {
            PaintAircraft(aircraft);
            liveAircraft[icaoid] = 1;
        });
        ExpireAircraft(liveAircraft);
    });
}    

{{end}}
