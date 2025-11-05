"""Data models for purge operations."""

from pydantic import BaseModel


class PurgeSimulation(BaseModel):
    """Info about what will be purged.

    This does not account for binaries linked from unrelated events
    so actual binary deletions will be smaller.
    """

    events: int


class PurgeResults(BaseModel):
    """Info about what was purged."""

    events_purged: int = 0
    binaries_kept: int = 0
    binaries_purged: int = 0
