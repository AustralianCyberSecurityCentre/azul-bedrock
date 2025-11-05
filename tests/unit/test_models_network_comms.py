from __future__ import annotations

import datetime
import json
import unittest
from typing import Any

from azul_bedrock import models_network as azm

# ##############################################
# Template for testing serialisation of objects


class _TupleConversionTests(unittest.TestCase):
    """Tests conversion between dataclass, dict, and json form as defined in the TEST_CASES tuple list."""

    TEST_CASES: list[tuple[Any, dict, str]]
    maxDiff = None

    @classmethod
    def setUpClass(cls) -> None:
        if cls is _TupleConversionTests:
            raise unittest.SkipTest("TupleConversionTest template does not execute directly")

    def test_values_to_dict(self):
        for num, (obj_val, dict_val, json_val) in enumerate(self.TEST_CASES):
            # We don't compare JSON output because the order of keys is not guaranteed between runs
            self.assertEqual(
                json.loads(obj_val.model_dump_json(exclude_defaults=True)),
                dict_val,
                "%s.test_to_dict failed for test case #%s" % (self.__class__.__name__, num + 1),
            )

    def test_values_from_dict(self):
        for num, (obj_val, dict_val, json_val) in enumerate(self.TEST_CASES):
            self.assertEqual(
                obj_val.__class__(**dict_val),
                obj_val,
                "%s.test_from_dict failed for test case #%s" % (self.__class__.__name__, num + 1),
            )

    def test_values_from_json(self):
        for num, (obj_val, dict_val, json_val) in enumerate(self.TEST_CASES):
            self.assertEqual(
                obj_val.__class__.model_validate_json(json_val),
                obj_val,
                "%s.test_from_json failed for test case #%s" % (self.__class__.__name__, num + 1),
            )


# ###############################################
# Feature declarations, values, and data streams


class TestFeatureDefs(_TupleConversionTests):
    """Tests the Feature class serialisation"""

    TEST_CASES = [
        # (Feature obj, dict form, json form)
        (
            azm.PluginEvent.Entity.Feature(
                name="test_int", desc="A test integer feature", type=azm.FeatureType.Integer
            ),
            dict(name="test_int", type="integer", desc="A test integer feature"),
            '{"name": "test_int", "desc": "A test integer feature", "type": "integer"}',
        ),
        (
            azm.PluginEvent.Entity.Feature(name="test_path", desc="A test URI feature", type=azm.FeatureType.Uri),
            dict(name="test_path", type="uri", desc="A test URI feature"),
            '{"name": "test_path", "desc": "A test URI feature", "type": "uri"}',
        ),
        (
            azm.PluginEvent.Entity.Feature(
                name="test_dt", desc="A test datetime feature", type=azm.FeatureType.Datetime
            ),
            dict(name="test_dt", type="datetime", desc="A test datetime feature"),
            '{"name": "test_dt", "type": "datetime", "desc": "A test datetime feature"}',
        ),
        (
            azm.PluginEvent.Entity.Feature(
                name="test_bytes", desc="A test bytes feature", type=azm.FeatureType.Binary
            ),
            dict(name="test_bytes", type="binary", desc="A test bytes feature"),
            '{"name": "test_bytes", "type": "binary", "desc": "A test bytes feature"}',
        ),
    ]


class TestAPIFeatureValue(_TupleConversionTests):
    TEST_CASES = [
        (
            azm.FeatureValue(name="some_feature", type="integer", value="5"),
            {"name": "some_feature", "type": "integer", "value": "5"},
            '{"name": "some_feature", "type": "integer", "value": "5"}',
        ),
        (
            azm.FeatureValue(name="other_feature", type="filepath", value="/bin/foo"),
            {"name": "other_feature", "type": "filepath", "value": "/bin/foo"},
            '{"name": "other_feature", "type": "filepath", "value": "/bin/foo"}',
        ),
        (
            azm.FeatureValue(name="other_feature", type="filepath", value="/bin/foo"),
            {"name": "other_feature", "type": "filepath", "value": "/bin/foo"},
            '{"name": "other_feature", "type": "filepath", "value": "/bin/foo"}',
        ),
        (
            azm.FeatureValue(name="feature", type="binary", value="TVo=", label="string1"),
            {"name": "feature", "type": "binary", "value": "TVo=", "label": "string1"},
            '{"name": "feature", "type": "binary", "value": "TVo=", "label": "string1"}',
        ),
    ]


class TestAPIContentEntry(_TupleConversionTests):
    # Includes testing of _BytesField
    TEST_CASES = (
        (
            azm.Datastream(
                label=azm.DataLabel.TEXT,
                size=9001,
                sha1="1",
                sha256="256",
                sha512="512",
                md5="5",
                mime="application/octet-stream",
                magic="abracadabra",
                file_format_legacy="something",
            ),
            dict(
                label=azm.DataLabel.TEXT,
                size=9001,
                sha1="1",
                sha256="256",
                sha512="512",
                md5="5",
                mime="application/octet-stream",
                magic="abracadabra",
                file_format_legacy="something",
            ),
            """
            {
                "label": "text",
                "size": 9001,
                "sha1": "1",
                "sha256": "256",
                "sha512": "512",
                "md5": "5",
                "mime": "application/octet-stream",
                "magic": "abracadabra",
                "file_format_legacy": "something"
            }
            """,
        ),
        (
            azm.Datastream(
                label=azm.DataLabel.TEXT,
                size=9001,
                sha1="1",
                sha256="256",
                sha512="512",
                md5="5",
                mime="application/octet-stream",
                magic="abracadabra",
                file_format_legacy="something",
                language="random value",
            ),
            dict(
                label=azm.DataLabel.TEXT,
                size=9001,
                sha1="1",
                sha256="256",
                sha512="512",
                md5="5",
                mime="application/octet-stream",
                magic="abracadabra",
                file_format_legacy="something",
                language="random value",
            ),
            """
             {
                 "label": "text",
                 "size": 9001,
                 "sha1": "1",
                 "sha256": "256",
                 "sha512": "512",
                 "md5": "5",
                 "mime": "application/octet-stream",
                 "magic": "abracadabra",
                 "file_format_legacy": "something",
                 "language": "random value"
             }
            """,
        ),
    )


# ########


class TestInputEntity(_TupleConversionTests):
    TEST_CASES = (
        (
            azm.BinaryEvent.Entity(
                sha256="foo_ent",
                features=[],
                datastreams=[],
            ),
            dict(
                sha256="foo_ent",
            ),
            '{"sha256": "foo_ent"}',
        ),
        (azm.BinaryEvent.Entity(sha256="blah"), dict(sha256="blah"), '{"sha256": "blah"}'),
        (
            azm.BinaryEvent.Entity(sha256="blah", info={"test": 5, "other value": ["list", "of", "strings"]}),
            dict(sha256="blah", info={"test": 5, "other value": ["list", "of", "strings"]}),
            '{"sha256": "blah", "info": {"test": 5, "other value": ["list", "of", "strings"]}}',
        ),
        (
            azm.BinaryEvent.Entity(
                sha256="entID",
                features=[
                    azm.FeatureValue(name="feature", type="string", value="string", label="labelfeat"),
                    azm.FeatureValue(name="feature", type="string", value="string", label="labelfeat", offset=16),
                    azm.FeatureValue(name="feature", type="binary", value="Ynl0ZXM=", label="otherfeat"),
                ],
                datastreams=[
                    azm.Datastream(
                        label=azm.DataLabel.TEXT,
                        size=7,
                        sha1="1",
                        sha256="256",
                        sha512="512",
                        md5="5",
                        mime="application/octet-stream",
                        magic="abracadabra",
                        file_format_legacy="something",
                    )
                ],
            ),
            {
                "sha256": "entID",
                "features": [
                    {"name": "feature", "type": "string", "value": "string", "label": "labelfeat"},
                    {"name": "feature", "type": "string", "value": "string", "label": "labelfeat", "offset": 16},
                    {"name": "feature", "type": "binary", "value": "Ynl0ZXM=", "label": "otherfeat"},
                ],
                "datastreams": [
                    dict(
                        label=azm.DataLabel.TEXT,
                        size=7,
                        sha1="1",
                        sha256="256",
                        sha512="512",
                        md5="5",
                        mime="application/octet-stream",
                        magic="abracadabra",
                        file_format_legacy="something",
                    )
                ],
            },
            """
            {
                "sha256": "entID",
                "features": [
                    {"name": "feature", "type": "string", "value": "string", "label": "labelfeat"},
                    {"name": "feature", "type": "string", "value": "string", "label": "labelfeat", "offset": 16},
                    {"name": "feature", "type": "binary", "value": "Ynl0ZXM=", "label": "otherfeat"}
                ],
                "datastreams": [{"label": "text", "size": 7, "sha1": "1", "sha256": "256", "sha512": "512", "md5": "5",
                          "mime": "application/octet-stream", "magic": "abracadabra",
                          "file_format_legacy": "something"}]
            }
            """,
        ),
    )


class TestAuthorDetails(_TupleConversionTests):
    TEST_CASES = (
        (
            azm.PluginEvent.Entity(
                name="Something",
                category="whatever",
                version="None",
                description="A test azm.AuthorDetails entry",
                features=[
                    azm.PluginEvent.Entity.Feature(name="feature", desc="A test feature", type=azm.FeatureType.String)
                ],
            ),
            dict(
                name="Something",
                category="whatever",
                version="None",
                description="A test azm.AuthorDetails entry",
                features=[{"name": "feature", "desc": "A test feature", "type": "string"}],
            ),
            """
            {
                "name": "Something",
                "category": "whatever",
                "version": "None",
                "description": "A test azm.AuthorDetails entry",
                "features": [{"name": "feature", "desc": "A test feature", "type": "string"}]
            }
            """,
        ),
        (
            # Test .summary(); should return an azm.Author class
            azm.PluginEvent.Entity(
                name="Something",
                category="whatever",
                version="None",
                description="A test azm.AuthorDetails entry",
                features=[
                    azm.PluginEvent.Entity.Feature(name="feature", desc="A test feature", type=azm.FeatureType.String)
                ],
            ).summary(),
            dict(
                name="Something",
                category="whatever",
                version="None",
            ),
            """
            {
                "name": "Something",
                "category": "whatever",
                "version": "None"
            }
            """,
        ),
    )


class TestStatusEntity(_TupleConversionTests):
    TEST_CASES = (
        (
            azm.StatusEvent.Entity(
                status="completed",
                input=azm.BinaryEvent(
                    model_version=azm.CURRENT_MODEL_VERSION,
                    kafka_key="abc",
                    dequeued="ccc",
                    action=azm.BinaryAction.Sourced,
                    flags={
                        "expedite": True,
                    },
                    author=azm.Author(name="system", category="something"),
                    entity=azm.BinaryEvent.Entity(sha256="42"),
                    source=azm.Source(
                        name="",
                        path=[],
                        references={"info1": "foo"},
                        timestamp=datetime.datetime(year=1999, month=12, day=31, tzinfo=datetime.timezone.utc),
                    ),
                    timestamp=datetime.datetime(year=1999, month=12, day=31, tzinfo=datetime.timezone.utc),
                ),
            ),
            dict(
                status="completed",
                input=dict(
                    model_version=azm.CURRENT_MODEL_VERSION,
                    kafka_key="abc",
                    dequeued="ccc",
                    action=azm.BinaryAction.Sourced,
                    flags={
                        "expedite": True,
                    },
                    author=dict(name="system", category="something"),
                    entity=dict(sha256="42"),
                    source=dict(name="", path=[], timestamp="1999-12-31T00:00:00+00:00", references={"info1": "foo"}),
                    timestamp="1999-12-31T00:00:00+00:00",
                ),
            ),
            """
            {
                "status": "completed",
                "input": {
                    "model_version":5,
                    "kafka_key":"abc",
                    "dequeued": "ccc",
                    "action": "sourced",
                    "flags": {"expedite": true},
                    "author": {"name": "system", "category": "something"},
                    "entity": {"sha256": "42"},
                    "source": {"name": "", "path": [], "timestamp": "1999-12-31T00:00:00+00:00",
                               "references": {"info1": "foo"}},
                    "timestamp": "1999-12-31T00:00:00+00:00"
                }
            }
            """,
        ),
    )


class TestStatusEvent(_TupleConversionTests):
    TEST_CASES = (
        (
            azm.StatusEvent(
                model_version=azm.CURRENT_MODEL_VERSION,
                kafka_key="abc",
                author=azm.Author(name="test", category="TEST"),
                timestamp=datetime.datetime(year=1992, month=5, day=17, tzinfo=datetime.timezone.utc),
                entity=azm.StatusEvent.Entity(
                    status="error-exception",
                    error="WORKER has fainted!",
                    input=azm.BinaryEvent(
                        model_version=azm.CURRENT_MODEL_VERSION,
                        kafka_key="abc",
                        dequeued="input-dummy-dequeued-id",
                        action=azm.BinaryAction.Extracted,
                        author=azm.Author(name="", category=""),
                        entity=azm.BinaryEvent.Entity(
                            sha256="foo",
                            md5="",
                            sha1="",
                            sha512="",
                            size=0,
                        ),
                        source=azm.Source(
                            name="",
                            path=[],
                            timestamp=datetime.datetime(year=1992, month=5, day=17, tzinfo=datetime.timezone.utc),
                        ),
                        timestamp=datetime.datetime(year=2001, month=8, day=24, tzinfo=datetime.timezone.utc),
                    ),
                ),
            ),
            dict(
                model_version=azm.CURRENT_MODEL_VERSION,
                kafka_key="abc",
                author=dict(name="test", category="TEST"),
                timestamp="1992-05-17T00:00:00+00:00",
                entity=dict(
                    status="error-exception",
                    error="WORKER has fainted!",
                    input=dict(
                        model_version=azm.CURRENT_MODEL_VERSION,
                        kafka_key="abc",
                        dequeued="input-dummy-dequeued-id",
                        action=azm.BinaryAction.Extracted,
                        author=dict(name="", category=""),
                        entity=dict(
                            sha256="foo",
                            md5="",
                            sha1="",
                            sha512="",
                            size=0,
                        ),
                        source=dict(name="", path=[], timestamp="1992-05-17T00:00:00+00:00"),
                        timestamp="2001-08-24T00:00:00+00:00",
                    ),
                ),
            ),
            """
            {
                "model_version": 5,
                "kafka_key":"abc",
                "author": {"name": "test", "category": "TEST"},
                "timestamp": "1992-05-17T00:00:00+00:00",
                "entity": {
                    "status": "error-exception",
                    "error": "WORKER has fainted!",
                    "input": {
                        "model_version": 5,
                        "kafka_key":"abc",
                        "dequeued": "input-dummy-dequeued-id",
                        "action": "extracted",
                        "author": {"name": "", "category": ""},
                        "entity": {
                            "sha256": "foo",
                            "md5": "",
                            "sha1": "",
                            "sha512": "",
                            "size": 0
                        },
                        "source": {"name": "", "path": [], "timestamp": "1992-05-17T00:00:00+00:00"},
                        "timestamp": "2001-08-24T00:00:00+00:00"
                    }
                }
            }
            """,
        ),
    )


class TestAuthor(_TupleConversionTests):
    TEST_CASES = (
        (
            azm.Author(
                name="TestPlugin",
                category="Plugin",
                version="1.0",
                security=None,
            ),
            dict(
                name="TestPlugin",
                category="Plugin",
                version="1.0",
            ),
            """
            {
                "name": "TestPlugin",
                "category": "Plugin",
                "version": "1.0"
            }
            """,
        ),
        (
            azm.Author(  # No version, but with security label
                name="TestPlugin",
                category="Plugin",
                security="LIMITED_ACCESS",
            ),
            dict(
                name="TestPlugin",
                category="Plugin",
                security="LIMITED_ACCESS",
            ),
            """
            {
                "name": "TestPlugin",
                "category": "Plugin",
                "security": "LIMITED_ACCESS"
            }
            """,
        ),
    )


class TestHistory(_TupleConversionTests):
    TEST_CASES = (
        (
            azm.PathNode(
                action=azm.BinaryAction.Extracted,
                timestamp=datetime.datetime(year=2345, month=7, day=8, tzinfo=datetime.timezone.utc),
                author=azm.Author(
                    name="User Insert",
                    category="Manual",
                ),
                sha256="<hash goes here>",
                relationship={"action": "extracted", "label": "deflate"},
            ),
            dict(
                action=azm.BinaryAction.Extracted,
                timestamp="2345-07-08T00:00:00+00:00",
                author=dict(
                    name="User Insert",
                    category="Manual",
                ),
                sha256="<hash goes here>",
                relationship={"action": "extracted", "label": "deflate"},
            ),
            """
            {
                "action": "extracted",
                "timestamp": "2345-07-08T00:00:00+00:00",
                "author": {
                    "name": "User Insert",
                    "category": "Manual"
                },
                "sha256": "<hash goes here>",
                "relationship": {"action": "extracted", "label": "deflate"}
            }
            """,
        ),
    )


class TestBinaryEvent(_TupleConversionTests):
    TEST_CASES = (
        (
            azm.BinaryEvent(
                model_version=azm.CURRENT_MODEL_VERSION,
                kafka_key="abc",
                dequeued="ccc",
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
            ),
            dict(
                model_version=azm.CURRENT_MODEL_VERSION,
                kafka_key="abc",
                dequeued="ccc",
                action=azm.BinaryAction.Sourced,
                timestamp="2014-09-21T00:00:00+00:00",
                author=dict(
                    name="some ingest process",
                    category="automatic_input",
                ),
                entity=dict(sha256="12345"),
                source=dict(
                    name="",
                    path=[],
                    timestamp="2014-09-21T00:00:00+00:00",
                ),
            ),
            # 'flags' is missing from this JSON to ensure that it correctly defaults to empty dict on load
            """
            {
                "model_version": 5,
                "kafka_key":"abc",
                "dequeued": "ccc",
                "action": "sourced",
                "timestamp": "2014-09-21T00:00:00+00:00",
                "author": {
                    "name": "some ingest process",
                    "category": "automatic_input"
                },
                "entity": {"sha256": "12345"},
                "source": {
                    "name": "",
                    "path": [],
                    "timestamp": "2014-09-21T00:00:00+00:00"
                }
            }
            """,
        ),
        (
            azm.BinaryEvent(
                model_version=azm.CURRENT_MODEL_VERSION,
                kafka_key="foobar-md5",
                dequeued="dummy-dequeued-id",
                flags={},
                action=azm.BinaryAction.Sourced,
                timestamp=datetime.datetime(year=2014, month=9, day=21, tzinfo=datetime.timezone.utc),
                author=azm.Author(name="some ingest process", category="automatic_input"),
                entity=azm.BinaryEvent.Entity(sha256="12345"),
                source=azm.Source(
                    name="",
                    path=[],
                    timestamp=datetime.datetime(year=2014, month=9, day=21, tzinfo=datetime.timezone.utc),
                ),
            ),
            dict(
                model_version=azm.CURRENT_MODEL_VERSION,
                kafka_key="foobar-md5",
                dequeued="dummy-dequeued-id",
                action=azm.BinaryAction.Sourced,
                timestamp="2014-09-21T00:00:00+00:00",
                author=dict(
                    name="some ingest process",
                    category="automatic_input",
                ),
                entity=dict(sha256="12345"),
                source=dict(
                    name="",
                    path=[],
                    timestamp="2014-09-21T00:00:00+00:00",
                ),
            ),
            """
            {
                "model_version": 5,
                "kafka_key":"foobar-md5",
                "dequeued": "dummy-dequeued-id",
                "flags": {},
                "action": "sourced",
                "timestamp": "2014-09-21T00:00:00+00:00",
                "author": {
                    "name": "some ingest process",
                    "category": "automatic_input"
                },
                "entity": {"sha256": "12345"},
                "source": {
                    "name": "",
                    "path": [],
                    "timestamp": "2014-09-21T00:00:00+00:00"
                }
            }
            """,
        ),
    )
