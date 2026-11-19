# Retrohunt index storage: Bedrock CLI proof of concept

This patch introduces `cmd/azul-index` and `gosrc/indexstorage`. It **does not
change BigGrep**, the existing PVC flow, any Helm values, or the Retrofit
plugins. The implementation is intentionally opt-in.

## Why this design

BigGrep `bgparse` expects local files; the CLI stages immutable `.bgi` files
from S3 (including SeaweedFS), Azure Blob, or a local test backend. It builds
on Bedrock's existing `gosrc/store.FileStorage` implementations, avoiding a
second set of cloud SDK clients and credentials.

The object structure is:

```
retrohunt/indices/<sha256>.bgi
retrohunt/indices/<sha256>.ready
```

The digest is the SHA-256 of the complete `.bgi` bytes, not the input binaries.
The tiny `.ready` marker contains the digest and is published **only after**
the object-store upload succeeds. `list` shows only completed indexes.
A failed `put` can be safely retried, without replacing a published index.

## Build

Use the version of Go required by this repository's `go.mod` (Go 1.27.3 or
newer) and the normal Azul Bedrock build environment:

```
go build -o azul-index ./cmd/azul-index
```

This repository has existing toolchain and native dependencies. Build in its
usual CI image or development environment.

## Configuration

Only set credentials with environment variables, Kubernetes Secrets or workload
identity; do not put credentials in command arguments or config checked in to Git.

### S3 / SeaweedFS

```
export AZUL_INDEX_BACKEND=s3
export AZUL_INDEX_S3_ENDPOINT='your-seaweedfs-filer-service:8333'
export AZUL_INDEX_S3_SECURE=false
export AZUL_INDEX_S3_BUCKET=retrohunt-indexes
export AZUL_INDEX_S3_REGION=ap-southeast-2
export AZUL_INDEX_S3_ACCESS_KEY='...'
export AZUL_INDEX_S3_SECRET_KEY='...'
```

For AWS S3, use the appropriate host, `AZUL_INDEX_S3_SECURE=true`, and leave
both access/secret variables unset to use the existing Bedrock IAM credential
path. The S3 bucket should be pre-created and permissions restricted even
though Bedrock's current store constructor can create a missing bucket.

### Azure Blob

```
export AZUL_INDEX_BACKEND=azure
export AZUL_INDEX_AZURE_ENDPOINT='https://youraccount.blob.core.windows.net/'
export AZUL_INDEX_AZURE_CONTAINER=retrohunt-indexes
# Optional when using a shared access key:
export AZUL_INDEX_AZURE_ACCOUNT='youraccount'
export AZUL_INDEX_AZURE_KEY='...'
```

With no Azure access key, Bedrock uses `DefaultAzureCredential` (workload
identity or configured service principal). The existing Azure client attempts
to create the container if missing; ensure the identity is appropriately scoped.

### Local smoke tests

```
export AZUL_INDEX_BACKEND=local
export AZUL_INDEX_LOCAL_ROOT=/tmp/azul-index-object-store
```

## Commands

```
# Retroindexer: generate a .bgi locally with existing bgindex, then publish it.
ID=$(azul-index put --file /tmp/generated.bgi)

# Retrohunt: discover ready indexes (one SHA-256 ID per line).
azul-index list

# Retrohunt: fetch once to an ephemeral cache, then give this path to bgparse.
azul-index fetch --id "$ID" --output "/var/cache/azul/indexes/$ID.bgi"
```

`fetch` hashes any existing cached file before using it, streams a missing
object to a same-directory temporary file, checks size and SHA-256, fsyncs it,
and renames it atomically. Linux `flock` prevents multiple local workers from
fetching the same index simultaneously when the destination path is identical.
The `*.lock` files can remain harmlessly in the cache directory.

## Suggested staged Retrohunt/Retroindexer integration

1. **Retroindexer:** keep running the current BigGrep `bgindex`; run
   `azul-index put` after the index is fully generated; publish index IDs to
   the normal Azul metadata/event mechanism. Record the digest and intended
   data-set/batch ID in the plugin's existing metadata; the CLI does not infer
   batch ownership or retention from object names.
2. **Retrohunt:** under a new opt-in setting, list or resolve the appropriate
   published index IDs, and invoke `azul-index fetch` once per required index.
   Reuse the resulting local `.bgi` path for every broad-phase task that needs
   it. Run the unchanged `bgparse` and YARA-X verification flows.
3. **PVC fallback:** maintain existing behaviour when the new flag is off;
   retain until correctness, performance and availability pass in dev.
4. **Kubernetes:** mount a writable `emptyDir` for index cache. Set an
   ephemeral-storage request/limit that covers expected concurrent index
   downloads and working sets. Consider cache retention by node/pod if cold
   starts become a bottleneck.

## Performance guidance

- Bedrock's current S3 and Blob uploaders already support streaming large
  files and concurrent multipart/chunked uploads. The CLI passes a known size.
- Downloads are streamed with bounded memory; this implementation does **not**
  load complete indexes into RAM.
- Per-path file locking prevents N workers in the same pod downloading one
  object N times. Different pods have independent caches.
- Cached files are SHA-256 verified each access. This costs sequential disk
  I/O but avoids silently searching corrupted indexes. A future trusted-cache
  metadata/mtime validation mode could improve repeated very-large-index
  performance, with a deliberate integrity trade-off.
- `list` currently returns all committed IDs and does not implement pruning or
  pagination at CLI level. Integrate with existing index metadata to select
  just the indexes needed for a hunt.

## Important limitations and follow-up work

- **Not yet integrated into Retrohunt/Retroindexer.** This is a reusable
  Bedrock CLI + Go package, not a complete migration.
- **No remote zero-copy mmap:** BigGrep still consumes local indexes.
- **No lifecycle or GC:** the marker and object are immutable. Add an external
  retention/GC process that only removes indexes no active hunt refers to.
- **No distributed cache lock:** locking applies within one filesystem/pod.
- **No timeout propagation:** the inherited `FileStorage` interface does not
  accept context for Put/Fetch; operations follow existing Bedrock behaviour.
- **No cross-component index registry:** reference IDs, batch metadata, index
  generation version and retention policies must be managed by the plugins.
- **No validation of `.bgi` internals:** verifies bytes, not BigGrep syntax.
- **Not a production atomic transaction:** object stores do not make a
  multi-object transaction. Marker publication gives safe discoverability
  under normal object store consistency; verify backend-specific semantics.
- The existing Bedrock S3/Blob constructors may create a missing bucket or
  container and have their own credential and timeout behaviour.
- Review GPL obligations if packaging existing BigGrep binaries alongside
  Bedrock tooling; this patch itself does not vendor BigGrep code.

## Validation

Unit tests under `gosrc/indexstorage` exercise publishing, fetching, listing,
cache corruption recovery, unfinished uploads and concurrent fetches against
Bedrock's local storage implementation. Run:

```
go test ./gosrc/indexstorage ./cmd/azul-index
```

Before production, exercise the exact SeaweedFS deployment and a real Azure
Blob account, benchmark against PVC with representative `.bgi` sizes, test
interrupted uploads and failed downloads, and verify the SeaweedFS writable-
volume issue previously observed is resolved.
