{{define "js-heatmap"}}

// Depends on "js-polling" and "js-textboxes"; abuses the 'notes' textbox.

// InitHeatmap();
// InitUsermap();  // Unique users, not complaints
// FetchAndPaintHeatmap(duration);                // One-off render
// PollForHeatmap(duration,interval,expirytime);  // redraw every 'interval' until 'expirytime'

// https://developers.google.com/maps/documentation/javascript/heatmaplayer

var heatmap = null;

var heatmapCoords;
var heatmapIcaoId = "";
var heatmapDurationAll;
var heatmapDurationSingle = "15m";
var heatmapInterval;
var heatmapIsPolling = false;

// These two are hideous hacks.
var heatmapOfUniqueUsers = false;  // Go for users, not complaints
var heatmapOfAllUsers = false;     // Modifier: go for all-time users, not active<duration.

function getDuration() {
    duration = "14m";
    if (heatmapIcaoId != "") {
        duration = heatmapDurationSingle;
    } else {
        duration = heatmapDurationAll;
    }
    return duration;
}

function InitHeatmap() {
    heatmapCoords = new google.maps.MVCArray([]);
    heatmap = new google.maps.visualization.HeatmapLayer({data: heatmapCoords});
    heatmap.setMap(map);
    heatmapIsPolling = false;
    heatmapIcaoId = "";
    heatmapOfUniqueUsers = false;
    heatmapOfAllUsers = false;
}

function InitUsermap(dur) {
    InitHeatmap();

    heatmapOfUniqueUsers = true;
    if (dur == "all") {
        heatmapOfAllUsers = true;
    }
}

function SetHeatmapIcaoId(icaoid) {
    icaoid = icaoid.replace('EE',''); // In case this is from an fr24 dataset
    heatmapIcaoId = icaoid;
}

// One-off, non-polling version
function FetchAndPaintHeatmap(duration) {
    if (duration == "") {
        duration = getDuration()
    } else {
        heatmapDurationAll = duration
    }
    
    fetchAndPaintNewHeatmap(duration, function(){
        PaintNotes(heatmapSummary());
    });
}

function PollForHeatmap(duration,intervalMillis,expires) {
    heatmapDurationAll = duration;
    heatmapInterval = intervalMillis;
    PaintPollingLabel();
    toggleHeatmapPolling(); // Start !
    // Actual polling is triggered by toggleHeatmapPolling
}

function heatmapSummary() {
    var str = "Complaints in last "+getDuration()+": "+heatmapCoords.length;
    if (heatmapIcaoId != "") {
        str += "<br/>IcaoId: "+heatmapIcaoId;
    }

    if (heatmapOfAllUsers == true) {
        str = "All users ever registered: "+heatmapCoords.length;
    } else if (heatmapOfUniqueUsers == true) {
        str = "Users active over past "+getDuration()+": "+heatmapCoords.length;
    }

    return str;
}

function PaintPollingLabel() {
    var label = ""
    if (heatmapIsPolling) {
        label = heatmapSummary();
        label += '<br/><button onclick="toggleHeatmapPolling()">Stop updating</button>';
    } else {
        label = '<button onclick="toggleHeatmapPolling()">Start Realtime Complaints</button>'
    }

    PaintNotes(label)
}

function toggleHeatmapPolling() {
    name = "complaintheatmap";
    if (heatmapIsPolling) {
        heatmapIsPolling = false;
        SetHeatmapIcaoId("");
        clearHeatmapCoords();
        StopPolling(name);

    } else {
        heatmapIsPolling = true;

        SetHeatmapIcaoId(CurrentlySelectedIcao());
        
        StartPolling( function(){
            fetchAndPaintNewHeatmap(getDuration(), PaintPollingLabel);
        }, heatmapInterval, name);
    }
    PaintPollingLabel()
}

function clearHeatmapCoords() {
    while(heatmapCoords.getLength() > 0) heatmapCoords.pop(); // How to empty an MVC array
}

function fetchAndPaintNewHeatmap(duration, callback) {
    var url = "https://stop.jetnoise.net/heatmap?d=" + duration;

    if (heatmapOfUniqueUsers == true) {
        url += "&uniques=1";
    }

    if (heatmapIcaoId != "") {
        url += "&icaoid=" + heatmapIcaoId;
    }

    if (heatmapOfAllUsers == true) {
        url += "&allusers=1" // This causes all other args to be ignored
    }
    
    $.getJSON(url, function(arrayData){
        clearHeatmapCoords();
        $.each(arrayData, function(idx, obj){
            heatmapCoords.push(new google.maps.LatLng(obj.Lat,obj.Long));
        });
    }).done(function(data) {
        // Only create this when the getJSON is done.
        if (callback != null) {
            callback();
        }
    });
}

{{end}}
