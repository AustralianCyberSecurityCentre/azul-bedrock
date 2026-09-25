"""Models for alert endpoints and redis storage."""

from enum import StrEnum
from typing import Annotated

from pydantic import AwareDatetime, BaseModel, ConfigDict, PlainSerializer

from azul_bedrock.models_network import BinaryAction, StatusEnum

# All models here have to map to gosrc/models/alert.go

# Redis key for the storing of the alerters configuration.
ALERTER_CONFIG_KEY = "config"
# Redis key for the storing of active alerts in redis
ALERTER_ALERT_KEY = "alerts"
# Redis Alerter database ID.
ALERTER_DB_ID = 4


class AlertRuleBase(BaseModel):
    """Base alert Rule."""

    # Message that should be displayed as part of the alert message when the alert is triggered.
    alert_message: str = ""
    # Status of the status event to match on.
    status: StatusEnum | None = None
    # Type of event to alert on.
    event_type: BinaryAction | None = None
    # Case sensitive plugin name to alert on.
    plugin_name: str | None = None
    # Case sensitive version of the plugin to alert on.
    plugin_version: str | None = None
    # Name of the source that the alert should trigger on.
    source_name: str | None = None
    # Source reference key/values that the alert should trigger on.
    source_reference_key_values: dict[str, str] | None = None
    # Feature name/value pairs that the alert should trigger on.
    feature_name_values: dict[str, str] | None = None


class AlertRulePatch(AlertRuleBase):
    """Alert rule."""

    model_config = ConfigDict(use_enum_values=True)

    id: str
    webhook_id: str | None = None


class AlertRuleCreate(AlertRuleBase):
    """Alert rule creation.

    NOTE that most fields are optional but if they aren't set the alert will trigger on everything.
    The trigger conditions work  like an AND filter, so all conditions must be met for the alert to fire.
    """

    model_config = ConfigDict(use_enum_values=True)

    # Id of the webhook to send the alert message to
    webhook_id: str


class AlertRule(AlertRuleCreate):
    """Alert rule."""

    id: str


class LoadedRules(BaseModel):
    """Loaded rules that are stored in redis."""

    rules: list[AlertRule]
    rules_compile_time: Annotated[
        AwareDatetime,
        PlainSerializer(lambda v: v.isoformat() if v else "", return_type=str),
    ]


class AlertHit(BaseModel):
    """Alert hit that can be stored in redis."""

    rule_id: str
    webhook_id: str
    alert_message: str = ""
    alert_attempt: int = 0
    sha256: str


class SupportedWebhookType(StrEnum):
    """Enums that can be used and are supported by alerter."""

    MSTeams = "msteams"
    Mattermost = "mattermost"


class WebhookMappingApi(BaseModel):
    """Mapping for wheat webhook should be used when sending out alerts, for displaying in the restapi."""

    model_config = ConfigDict(use_enum_values=True)

    id: str
    description: str = ""
    webhook_type: SupportedWebhookType = SupportedWebhookType.Mattermost
    # Field used when posting a message to the webhook.
    message_field: str = "text"


class WebhookMapping(WebhookMappingApi):
    """Mapping for wheat webhook should be used when sending out alerts."""

    url: str
