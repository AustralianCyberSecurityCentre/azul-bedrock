from unittest import mock, TestCase

from azul_bedrock import datastore
from azul_bedrock import exceptions_metastore
from azul_bedrock.exception_enums import ExceptionCodeEnum


class TestUtil(TestCase):
    def test_credentials_to_access(self):
        self.assertRaises(
            exceptions_metastore.BadCredentialsException, datastore.credentials_to_access, {"random": "stuff"}
        )
        self.assertRaises(
            exceptions_metastore.BadCredentialsException, datastore.credentials_to_access, {"format": "stuff"}
        )
        self.assertRaises(
            exceptions_metastore.BadCredentialsException, datastore.credentials_to_access, {"format": "basic"}
        )
        self.assertRaises(
            exceptions_metastore.BadCredentialsException, datastore.credentials_to_access, {"format": "jwt"}
        )
        self.assertRaises(
            exceptions_metastore.BadCredentialsException, datastore.credentials_to_access, {"format": "oauth"}
        )

        c = datastore.credentials_to_access({"format": "basic", "username": "user", "password": "pass"})
        self.assertEqual(("user", "pass"), c["http_auth"])

        c = datastore.credentials_to_access({"format": "jwt", "token": "tok"})
        self.assertEqual("tok", c["headers"]["Authorization"])

        c = datastore.credentials_to_access({"format": "oauth", "token": "password"})
        self.assertEqual("Bearer password", c["headers"]["Authorization"])

    @mock.patch("opensearchpy.OpenSearch")
    @mock.patch("azul_bedrock.datastore.credentials_to_access")
    def test_credentials_to_es(self, _cta, _es):
        _cta.side_effect = exceptions_metastore.BadCredentialsException(
            internal=ExceptionCodeEnum.MetastoreOpensearchCantGetUserAccount
        )
        self.assertRaises(
            exceptions_metastore.BadCredentialsException, datastore.credentials_to_es, {"unique": "blah"}
        )

        _cta.side_effect = None
        _cta.return_value = {"http_auth": ("user", "pass")}
        datastore.credentials_to_es({"unique": "blah"})
