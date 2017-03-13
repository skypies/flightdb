# flightdb2 - a historical flighttrack database (v2)

To deploy all this ...

    $ goapp deploy              ui/
    $ goapp deploy              backend/

    $ appcfg.py update_cron     backend/
    $ appcfg.py update_indexes  backend/
    $ appcfg.py update_queues   backend/
    $ appcfg.py update_dispatch backend/

Layout guidelines

a FormValue method parses and instantiates (or errors). Does not do DS
reads.

Magic loading only happens at the ui/backend level.


* flightdb2: may not depend on anything in fgae, or needing a DB or context.
* flightdb2/fgae: depends on fdb and appengine
* flightdb2/report :
* flightdb2/{ui,backend}: can mix fgae & flightdb2
