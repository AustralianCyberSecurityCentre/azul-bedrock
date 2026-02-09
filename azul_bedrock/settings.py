"""Base settings for bedrock."""

from enum import StrEnum

from pydantic_settings import BaseSettings, SettingsConfigDict


class LanguageCatalogsEnum(StrEnum):
    """String enum of possible language catalogs that can be loaded.

    The string value is the name of the file in the language catalog directory.
    """

    English = "english.po"


class BedrockSettings(BaseSettings):
    """Base settings for bedrock module."""

    model_config = SettingsConfigDict(env_prefix="bedrock_")

    language: LanguageCatalogsEnum = LanguageCatalogsEnum.English
