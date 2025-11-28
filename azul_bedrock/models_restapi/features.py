"""Data models for feature restapi."""

from pydantic import ConfigDict

from azul_bedrock.models_restapi.basic import BaseModelRepr


class FeatureMulticountRet(BaseModelRepr):
    """Return data for multicount feature operation."""

    name: str
    values: int | None = None
    entities: int | None = None


class ValueCountItem(BaseModelRepr):
    """Value item for multicount."""

    name: str
    value: str

    model_config = ConfigDict(coerce_numbers_to_str=True)


class ValueCountRet(ValueCountItem):
    """Return data for multicounut value operation."""

    entities: int


class ValuePartCountItem(BaseModelRepr):
    """Value item for multicount."""

    value: str
    part: str

    model_config = ConfigDict(coerce_numbers_to_str=True)


class ValuePartCountRet(ValuePartCountItem):
    """Return data for multicounut value operation."""

    entities: int


class ReadFeatureValueTagsTag(BaseModelRepr):
    """Feature value tag in system."""

    tag: str
    num_feature_values: int


class ReadFeatureValueTags(BaseModelRepr):
    """Collection of tags in the system, including entity counts."""

    tags: list[ReadFeatureValueTagsTag]
    num_tags: int


class FeatureValueTag(BaseModelRepr):
    """A tag attached to a feature value."""

    feature_name: str
    feature_value: str
    tag: str
    type: str

    owner: str | None = None
    timestamp: str
    security: str

    model_config = ConfigDict(coerce_numbers_to_str=True)


class ReadFeatureTagValues(BaseModelRepr):
    """Feature value tags."""

    items: list[FeatureValueTag]


class FeatureDescription(BaseModelRepr):
    """Description of feature."""

    author_type: str | None = None
    author_name: str | None = None
    author_version: str | None = None
    timestamp: str | None = None
    desc: str | None = None
    type: str | None = None


class Feature(BaseModelRepr):
    """Compilation of information for a specific feature."""

    name: str
    descriptions: list[FeatureDescription] | None = None
    tags: list[str] | None = None
    security: list[str] | None = None


class Features(BaseModelRepr):
    """Features produced by plugins."""

    items: list[Feature]


class ReadFeatureValuesValue(BaseModelRepr):
    """Value for feature."""

    name: str
    value: str
    tags: list[FeatureValueTag] = []
    newest_processed: str
    score: int = 0

    model_config = ConfigDict(coerce_numbers_to_str=True)


class ReadFeatureValues(BaseModelRepr):
    """Set of values for a specific feature."""

    name: str
    type: str | None = None
    values: list[ReadFeatureValuesValue]
    is_search_complete: bool = False
    after: str | None = None  # pagination key for next request
    is_total_approx: bool = False  # is the total an approximation, occurs if a term query is done.
    total: int | None = None  # on first request only, expected number of items if all pages are collected


class FeaturePivotRequest(BaseModelRepr):
    """Format of the request body for a feature pivot request."""

    feature_name: str
    feature_value: str

    model_config = ConfigDict(coerce_numbers_to_str=True)


class FeaturePivotValueCount(BaseModelRepr):
    """The value and count for a binary."""

    feature_value: str
    entity_count: str

    model_config = ConfigDict(coerce_numbers_to_str=True)


class FeaturePivotNameWithValueCount(BaseModelRepr):
    """Feature name, value and total count."""

    feature_name: str
    feature_description: str
    values_and_counts: list[FeaturePivotValueCount]


class FeaturePivotResponse(BaseModelRepr):
    """Response from the Feature pivot containing a list of matching features and their counts."""

    feature_value_counts: list[FeaturePivotNameWithValueCount] = []
    incomplete_query: bool = False
    reason: str = ""
