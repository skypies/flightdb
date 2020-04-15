# flightdb - a database for flight tracks

Prerequisites:
* the [Go programming language](https://golang.org/dl/)
* define your Go workspace: `export GOPATH=~/go`

Download and run things locally
* `go get github.com/skypies/flightdb/app/frontend` (pulls down all dependencies)
* `go run $GOPATH/github.com/skypies/flightdb/app/frontend/*.go` (build & run locally)
* Look at <http://localhost:8080/>

To deploy everything into a Google Cloud project:

```
gcloud app deploy --project=serfr0-fdb app/frontend --version=one
gcloud app deploy --project=serfr0-fdb app/backend  --version=one

gcloud app deploy --project=serfr0-fdb app/dispatch.yaml
gcloud app deploy --project=serfr0-fdb app/queues.yaml
gcloud app deploy --project=serfr0-fdb app/cron.yaml
gcloud app deploy --project=serfr0-fdb app/index.yaml
```

If you want it to accumulate realtime flight track data, you'll also want to:
* deploy `github.com/skypies/pi/skypi` onto some Raspberry Pi receivers
* deploy `github.com/skypies/pi/consolidator` into a VM inside your project

The skypies will post bundles of received ADSB (and perhaps MLAT)
messages up to Google PubSub, every second or so. The consolidator
will read those bundles, group them by airframe, and add them into the
database.

If you have CSV dumps of historical flight track data (perhaps from
the FAA), you can import it using the code in
`github.com/skypies/flightdb/app/backend/foia.go`.
