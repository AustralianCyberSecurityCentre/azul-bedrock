"""Exceptions and errors."""

import contextlib
import uuid

import httpx
from fastapi import HTTPException
from pydantic import BaseModel, Field, ValidationError
from starlette.status import HTTP_500_INTERNAL_SERVER_ERROR

from azul_bedrock.models_api import DispatcherApiErrorModel


class NetworkDataException(Exception):
    """Raised when data to be sent via network is invalid."""

    pass


class BaseError(BaseModel):
    """Standard Azul REST API Error format."""

    id: str = Field(description="A unique identifier for this particular occurrence of the problem.")
    ref: str = Field(description="An application-specific error reference.")
    internal: str = Field(description="Message to return to user of api")
    external: str | None = Field(
        "Error details can be found in server logs.",
        description="Message to log to restapi stderr",
    )


class ApiException(HTTPException):
    """Generic exception for Azul restapi."""

    detail: dict[str, str]

    def __init__(
        self,
        *,
        status_code: int = HTTP_500_INTERNAL_SERVER_ERROR,
        ref: str,
        internal: str,
        external: str | None = None,
    ) -> None:
        """Init."""
        detail = BaseError(
            id=str(uuid.uuid4()),
            ref=ref,
            external=external,
            internal=internal,
            # meta=meta,
        ).model_dump(exclude_unset=True)
        super().__init__(status_code=status_code, detail=detail, headers=None)

    def __repr__(self) -> str:
        """Repr."""
        class_name = self.__class__.__name__
        return (
            f"{class_name}("
            f"status_code={self.status_code!r}, "
            f"id={self.detail['id']!r}, "
            f"ref={self.detail['ref']!r}, "
            f"internal={self.detail['internal']!r}"
            f")"
        )


class DispatcherApiException(ApiException):
    """Exceptions raised when failures occur when interacting with dispatchers API."""

    def __init__(
        self,
        *,
        ref: str,
        internal: str,
        response: httpx.Response | None = None,  # Dispatchers status code if part of exception.
        external: str | None = None,
    ):
        self.response = response
        status_code = HTTP_500_INTERNAL_SERVER_ERROR
        if self.response is not None:
            status_code = self.response.status_code

        # Attempt to set external to the recommended value from dispatcher, only do with synchronous requests.
        if response and not external and isinstance(response.stream, httpx.SyncByteStream):
            with contextlib.suppress(ValidationError):
                d_err = DispatcherApiErrorModel.model_validate_json(self.response.content)
                if d_err.title and d_err.detail:
                    external = f"{d_err.title}: {d_err.detail}"
                elif d_err.title:
                    external = f"{d_err.title}"
                elif d_err.detail:
                    external = f"{d_err.detail}"

        super().__init__(status_code=status_code, ref=ref, internal=internal, external=external)
