"""Mocked Dispatcher.

This mock has endpoints that track a limited amount of state.
It is primarily used to test individual requests are valid, not a sequence of actions.
"""

from .dp import MockDispatcher
from .helper import Editor

__all__ = [
    "MockDispatcher",
    "Editor",
]
