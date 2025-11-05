import unittest

from azul_bedrock import models_api


class TestBasic(unittest.TestCase):
    def test_import(self):
        self.assertTrue(models_api.__name__)
