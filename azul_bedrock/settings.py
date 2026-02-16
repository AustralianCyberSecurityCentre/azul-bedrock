"""Base settings for bedrock."""

import logging
from enum import StrEnum
from functools import lru_cache

from pydantic_settings import BaseSettings, SettingsConfigDict

printed = False


class LanguageCatalogsEnum(StrEnum):
    """String enum of possible language catalogs that can be loaded.

    The string value is the name of the file in the language catalog directory.
    """

    English = "english.po"


class BedrockSettings(BaseSettings):
    """Base settings for bedrock module."""

    model_config = SettingsConfigDict(env_prefix="bedrock_")

    language: LanguageCatalogsEnum = LanguageCatalogsEnum.English


class OpensearchSettings(BaseSettings):
    """Opensearch settings that are used for accessing Opensearch."""

    model_config = SettingsConfigDict(env_prefix="metastore_")

    def __init__(self):
        """Init function."""
        global printed
        super().__init__()
        logger = logging.getLogger(__name__)
        # prevent duplicate printing for each read of settings
        if not printed:
            if not self.opensearch_url:
                logger.warning("no opensearch url set for metastore!")
            if not self.certificate_verification:
                logger.warning("certificate verification disabled!")

            if self.opensearch_url.startswith("http:"):
                logger.warning(f"host not under ssl! {self.opensearch_url}")
            printed = True

    # location of opensearch cluster that can be queried
    # can also be a load balancer
    opensearch_url: str = ""

    # Opensearch
    opensearch_azul_security_username: str = "azul_security"
    opensearch_azul_security_password: str = ""  # noqa: S105

    # admin credentials to create roles and rolemappings (must be used in conjunction with no-input flag)
    opensearch_admin_username: str = "azul_admin"
    opensearch_admin_password: str = ""  # noqa: S105

    # intended for local testing only
    certificate_verification: bool = True


@lru_cache()
def get_opensearch():
    """Return a cached copy of bedrock Opensearch settings."""
    return OpensearchSettings()
