# flightdb - a database for flight tracks

Prerequisites:
* the [Go programming language](https://golang.org/dl/)
* the [Go appengine SDK](https://cloud.google.com/appengine/docs/go/), and add it to your `$PATH`
* define your Go workspace: `export GOPATH=~/go`

Download and run things locally
* `go get github.com/skypies/flightdb/app/frontend` (pulls down all dependencies)
* `goapp serve $GOPATH/github.com/skypies/flightdb/app/frontend` (build & run locally)
* Look at <http://localhost:8080/> (appengine admin panel is <http://localhost:8000/>)

To deploy everything into a Google Cloud project:

    $ goapp deploy              app/frontend
    $ goapp deploy              app/backend

    $ appcfg.py update_cron     app/backend
    $ appcfg.py update_indexes  app/backend
    $ appcfg.py update_queues   app/backend
    $ appcfg.py update_dispatch app/backend

If you want it to accumulate realtime flight track data, you'll also want to:
* deploy `github.com/skypies/pi/skypi` onto some Raspberry Pi receivers
* deploy `github.com/skypies/pi/consolidator` onto an AppEngine Flexible Environment

The skypies will post bundles of received ADSB (and perhaps MLAT)
messages up to Google PubSub, every second or so. The consolidator
will read those bundles, group them by airframe, and add them into the
database.

If you have CSV dumps of historical flight track data (perhaps from
the FAA), you can import it using the code in
`github.com/skypies/flightdb/app/backend/foia.go`.
