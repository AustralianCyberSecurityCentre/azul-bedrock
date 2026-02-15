"""Exceptions specific to the azul_security repository."""

from azul_bedrock.exceptions_bedrock import BaseAzulException


class SecurityException(BaseAzulException):
    """Something went wrong with handling security."""

    pass


class SecurityConfigException(SecurityException):
    """The friendly config transform had errors."""

    pass


class SecurityParseException(SecurityException):
    """The friendly config transform had errors."""

    pass


class SecurityAccessException(SecurityException):
    """User not permitted to access object."""

    pass
