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
    elif action == DownloadAction.SkippedAlreadyPresent:
        return StatusEnum.OPT_OUT
    elif action == DownloadAction.Requested:
        return StatusEnum.DOWNLOAD_REQUESTED
    elif action == DownloadAction.Success:
        return StatusEnum.COMPLETED
    return StatusEnum.ERROR_EXCEPTION


def convert_download_action_to_message(action: DownloadAction) -> str:
    """Get a message for a download action."""
    # Add a message associated with the download status.
    # FUTURE: could get this through the runner framework and allow plugins to override message.
    download_message = ""
    if action == DownloadAction.Failed:
        download_message = "The download request has failed with an error."
    elif action == DownloadAction.FailedNotFound:
        download_message = "Download was attempted but the requested sha256 was not found."
    elif action == DownloadAction.FailedNotFound:
        download_message = "Download was attempted but the requested sha256 was not found."
    elif action == DownloadAction.Requested:
        download_message = "Download was requested and is pending."
    elif action == DownloadAction.Success:
        download_message = "Download has successfully found the file and completed."
    return download_message
