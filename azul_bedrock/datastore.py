"""Datastore common functionallity."""

import opensearchpy

from azul_bedrock import exceptions_metastore, settings
from azul_bedrock.exception_enums import ExceptionCodeEnum


def credentials_to_access(c: dict) -> dict:
    """Return credentials in format used to get access."""
    access = {}
    format = c.get("format")
    try:
        if format == "basic":
            access["http_auth"] = (c["username"], c["password"])
        elif format == "jwt":
            access["headers"] = {"Authorization": c["token"]}
        elif format == "oauth":
            access["headers"] = {"Authorization": f"Bearer {c['token']}"}
        else:
            raise exceptions_metastore.BadCredentialsException(
                ref=f"unrecognised credential format: {c['format']}",
                internal=ExceptionCodeEnum.MetastoreSearchDataBadCredentials,
                parameters={"credential_format": str(c["format"])},
            )
    except KeyError as e:
        raise exceptions_metastore.BadCredentialsException(
            ref=f"missing/bad parameter: {str(e)}",
            internal=ExceptionCodeEnum.MetastoreSearchDataMissingOrBadParameters,
            parameters={"inner_exception": str(e)},
        ) from e

    return access


def credentials_to_es(c: dict) -> opensearchpy.OpenSearch:
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
