import base64
import datetime
import json
import os
import unittest

import yaml
from pydantic import BaseModel

from azul_bedrock import models_settings

BASE_FILE_DIR = os.path.join(os.path.dirname(__file__), "..", "..", "testdata")


def read_data(path: str) -> bytes:
    with open(os.path.join(BASE_FILE_DIR, path), "rb") as f:
        data = f.read()
    return data


def jsondict(d: BaseModel):
    return json.loads(d.model_dump_json(exclude_defaults=True, exclude_unset=True))


class TestBasic(unittest.TestCase):
    def assertJsonDict(self, in1: BaseModel, in2: dict):
        try:
            self.assertEqual(jsondict(in1), in2)
        except Exception:
            print(f"failed, was actually:\n{jsondict(in1)}")
            raise

    def test_load_sources(self):
        sources_yaml = read_data("sources/sources1.yaml")
        loaded = yaml.safe_load(sources_yaml)
        sources = models_settings.Sources(**loaded)
        self.assertEqual(len(sources.sources), 7)
        self.assertEqual(
            set(sources.sources.keys()),
            {"testing", "incidents", "reporting", "samples", "tasking", "virustotal", "watch"},
        )
        self.assertEqual(sources.sources["testing"].description, "Files submitted during testing of Azul")
        self.assertEqual(sources.sources["testing"].exclude_from_backup, True)
        self.assertEqual(sources.sources["virustotal"].exclude_from_backup, True)
        self.assertEqual(sources.sources["tasking"].exclude_from_backup, False)

        # Verify the expire_events_ms data loads correctly.
        self.assertEqual(sources.sources["testing"].expire_events_after, "7 days")
        self.assertEqual(sources.sources["testing"].expire_events_ms, 604800000)
        self.assertEqual(
            sources.sources["testing"].kafka_config_full,
            {"retention.ms": "604800000", "cleanup.policy": "delete", "segment.bytes": "1073741824"},
        )

        self.assertEqual(sources.sources["incidents"].expire_events_after, "0")
        self.assertEqual(sources.sources["incidents"].expire_events_ms, -1)
        self.assertEqual(
            sources.sources["incidents"].kafka_config_full, {"retention.ms": "-1", "cleanup.policy": "compact"}
        )

        self.assertEqual(sources.sources["reporting"].expire_events_after, "0")
        self.assertEqual(sources.sources["reporting"].expire_events_ms, -1)
        self.assertEqual(
            sources.sources["reporting"].kafka_config_full, {"retention.ms": "-1", "cleanup.policy": "compact"}
        )

        self.assertEqual(sources.sources["virustotal"].expire_events_after, "3 months")
        self.assertEqual(sources.sources["virustotal"].expire_events_ms, 7776000000)
        self.assertEqual(
            sources.sources["virustotal"].kafka_config_full, {"retention.ms": "7776000000", "cleanup.policy": "delete"}
        )

    def _load_source(self, path: str) -> models_settings.Sources:
        sources_yaml = read_data(f"sources/{path}")
        loaded = yaml.safe_load(sources_yaml)
        source = models_settings.Sources(**loaded)
        self.assertEqual(len(source.sources), 1)
        return source

    def test_load_bad_sources(self):
        """Test cases where the input duration is bad and will fail to load."""
        src = self._load_source("sources_bad_expiry_inner_spaces.yaml")
        self.assertEqual(src.sources["testing"].expire_events_after, "7  days")
        with self.assertRaises(
            ValueError, msg="Expected error when there is multiple spaces between number and unit."
        ):
            src.sources["testing"].expire_events_ms

        src = self._load_source("sources_bad_expiry_no_number.yaml")
        self.assertEqual(src.sources["testing"].expire_events_after, "seven days")
        with self.assertRaises(ValueError, msg="Expected error with no number"):
            src.sources["testing"].expire_events_ms

        src = self._load_source("sources_bad_expiry_outer_spaces_lead.yaml")
        self.assertEqual(src.sources["testing"].expire_events_after, " 7 days")
        with self.assertRaises(ValueError, msg="Expected error with leading space"):
            src.sources["testing"].expire_events_ms

        src = self._load_source("sources_bad_expiry_outer_spaces_trail.yaml")
        self.assertEqual(src.sources["testing"].expire_events_after, "7 days ")
        with self.assertRaises(ValueError, msg="Expected error with trailing space."):
            src.sources["testing"].expire_events_ms
