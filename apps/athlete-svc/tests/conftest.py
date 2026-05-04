"""Shared fixtures.

Tests in this repo are *not* mocked at the broker boundary. They run
against the live dev compose stack (postgres / kafka / schema-registry).
This is the convention every event-publishing test follows — the goal is
to catch wire-format / schema / serialization bugs that mocks would hide.

Run `make up && make register-schemas` from the repo root before running.
"""

from __future__ import annotations

import json
import uuid
from pathlib import Path

import pytest
from confluent_kafka import Consumer, TopicPartition
from confluent_kafka.schema_registry import SchemaRegistryClient
from confluent_kafka.schema_registry.avro import AvroDeserializer
from confluent_kafka.serialization import MessageField, SerializationContext
from django.conf import settings


@pytest.fixture(autouse=True)
def _ensure_test_db_clean(db):
    """pytest-django gives us a clean test database per test."""
    yield


@pytest.fixture
def kafka_consumer():
    """Confluent Consumer subscribed to a unique consumer group so it
    starts at the latest offset and doesn't see prior test pollution.
    """
    cfg = {
        **settings.KAFKA_PRODUCER_CONFIG,
        "group.id": f"athlete-svc-tests-{uuid.uuid4()}",
        "auto.offset.reset": "earliest",
        "enable.auto.commit": False,
    }
    # Strip producer-only keys.
    for k in ("client.id", "enable.idempotence", "acks"):
        cfg.pop(k, None)
    cfg["bootstrap.servers"] = settings.KAFKA_PRODUCER_CONFIG["bootstrap.servers"]
    consumer = Consumer(cfg)
    yield consumer
    consumer.close()


@pytest.fixture
def avro_deserializer():
    """Returns a function: bytes -> dict, decoding Confluent-format Avro
    using the schema registry to look up the schema by ID embedded in the
    payload header.
    """
    sr = SchemaRegistryClient(settings.SCHEMA_REGISTRY_CONFIG)

    def _decode(value: bytes, topic: str) -> dict:
        deser = AvroDeserializer(sr)
        return deser(value, SerializationContext(topic, MessageField.VALUE))

    return _decode


@pytest.fixture
def schema_text():
    """Loads an Avro schema by filename from schemas/."""
    def _load(filename: str) -> dict:
        return json.loads((Path(settings.SCHEMAS_DIR) / filename).read_text())
    return _load
