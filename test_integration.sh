#/bin/bash
# local testing of integration
# first argument is prefix of tests to run, if empty runs all

BED__LOG_LEVEL=TRACE \
BED__LOG_PRETTY=TRUE \
DP__STREAMS__S3__ENDPOINT=localhost:9000 \
DP__STREAMS__S3__ACCESS_KEY=minio-root-user \
DP__STREAMS__S3__SECRET_KEY=minio-root-password \
DP__STREAMS__S3__SECURE="false" \
DP__STREAMS__API_ALLOW_DELETE=TRUE \
go test ./... -count=1 -tags=integration -run=$1 -p 1 -failfast

# The following runs integration tests on Azure Blob, ensure to set the <set-value> env's
# BED__LOG_LEVEL=TRACE \
# BED__LOG_PRETTY=TRUE \
# DP__STREAMS__AZURE__ACCESS_KEY=<value-here> \
# DP__STREAMS__AZURE__CONTAINER=<value-here> \
# DP__STREAMS__AZURE__STORAGE_ACCOUNT=<value-here \
# DP__STREAMS__AZURE__ENDPOINT=<value-here> \
# go test ./... -count=1 -tags=integration_azure -run=$1
