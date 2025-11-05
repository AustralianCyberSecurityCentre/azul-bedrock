"""Models for common Azul settings."""

import enum
from functools import cached_property

import pendulum
from pydantic import BaseModel, computed_field


class PartitionUnitEnum(str, enum.Enum):
    """Valid partition unit settings for event types."""

    all = "all"
    year = "year"
    month = "month"
    week = "week"
    day = "day"


def convert_string_to_duration_ms(input_duration: str) -> int:
    """Convert an input string into a total duration in milliseconds.

    Input string format is expected to be
    <int> <years|months|weeks|days>
    e.g:
    10 days
    1 years
    11 weeks
    """
    if input_duration == "0":
        return -1
    split_string = input_duration.split(" ")
    if len(split_string) != 2:
        raise ValueError(
            f"provided input '{input_duration}' split into '{len(split_string)}' strings it must split"
            + f" into 2, the split was actually {split_string}"
        )
    duration, unit = input_duration.split(" ")
    try:
        duration = int(duration)
    except Exception:
        raise ValueError(
            f"Invalid duration for expire_events_after '{duration=}' duration should be an integer value."
        )

    valid_units = ["days", "weeks", "months", "years"]
    if unit not in valid_units:
        raise ValueError(f"Invalid unit for expire_events_after {unit=} valid values are {valid_units}")
    pend_dur = pendulum.Duration(**{unit: duration})
    return int(pend_dur.total_seconds() * 1000)


class Source(BaseModel):
    """Information for a particular data source for Azul binaries."""

    class SourceReference(BaseModel):
        """Specific reference information."""

        name: str
        required: bool
        description: str
        highlight: bool = False

    class SourceKafka(BaseModel):
        """Kafka source configuration."""

        numpartitions: int = 0
        replicationfactor: int = 0
        config: dict[str, str] = {}

    icon_class: str = "bug"
    references: list[SourceReference] = []
    kafka: SourceKafka = SourceKafka()

    @computed_field
    @property
    def kafka_config_full(self) -> dict[str, str]:
        """Get full kafka config including options added based on expiry settings."""
        out = dict(self.kafka.config)
        if self.expire_events_ms <= 0:
            out["cleanup.policy"] = "compact"
        else:
            out["cleanup.policy"] = "delete"
        out["retention.ms"] = str(self.expire_events_ms)
        return out

    description: str = ""
    # time box indices
    partition_unit: PartitionUnitEnum = PartitionUnitEnum.year
    # keep events for this duration and no longer.
    expire_events_after: str = "0"

    @computed_field
    @cached_property
    def expire_events_ms(self) -> int:
        """Return the duration after which events should be removed from opensearch and kafka.

        -1 means never.
        """
        return convert_string_to_duration_ms(self.expire_events_after)

    # override settings for the elastic indices associated with the source
    elastic: dict = {}
    # if true, the source will not be backed up with azul-backup
    exclude_from_backup: bool = False


class Sources(BaseModel):
    """Collection of Azul sources stored by name."""

    sources: dict[str, Source] = {}
