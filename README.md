# flightdb2 - a historical flighttrack database (v2)

To deploy all this ...

    $ goapp deploy              ui/
    $ goapp deploy              backend/

    $ appcfg.py update_cron     backend/
    $ appcfg.py update_indexes  backend/
    $ appcfg.py update_queues   backend/
    $ appcfg.py update_dispatch backend/
