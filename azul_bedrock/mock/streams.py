"""Dispatcher API Mocking."""

import hashlib
import json
import re
from typing import Callable

from fastapi import APIRouter, Request, Response

from azul_bedrock import models_api as azapi
from azul_bedrock import models_network as azm

from . import state

router = APIRouter()


REQUEST_RANGE_RE = re.compile(r"bytes=([0-9]*)-([0-9]*)")


class RawResponse(Response):
    """Wraps generic response as an octet stream."""

    media_type = "binary/octet-stream"

    def render(self, content: bytes) -> bytes:
        """Wraps generic response as an octet stream."""
        if not isinstance(content, bytes):
            raise Exception(f"bad response type {type(content)} {content}")
        return content


def _parse_range(range_str: str) -> tuple[int, int | None]:
    """Parses a range header string and returns a tuple of (start, end).

    End may be None if unspecified, and start may be negative if a suffix request was made
        (in which case end will always be None).
    """
    match = REQUEST_RANGE_RE.match(range_str)
    if not match:
        raise RuntimeError("Could not parse request: 'Range: %s'" % range_str)
    start_str, end_str = match.groups()
    if start_str == "" and end_str == "":
        raise RuntimeError("start and end are both empty: 'Range %s'" % range_str)
    if start_str == "":
        return -int(end_str), None
    return int(start_str), (None if not end_str else int(end_str))


def _get_file_generic(size: int, datafunc: Callable[[int, int], bytes], request: Request, response: Response) -> bytes:
    """Handles a request and calculates the required range.

    Returns a 416 if appropriate, otherwise calls
    datafunc(slice_start, slice_end) to retrieve the data. `size` is the size of the file to be fetched from.
    (slice_end is an exclusive value, suitable for doing bytes_data[start:end] for example)
    """
    response.headers["content-type"] = "application/octet-stream"
    if "Range" not in request.headers:
        state._last_req.range = None
        response.status_code = 200
        return datafunc(0, size)
    start, end = _parse_range(request.headers["range"])
    state._last_req.range_raw = request.headers["range"]
    state._last_req.range = [start, end]
    if start < 0:
        start = max(size + start, 0)
    if end is None or end >= size:
        end = size - 1

    if start < 0 or start >= size or end < start:
        # Requested Range Not Satisfiable (or invalid)
        state._last_req.range = None
        response.status_code = 416
        response.headers["content-range"] = "bytes */%s" % size
        return b""
        # return response(status=416, headers={'content-range': 'bytes */%s' % size})

    state._last_req.range = (start, end)
    response.status_code = 206
    response.headers["content-range"] = "bytes %s-%s/%s" % (start, end, size)
    return datafunc(start, end + 1)


@router.post("/mock/set_stream/{data_hash}/{code}")
async def mock_set_stream(data_hash: str, code: int, request: Request) -> None:
    """Set a response."""
    data = await request.body()
    state._posted_files[data_hash] = (code, data)


@router.post("/api/v3/stream/{source}/{label}")
async def post_content_generic(request: Request, response: Response) -> azapi.DispatcherData:
    """Mocked endpoint."""
    data = await request.body()
    data_hash = hashlib.sha256(data).hexdigest()
    state._posted_files[data_hash] = (200, data)

    await state.update_last(request)
    if ret := state.get_default_resp(response):
        return json.loads(ret)

    # For testing purposes, this will return calculated hashes etc in tags, but no content
    hashes = {}
    for alg in ("md5", "sha1", "sha256", "sha512"):
        hashes[alg] = hashlib.new(alg, data, usedforsecurity=False).hexdigest()
    return azapi.DispatcherData(
        data=azm.Datastream(
            identify_version=1,
            label=azm.DataLabel.CONTENT,  # dispatcher currently always returns this
            size=len(data),
            mime="#TESTONLY",
            magic="#TESTONLY",
            file_format_legacy="#TESTONLY",
            file_format="#TEST/ONLY",
            file_extension="tonly",
            # We currently never return the encoded content, regardless of size
            **hashes,
        )
    )


@router.get("/api/v3/stream/{source}/{label}/{data_hash}")
async def get_content_generic(data_hash: str, request: Request, response: Response) -> bytes | None:
    """Mocked endpoint."""
    await state.update_last(request)
    if ret := state.get_default_resp(response):
        return ret
    if data_hash not in state._posted_files:
        response.status_code = 404
        return
    code, data = state._posted_files[data_hash]
    response.status_code = code
    if code < 200 or code > 299:
        return
    content = _get_file_generic(len(data), lambda start, end: data[start:end], request, response)
    return RawResponse(content, status_code=response.status_code, headers=response.headers)


@router.patch("/api/v3/stream/{source1}/{source2}/{label}/{data_hash}")
async def copy_binary(request: Request, response: Response) -> dict:
    """Mocked endpoint."""
    await state.update_last(request)
    if ret := state.get_default_resp(response):
        return json.loads(ret)
    return {}


@router.delete("/api/v3/stream/{source}/{label}/{data_hash}")
async def delete_binary(request: Request, response: Response) -> dict:
    """Mocked endpoint."""
    await state.update_last(request)
    if ret := state.get_default_resp(response):
        return json.loads(ret)
    return {}


@router.head("/api/v3/stream/{source}/{label}/{data_hash}")
async def has_binary(request: Request, response: Response) -> dict:
    """Mocked endpoint."""
    await state.update_last(request)
    if ret := state.get_default_resp(response):
        return json.loads(ret)
    return {}
