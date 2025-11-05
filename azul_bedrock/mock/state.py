"""Dispatcher API Mocking."""

from fastapi import APIRouter, Request, Response
from pydantic import BaseModel

router = APIRouter()


class LastReq(BaseModel):
    """Holds information about the last request to mocked dispatcher."""

    url: str = ""
    headers: dict[str, str] = {}
    body: bytes = b""
    range_raw: str = ""
    range: list[int] | None = []


# Stores files for later retrieval
_posted_files: dict[str, tuple[int, bytes]] = {}
_last_req = LastReq()
_next_resp_code: int = 0
_next_resp_body: bytes = b""


def clear():
    """Clears out any dispatcher state (that isn't set from last request)."""
    global _last_req, _posted_files, _next_resp_body, _next_resp_code
    _posted_files = {}
    _last_req = LastReq()
    _next_resp_code = 0
    _next_resp_body = b""


clear()


def get_default_resp(response: Response) -> bytes:
    """Return the default response for a route."""
    global _next_resp_code, _next_resp_body
    if not _next_resp_code:
        return False
    response.status_code = _next_resp_code
    body = _next_resp_body
    # reset
    _next_resp_code = 0
    _next_resp_body = b""
    return body


async def update_last(request: Request):
    """Update record of last request."""
    _last_req.body = await request.body()
    _last_req.url = str(request.url)
    _last_req.headers = {
        x: y for x, y in dict(request.headers).items() if x not in ["host", "accept", "accept-encoding", "connection"]
    }
    if "range" in _last_req.headers:
        _last_req.range_raw = _last_req.headers["range"]
    else:
        _last_req.range_raw = ""


@router.get("/mock/last_req")
async def mock_get_last_req() -> LastReq:
    """Get last request info."""
    return _last_req


@router.post("/mock/set_resp/{code}")
async def mock_set_resp(code: int, request: Request) -> None:
    """Set a response."""
    global _next_resp_code, _next_resp_body
    body = await request.body()
    _next_resp_code = code
    _next_resp_body = body
