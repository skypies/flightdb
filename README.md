# flightdb2 - a historical flighttrack database (v2)

To deploy all this ...

    $ goapp deploy              ui/ui.yaml
    $ goapp deploy              backend/backend.yaml
    $ appcfg.py update_cron     backend/
    $ appcfg.py update_indexes  backend/
    $ appcfg.py update_queues   backend/
