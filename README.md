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
sudo apt install autoconf automake autotools-dev gcc libtool make zlib1g-dev
git clone --depth 1 --branch FILE5_47 https://github.com/file/file
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

### yara-x

Library required for identify to work, must be manually installed for golang not for python.

```bash
# Install yara-x for identify - needed for golang bedrock
# Install Rust and yara-x
export RUST_VERSION=1.96.0
gpg --keyserver hkps://keyserver.ubuntu.com --recv-keys 85AB96E6FA1BE5FE
# Download Rust tarball + signature
curl -O https://static.rust-lang.org/dist/rust-${RUST_VERSION}-x86_64-unknown-linux-gnu.tar.gz \
    && curl -O https://static.rust-lang.org/dist/rust-${RUST_VERSION}-x86_64-unknown-linux-gnu.tar.gz.asc
# Verify signature
gpg --verify rust-${RUST_VERSION}-x86_64-unknown-linux-gnu.tar.gz.asc

# perform rust install
tar xzf rust-${RUST_VERSION}-x86_64-unknown-linux-gnu.tar.gz
rust-${RUST_VERSION}-x86_64-unknown-linux-gnu/install.sh --prefix=/usr/local --without=rust-docs
rm -rf rust-${RUST_VERSION}-*

cargo install cargo-c
git clone -b v1.18.0 https://github.com/VirusTotal/yara-x.git && \
cd yara-x
cargo cinstall -p yara-x-capi --release --prefix /usr/local
sudo cp -r /tmp/yara-build/lib/x86_64-linux-gnu/* /usr/local/lib
sudo cp -r /tmp/yara-build/include/yara_x.h /usr/include/yara_x.h
cd ..
rm -rf yara-x

# Verify that yara-x install was successfull
cat <<'EOF' > test.c
#include <yara_x.h>
int main() {
    YRX_RULES* rules;
    yrx_compile("rule dummy { condition: true }", &rules);
    yrx_rules_destroy(rules);
}
EOF
gcc `pkg-config --cflags yara_x_capi` test.c `pkg-config --libs yara_x_capi`
rm test.c
# End of yara-x/rust install
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

## Dependency management

Dependencies are managed in the pyproject.toml and debian.txt file.

Version pinning is achieved using the `uv.lock` file.
Because the `uv.lock` file is configured to use a private UV registry, external developers using UV will need to delete the existing `uv.lock` file and update the project configuration to point to the publicly available PyPI registry instead.

To add new dependencies it's recommended to use uv with the command `uv add <new-package>`
    or for a dev package `uv add --dev <new-dev-package>`

The tool used for linting and managing styling is `ruff` and it is configured via `pyproject.toml`

The debian.txt file manages the debian dependencies that need to be installed on development systems and docker images.

Sometimes the debian.txt file is insufficient and in this case the Dockerfile may need to be modified directly to
install complex dependencies.
