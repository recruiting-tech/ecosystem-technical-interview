"""Kafka publisher used by the outbox dispatcher (NOT by use cases directly).

Drains a single EventOutbox row to Kafka with Confluent Avro encoding.
Reads schemas from `schemas/` (the shared cross-service catalog) and trusts
that they've already been registered (`make register-schemas`). We do NOT
auto-register here — keeping the schema lifecycle out of the runtime path
makes incompatible changes loud at deploy time, not at first publish.
"""

from __future__ import annotations

import json
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from confluent_kafka import Producer
from confluent_kafka.schema_registry import SchemaRegistryClient
from confluent_kafka.schema_registry.avro import AvroSerializer
from confluent_kafka.serialization import MessageField, SerializationContext
from django.conf import settings
from fastavro import validation

from athlete_svc.core.domain.events import AthleteCreatedEvent


@dataclass(frozen=True)
class KafkaEventConfig:
    topic: str
    schema_subject: str
    schema_file: str  # filename inside settings.SCHEMAS_DIR


# Single source of truth for "this domain event class maps to this topic +
# schema subject + schema file." When you add a new event, you add an entry
# here; the publisher picks it up automatically.
KAFKA_EVENT_MAPPING: dict[type, KafkaEventConfig] = {
    AthleteCreatedEvent: KafkaEventConfig(
        topic="AthleteCreated.V1",
        schema_subject="AthleteCreated.V1-value",
        schema_file="AthleteCreated.V1.avsc",
    ),
}


class KafkaPublisher:
    """One instance per dispatcher run. Holds a Producer + SR client."""

    def __init__(self) -> None:
        self.producer = Producer(settings.KAFKA_PRODUCER_CONFIG)
        self.sr_client = SchemaRegistryClient(settings.SCHEMA_REGISTRY_CONFIG)
        # Cache schema text by subject so we don't re-read files / re-fetch.
        self._schema_cache: dict[str, str] = {}

    def publish_one(self, event: Any, metadata: dict[str, Any]) -> None:
        config = KAFKA_EVENT_MAPPING.get(type(event))
        if config is None:
            raise ValueError(
                f"No KAFKA_EVENT_MAPPING entry for {type(event).__name__}. "
                "Add one before publishing."
            )

        schema_str = self._load_schema(config)

        # Validate first — AvroSerializer happily serializes loose values.
        # An explicit validation step turns "weird Avro errors" into clear
        # "your event doesn't match the schema" errors.
        from dataclasses import asdict
        event_dict = _normalize(asdict(event))
        validation.validate(event_dict, json.loads(schema_str))

        serializer = AvroSerializer(
            self.sr_client,
            schema_str,
            conf={"auto.register.schemas": False, "use.latest.version": True},
        )
        ctx = SerializationContext(config.topic, MessageField.VALUE)
        value = serializer(event_dict, ctx)
        key = getattr(event, "key", None)
        if key is None:
            raise ValueError(
                f"{type(event).__name__} has no `key` attribute — "
                "every event must define one for partitioning"
            )
        headers = [("schema_subject", config.schema_subject.encode())]
        for k, v in (metadata or {}).items():
            if v is not None:
                headers.append((str(k), str(v).encode()))

        self.producer.produce(config.topic, value, str(key), headers=headers)

    def flush(self) -> None:
        self.producer.flush()

    def _load_schema(self, config: KafkaEventConfig) -> str:
        if config.schema_subject in self._schema_cache:
            return self._schema_cache[config.schema_subject]
        path = Path(settings.SCHEMAS_DIR) / config.schema_file
        text = path.read_text()
        self._schema_cache[config.schema_subject] = text
        return text


def _normalize(d: dict[str, Any]) -> dict[str, Any]:
    """fastavro validation expects native Python types. Avro datetime
    logical types accept the int representation our outbox already stored
    after dataclass serialization, but our outbox writes ISO strings. So
    we let AvroSerializer handle the int conversion when it sees a string
    and the schema is logical-type long timestamp-millis. fastavro's
    validate is permissive here so we just pass through."""
    return d
