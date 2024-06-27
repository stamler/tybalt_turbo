/**
  Steps to export Firestore data to Parquet format in GCS
  
  1. Export the data from firestore to GCS
    - visit https://console.cloud.google.com/firestore/databases/-default-/import-export?project=charade-ca63f
    - click the "Export" button
    - select all collections to export from "export one or more collection groups"
    - select the destination "tybalt_firestore_collection_exports"
    - click "Export"

  2. Import the data from GCS into BigQuery
    - visit https://console.cloud.google.com/bigquery?project=charade-ca63f
    - click the "tybalt" dataset
    - click the "Create Table" button
    - select "Googld Cloud Storage" as the source
    - select the appropriate *.export_metadata file from the GCS bucket for the kind of data you are importing
    - select "Cloud Datastore Backup" as the file format
    - in Destination, give the table a name (likely same as the kind of data you are importing)
    - click "Create Table" 

  3. Export the data from BigQuery to GCS in Parquet format using the below query
*/

EXPORT DATA OPTIONS (
  uri = 'gs://tybalt_firestore_parquet/*.parquet',
  format = 'PARQUET',
  compression = 'SNAPPY',
  overwrite = true
) AS (
  SELECT *
    , __key__.kind AS kind
    , __key__.name AS keyname
    , __key__.id AS keyid
  FROM `charade-ca63f.tybalt.Jobs`
);