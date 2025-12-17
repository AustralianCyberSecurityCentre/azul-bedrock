"""Dispatcher API Mocking."""

import contextlib
import logging
import multiprocessing
import socket

import uvicorn
from fastapi import FastAPI
from fastapi.testclient import TestClient

from . import events, state, streams

logger = logging.getLogger(__name__)


app = FastAPI()


app.include_router(events.router)
app.include_router(streams.router)
app.include_router(state.router)


client = TestClient(app)


class MockDispatcher(multiprocessing.Process):
    """Allows for running the mock dispatcher as a process rather than using the TestClient."""

    def __init__(self, host="localhost", port="42081"):
        super().__init__()
        self.host = host
        with contextlib.closing(socket.socket(socket.AF_INET, socket.SOCK_STREAM)) as s:
            s.bind(("", 0))
            self.port = s.getsockname()[1]
        logger.debug("Dummy server using port %s" % self.port)

    def stop(self):
        """Stop the server."""
        self.terminate()

    def run(self, *args, **kwargs):
        """Run the server."""
        config = uvicorn.Config(app, host=self.host, port=self.port)
        self.server = uvicorn.Server(config=config)
        self.config = config
        self.server.run()
