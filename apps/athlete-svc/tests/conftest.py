"""Shared fixtures.

Tests in this repo are *not* mocked at the broker boundary — that
convention is load-bearing: every "passed but broke in prod" bug we've
debugged in this codebase was a mocked broker. The goal is to catch
wire-format / schema / serialization bugs that mocks would hide.

But running tests against the dev compose stack would leak state
(Kafka topic offsets in particular have no transactional rollback). So
each test session spins up its own throwaway Kafka + Schema Registry
via Redpanda (testcontainers). This mirrors apps/api-svc's
testcontainers-Postgres pattern. Tests still hit a real broker — just
not the one shared with `make dev`.

Cost: ~8s of session startup. Benefit: `make test` leaves the dev
compose stack untouched.

Requirements: Docker available. You do *not* need `make up` to run
these tests.
"""

from __future__ import annotations

import json
import os
import time
import urllib.error
import urllib.request
import uuid
from pathlib import Path

# ---------------------------------------------------------------------------
# Per-session Redpanda container (Kafka + Schema Registry in one image).
# This block runs at conftest-import time, BEFORE Django settings load,
# so we can override KAFKA_BROKERS / SCHEMA_REGISTRY_URL via env vars.
# ---------------------------------------------------------------------------

from testcontainers.kafka import RedpandaContainer

_REPO_ROOT = Path(__file__).resolve().parents[3]
_SCHEMAS_DIR = _REPO_ROOT / "schemas"

_redpanda = RedpandaContainer()
_redpanda.start()

_BOOTSTRAP = _redpanda.get_bootstrap_server()
_SR_URL = _redpanda.get_schema_registry_address()

# Set env vars in case anything reads them directly. Not load-bearing —
# pytest-django imports settings during its pytest_configure (before this
# conftest's module body runs), so settings.KAFKA_PRODUCER_CONFIG is
# already populated from os.environ at *that* moment with dev defaults.
# We have to mutate the loaded dicts in place (below) to redirect.
os.environ["KAFKA_BROKERS"] = _BOOTSTRAP
os.environ["SCHEMA_REGISTRY_URL"] = _SR_URL

from django.conf import settings as _django_settings  # noqa: E402

_django_settings.KAFKA_PRODUCER_CONFIG["bootstrap.servers"] = _BOOTSTRAP
_django_settings.SCHEMA_REGISTRY_CONFIG["url"] = _SR_URL


def _register_schemas() -> None:
    """Register every schemas/*.avsc as <stem>-value with BACKWARD compat.

    Mirrors what scripts/register-schemas.sh does for the dev SR. We
    default everything to BACKWARD because that's what schemas.yaml uses
    today; if a future subject needs a different policy, parse it from
    the YAML.
    """
    sr_url = _SR_URL
    deadline = time.time() + 30
    while True:
        try:
            urllib.request.urlopen(f"{sr_url}/subjects", timeout=2).read()
            break
        except (urllib.error.URLError, ConnectionError, TimeoutError):
            if time.time() > deadline:
                raise RuntimeError(f"schema registry unreachable at {sr_url}")
            time.sleep(0.5)

    headers = {"Content-Type": "application/vnd.schemaregistry.v1+json"}
    for avsc in sorted(_SCHEMAS_DIR.glob("*.avsc")):
        subject = f"{avsc.stem}-value"
        urllib.request.urlopen(urllib.request.Request(
            f"{sr_url}/config/{subject}",
            method="PUT",
            headers=headers,
            data=json.dumps({"compatibility": "BACKWARD"}).encode(),
        ))
        urllib.request.urlopen(urllib.request.Request(
            f"{sr_url}/subjects/{subject}/versions",
            method="POST",
            headers=headers,
            data=json.dumps({
                "schemaType": "AVRO",
                "schema": avsc.read_text(),
            }).encode(),
        ))


_register_schemas()


def pytest_sessionfinish(session, exitstatus):  # noqa: ARG001
    _redpanda.stop()


# ---------------------------------------------------------------------------
# Existing fixtures (now talking to the ephemeral broker).
# ---------------------------------------------------------------------------

import pytest
from confluent_kafka import Consumer
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
