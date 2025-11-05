import unittest

from azul_bedrock import models_auth


class TestBasic(unittest.TestCase):
    def test_import(self):
        m = models_auth.UserInfo(unique_id="test")
        self.assertTrue(m)
