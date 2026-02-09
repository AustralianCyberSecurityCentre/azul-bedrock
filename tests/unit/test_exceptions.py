import unittest

from azul_bedrock import exceptions_bedrock


class TestBasic(unittest.TestCase):
    def test_import(self):
        self.assertTrue(exceptions_bedrock.__name__)

    def test_exceptions(self):
        """Ensure Dispatcher API Exception works on basic case (regression test)"""
        exceptions_bedrock.DispatcherApiException(
            ref="", internal=exceptions_bedrock.ExceptionCodeEnum.DPGetEventFailStatusCode, response=None
        )
