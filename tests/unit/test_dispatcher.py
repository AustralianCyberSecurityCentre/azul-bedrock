"""Test that Dispatcher API returns and raises appropriate exceptions/responses."""

import datetime
import io
import json
import time
import unittest
from typing import AsyncIterable, ClassVar

import httpx
import pendulum
from fastapi import UploadFile

from azul_bedrock import mock as md
from azul_bedrock import models_api as azapi
from azul_bedrock import models_network as azm
from azul_bedrock.dispatcher import DispatcherAPI
from azul_bedrock.exceptions import DispatcherApiException, NetworkDataException
from azul_bedrock.mock import streams

an_event = azm.BinaryEvent(
    model_version=azm.CURRENT_MODEL_VERSION,
    kafka_key="abc",
    dequeued="thing",
    action=azm.BinaryAction.Sourced,
    flags={},
    timestamp=datetime.datetime(year=2014, month=9, day=21, tzinfo=datetime.timezone.utc),
    author=azm.Author(name="some ingest process", category="automatic_input"),
    entity=azm.BinaryEvent.Entity(sha256="12345"),
    source=azm.Source(
        name="",
        path=[],
        timestamp=datetime.datetime(year=2014, month=9, day=21, tzinfo=datetime.timezone.utc),
    ),
)
an_event_cooked = an_event.model_dump()


class TestDispatcherApi(unittest.IsolatedAsyncioTestCase):
    """Mandatory."""

    mock_server: ClassVar[md.MockDispatcher]
    server: ClassVar[str]  # Endpoint to the mock server in the form 'http://host:port'
    maxDiff = None

    @classmethod
    def setUpClass(cls) -> None:
        cls.mock_server = md.MockDispatcher()
        cls.mock_server.start()
        while not cls.mock_server.is_alive():
            time.sleep(0.2)  # Wait for server to start
        cls.server = "http://%s:%s" % (cls.mock_server.host, cls.mock_server.port)
        cls.editor = md.Editor(cls.server)

        # Wait for server to be ready to respond
        tries = 0
        while True:
            time.sleep(0.2)
            tries += 1
            try:
                _ = httpx.get(cls.server + "/info")
                break  # Exit loop if successful
            except httpx.TimeoutException:
                if tries > 20:  # Time out after about 4 seconds
                    raise RuntimeError("Timed out waiting for mock server to be ready")
            except httpx.ConnectError:
                if tries > 20:  # Time out after about 4 seconds
                    raise RuntimeError("Connection error waiting for mock server to be ready")

    async def asyncSetUp(self):
        self.dp = DispatcherAPI(
            events_url=self.server,
            data_url=self.server,
            retry_count=2,
            timeout=2,
            author_name="plugin1",
            author_version="1.0",
            deployment_key="plugin1key",
        )

    @classmethod
    def tearDownClass(cls) -> None:
        cls.mock_server.stop()
        cls.mock_server.kill()

    def test_parse_range(self):
        _pr = streams._parse_range
        self.assertEqual(_pr("bytes=50-100"), (50, 100))
        self.assertEqual(_pr("bytes=999-"), (999, None))
        self.assertEqual(_pr("bytes=-500"), (-500, None))
        self.assertRaisesRegex(RuntimeError, "Could not parse request", _pr, "50-100")
        self.assertRaisesRegex(RuntimeError, "Could not parse request", _pr, "bytes=jabberwocky")
        self.assertRaisesRegex(RuntimeError, "Could not parse request", _pr, "bytes=hyphenated-string")
        self.assertRaisesRegex(RuntimeError, "start and end are both empty", _pr, "bytes=-")

    def test_get_binary_events(self):
        generic_events = {"events": []}
        # Test main implementation
        self.editor.set_response(200, generic_events)
        respInfo, respEvents = self.dp.get_binary_events(is_task=True)
        info = self.editor.get_last_request()
        self.assertEqual(info.body, b"")
        self.assertEqual(
            info.url,
            self.server
            + "/api/v2/event/binary/active?name=plugin1&version=1.0&count=1&deadline=1&deployment_key=plugin1key",
        )
        self.assertEqual(info.headers["user-agent"], self.dp.user_agent)
        self.assertEqual(info.headers["user-agent"], "dpclient-py-plugin1")
        # check response
        self.assertEqual(respInfo.fetched, 0)
        self.assertEqual(len(respEvents), 0)

        generic_events = {"events": [an_event_cooked, an_event_cooked, an_event_cooked]}
        # Test main implementation
        self.editor.set_response(200, generic_events)
        respInfo, respEvents = self.dp.get_binary_events(is_task=True)
        info = self.editor.get_last_request()
        self.assertEqual(info.body, b"")
        self.assertEqual(
            info.url,
            self.server
            + "/api/v2/event/binary/active?name=plugin1&version=1.0&count=1&deadline=1&deployment_key=plugin1key",
        )
        self.assertEqual(info.headers["user-agent"], self.dp.user_agent)
        self.assertEqual(info.headers["user-agent"], "dpclient-py-plugin1")
        # check response
        self.assertEqual(respInfo.fetched, 3)
        self.assertEqual(len(respEvents), 3)

        self.editor.set_response(200, generic_events)
        respInfo, respEvents = self.dp.get_binary_events(is_task=False)
        info = self.editor.get_last_request()
        self.assertEqual(info.body, b"")
        self.assertEqual(
            info.url,
            self.server
            + "/api/v2/event/binary/passive?name=plugin1&version=1.0&count=1&deadline=1&deployment_key=plugin1key",
        )
        self.assertEqual(info.headers["user-agent"], self.dp.user_agent)
        self.assertEqual(info.headers["user-agent"], "dpclient-py-plugin1")

        # deny historic
        self.editor.set_response(200, generic_events)
        respInfo, respEvents = self.dp.get_binary_events(
            require_expedite=True,
            require_live=True,
            is_task=True,
            require_under_content_size=100,
            require_over_content_size=50,
            require_actions=["extracted"],
            deny_actions=["binary_enhanced"],
            deny_self=True,
            require_streams=["content,application/windows/exe23", "*,application/blah"],
        )
        info = self.editor.get_last_request()
        self.assertEqual(info.body, b"")
        self.assertEqual(
            info.url,
            self.server
            + "/api/v2/event/binary/active?name=plugin1&version=1.0&count=1&deadline=1&deployment_key=plugin1key&d-action=binary_enhanced&d-self=true&r-expedite=true&r-live=true&r-under-content-size=100&r-over-content-size=50&r-action=extracted&r-streams=content%2Capplication%2Fwindows%2Fexe23&r-streams=%2A%2Capplication%2Fblah",
            info.url,
        )

        # require historic
        self.editor.set_response(200, generic_events)
        respInfo, respEvents = self.dp.get_binary_events(
            require_historic=True,
            is_task=False,
        )
        info = self.editor.get_last_request()
        self.assertEqual(info.body, b"")
        self.assertEqual(
            info.url,
            self.server
            + "/api/v2/event/binary/passive?name=plugin1&version=1.0&count=1&deadline=1&deployment_key=plugin1key&r-historic=true",
            info.url,
        )

        # Test passive
        self.editor.set_response(200, generic_events)
        respInfo, respEvents = self.dp.get_binary_events(is_task=False, count=100)
        info = self.editor.get_last_request()
        self.assertEqual(
            info.url,
            self.server
            + "/api/v2/event/binary/passive?name=plugin1&version=1.0&count=100&deadline=1&deployment_key=plugin1key",
        )

        # Test active
        self.editor.set_response(200, generic_events)
        respInfo, respEvents = self.dp.get_binary_events(is_task=True)
        info = self.editor.get_last_request()
        self.assertEqual(
            info.url,
            self.server
            + "/api/v2/event/binary/active?name=plugin1&version=1.0&count=1&deadline=1&deployment_key=plugin1key",
        )

        # test failures
        self.editor.set_response(500, {})
        with self.assertRaises(DispatcherApiException) as ex:
            self.dp.get_binary_events(is_task=True)
        info = self.editor.get_last_request()
        self.assertEqual(ex.exception.status_code, 500)
        self.assertEqual(
            info.url,
            self.server
            + "/api/v2/event/binary/active?name=plugin1&version=1.0&count=1&deadline=1&deployment_key=plugin1key",
        )

        self.editor.set_response(400, {})
        with self.assertRaises(DispatcherApiException) as ex:
            self.dp.get_binary_events(is_task=True)
        info = self.editor.get_last_request()
        self.assertEqual(ex.exception.status_code, 400)
        self.assertEqual(
            info.url,
            self.server
            + "/api/v2/event/binary/active?name=plugin1&version=1.0&count=1&deadline=1&deployment_key=plugin1key",
        )

    def test_get_delete_events(self):
        generic_events = {
            "events": [
                azm.DeleteEvent(
                    model_version=azm.CURRENT_MODEL_VERSION,
                    kafka_key="tmp",
                    author=azm.Author(
                        name="thing",
                        category="user",
                        security=None,
                    ),
                    timestamp=pendulum.now(pendulum.UTC).to_iso8601_string(),
                    entity=azm.DeleteEvent.DeleteEntity(reason="thingo"),
                    action=azm.DeleteAction.author,
                ).model_dump()
            ]
        }
        self.editor.set_response(200, generic_events)
        respInfo, respEvents = self.dp.get_delete_events(is_task=False)
        info = self.editor.get_last_request()
        self.assertEqual(
            info.url,
            self.server
            + "/api/v2/event/delete/passive?name=plugin1&version=1.0&count=1&deadline=1&deployment_key=plugin1key",
        )
        # check response
        self.assertEqual(respInfo.fetched, 1)
        self.assertEqual(len(respEvents), 1)

    def test_get_status_events(self):
        generic_events = {
            "events": [
                azm.StatusEvent(
                    model_version=azm.CURRENT_MODEL_VERSION,
                    kafka_key="tmp",
                    author=azm.Author(
                        name="thing",
                        category="user",
                        security=None,
                    ),
                    timestamp=pendulum.now(pendulum.UTC).to_iso8601_string(),
                    entity=azm.StatusEvent.Entity(status="heartbeat", runtime=10, input=an_event),
                ).model_dump()
            ]
        }
        self.editor.set_response(200, generic_events)
        respInfo, respEvents = self.dp.get_status_events(is_task=False)
        info = self.editor.get_last_request()
        self.assertEqual(
            info.url,
            self.server
            + "/api/v2/event/status/passive?name=plugin1&version=1.0&count=1&deadline=1&deployment_key=plugin1key",
        )
        # check response
        self.assertEqual(respInfo.fetched, 1)
        self.assertEqual(len(respEvents), 1)

    def test_get_plugin_events(self):
        generic_events = {
            "events": [
                azm.PluginEvent(
                    model_version=azm.CURRENT_MODEL_VERSION,
                    kafka_key="tmp",
                    author=azm.Author(
                        name="thing",
                        category="user",
                        security=None,
                    ),
                    timestamp=pendulum.now(pendulum.UTC).to_iso8601_string(),
                    entity=azm.PluginEvent.Entity(
                        category="plugin",
                        name="generic_plugin",
                        version="2021-01-01T12:00:00+00:00",
                        contact="generic_contact",
                        description="generic_description",
                        features=[
                            dict(
                                name="generic_feature",
                                desc="generic_description",
                                type="string",
                            )
                        ],
                        security="LOW TLP:CLEAR",
                        config={},
                    ),
                ).model_dump()
            ]
        }
        self.editor.set_response(200, generic_events)
        respInfo, respEvents = self.dp.get_plugin_events(is_task=False)
        info = self.editor.get_last_request()
        self.assertEqual(
            info.url,
            self.server
            + "/api/v2/event/plugin/passive?name=plugin1&version=1.0&count=1&deadline=1&deployment_key=plugin1key",
        )
        # check response
        self.assertEqual(respInfo.fetched, 1)
        self.assertEqual(len(respEvents), 1)

    def test_get_generic_events(self):
        generic_events = {"events": [{"favourite_fruit": "mandarin"}]}
        self.editor.set_response(200, generic_events)
        respInfo, respEvents = self.dp.get_generic_events(is_task=False, model=azm.ModelType.Retrohunt)
        info = self.editor.get_last_request()
        self.assertEqual(
            info.url,
            self.server
            + "/api/v2/event/retrohunt/passive?name=plugin1&version=1.0&count=1&deadline=1&deployment_key=plugin1key",
        )
        # check response
        self.assertEqual(respInfo.fetched, 1)
        self.assertEqual(len(respEvents), 1)

    def test_submit_events(self):
        """Mandatory."""
        # Check it runs without issue
        self.editor.set_response(200, {"total_ok": 1, "total_failures": 0, "failures": []})
        resp = self.dp.submit_events([], model=azm.ModelType.Binary)
        self.assertEqual(resp.total_ok, 1)
        self.assertEqual(resp.total_failures, 0)

        # Check simple error from Dispatcher works
        self.editor.set_response(
            500, azapi.DispatcherApiErrorModel(detail="Dispatcher sourced error message").model_dump_json()
        )
        with self.assertRaises(DispatcherApiException) as ex:
            self.dp.submit_events([], model=azm.ModelType.Binary)
        self.assertEqual(ex.exception.detail.get("external"), "Dispatcher sourced error message")

        self.editor.set_response(
            500,
            azapi.DispatcherApiErrorModel(title="title", detail="Dispatcher sourced error message").model_dump_json(),
        )
        with self.assertRaises(DispatcherApiException) as ex:
            self.dp.submit_events([], model=azm.ModelType.Binary)
        self.assertEqual(ex.exception.detail.get("external"), "title: Dispatcher sourced error message")

        tmp = "a" * (1024 * 1024 * 2 + 512)
        too_large_event = azm.BinaryEvent(
            model_version=azm.CURRENT_MODEL_VERSION,
            kafka_key="abc",
            dequeued=tmp,
            action=azm.BinaryAction.Sourced,
            flags={},
            timestamp=datetime.datetime(year=2014, month=9, day=21, tzinfo=datetime.timezone.utc),
            author=azm.Author(name="some ingest process", category="automatic_input"),
            entity=azm.BinaryEvent.Entity(sha256="12345"),
            source=azm.Source(
                name="",
                path=[],
                timestamp=datetime.datetime(year=2014, month=9, day=21, tzinfo=datetime.timezone.utc),
            ),
        )
        with self.assertRaises(NetworkDataException) as ex:
            self.dp.submit_events([too_large_event], model=azm.ModelType.Binary)

        def check_exception(expected_status):
            self.editor.set_response(expected_status, {})
            with self.assertRaises(DispatcherApiException) as ex:
                self.dp.submit_events([], model=azm.ModelType.Binary)
            self.assertEqual(ex.exception.status_code, expected_status)

        check_exception(500)
        check_exception(308)
        check_exception(404)

        # several events at once, over size limit of single event
        self.editor.set_response(200, {"total_ok": 1, "total_failures": 0, "failures": []})
        tmp = "a" * (1024 * 1024 * 2 - 1024)
        large_event1 = azm.BinaryEvent(
            model_version=azm.CURRENT_MODEL_VERSION,
            kafka_key="abc",
            dequeued=tmp,
            action=azm.BinaryAction.Sourced,
            flags={},
            timestamp=datetime.datetime(year=2014, month=9, day=21, tzinfo=datetime.timezone.utc),
            author=azm.Author(name="some ingest process", category="automatic_input"),
            entity=azm.BinaryEvent.Entity(sha256="12345"),
            source=azm.Source(
                name="",
                path=[],
                timestamp=datetime.datetime(year=2014, month=9, day=21, tzinfo=datetime.timezone.utc),
            ),
        )
        large_event2 = azm.BinaryEvent(
            model_version=azm.CURRENT_MODEL_VERSION,
            kafka_key="abcd",
            dequeued=tmp,
            action=azm.BinaryAction.Sourced,
            flags={},
            timestamp=datetime.datetime(year=2014, month=9, day=21, tzinfo=datetime.timezone.utc),
            author=azm.Author(name="some ingest process", category="automatic_input"),
            entity=azm.BinaryEvent.Entity(sha256="12345"),
            source=azm.Source(
                name="",
                path=[],
                timestamp=datetime.datetime(year=2014, month=9, day=21, tzinfo=datetime.timezone.utc),
            ),
        )
        self.dp.submit_events([large_event1, large_event2], model=azm.ModelType.Binary)
        info = self.editor.get_last_request()
        self.assertGreater(len(info.body), 4110000)

        # test invalid handling
        self.editor.set_response(
            200,
            {"total_ok": 1, "total_failures": 1, "failures": [{"error": "this errored", "event": "{}"}]},
        )
        # should raise exception
        with self.assertRaises(DispatcherApiException):
            self.dp.submit_events([], model=azm.ModelType.Binary)
        # shouldn't raise exception
        self.editor.set_response(
            200,
            {"total_ok": 1, "total_failures": 1, "failures": [{"error": "this errored", "event": "{}"}]},
        )
        resp = self.dp.submit_events([], model=azm.ModelType.Binary, raise_invalid=False)
        self.assertEqual(resp.total_failures, 1)

        # nonstandard event type
        self.editor.set_response(200, {"total_ok": 1, "total_failures": 0, "failures": []})

        class CustomEvent(azm.BaseEvent):
            class CustomEntity(azm.BaseModelStrict):
                ipsum: int

            entity: CustomEntity = None

        ev = CustomEvent(
            model_version=azm.CURRENT_MODEL_VERSION,
            entity=CustomEvent.CustomEntity(ipsum=5),
            kafka_key="",
            timestamp=datetime.datetime(year=2014, month=9, day=21, tzinfo=datetime.timezone.utc),
            author=azm.Author(name="some ingest process", category="automatic_input"),
        )
        self.dp.submit_events([ev], model=azm.ModelType.Retrohunt)

    def test_simulate_consumers_on_event(self):
        event1 = azm.BinaryEvent(
            model_version=azm.CURRENT_MODEL_VERSION,
            kafka_key="abc",
            dequeued="tmp",
            action=azm.BinaryAction.Sourced,
            flags={},
            timestamp=datetime.datetime(year=2014, month=9, day=21, tzinfo=datetime.timezone.utc),
            author=azm.Author(name="some ingest process", category="automatic_input"),
            entity=azm.BinaryEvent.Entity(sha256="12345"),
            source=azm.Source(
                name="",
                path=[],
                timestamp=datetime.datetime(year=2014, month=9, day=21, tzinfo=datetime.timezone.utc),
            ),
        )

        # Check it runs without issue
        self.editor.set_response(200, {})
        self.dp.simulate_consumers_on_event(event1)

        def check_exception(expected_status):
            self.editor.set_response(expected_status, {})
            with self.assertRaises(DispatcherApiException) as ex:
                self.dp.simulate_consumers_on_event(event1)
            self.assertEqual(ex.exception.status_code, expected_status)

        check_exception(500)
        check_exception(308)
        check_exception(404)

        # several events at once, over size limit of single event
        self.editor.set_response(
            200,
            {
                "consumers": [
                    {
                        "name": "magrog",
                        "version": "1",
                        "filter_out": True,
                        "filter_out_trigger": "MyPipeline-not_suitable",
                    }
                ]
            },
        )
        simulation = self.dp.simulate_consumers_on_event(event1)
        print(simulation)
        self.assertEqual(len(simulation.consumers), 1)
        self.assertEqual(
            simulation,
            azapi.EventSimulate(
                consumers=[
                    azapi.EventSimulateConsumer(
                        name="magrog", version="1", filter_out=True, filter_out_trigger="MyPipeline-not_suitable"
                    )
                ]
            ),
        )

    def test_has_binary(self):
        """Mandatory."""
        # Check run with no issue
        self.editor.set_response(200, {})
        self.dp.has_binary("source", azm.DataLabel.TEST, "not-real-sha256")

        self.editor.set_response(206, {})
        self.dp.has_binary("source", azm.DataLabel.TEST, "not-real-sha256")

        def check_exception(expected_status):
            self.editor.set_response(expected_status, {})
            with self.assertRaises(DispatcherApiException) as ex:
                self.dp.has_binary("source", azm.DataLabel.TEST, "not-real-sha256")
            self.assertEqual(ex.exception.status_code, expected_status)

        check_exception(500)
        check_exception(308)
        check_exception(404)

    def test_get_binary(self):
        """Mandatory."""
        # Ok
        self.editor.set_stream("not-real-sha256", 200, b"abcdef")
        resp = self.dp.get_binary("source", azm.DataLabel.TEST, "not-real-sha256")
        self.assertEqual(resp.status_code, 200)

        # Bad
        def check_exception(expected_status):
            self.editor.set_stream("not-real-sha256", expected_status, b"abcdef")
            with self.assertRaises(DispatcherApiException) as ex:
                self.dp.get_binary("source", azm.DataLabel.TEST, "not-real-sha256")
            self.assertEqual(ex.exception.status_code, expected_status)

        check_exception(500)
        check_exception(308)
        check_exception(404)

    async def get_async_iterable_content(self, asyncIterBytes: AsyncIterable[bytes]) -> bytes:
        data = b""
        async for cur_byte in asyncIterBytes:
            data += cur_byte
        return data

    async def test_async_get_binary(self):
        """Mandatory."""
        # Ok
        self.editor.set_stream("not-real-sha256", 200, b"abcdef")
        asyncIterBytes = await self.dp.async_get_binary("source", azm.DataLabel.TEST, "not-real-sha256")
        self.assertEqual(await self.get_async_iterable_content(asyncIterBytes), b"abcdef")

        self.editor.set_stream("not-real-sha256", 200, b"206abcdef")
        asyncIterBytes = await self.dp.async_get_binary("source", azm.DataLabel.TEST, "not-real-sha256")
        self.assertEqual(await self.get_async_iterable_content(asyncIterBytes), b"206abcdef")

        # Error cases
        async def check_exception(expected_status):
            self.editor.set_stream("not-real-sha256", expected_status, b"")
            with self.assertRaises(DispatcherApiException) as ex:
                asyncIterBytes = await self.dp.async_get_binary("source", azm.DataLabel.TEST, "not-real-sha256")
                await self.get_async_iterable_content(asyncIterBytes)
            self.assertEqual(ex.exception.status_code, expected_status)

        await check_exception(500)
        await check_exception(308)
        await check_exception(404)

    async def test_async_get_binary_verify_range(self):
        """Setup side affects that verify the headers are expected values."""
        start_val = 10
        end_val = 200

        # Verify the range works with various options set.
        self.editor.set_stream("not-real-sha256", 200, b"abcdef" * 40)
        asyncIterBytes = await self.dp.async_get_binary("source", azm.DataLabel.TEST, "not-real-sha256")
        await self.get_async_iterable_content(asyncIterBytes)

        last_req = self.editor.get_last_request()
        self.assertEqual(last_req.range_raw, "")
        self.assertEqual(last_req.headers["user-agent"], self.dp.user_agent)

        asyncIterBytes = await self.dp.async_get_binary(
            "source", azm.DataLabel.TEST, "not-real-sha256", end_pos=end_val
        )
        await self.get_async_iterable_content(asyncIterBytes)
        last_req = self.editor.get_last_request()
        self.assertEqual(last_req.range_raw, f"bytes=-{end_val}")

        asyncIterBytes = await self.dp.async_get_binary(
            "source", azm.DataLabel.TEST, "not-real-sha256", start_pos=start_val
        )
        await self.get_async_iterable_content(asyncIterBytes)
        last_req = self.editor.get_last_request()
        self.assertEqual(last_req.range_raw, f"bytes={start_val}-")

        asyncIterBytes = await self.dp.async_get_binary(
            "source", azm.DataLabel.TEST, "not-real-sha256", start_pos=start_val, end_pos=end_val
        )
        await self.get_async_iterable_content(asyncIterBytes)
        last_req = self.editor.get_last_request()
        self.assertEqual(last_req.range_raw, f"bytes={start_val}-{end_val}")

    def test_submit_binary(self):
        """Mandatory."""
        # Test for successfully downloading file info to data.
        finfo = azm.Datastream(
            label=azm.DataLabel.TEXT,
            size=10,
            sha512="a",
            sha256="b",
            md5="c",
            sha1="d",
            file_format_legacy="f",
            magic="g",
            mime="h",
        )
        self.editor.set_response(200, {"data": finfo.model_dump()})
        val = self.dp.submit_binary("source", azm.DataLabel.TEST, b"hello")
        info = self.editor.get_last_request()
        self.assertEqual(val, finfo)
        self.assertEqual(info.body, b"hello")

        # Submit a io reader
        self.editor.set_response(200, {"data": finfo.model_dump()})
        val = self.dp.submit_binary("source", azm.DataLabel.TEST, io.BytesIO(b"hello"))
        info = self.editor.get_last_request()
        self.assertEqual(val, finfo)
        self.assertEqual(info.body, b"hello")
        # Submit a using a buffered random (like azul-runner) + extra timeout
        self.editor.set_response(200, {"data": finfo.model_dump()})
        val = self.dp.submit_binary("source", azm.DataLabel.TEST, io.BufferedRandom(io.BytesIO(b"hello2")), timeout=20)
        info = self.editor.get_last_request()
        self.assertEqual(val, finfo)
        self.assertEqual(info.body, b"hello2")

        # Submit a io reader partial seek ( will result in content too-long error if not set back to 0)
        self.editor.set_response(200, {"data": finfo.model_dump()})
        seeked = io.BytesIO(b"hello")
        seeked.read(2)
        val = self.dp.submit_binary("source", azm.DataLabel.TEST, seeked)
        info = self.editor.get_last_request()
        self.assertEqual(val, finfo)
        self.assertEqual(info.body, b"hello")

        # Bad key to access data.
        self.editor.set_response(200, {"bad-key": finfo.model_dump()})
        with self.assertRaises(DispatcherApiException) as ex:
            self.dp.submit_binary("source", azm.DataLabel.TEST, b"hello")
        info = self.editor.get_last_request()
        self.assertEqual(info.body, b"hello")

        # Assert various exception cases.
        self.editor.set_response(500, {})
        with self.assertRaises(DispatcherApiException) as ex:
            self.dp.submit_binary("source", azm.DataLabel.TEST, b"hello")
        info = self.editor.get_last_request()
        self.assertEqual(ex.exception.status_code, 500)
        self.assertEqual(info.body, b"hello")

        # Bad data type
        self.editor.set_response(200, {"data": finfo.model_dump()})
        self.assertRaises(ValueError, self.dp.submit_binary, "source", azm.DataLabel.TEST, {"hello": "Goodbye"})

        # Bad string like buffer
        self.editor.set_response(200, {"data": finfo.model_dump()})
        self.assertRaises(
            ValueError, self.dp.submit_binary, "source", azm.DataLabel.TEST, io.TextIOWrapper(io.BytesIO(b"hello2"))
        )

        # check skip identify & expected sha256
        self.editor.set_response(200, {"data": finfo.model_dump()})
        val = self.dp.submit_binary(
            "source", azm.DataLabel.TEST, io.BytesIO(b"hello"), skip_identify=True, expected_sha256="ffff"
        )
        info = self.editor.get_last_request()
        self.assertEqual(val, finfo)
        self.assertEqual(info.body, b"hello")
        self.assertEqual(
            info.url,
            self.server + "/api/v3/stream/source/test?skip-identify=true&expected-sha256=ffff",
        )

    async def test_async_submit_binary(self):
        """Mandatory."""
        # Test for successfully downloading file info to data.
        finfo = azm.Datastream(
            label=azm.DataLabel.TEXT,
            size=10,
            sha512="a",
            sha256="b",
            md5="c",
            sha1="d",
            file_format_legacy="f",
            magic="g",
            mime="h",
        )
        self.editor.set_response(200, {"data": finfo.model_dump()})
        val = await self.dp.async_submit_binary("source", azm.DataLabel.TEST, b"hello")
        info = self.editor.get_last_request()
        self.assertEqual(val, finfo)
        self.assertEqual(info.body, b"hello")

        # Submit a io reader
        self.editor.set_response(200, {"data": finfo.model_dump()})
        val = await self.dp.async_submit_binary("source", azm.DataLabel.TEST, io.BytesIO(b"hello"))
        info = self.editor.get_last_request()
        self.assertEqual(val, finfo)
        self.assertEqual(info.body, b"hello")

        # Submit a using a buffered random (like azul-runner)
        self.editor.set_response(200, {"data": finfo.model_dump()})
        val = await self.dp.async_submit_binary("source", azm.DataLabel.TEST, io.BufferedRandom(io.BytesIO(b"hello2")))
        info = self.editor.get_last_request()
        self.assertEqual(val, finfo)
        self.assertEqual(info.body, b"hello2")

        # Submit a fastapi UploadFile type
        self.editor.set_response(200, {"data": finfo.model_dump()})
        val = await self.dp.async_submit_binary(
            "source", azm.DataLabel.TEST, UploadFile(io.BytesIO(b"helloHelloMrBondImAUploadFile"))
        )
        info = self.editor.get_last_request()
        self.assertEqual(val, finfo)
        self.assertEqual(info.body, b"helloHelloMrBondImAUploadFile")

        # Generic async iterable
        async def iter_bytes() -> AsyncIterable[bytes]:
            for c in [b"hello", b"i'm an iterable", b"bytes", b"thats async"]:
                yield c

        self.editor.set_response(200, {"data": finfo.model_dump()})
        # Extend timeout as well as async iterable
        val = await self.dp.async_submit_binary("source", azm.DataLabel.TEST, iter_bytes(), 600.0)
        info = self.editor.get_last_request()
        self.assertEqual(val, finfo)
        self.assertEqual(info.body, b"helloi'm an iterablebytesthats async")

        # Submit a io reader partial seek ( will result in content too-long error if not set back to 0)
        self.editor.set_response(200, {"data": finfo.model_dump()})
        seeked = io.BytesIO(b"hello")
        seeked.read(2)
        val = await self.dp.async_submit_binary("source", azm.DataLabel.TEST, seeked)
        info = self.editor.get_last_request()
        self.assertEqual(val, finfo)
        self.assertEqual(info.body, b"hello")

        # Bad key to access data.
        self.editor.set_response(200, {"bad-key": finfo.model_dump()})
        with self.assertRaises(DispatcherApiException) as ex:
            await self.dp.async_submit_binary("source", azm.DataLabel.TEST, b"hello")
        info = self.editor.get_last_request()
        self.assertEqual(info.body, b"hello")

        # Assert various exception cases.
        self.editor.set_response(500, {})
        with self.assertRaises(DispatcherApiException) as ex:
            await self.dp.async_submit_binary("source", azm.DataLabel.TEST, b"hello")
        info = self.editor.get_last_request()
        self.assertEqual(ex.exception.status_code, 500)
        self.assertEqual(info.body, b"hello")

        # Bad data type
        self.editor.set_response(200, {"data": finfo.model_dump()})
        with self.assertRaises(ValueError):
            await self.dp.async_submit_binary("source", azm.DataLabel.TEST, {"hello": "Goodbye"})

        # Bad string like buffer
        self.editor.set_response(200, {"data": finfo.model_dump()})
        with self.assertRaises(ValueError):
            await self.dp.async_submit_binary("source", azm.DataLabel.TEST, io.TextIOWrapper(io.BytesIO(b"hello2")))

    def test_delete_binary(self):
        """Mandatory."""
        random_date = pendulum.now(datetime.timezone.utc)
        # good response
        self.editor.set_response(200, {"deleted": True})
        resp = self.dp.delete_binary("source", azm.DataLabel.TEST, "not-sha256", random_date)
        self.assertEqual(resp, (True, True))

        # binary not in dispatcher (404 from dispatcher)
        self.editor.set_response(404, {})
        resp = self.dp.delete_binary("source", azm.DataLabel.TEST, "not-sha256", random_date)
        self.assertEqual(resp, (False, False))

        # Non 200 response
        self.editor.set_response(500, {})
        with self.assertRaises(DispatcherApiException) as ex:
            self.dp.delete_binary("source", azm.DataLabel.TEST, "not-sha256", random_date)
        self.assertEqual(ex.exception.status_code, 500)

    def test_copy_binary(self):
        """Mandatory."""
        sourceA = "sourceA"
        sourceB = "sourceB"
        label = azm.DataLabel.TEST
        hash = "probs-a-sha256"

        self.editor.set_response(200, {})
        self.dp.copy_binary(sourceA, sourceB, label, hash)

        hash = "missing-sha256"
        self.editor.set_response(500, {"title": "copy error"})
        with self.assertRaises(DispatcherApiException) as ex:
            self.dp.copy_binary(sourceA, sourceB, label, hash)
        self.assertEqual(ex.exception.status_code, 500)
