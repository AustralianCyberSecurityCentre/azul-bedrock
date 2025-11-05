"""Data models for Azul statistics."""

from pydantic import BaseModel


class StatisticSummary(BaseModel):
    """Summary statistics about this instance of Azul."""

    binary_count: int


class StatisticContainer(BaseModel):
    """Container for statistics, containing both arbitrary data and a timestamp."""

    timestamp: int
    data: StatisticSummary
