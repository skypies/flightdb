{{define "js-polling"}}

// Routines to make adding AJAX pollers easy.

// StartPolling( function(){
//        console.log("Hi ho!");
//   }, 5000, "hiho");
//
// StopPolling("hiho");

var pleaseStopFlags = [];

function StopPolling(name) { pleaseStopFlags[name] = true }

function StartPolling(func, intervalMillis, name) {
    delete pleaseStopFlags[name];
    var SleepUntil = time => new Promise(resolve => setTimeout(resolve, time))
    var LoopUntil = (promiseFn, time, name) => promiseFn().then(
        SleepUntil(time).then(
            function() {
                if (! pleaseStopFlags[name]) {
                    LoopUntil(promiseFn, time, name);
                }
            }
        )
    )

    // When the tab stops being visible, stop polling; if it becomes visible, then
    // resume
    document.addEventListener('visibilitychange', function () {
        var currentdate = new Date();
        if (document.hidden) {
            console.log("Hidden ! at " + currentdate + "("+name+")")
            StopPolling(name)
        } else {
            console.log("Unhidden ! at " + currentdate + "("+name+")")
            StartPolling(func, intervalMillis, name)
        }
    });
    
    LoopUntil( () => new Promise(()=>func()), intervalMillis, name )
}

{{end}}
