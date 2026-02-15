"""Exceptions and errors."""

import contextlib
import uuid
from collections import defaultdict

import httpx
from fastapi import HTTPException
from pydantic import BaseModel, Field, ValidationError, computed_field
from starlette.status import HTTP_500_INTERNAL_SERVER_ERROR

from azul_bedrock.exception_enums import ExceptionCodeEnum
from azul_bedrock.language_catalogs.catalog_creation import get_catalog
from azul_bedrock.settings import BedrockSettings, LanguageCatalogsEnum

settings = BedrockSettings()

PARAMETER_TYPE = dict[str, str | int | float | bool] | None


class BaseAzulException(Exception):
    """Base Exception of Azul."""

    def __init__(self, *, internal: ExceptionCodeEnum, ref: str = "", parameters: PARAMETER_TYPE = None):
        """Creation of exception."""
        self.ref = ref
        self.internal = internal
        self.parameters = parameters
        self.external = self._external(internal, parameters)
        super().__init__()

    @staticmethod
    def _external(internal: ExceptionCodeEnum, parameters: PARAMETER_TYPE) -> str:
        """Get the value of the message meant to be provided to a user."""
        return get_message_from_error_code(internal, parameters, settings.language)

    def __str__(self) -> str:
        """String."""
        if self.external:
            return self.external
        return self.__repr__()

    def __repr__(self) -> str:
        """Repr."""
        class_name = self.__class__.__name__
        return f"{class_name}(ref={self.ref!r}, internal={self.internal.value!r}, external={self.external!r}, )"


class NetworkDataException(BaseAzulException):
    """Raised when data to be sent via network is invalid."""

    pass


class AzulRuntimeException(BaseAzulException):
    """Raised when a runtime type error occurs in Azul."""

    pass


class BaseError(BaseModel):
    """Standard Azul REST API Error format."""

    id: str = Field(default="", description="A unique identifier for this particular occurrence of the problem.")
    ref: str = Field(default="", description="An application-specific error reference.")
    internal: ExceptionCodeEnum = Field(
        description="Specific error type which has an error method associated with it."
    )
    # Override external text with something from dispatcher if required.
    external_override: str | None = ""

    parameters: PARAMETER_TYPE = Field(
        default=None, description="Keyword parameters to be used when formatting the error message."
    )

    @computed_field
    @property
    def external(self) -> str:
        """Get the value of the message meant to be provided to a user."""
        if self.external_override:
            return self.external_override
        return get_message_from_error_code(self.internal, self.parameters, settings.language)


class ApiException(HTTPException):
    """Generic exception for Azul restapi."""

    detail: dict[str, str]

    def __init__(
        self,
        *,
        status_code: int = HTTP_500_INTERNAL_SERVER_ERROR,
        ref: str = "",
        internal: ExceptionCodeEnum,
        parameters: PARAMETER_TYPE = None,
        external_override: str | None = None,
    ) -> None:
        """Init."""
        if parameters:
            parameters = parameters.copy()
        else:
            parameters = {}
        parameters["status_code"] = status_code
        detail = BaseError(
            id=str(uuid.uuid4()),
            ref=ref,
            parameters=parameters,
            external_override=external_override,
            internal=internal,
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
            f"external={self.detail['external']!r}"
            f")"
        )

    def __str__(self) -> str:
        """String."""
        detail = self.detail.get("external")
        if detail:
            return detail
        return self.__repr__()


class DispatcherApiException(ApiException):
    """Exceptions raised when failures occur when interacting with dispatchers API."""

    def __init__(
        self,
        *,
        internal: ExceptionCodeEnum,
        ref: str = "",
        parameters: PARAMETER_TYPE | None = None,
        response: httpx.Response | None = None,  # Dispatchers status code if part of exception.
    ):
        self.response = response
        status_code = HTTP_500_INTERNAL_SERVER_ERROR
        if self.response is not None:
            status_code = self.response.status_code

        if parameters:
            parameters = parameters.copy()
        else:
            parameters = {}
        parameters["status_code"] = status_code

        external_override = ""
        # Attempt to set external to the recommended value from dispatcher, only do with synchronous requests.
        if self.response and isinstance(self.response.stream, httpx.SyncByteStream):
            with contextlib.suppress(ValidationError):
                # Delayed import to avoid circular imports
                from azul_bedrock.models_api import DispatcherApiErrorModel

                d_err = DispatcherApiErrorModel.model_validate_json(self.response.content)
                if d_err.title and d_err.detail:
                    external_override = f"{d_err.title}: {d_err.detail}"
                elif d_err.title:
                    external_override = f"{d_err.title}"
                elif d_err.detail:
                    external_override = f"{d_err.detail}"

        super().__init__(
            status_code=status_code,
            ref=ref,
            internal=internal,
            parameters=parameters,
            external_override=external_override,
        )


class AzulValueError(BaseAzulException):
    """Custom value error to allow for provision of status messages."""

    def __init__(
        self,
        *,
        internal: ExceptionCodeEnum,
        ref: str = "",
        parameters: PARAMETER_TYPE = None,
    ) -> None:
        """Init."""
        super().__init__(ref=ref, internal=internal, parameters=parameters)


class AzulDispatcherRawResponseException(BaseAzulException):
    """Error response from Dispatcher with content provided that caused an error."""

    content: bytes

    def __init__(
        self,
        *,
        content: bytes,
        ref: str = "",
        internal: ExceptionCodeEnum,
        parameters: PARAMETER_TYPE = None,
    ) -> None:
        """Init."""
        self.content = content
        super().__init__(ref=ref, internal=internal, parameters=parameters)


def get_message_from_error_code(
    exceptionCodeEnum: ExceptionCodeEnum,
    parameters: PARAMETER_TYPE = None,
    language: LanguageCatalogsEnum = settings.language,
) -> str:
    """Convert an error code into an error message."""
    message = get_catalog(language).get(exceptionCodeEnum.value)
    if not message:
        return ""
    msg = message.string
    if not msg:
        return ""
    if parameters:
        # Default dictionary which enures formatting will occur event if parameters are missing or values change.
        default_params = defaultdict(str, parameters)
        return str(msg).format_map(default_params)
    return str(msg)
