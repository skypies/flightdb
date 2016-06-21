PROJECT=serfr0-1000
DATASET=public
TABLE=flights

SCHEMA=./flight-schema.json

# https://cloud.google.com/bigquery/quickstart-command-line

ls:
	bq --project_id ${PROJECT} show ${DATASET}.${TABLE}

mkdataset:
	bq --project_id ${PROJECT} mk ${DATASET}

mktable:
	bq \
	  --project_id ${PROJECT} \
	  mk \
	  --schema ${SCHEMA} \
	  --time_partitioning_type=DAY \
	  ${DATASET}.${TABLE}

rmtable:
	bq --project_id ${PROJECT} rm -f ${DATASET}.${TABLE}

newtable: rmtable mktable
