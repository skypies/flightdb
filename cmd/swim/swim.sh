#!/bin/sh

# The SWIM Jumpstart is a tarball with a handy java program that dequeues stuff from JMS
# and, if configured, sends it to STDOUT as JSON.
JUMPSTART=~/swim
CONFIG=${JUMPSTART}/application.conf

export GOOGLE_DEFAULT_CREDENTIALS=~/serfr0-fdb-auth.json
SWIMBIN=~/repo/skypies/flightdb/cmd/swim/swim

cd ${JUMPSTART}
./bin/run -Dconfig.file=${CONFIG} | ${SWIMBIN}
