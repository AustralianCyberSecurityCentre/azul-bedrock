# Azul Restapi Common

Shared library for Azul models and operations.

## Design

The models are not in a nested folder as this gives nicer import names.

e.g.

```python
# much clearer naming of models on import, prevents confusion
from azul_thingo import api
from azul_bedrock import model_api

api.send_thingo(model_api.Thingo())

# annoying and depends on developer, likely will fragment into different names on import
from azul_thingo import api
from azul_bedrock.models import api as mapi

api.send_thingo(mapi.Thingo())
```

## Requirements

### libmagic

# default libmagic for debian can get out of date
contains a number of bugs for office and archive file types

```bash
git clone --depth 1 --branch FILE5_46 https://github.com/file/file
cd file/
autoreconf -f -i
./configure --disable-silent-rules
make -j4
sudo make install
sudo cp ./magic/magic.mgc /etc/magic
cd -
# check it worked
file --version
```

### yara

```bash
sudo apt-get install automake libtool make gcc pkg-config git flex bison -y
mkdir -p ./yara
git clone --branch v4.3.2 https://github.com/VirusTotal/yara ./yara
cd ./yara
./bootstrap.sh
./configure
make
sudo make install
cd -
## check it worked
yara -v
```

```bash
# you should put this in bashrc
export LD_LIBRARY_PATH="$LD_LIBRARY_PATH:/usr/local/lib"
```

## Install

```bash
pip install azul-bedrock
```

## Mocks

Using Mockery to generate mocks for golang client.

This is for use in upstream packages.

```bash
go install github.com/vektra/mockery/v2@v2.53.5
mockery
```

## Test Files

To run tests based off of bedrock you will need to set environment variables to allow you to download and use test files.

This requires you to be able to download files from Virustotal.
To save on download quotas files are also cached to an S3 cluster and your local file system.

The full set of environment variables for configuring this setup are here:

```bash
# Timeout for requests made by the file manager.
export file_manager_request_timeout: int = 30
# The URL to Virustotal's V3 API (guarantee no trailing slash).
export file_manager_virustotal_api_url="https://www.virustotal.com/api/v3"
# Virustotal API key used to download files from Virustotal
export file_manager_virustotal_api_key = ""
# whether to attempt to download files from virustotal or not.
export file_manager_virustotal_enabled="True"
# Directory where files are cached on the local file system when downloaded. (stored as carts)
export file_manager_file_cache_dir="/var/tmp/azul"
# Flag used to enable/disable the caching of test files.
export file_manager_file_caching_enabled="True"
# URL from Azure storage blob (storage account name address)
export file_manager_azure_storage_account_address=""
# Storage account Access key. (SAS key) used to access the azure storage.
export file_manager_azure_storage_access_key=""
# Name of the storage container within the blob storage.
export file_manager_azure_container_name="azul-test-file-cache"
# Flag used to enable/disable bucket caching.
export file_manager_azure_blob_cache_enabled="True"
```

It is recommended when you set these environment variables you add them to your `~/.bashrc` as it makes it easier when running tests.

## Avro Schema changes

If you modify the Avro schema you need to ensure you consider legal schema changes to maintain backwards compatibility.

Refer here for a guide: https://docs.confluent.io/platform/current/schema-registry/fundamentals/schema-evolution.html#compatibility-types

Azul must maintain BACKWARD_TRANSITIVE compatibility at all times with it's avro models.
This means fields can be deleted from the schema and OPTIONAL fields added.

If this is not achievable for a change the model version must be incremented and an upgrade path for old events needs to
be added to `msginflight/conversion_avro.go` for the model type that has been upgraded.

Test cases for the upgrade path will needed to be added.

Also if enough old versions build up the following release of Azul will need to require a kafka reprocess to be run
and assuming that has been run the old upgrade code and previous schemas could then be cleared.

## Integration tests

To run the golang integration test suite the docker-compose.yaml file must first be used to stand up,
a minio pod to act as an S3 datastore.

With the command:

` docker compose up`

Integration tests should be run with the script `test_integration.sh` with the
missing environment variables set to run the azure storage related tests.
