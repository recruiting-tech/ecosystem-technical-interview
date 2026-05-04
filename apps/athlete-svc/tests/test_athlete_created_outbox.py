"""End-to-end test of the AthleteCreated event flow:

  POST /athletes
    → CreateAthleteUseCase.execute (atomic: Athlete row + outbox row)
    → poll_event_outbox management command (atomic: Kafka publish + mark processed)
    → Kafka receives an AthleteCreated.V1 message with the right Avro payload

Tests run against the live compose stack — Postgres, Kafka, Schema Registry —
which is the convention this repo enforces. If you're tempted to swap in a
Mock Producer here, please don't: every test we've debugged in this codebase
that "passed but broke in prod" was a mocked broker.
"""

from __future__ import annotations

import time

import pytest
from django.core.management import call_command
from django.test import Client

from athlete_svc.core.infrastructure.persistence.models import Athlete, EventOutbox

pytestmark = pytest.mark.integration

TOPIC = "AthleteCreated.V1"


def _wait_for_event(consumer, deserializer, topic: str, athlete_id: str, timeout_s: float = 10.0):
    """Drain messages from the topic until we find one matching athlete_id.
    Tolerates leftover messages from prior local test runs."""
    consumer.subscribe([topic])
    deadline = time.time() + timeout_s
    while time.time() < deadline:
        msg = consumer.poll(1.0)
        if msg is None or msg.error():
            continue
        decoded = deserializer(msg.value(), topic)
        if decoded.get("id") == athlete_id:
            return msg, decoded
    raise AssertionError(f"no message with id={athlete_id} on {topic} within {timeout_s}s")


def test_create_athlete_publishes_event_via_outbox(kafka_consumer, avro_deserializer):
    # Subscribe BEFORE producing so we don't miss the message.
    kafka_consumer.subscribe([TOPIC])

    client = Client()
    resp = client.post(
        "/athletes",
        data={
            "first_name": "Ada",
            "last_name": "Lovelace",
            "sport": "basketball",
            "email": "ada@example.com",
            "date_of_birth": "2008-12-10",
        },
        content_type="application/json",
    )
    assert resp.status_code == 201, resp.content
    athlete_id = resp.json()["id"]

    # The athlete row landed.
    assert Athlete.objects.filter(id=athlete_id).exists()
    # Exactly one unprocessed outbox row.
    outbox_qs = EventOutbox.objects.filter(processed_at__isnull=True)
    assert outbox_qs.count() == 1
    row = outbox_qs.first()
    assert row.event_class.endswith(".AthleteCreatedEvent")
    assert "email" not in row.payload, "PII must not be on the event payload"

    # Run the dispatcher; it should publish exactly one message.
    call_command("poll_event_outbox")

    # Outbox row is now processed.
    row.refresh_from_db()
    assert row.processed_at is not None

    # And the Kafka topic has the message with the expected shape.
    msg, decoded = _wait_for_event(kafka_consumer, avro_deserializer, TOPIC, athlete_id)
    assert decoded["first_name"] == "Ada"
    assert decoded["last_name"] == "Lovelace"
    assert decoded["sport"] == "basketball"
    assert "email" not in decoded
    # Kafka key carries the partition key (athlete id), not a UUID.
    assert msg.key().decode() == athlete_id
