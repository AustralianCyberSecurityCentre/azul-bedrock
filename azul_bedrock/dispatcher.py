"""Handle interaction with dispatcher restapi."""

import io
import json
import logging
import typing
from collections.abc import AsyncIterable
from typing import Any, Callable

import httpx
import multipart
from pendulum.datetime import DateTime
from starlette.datastructures import UploadFile
from starlette.status import HTTP_200_OK, HTTP_206_PARTIAL_CONTENT, HTTP_404_NOT_FOUND

from azul_bedrock import models_api as azapi
from azul_bedrock import models_network as azm
from azul_bedrock.exceptions import DispatcherApiException, NetworkDataException

from . import dispatcher_params as DPP

logger = logging.getLogger(__name__)

# max message size is 2MB minus some slack
# azul expects kafka to allow 2MB max message size rather than default 1MB
MAX_MESSAGE_SIZE = 1024 * 1024 * 2 - 512
# 1 MB buffer when uploading content.
MAX_BUFFER_BYTES = 1024 * 1024


class BadResponseException(Exception):
    """Response from dispatcher was not great."""

    content: bytes


class DispatcherAPI:
    """Manage access to azul dispatcher restapi."""

    def __init__(
        self,
        *,
        events_url: str,
        data_url: str,
        retry_count: int,
        timeout: float,
        author_name: str,
        author_version: str,
        deployment_key: str,
    ) -> None:
        self._author_name = author_name
        self._author_version = author_version
        self._deployment_key = deployment_key
        self._events_url = events_url
        self._data_url = data_url
        self.user_agent = f"dpclient-py-{self._author_name}"
        self._client = httpx.Client(
            mounts={
                "https://": httpx.HTTPTransport(retries=retry_count),
                "http://": httpx.HTTPTransport(retries=retry_count),
            },
            headers={
                # custom user agent
                "user-agent": self.user_agent,
            },
            timeout=timeout,
        )
        # Session to allow connection reuse and retry
        self._async_client = httpx.AsyncClient(
            mounts={
                "https://": httpx.AsyncHTTPTransport(retries=retry_count),
                "http://": httpx.AsyncHTTPTransport(retries=retry_count),
            },
            headers={
                # custom user agent
                "user-agent": self.user_agent,
            },
            timeout=timeout,
        )

    def _get_events(
        self,
        *,
        model: str,
        loader: Callable[[bytes], list[Any]] = None,
        count: int = 1,
        deadline: int = 1,
        is_task: bool = False,
        deny_actions: typing.Optional[list[str]] = None,
        deny_self: bool = False,
        require_expedite: bool = False,
        require_live: bool = False,
        require_historic: bool = False,
        require_content: bool = False,
        require_under_content_size: int = 0,
        require_over_content_size: int = 0,
        require_actions: typing.Optional[list[str]] = None,
        require_streams: typing.Optional[list[str]] = None,
    ) -> tuple[azapi.GetEventsInfo, list[Any]]:
        """Get azul events from dispatcher."""
        if model not in azm.ModelType:
            raise Exception(f"invalid model for _get_events: {model}")

        if not require_actions:
            require_actions = []
        if not deny_actions:
            deny_actions = []
        if not require_streams:
            require_streams = []

        endpoint = DPP.GET_EVENTS_ENDPOINT_ACTIVE
        if not is_task:
            endpoint = DPP.GET_EVENTS_ENDPOINT_PASSIVE

        params_opt = {
            DPP.GetEvent.Name: self._author_name,
            DPP.GetEvent.Version: self._author_version,
            DPP.GetEvent.Count: count,
            DPP.GetEvent.Deadline: deadline,
            DPP.GetEvent.DeploymentKey: self._deployment_key,
            # python client can't handle avro
            # this gets dropped in params because falsey is default
            DPP.GetEvent.AvroFormat: False,
            DPP.GetEvent.DenyActions: deny_actions,
            DPP.GetEvent.DenySelf: deny_self,
            DPP.GetEvent.RequireExpedite: require_expedite,
            DPP.GetEvent.RequireLive: require_live,
            DPP.GetEvent.RequireHistoric: require_historic,
            DPP.GetEvent.RequireContent: require_content,
            DPP.GetEvent.RequireUnderContentSize: require_under_content_size,
            DPP.GetEvent.RequireOverContentSize: require_over_content_size,
            DPP.GetEvent.RequireActions: require_actions,
            DPP.GetEvent.RequireStreams: require_streams,
        }
        for k in list(params_opt):
            if not params_opt[k]:
                params_opt.pop(k)

        resp = self._client.get(
            f"{self._events_url}/api/v2/event/{model}/{endpoint}",
            params=params_opt,
        )

        if resp.status_code != HTTP_200_OK:
            raise DispatcherApiException(
                response=resp,
                ref="Unable to get events from dispatcher",
                internal=str(resp.text),
            )

        filecontent = None  # needs to exist for certain exceptions
        filetype = None  # needs to exist for certain exceptions
        try:
            content_type, options = multipart.parse_options_header(resp.headers["Content-Type"])
            if content_type != "multipart/form-data":
                raise DispatcherApiException(
                    response=resp,
                    ref="not form data",
                    internal=str(resp.text),
                )
            if "boundary" not in options:
                raise DispatcherApiException(
                    response=resp,
                    ref="no form data boundary",
                    internal=str(resp.text),
                )

            boundary = options["boundary"]
            stream = io.BytesIO(resp.content)
            parser = multipart.MultipartParser(stream, boundary)
            respInfo = None
            respEvents = None
            for part in parser:
                filecontent = part.raw
                if part.name == DPP.GET_EVENTS_RESP_INFO:
                    respInfo = azapi.GetEventsInfo.model_validate_json(filecontent)
                elif part.name == DPP.GET_EVENTS_RESP_EVENTS:
                    respEvents = loader(filecontent)

            if respInfo is None:
                raise Exception("no info filepart from dispatcher response")
            if respEvents is None:
                raise Exception(f"no events filepart from dispatcher response (bad loader?):\n{respInfo=}")

        except Exception as e:
            # ensure that bad content is within the exception
            err = BadResponseException(f"invalid response content for {filetype}: {str(e)}")
            err.content = filecontent or resp.content
            raise err from e

        return respInfo, respEvents

    def get_binary_events(self, **kwargs) -> tuple[azapi.GetEventsInfo, list[azm.BinaryEvent]]:
        """Get binary events from dispatcher."""
        return self._get_events(
            **kwargs, model=azm.ModelType.Binary, loader=lambda x: azapi.GetEventsBinary.model_validate_json(x).events
        )

    def get_delete_events(self, **kwargs) -> tuple[azapi.GetEventsInfo, list[azm.DeleteEvent]]:
        """Get delete events from dispatcher."""
        return self._get_events(
            **kwargs, model=azm.ModelType.Delete, loader=lambda x: azapi.GetEventsDelete.model_validate_json(x).events
        )

    def get_download_events(self, **kwargs) -> tuple[azapi.GetEventsInfo, list[azm.DownloadEvent]]:
        """Get download events from dispatcher."""
        return self._get_events(
            **kwargs,
            model=azm.ModelType.Download,
            loader=lambda x: azapi.GetEventsDownload.model_validate_json(x).events,
        )

    def get_insert_events(self, **kwargs) -> tuple[azapi.GetEventsInfo, list[azm.InsertEvent]]:
        """Get insert events from dispatcher."""
        return self._get_events(
            **kwargs, model=azm.ModelType.Insert, loader=lambda x: azapi.GetEventsInsert.model_validate_json(x).events
        )

    def get_plugin_events(self, **kwargs) -> tuple[azapi.GetEventsInfo, list[azm.PluginEvent]]:
        """Get register events from dispatcher."""
        return self._get_events(
            **kwargs,
            model=azm.ModelType.Plugin,
            loader=lambda x: azapi.GetEventsPlugin.model_validate_json(x).events,
        )

    def get_status_events(self, **kwargs) -> tuple[azapi.GetEventsInfo, list[azm.StatusEvent]]:
        """Get status events from dispatcher."""
        return self._get_events(
            **kwargs, model=azm.ModelType.Status, loader=lambda x: azapi.GetEventsStatus.model_validate_json(x).events
        )

    def get_generic_events(self, *, model: str, **kwargs) -> tuple[azapi.GetEventsInfo, list[dict]]:
        """Get status events from dispatcher."""
        return self._get_events(**kwargs, model=model, loader=lambda x: json.loads(x)["events"])

    def submit_events(
        self,
        events: list[azm.BaseEvent],
        *,
        model: str,
        sync: bool = False,
        include_ok: bool = False,
        raise_invalid: bool = True,
        pause_plugins: bool = False,
        params: dict | None = None,
    ) -> azapi.ResponsePostEvent:
        """Submit events to the dispatcher.

        If sync, only return after data is sent to kafka.
        If include_ok, include the events saved to dispatcher after any transformations and enrichment.
        If raise_invalid, when any event is judged invalid by dispatcher, an exception is raised.
        If pause_plugins, prevent all plugins from receiving events for the next 10 minutes.
        Once the pause ends skip over all messages to the current latest.
        """
        if not params:
            params = {}
        # our python client can't handle avro
        params["avro-format"] = False
        if model not in azm.ModelType:
            raise Exception(f"invalid model for submit_events: {model}")
        params["model"] = model
        if sync:
            params["sync"] = sync
        if include_ok:
            params["include_ok"] = include_ok
        if pause_plugins:
            params["pause_plugins"] = pause_plugins

        encoded_events = [x.model_dump_json(exclude_defaults=True, exclude_unset=True).encode() for x in events]
        for event in encoded_events:
            if len(event) > MAX_MESSAGE_SIZE:
                raise NetworkDataException(
                    f"an event to submit to dispatcher was too large: {len(event)}b > {MAX_MESSAGE_SIZE}b"
                )

        data = b"[" + b",".join(encoded_events) + b"]"

        try:
            rsp = self._client.post(
                f"{self._events_url}/api/v2/event",
                content=data,
                headers={"Content-Type": "application/json"},
                params=params,
            )
        except Exception as e:
            raise DispatcherApiException(
                ref="Unable to contact dispatcher",
                internal=str(e),
            ) from e

        if 400 <= rsp.status_code <= 499:
            logger.error(
                "failed to post events, client error %s '%s', raw data as follows:\n%s",
                rsp.status_code,
                rsp.content,
                data,
            )

        if rsp.status_code != HTTP_200_OK:
            raise DispatcherApiException(
                response=rsp,
                ref="Unable to submit event to dispatcher",
                internal=str(rsp.text),
            )
        resp = azapi.ResponsePostEvent.model_validate_json(rsp.content)

        # if we see any invalid events, raise an exception
        if raise_invalid and resp.total_failures > 0:
            logger.error("submitted events were invalid:\n%s", resp)
            raise DispatcherApiException(
                response=rsp,
                ref="submitted events were invalid",
                internal=str(resp),
            )

        return resp

    def simulate_consumers_on_event(self, event: azm.BaseEvent, params: dict | None = None) -> azapi.EventSimulate:
        """Returns information about which consumers would process the provided event."""
        data = event.model_dump_json(exclude_defaults=True, exclude_unset=True).encode()
        try:
            rsp = self._client.post(
                f"{self._events_url}/api/v2/event/simulate",
                content=data,
                headers={"Content-Type": "application/json"},
                params=params,
            )
        except Exception as e:
            raise DispatcherApiException(
                ref="Unable to contact dispatcher",
                internal=str(e),
            ) from e

        if 400 <= rsp.status_code <= 499:
            logger.error(
                "failed to post events, client error %s '%s', raw data as follows:\n%s",
                rsp.status_code,
                rsp.content,
                data,
            )

        if rsp.status_code != HTTP_200_OK:
            raise DispatcherApiException(
                response=rsp,
                ref="Unable to submit event to dispatcher",
                internal=str(rsp.text),
            )
        return azapi.EventSimulate.model_validate_json(rsp.content)

    def has_binary(self, source: str, label: azm.DataLabel, sha256: str):
        """Attempt to check if dispatcher has the binary file available.

        Raises an exception if the binary is not available.

        :param sha256: sha256 of file
        """
        try:
            rsp = self._client.head(f"{self._data_url}/api/v3/stream/{source}/{label}/{sha256.lower()}")
        except Exception as e:
            # remote disconnects without response if offset is out of range
            raise DispatcherApiException(ref="Unable to request file", internal=str(e)) from e

        if rsp.status_code == HTTP_404_NOT_FOUND:
            raise DispatcherApiException(response=rsp, ref="Binary content not found", internal=rsp.text)
        if rsp.status_code != HTTP_200_OK and rsp.status_code != HTTP_206_PARTIAL_CONTENT:
            raise DispatcherApiException(
                response=rsp,
                ref=f"Unable to request file bad dispatcher status code {rsp.status_code}",
                internal=rsp.text,
            )

    def get_binary(
        self, source: str, label: azm.DataLabel, sha256: str, start_pos: int | None = None, end_pos: int | None = None
    ) -> httpx.Response:
        """Retrieve binary from dispatcher with range arguments.

        Documentation for range is as follows https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Range

        Valid values are:
        10- # get all bytes after the first 10bytes
        -100 # get last 100 bytes
        100-200 # get all bytes in the range 100 to 200.

        :param sha256: sha256 of file
        :param start_pos: offset in bytes from start of file to retrieve
        :param end_pos: bytes to take up to, if None uses EOF
        """
        req_header = {}
        # take range
        if start_pos is None and end_pos is None:
            # Don't add a range header if there is no start or end range provided.
            pass
        elif start_pos is not None or end_pos is not None:
            # If the start or end is not None a range can be created.
            if start_pos is None:
                start_pos = ""
            if end_pos is None:
                end_pos = ""
            req_header = {"range": f"bytes={start_pos}-{end_pos}"}

        try:
            rsp = self._client.get(
                f"{self._data_url}/api/v3/stream/{source}/{label}/{sha256.lower()}", headers=req_header
            )
        except Exception as e:
            # remote disconnects without response if offset is out of range
            raise DispatcherApiException(
                ref="Unable to request file, is offset too large?",
                internal=str(e),
            ) from e

        if rsp.status_code == HTTP_404_NOT_FOUND:
            raise DispatcherApiException(response=rsp, ref="Binary content not found", internal=rsp.text)

        if rsp.status_code != HTTP_200_OK and rsp.status_code != HTTP_206_PARTIAL_CONTENT:
            raise DispatcherApiException(
                response=rsp,
                ref="Unable to request file",
                internal=rsp.text,
            )

        return rsp

    async def _get_error_message(self, rsp: httpx.Response) -> str:
        """Get error message for an async request."""
        MAX_ERROR_MESSAGE_LENGTH = 2000
        data = ""
        async for c in rsp.aiter_text():
            data += c
            if len(data) > MAX_ERROR_MESSAGE_LENGTH:
                return data
        return data

    async def async_get_binary(
        self, source: str, label: azm.DataLabel, sha256: str, start_pos: int | None = None, end_pos: int | None = None
    ) -> AsyncIterable[bytes]:
        """Retrieve binary from dispatcher with range arguments.

        Documentation for range is as follows https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Range

        Valid values are:
        10- # get all bytes after the first 10bytes
        -100 # get last 100 bytes
        100-200 # get all bytes in the range 100 to 200.

        :param sha256: sha256 of file
        :param start_pos: offset in bytes from start of file to retrieve
        :param end_pos: bytes to take up to, if None uses EOF
        """
        req_header = {}
        # take range
        if start_pos is None and end_pos is None:
            # Don't add a range header if there is no start or end range provided.
            pass
        elif start_pos is not None or end_pos is not None:
            # If the start or end is not None a range can be created.
            if start_pos is None:
                start_pos = ""
            if end_pos is None:
                end_pos = ""
            req_header = {"range": f"bytes={start_pos}-{end_pos}"}

        req = self._async_client.build_request(
            "GET", f"{self._data_url}/api/v3/stream/{source}/{label}/{sha256.lower()}", headers=req_header
        )
        rsp = await self._async_client.send(req, stream=True)
        # Handle any potential status code errors immediately.
        try:
            if rsp.status_code == HTTP_404_NOT_FOUND:
                logger.warning(f"Failed to download file due to 404 with {source=}, {label=}, {sha256=}")
                raise DispatcherApiException(
                    response=rsp, ref="Binary content not found", internal=await self._get_error_message(rsp)
                )
            elif rsp.status_code != HTTP_200_OK and rsp.status_code != HTTP_206_PARTIAL_CONTENT:
                logger.warning(
                    "Failed to download file with statsu code to "
                    + f"{rsp.status_code} with {source=}, {label=}, {sha256=}"
                )
                raise DispatcherApiException(
                    response=rsp,
                    ref="Unable to request file",
                    internal=await self._get_error_message(rsp),
                )
        except Exception:
            # Close the response body if there is an exception.
            # NOTE - a memory leak could occur if aclose is never called.
            await rsp.aclose()
            raise

        async def content_generator() -> AsyncIterable[bytes]:
            """Coroutine to yield the contents as the status code is 200 or 206.

            Done this way to allow handling of status code exceptions.
            """
            try:
                async for cur_bytes in rsp.aiter_bytes():
                    yield cur_bytes
            except Exception as e:
                # remote disconnects without response if offset is out of range
                raise DispatcherApiException(
                    ref="Unable to request file, is offset too large?",
                    internal=str(e),
                ) from e
            finally:
                # Ensure body of response closes.
                # NOTE - a memory leak could occur if aclose is never called.
                await rsp.aclose()

        return content_generator()

    @staticmethod
    async def async_convert_to_async_iterable(
        data: bytes | typing.BinaryIO | UploadFile | AsyncIterable,
    ) -> typing.AsyncIterable:
        """Buffer and yield binary data in a sensible fashion. Data must be seeked to zero."""
        if isinstance(data, bytes):
            yield data
        elif isinstance(data, io.IOBase):
            while next_data := data.read(MAX_BUFFER_BYTES):
                yield next_data
        elif issubclass(type(data), UploadFile):
            while next_data := await data.read(MAX_BUFFER_BYTES):
                yield next_data
        elif isinstance(data, AsyncIterable):
            async for b in data:
                yield b
        else:
            raise ValueError(f"Bad type for _yield_data, Unexpected type when submitting binary {type(data)}")

    async def async_submit_binary(
        self,
        source: str,
        label: azm.DataLabel,
        data: bytes | typing.BinaryIO | UploadFile | AsyncIterable,
        timeout: float | None = None,
        *,
        skip_identify: bool = False,
        expected_sha256: str = "",
    ) -> azm.Datastream:
        """Asynchronously submit the binary data file to the dispatcher and return metadata."""
        # verify data is actually sendable
        if isinstance(data, bytes):
            pass
        elif isinstance(data, io.IOBase):
            if isinstance(data, io.TextIOBase):
                raise ValueError(
                    f"Unexpected string buffer when submitting binary {type(data)}"
                    + ", valid types are binary io.IOBase, UploadFile, AsyncIterable classes and bytes"
                )
            data.seek(0)
        elif issubclass(type(data), UploadFile):
            await data.seek(0)
        elif isinstance(data, AsyncIterable):
            pass
        else:
            raise ValueError(
                f"Unexpected type when submitting binary {type(data)}"
                + ", valid types are binary io.IOBase, UploadFile, AsyncIterable classes and bytes"
            )

        # prepare query params
        params_opt = {DPP.PostStream.SkipIdentify: skip_identify, DPP.PostStream.ExpectedSha256: expected_sha256}
        for k in list(params_opt):
            if not params_opt[k]:
                params_opt.pop(k)

        # perform request
        extra_kwargs = dict()
        if timeout:
            extra_kwargs["timeout"] = timeout

        try:
            rsp = await self._async_client.post(
                f"{self._data_url}/api/v3/stream/{source}/{label}",
                content=self.async_convert_to_async_iterable(data),
                params=params_opt,
                **extra_kwargs,
            )
        except Exception as e:
            raise DispatcherApiException(ref="Unable to contact dispatcher", internal=str(e)) from e

        if rsp.status_code != HTTP_200_OK:
            raise DispatcherApiException(
                response=rsp,
                ref="Unable to submit file",
                internal=rsp.text,
            )

        try:
            bin_info = rsp.json()["data"]
        except KeyError as e:
            raise DispatcherApiException(
                ref="Unable to submit file",
                internal="Error submitting file to DISPATCHER service",
            ) from e

        return azm.Datastream(**bin_info)

    def submit_binary(
        self,
        source: str,
        label: azm.DataLabel,
        data: bytes | typing.BinaryIO,
        timeout: float | None = None,
        *,
        skip_identify: bool = False,
        expected_sha256: str = "",
    ) -> azm.Datastream:
        """Submit the binary data file to the dispatcher and return metadata."""
        if isinstance(data, bytes):
            pass
        elif isinstance(data, io.IOBase):
            if isinstance(data, io.TextIOBase):
                raise ValueError(
                    f"Unexpected string buffer when submitting binary {type(data)}"
                    + ", valid types are binary io.IOBase classes and bytes"
                )
            data.seek(0)
        else:
            raise ValueError(
                f"Unexpected type when submitting binary {type(data)}"
                + ", valid types are binary io.IOBase classes and bytes"
            )

        # prepare query params
        params_opt = {DPP.PostStream.SkipIdentify: skip_identify, DPP.PostStream.ExpectedSha256: expected_sha256}
        for k in list(params_opt):
            if not params_opt[k]:
                params_opt.pop(k)

        # perform request
        extra_kwargs = dict()
        if timeout:
            extra_kwargs["timeout"] = timeout

        try:
            rsp = self._client.post(
                f"{self._data_url}/api/v3/stream/{source}/{label}",
                content=data,
                params=params_opt,
                **extra_kwargs,
            )
        except Exception as e:
            raise DispatcherApiException(ref="Unable to contact dispatcher", internal=str(e)) from e
        if rsp.status_code != HTTP_200_OK:
            raise DispatcherApiException(
                response=rsp,
                ref="Unable to submit file",
                internal=rsp.text,
            )
        try:
            bin_info = rsp.json()["data"]
        except KeyError as e:
            raise DispatcherApiException(
                ref="Unable to submit file",
                internal="Error submitting file to DISPATCHER service",
            ) from e

        return azm.Datastream(**bin_info)

    def copy_binary(self, sourceA: str, sourceB: str, label: azm.DataLabel, sha256: str):
        """Copies a binary from once source to another."""
        rsp = self._client.patch(
            f"{self._data_url}/api/v3/stream/{sourceA}/{sourceB}/{label}/{sha256}",
        )

        if rsp.status_code != HTTP_200_OK:
            raise DispatcherApiException(
                response=rsp,
                ref="Unable to copy file",
                internal=rsp.text,
            )

    def delete_binary(
        self, source: str, label: azm.DataLabel, sha256: str, ifOlderThan: DateTime
    ) -> tuple[bool, bool]:
        """Delete binary from the dispatcher if older than a certain time.

        Return (wasFound, wasDeleted).
        """
        # avoid timezone issues from client side
        ifOlderThan = ifOlderThan.in_tz("UTC")
        rsp = self._client.delete(
            f"{self._data_url}/api/v3/stream/{source}/{label}/{sha256.lower()}",
            params={"ifOlderThan": int(ifOlderThan.timestamp())},
        )
        if rsp.status_code == HTTP_200_OK:
            logger.debug(
                f"binary {sha256} was found in dispatcher, {ifOlderThan=} and deleted: {rsp.json()['deleted']}"
            )
            return True, rsp.json()["deleted"]
        elif rsp.status_code == HTTP_404_NOT_FOUND:
            logger.debug(f"binary {sha256} was not found in dispatcher")
            return False, False
        else:
            msg = f"unknown dispatcher response {rsp.status_code}: {rsp.content}"
            raise DispatcherApiException(response=rsp, ref=msg, internal=msg)
