"""Restapi models for downloading binaries ."""

from typing import Annotated

from pydantic import BaseModel
from pydantic.functional_serializers import PlainSerializer
from pydantic.types import AwareDatetime

from azul_bedrock.models_network import DownloadAction, StatusEnum, StatusEvent


class DownloadResponse(BaseModel):
    """Response from a download request."""

    # Sha256 that was requested for download.
    sha256: str
    # Security classification associated with the download event.
    last_download_security: str
    # Last download request time
    last_download_timestamp: Annotated[AwareDatetime, PlainSerializer(lambda v: v.isoformat(), return_type=str)]
    # Last download requested by specified user
    last_download_author_name: str
    # All the statuses from download plugins
    plugin_statuses: list[StatusEvent]


def convert_download_action_to_status(action: DownloadAction) -> StatusEnum:
    """Convert a download action to a sensible status."""
    if action == DownloadAction.Failed:
        return StatusEnum.ERROR_EXCEPTION
    elif action == DownloadAction.FailedNotFound:
        return StatusEnum.OPT_OUT
    elif action == DownloadAction.Requested:
        return StatusEnum.DOWNLOAD_REQUESTED
    elif action == DownloadAction.Success:
        return StatusEnum.COMPLETED
    return StatusEnum.ERROR_EXCEPTION
