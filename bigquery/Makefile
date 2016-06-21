PROJECT=serfr0-1000
DATASET=public
TABLE=flights

SCHEMA=./bigquery-schema.json

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
