# flightdb2 - the temp name for the flightdb

To deploy all this ...

$ goapp deploy              ui/ui.yaml
$ goapp deploy              backend/backend.yaml
$ appcfg.py update_cron     backend/
$ appcfg.py update_indexes  backend/


# Todo

Move timeslot rounding into util/date
Move timeslot duration out of fgae; the UI needs it (to round things down)
Implement lookup.ByTime with single time, not just range
UI: better 'recent' view
UI: render a flight as lines (and impose min length ?)
UI: render multiple flights as lots of lines ?

Poller: implement a fr24 poller
  Takes snapshots (see just below) - probably only every 10m or so.
  A] Update an Icao24 -> registration database.
  B] Lookup flight in our DB, by Mode-S ! (or flightnumber or callsign ?)
  C] update/insert: add flight identity, provisional fr24 foreign-key, and sched dep/arr
   ] (perhaps update/replace the fr24 foreign-key ?)
  D] add to a workqueue for trackfetch

  Add the flight to a workqueue, for track fetching
  Add a thingy that uses fr24 to lookup a registration: https://www.flightradar24.com/reg/n903sw
   and turns it into a flightnumber ? (what for ??)

Workqueue - trackfetch:
  A] go to fr24, using most-recent fr24-key; plausible match to anything already there ?
  B] if interesting, go to fa, also do a plausible match

Workqueue: incorporate flightaware data (either flightinfoex, track, or both)

Workqueue - flightpath [re]analysis
  A] which waypoints ? Tag with well-known procedures
  B] Class B

1. Normal scheduled flights
["7624382","AC7BF6",37.7370,-122.4019,195,6775,269,"3253","T-KSFO1","CRJ2","N903SW",1441900518,"SFO","BFL","UA5613",0,2176,"",0]
["76319bb","A6E88B",37.6254,-122.3963,276,74,9,    "1414","T-MLAT2","B752","N544UA",1441940807,"OGG","SFO","UA738", 1,0,   "UAL738",0]

2. Unscheduled flights, but with ModeS and registration
["7638091","A8A763",37.6081,-122.3855,197,74,7,    "6337","T-MLAT2","B762","N657GT",1441940842,"","","",            1,0,   "",0]
["76375b8","A1B8B8",37.6351,-122.3929,332,100,10,  "4262","T-MLAT7","B190","N21RZ", 1441940793,"","","",            1,0,   "",0]

3. Anonymous flights, with nothing but a crappy callsign (private jets / general aviation ?)
["7624195","",      37.6762,-122.5215,275,4143,142,"3347","T-MLAT2","GLF4","",      1441900519,"","","",            0,2048,"GLF4",0]
["76226db","",      37.6278,-122.3826,163,0,0,     "3226","F-KSFO1","E55P","",      1441900520,"","","",            1,0,   "E55P",0]



