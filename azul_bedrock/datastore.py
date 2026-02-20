"""Datastore common functionallity."""

import copy

import httpx
import opensearchpy

from azul_bedrock import exceptions_bedrock, exceptions_metastore, settings
from azul_bedrock.exception_enums import ExceptionCodeEnum
from azul_bedrock.models_auth import CredentialFormat, Credentials


def credentials_to_access(c: Credentials) -> dict:
    """Return credentials in format used to get access."""
    if type(c) != Credentials:
        raise exceptions_bedrock.AzulValueError(
            internal=ExceptionCodeEnum.BedrockInvalidCredentialType, parameters={"type": str(type(c))}
        )
    access = {}
    raise_bad_creds = True
    if c.format == CredentialFormat.basic and c.username and c.password:
        access["http_auth"] = (c.username, c.password)
        raise_bad_creds = False
    elif c.format == CredentialFormat.jwt and c.token:
        access["headers"] = {"Authorization": c.token}
        raise_bad_creds = False
    elif c.format == CredentialFormat.oauth and c.token:
        access["headers"] = {"Authorization": f"Bearer {c.token}"}
        raise_bad_creds = False

    if c.format == CredentialFormat.none:
        raise exceptions_metastore.BadCredentialsException(
            internal=ExceptionCodeEnum.MetastoreSearchDataBadCredentials,
            parameters={"credential_format": str(c.format)},
        )
    if raise_bad_creds:
        print(c.model_dump())
        raise exceptions_metastore.BadCredentialsException(
            internal=ExceptionCodeEnum.BedrockBadOpensearchCredential,
            parameters={"method": c.format},
        )

    return access


def credentials_to_es(c: Credentials) -> opensearchpy.OpenSearch:
    """Turn credentials into an opensearch object and cache. Invalidate after some time."""
    access = credentials_to_access(c)
    s = settings.get_opensearch()
    # unfortunately opensearchpy does not use requests library, but urlli3 directly
    # we have to manually reference certificates not found in Certifi for consistency
    # as we query opensearch via both requests and opensearchpy
    if not s.certificate_verification:
        access["verify_certs"] = False
        access["ssl_show_warn"] = False
    access["timeout"] = 120

    # enable transport compression
    return opensearchpy.OpenSearch(hosts=s.opensearch_url, http_compress=True, **access)


def get_user_account(user_auth: dict) -> dict:
    """Get user account information from opensearch."""
    s = settings.get_opensearch()
    # opensearch path to get account information
    api_path = "_plugins/_security/api/account"

    raw = copy.deepcopy(user_auth)
    if "http_auth" in raw:
        raw["auth"] = httpx.BasicAuth(*raw.pop("http_auth"))
    try:
        endpoint = f"{s.opensearch_url}/{api_path}"
        resp = httpx.get(endpoint, verify=s.certificate_verification, timeout=10, **raw)
        if resp.status_code not in {200, 201}:
            raise exceptions_bedrock.BaseAzulException(
                internal=ExceptionCodeEnum.MetastoreOpensearchCantGetUserAccountInner,
                parameters={"status_code": str(resp.status_code), "response_text": str(resp.content)},
            )
        return resp.json()
    except Exception as e:
        raise exceptions_bedrock.BaseAzulException(
            ref=f"could not get user account: {str(e)}",
            internal=ExceptionCodeEnum.MetastoreOpensearchCantGetUserAccount,
            parameters={"inner_exception": str(e)},
        ) from e
