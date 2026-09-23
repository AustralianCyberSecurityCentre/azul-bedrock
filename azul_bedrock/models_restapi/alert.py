"""Models for alert endpoints and redis storage."""

import datetime

from pydantic import BaseModel, ConfigDict

from azul_bedrock.models_network import BinaryAction

# All models here have to map to gosrc/models/alert.go

# Redis key for the storing of the alerters configuration.
ALERTER_CONFIG_KEY = "config"
# Redis key for the storing of active alerts in redis
ALERTER_ALERT_KEY = "alerts"
# Redis Alerter database ID.
ALERTER_DB_ID = 4


class AlertRulePatch(BaseModel):
    """Alert rule."""

    model_config = ConfigDict(use_enum_values=True)

    endpoint: str | None = None
    event_type: BinaryAction | None = None
    plugin_name: str | None = None
    plugin_version: str | None = None
    source_name: str | None = None
    source_reference_key_values: dict[str, str] | None = None
    feature_name_values: dict[str, str] | None = None


class AlertRule(BaseModel):
    """Alert rule."""

    model_config = ConfigDict(use_enum_values=True)

    id: str
    endpoint: str
    event_type: BinaryAction
    plugin_name: str
    plugin_version: str
    source_name: str
    source_reference_key_values: dict[str, str]
    feature_name_values: dict[str, str]


class LoadedRules(BaseModel):
    """Loaded rules that are stored in redis."""

    rules: list[AlertRule]
    rules_compile_time: datetime.datetime


class AlertHit(BaseModel):
    """Alert hit that can be stored in redis."""

    rule: AlertRule
    sha256: str
