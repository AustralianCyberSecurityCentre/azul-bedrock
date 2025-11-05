"""Dispatcher API Mocking."""

import json

import httpx

from .state import LastReq


class Editor:
    """Manipulate or view internal state of mock dispatcher."""

    def __init__(self, server: str):
        self.server = server

    def set_stream(self, sha256: str, code: int, data: bytes):
        """Set the content for the given sha256."""
        return httpx.post(f"{self.server}/mock/set_stream/{sha256}/{code}", content=data)

    def set_response(self, code: int, body: bytes | dict):
        """Set the response values for the next request."""
        if isinstance(body, dict):
            body = json.dumps(body)
        httpx.post(f"{self.server}/mock/set_resp/{code}", content=body)

    def get_last_request(self) -> LastReq:
        """Return last request info."""
        resp = httpx.get(f"{self.server}/mock/last_req")
        return LastReq.model_validate_json(resp.content)
