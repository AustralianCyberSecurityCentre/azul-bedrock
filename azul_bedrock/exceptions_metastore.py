"""Exceptions specific to the azul_metastore repository."""

from azul_bedrock.exception_enums import ExceptionCodeEnum
from azul_bedrock.exceptions_bedrock import ApiException, BaseAzulException


class NoWriteException(BaseAzulException):
    """User does not have permission to write documents."""

    pass


class DataException(BaseAzulException):
    """Error getting data from dispatcher."""

    pass


class ConfigException(BaseAzulException):
    """Problem with the configuration of Azul Metastore."""

    pass


class BadSourceRefsException(BaseAzulException):
    """Source references are invalid."""

    pass


class FeatureEncodeException(BaseAzulException):
    """Something went wrong while encoding features."""

    pass


class ExtractException(BaseAzulException):
    """Supplied file could not be extracted."""

    pass


class CacheAlreadyExistsException(BaseAzulException):
    """Cache already exists and is being re-created."""

    pass


class BadCredentialsException(BaseAzulException):
    """Supplied credentials were invalid."""

    pass


class InvalidSearchException(BaseAzulException):
    """Arguments to search were invalid."""

    pass


class IndexException(BaseAzulException):
    """Exception when attempting to index the documents."""

    pass


class InitFailure(BaseAzulException):
    """Opensearch indices could not be verified/initialised for metastore."""

    pass


class InvalidAnnotation(BaseAzulException):
    """The supplied annotation is not valid."""

    pass


class PreprocessException(BaseAzulException):
    """The object is unable to be encoded for opensearch."""

    pass


class BadSourceException(PreprocessException):
    """The supplied source was not specified in configuration."""

    pass


class InvalidPurgeException(PreprocessException):
    """The provided arguments to the purge are invalid."""

    pass


def convert_exception_to_api_exception(
    base_exception: BaseAzulException, new_error_enum: ExceptionCodeEnum, status_code: int
) -> ApiException:
    """Convert an Azul exception into an azul API exception."""
    return ApiException(
        status_code=status_code, internal=new_error_enum, parameters={"inner_exception": base_exception.external}
    )
