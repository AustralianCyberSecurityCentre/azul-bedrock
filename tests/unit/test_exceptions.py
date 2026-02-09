import unittest

from azul_bedrock import exceptions


class TestBasic(unittest.TestCase):
    def test_import(self):
        self.assertTrue(exceptions.__name__)

    def test_exceptions(self):
        """Ensure Dispatcher API Exception works on basic case (regression test)"""
        exceptions.DispatcherApiException(
            ref="", internal=exceptions.ExceptionCodeEnum.DPGetEventFailStatusCode, response=None
        )
