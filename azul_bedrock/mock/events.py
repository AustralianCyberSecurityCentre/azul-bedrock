"""Dispatcher API Mocking."""

import json

import urllib3
from fastapi import APIRouter, Request, Response

from azul_bedrock import models_api as azapi

from . import state

router = APIRouter()


@router.get("/api/v2/event/{rest:path}")
async def get_events(rest: str, request: Request, response: Response) -> bytes | list | dict | None:
    """Mocked endpoint."""
    await state.update_last(request)
    response.status_code = state._next_resp_code
    if response.status_code != 200:
        return state._next_resp_body

    loaded = json.loads(state._next_resp_body)
    fetched = len(loaded["events"])
    content, response.headers["Content-Type"] = urllib3.encode_multipart_formdata(
        {
            "info": (None, azapi.GetEventsInfo(filtered=0, fetched=fetched, ready=0).model_dump_json()),
            "events": (None, state._next_resp_body),
        }
    )
    return Response(content=content, media_type=response.headers["Content-Type"])


@router.post("/api/v2/event")
async def post_content_generic(request: Request, response: Response) -> dict:
    """Mocked endpoint."""
    await state.update_last(request)
    if ret := state.get_default_resp(response):
        return json.loads(ret)
    return {}


@router.post("/api/v2/event/simulate")
async def has_binary(request: Request, response: Response) -> dict:
    """Mocked endpoint."""
    await state.update_last(request)
    if ret := state.get_default_resp(response):
        return json.loads(ret)
    return {}
