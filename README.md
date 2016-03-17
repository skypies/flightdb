# flightdb2 - a historical flighttrack database (v2)

To deploy all this ...

    $ goapp deploy              ui/ui.yaml
    $ goapp deploy              backend/backend.yaml
    $ appcfg.py update_cron     backend/
    $ appcfg.py update_indexes  backend/
    $ appcfg.py update_queues   backend/
    $ appcfg.py update_dispatch backend/


MLAT stuff

* ensure the addfrag stuff is robust
* ensure it won't get out of db v2

Puzzles

1. http://fdb.serfr1.org/fdb/debug?idspec=A835D1@1457364600
- why not tagged with EPICK ?
- why not tagged with AL ?
