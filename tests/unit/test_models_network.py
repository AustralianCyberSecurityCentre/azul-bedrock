import base64
import datetime
import json
import unittest

from pydantic import BaseModel

from azul_bedrock import models_network as azm


def jsondict(d: BaseModel):
    return json.loads(d.model_dump_json(exclude_defaults=True, exclude_unset=True))


atime = datetime.datetime(year=2012, month=1, day=1, tzinfo=datetime.timezone.utc)


class TestBasic(unittest.TestCase):
    maxDiff = None

    def assertJsonDict(self, in1: BaseModel, in2: dict):
        try:
            self.assertEqual(jsondict(in1), in2)
        except Exception:
            print(f"failed, was actually:\n{jsondict(in1)}")
            raise

    def test_import(self):
        self.assertTrue(azm.__name__)

    def test_values(self):
        self.assertEqual(f"{azm.FeatureType.Float}", "float")
        self.assertEqual(f"{azm.StatusEnum.OPT_OUT}", "opt-out")

    def test_contains(self):
        self.assertTrue(azm.FeatureType.Float in azm.FeatureType)
        self.assertTrue(azm.FeatureType.Integer in azm.FeatureType)
        self.assertTrue("gtrjgj" not in azm.FeatureType)

    def test_commonfilecontent_to_input_entity(self):
        cfc = azm.FileInfo(
            md5="md5.",
            sha1="sha1.",
            sha256="sha256.",
            sha512="sha512.",
            ssdeep="ssdeep.",
            tlsh="current_tlsh.",
            size=4444,
            file_format_legacy="file_format_legacy.",
            file_format="file_format.",
            file_extension="f_extension.",
            mime="mime.",
            magic="magic.",
        )
        ent = jsondict(cfc.to_input_entity())
        print(ent)
        self.assertEqual(
            ent,
            {
                "sha256": "sha256.",
                "sha512": "sha512.",
                "sha1": "sha1.",
                "md5": "md5.",
                "ssdeep": "ssdeep.",
                "tlsh": "current_tlsh.",
                "size": 4444,
                "file_format_legacy": "file_format_legacy.",
                "file_format": "file_format.",
                "file_extension": "f_extension.",
                "mime": "mime.",
                "magic": "magic.",
                "features": [
                    {"name": "file_format", "type": "string", "value": "file_format."},
                    {"name": "file_format_legacy", "type": "string", "value": "file_format_legacy."},
                    {"name": "file_extension", "type": "string", "value": "f_extension."},
                    {"name": "magic", "type": "string", "value": "magic."},
                    {"name": "mime", "type": "string", "value": "mime."},
                ],
            },
        )

        to_input_ent = cfc.to_input_entity()
        to_input_ent.features.append(azm.FeatureValue(name="new_feature", type=azm.FeatureType.String, value="NEW!"))
        to_input_ent.info = {"value": "info-value is here."}
        to_input_ent.datastreams = []

        back_to_file_info = to_input_ent.to_file_info()
        # Ensure the new and original are still equal
        self.assertEqual(cfc.model_dump(), back_to_file_info.model_dump())

    def test_file_info_to_input_entity(self):
        fi = azm.Datastream(
            label=azm.DataLabel.CONTENT,
            size=4444,
            md5="md5.",
            sha1="sha1.",
            sha256="sha256.",
            sha512="sha512.",
            mime="mime.",
            magic="magic.",
            file_format_legacy="file_format_legacy.",
            file_format="file_format.",
            file_extension="f_extension.",
            tlsh="current_tlsh.",
        )
        ent = jsondict(fi.to_input_entity())
        print(ent)
        self.assertEqual(
            ent,
            {
                "sha256": "sha256.",
                "sha512": "sha512.",
                "sha1": "sha1.",
                "md5": "md5.",
                "tlsh": "current_tlsh.",
                "size": 4444,
                "file_format_legacy": "file_format_legacy.",
                "file_format": "file_format.",
                "file_extension": "f_extension.",
                "mime": "mime.",
                "magic": "magic.",
                "features": [
                    {"name": "file_format", "type": "string", "value": "file_format."},
                    {"name": "file_format_legacy", "type": "string", "value": "file_format_legacy."},
                    {"name": "file_extension", "type": "string", "value": "f_extension."},
                    {"name": "magic", "type": "string", "value": "magic."},
                    {"name": "mime", "type": "string", "value": "mime."},
                ],
                "datastreams": [
                    {
                        "sha256": "sha256.",
                        "sha512": "sha512.",
                        "sha1": "sha1.",
                        "md5": "md5.",
                        "tlsh": "current_tlsh.",
                        "size": 4444,
                        "file_format_legacy": "file_format_legacy.",
                        "file_format": "file_format.",
                        "file_extension": "f_extension.",
                        "mime": "mime.",
                        "magic": "magic.",
                        "label": "content",
                    }
                ],
            },
        )

    def test_feature_value(self):
        fv = azm.FeatureValue(name="name", type=azm.FeatureType.String, value="value")
        self.assertJsonDict(fv, {"name": "name", "type": azm.FeatureType.String, "value": "value"})

    def test_path(self):
        tmp = azm.PathNode(
            action=azm.BinaryAction.Mapped,
            timestamp=atime,
            author=azm.Author(name="name"),
            sha256="sha256",
        )
        self.assertEqual(tmp.relationship, {})
        self.assertJsonDict(
            tmp,
            {
                "action": "mapped",
                "timestamp": "2012-01-01T00:00:00+00:00",
                "author": {"name": "name"},
                "sha256": "sha256",
            },
        )

    def test_feature_value_coding(self):
        cases = [
            (azm.FeatureType.Binary, b"abacus\x00", "YWJhY3VzAA=="),
            (
                azm.FeatureType.Datetime,
                datetime.datetime(2000, 1, 1, 1, 1, 1, 0, datetime.UTC),
                "2000-01-01T01:01:01+00:00",
            ),
            (azm.FeatureType.Integer, 55565, "55565"),
            (azm.FeatureType.Float, 0.55565, "0.55565"),
            (
                azm.FeatureType.String,
                "this is my string and you can't have it",
                "this is my string and you can't have it",
            ),
            (azm.FeatureType.Filepath, "/private/files", "/private/files"),
            (azm.FeatureType.Uri, "https://unpatched.apache.internal", "https://unpatched.apache.internal"),
        ]
        for _type, val, expected in cases:
            encoded = azm.value_encode(val)
            self.assertEqual(encoded, expected, _type)
            decoded = azm.value_decode(_type, encoded)
            self.assertEqual(val, decoded, _type)

    def test_surrogate_string_feature_value_coding(self):
        """Surrogate string (hard to encode unicode values.) ensure they get escaped when they won't encode."""
        input_value = "\ud83d\ude4f"
        value_after_encoding = "\\ud83d\\ude4f"
        # Verify that encode/decode mutates the input value by escaping the complex characters.
        encoded = azm.value_encode(input_value)
        self.assertEqual(encoded, value_after_encoding)
        decoded = azm.value_decode(azm.FeatureType.String, encoded)
        self.assertEqual(value_after_encoding, decoded)

        # Verify that subsequent encode/decode have no affect.
        encoded2 = azm.value_encode(decoded)
        decoded2 = azm.value_decode(azm.FeatureType.String, encoded2)
        self.assertEqual(decoded, decoded2)

    def test_source(self):
        tmp = azm.Source(name="id", path=[], timestamp=atime)
        self.assertEqual(tmp.references, {})
        self.assertJsonDict(tmp, {"name": "id", "path": [], "timestamp": "2012-01-01T00:00:00+00:00"})

    def test_input_event(self):
        tmp = azm.BinaryEvent(
            model_version=azm.CURRENT_MODEL_VERSION,
            kafka_key="id",
            action=azm.BinaryAction.Sourced,
            timestamp=atime,
            source=azm.Source(name="id", path=[], timestamp=atime),
            author=azm.Author(name="name"),
            entity=azm.BinaryEvent.Entity(sha256="id"),
        )
        self.assertEqual(tmp.entity.info, {})
        self.assertEqual(tmp.entity.features, [])
        self.assertEqual(tmp.entity.datastreams, [])
        self.assertJsonDict(
            tmp,
            {
                "model_version": 5,
                "kafka_key": "id",
                "action": "sourced",
                "timestamp": "2012-01-01T00:00:00+00:00",
                "source": {"name": "id", "path": [], "timestamp": "2012-01-01T00:00:00+00:00"},
                "author": {"name": "name"},
                "entity": {"sha256": "id"},
            },
        )

    def test_input_event(self):
        tmp = azm.BinaryEvent(
            model_version=azm.CURRENT_MODEL_VERSION,
            kafka_key="id",
            action=azm.BinaryAction.Sourced,
            timestamp=atime,
            source=azm.Source(name="id", path=[], timestamp=atime),
            author=azm.Author(name="name"),
            entity=azm.BinaryEvent.Entity(sha256="id"),
        )
        self.assertEqual(tmp.entity.info, {})
        self.assertEqual(tmp.entity.features, [])
        self.assertEqual(tmp.entity.datastreams, [])
        self.assertJsonDict(
            tmp,
            {
                "model_version": 5,
                "kafka_key": "id",
                "action": "sourced",
                "timestamp": "2012-01-01T00:00:00+00:00",
                "source": {"name": "id", "path": [], "timestamp": "2012-01-01T00:00:00+00:00"},
                "author": {"name": "name"},
                "entity": {"sha256": "id"},
            },
        )

    def test_status_event(self):
        tmp = azm.StatusEvent(
            model_version=azm.CURRENT_MODEL_VERSION,
            kafka_key="id",
            timestamp=atime,
            author=azm.Author(name="name"),
            entity=azm.StatusEvent.Entity(
                input=azm.BinaryEvent(
                    model_version=azm.CURRENT_MODEL_VERSION,
                    kafka_key="id",
                    action=azm.BinaryAction.Sourced,
                    timestamp=atime,
                    source=azm.Source(name="id", path=[], timestamp=atime),
                    author=azm.Author(name="name"),
                    entity=azm.BinaryEvent.Entity(sha256="id"),
                ),
                status="error-exception",
            ),
        )
        self.assertEqual(tmp.entity.results, [])
        self.assertJsonDict(
            tmp,
            {
                "model_version": 5,
                "kafka_key": "id",
                "timestamp": "2012-01-01T00:00:00+00:00",
                "author": {"name": "name"},
                "entity": {
                    "input": {
                        "model_version": 5,
                        "kafka_key": "id",
                        "action": "sourced",
                        "timestamp": "2012-01-01T00:00:00+00:00",
                        "source": {"name": "id", "path": [], "timestamp": "2012-01-01T00:00:00+00:00"},
                        "author": {"name": "name"},
                        "entity": {"sha256": "id"},
                    },
                    "status": "error-exception",
                },
            },
        )

    def test_author_event(self):
        tmp = azm.PluginEvent(
            model_version=azm.CURRENT_MODEL_VERSION,
            kafka_key="id",
            timestamp=atime,
            author=azm.Author(name="name"),
            entity=azm.PluginEvent.Entity(name="name"),
        )
        self.assertEqual(tmp.entity.features, [])
        self.assertJsonDict(
            tmp,
            {
                "model_version": 5,
                "kafka_key": "id",
                "timestamp": "2012-01-01T00:00:00+00:00",
                "author": {"name": "name"},
                "entity": {"name": "name"},
            },
        )

    def test_insert_event(self):
        tmp = azm.InsertEvent(
            model_version=azm.CURRENT_MODEL_VERSION,
            kafka_key="id",
            timestamp=atime,
            author=azm.Author(name="name"),
            entity=azm.InsertEvent.Entity(
                original_source="tasking",
                parent_sha256="parent_sha256",
                child=azm.BinaryEvent.Entity(sha256="id"),
                child_history=azm.PathNode(
                    author=azm.Author(name="name"),
                    action=azm.BinaryAction.Sourced,
                    sha256="sha256",
                    relationship={},
                    timestamp=atime,
                ),
            ),
        )
        self.assertJsonDict(
            tmp,
            {
                "model_version": 5,
                "kafka_key": "id",
                "timestamp": "2012-01-01T00:00:00+00:00",
                "author": {"name": "name"},
                "entity": {
                    "original_source": "tasking",
                    "parent_sha256": "parent_sha256",
                    "child": {"sha256": "id"},
                    "child_history": {
                        "action": "sourced",
                        "timestamp": "2012-01-01T00:00:00+00:00",
                        "author": {"name": "name"},
                        "sha256": "sha256",
                    },
                },
            },
        )
