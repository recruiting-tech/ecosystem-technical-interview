"""Buffers domain events and writes them to the EventOutbox in a DB
transaction. Production wires this as the publisher passed to use cases —
the Kafka publisher is invoked separately by the poll_event_outbox worker.

This is the single most important contract in this service: the use case
never talks to Kafka directly. If we crash between creating an athlete row
and publishing to Kafka, the outbox row guarantees the event will be
delivered eventually.
"""

from dataclasses import asdict, dataclass, field, is_dataclass
from datetime import date, datetime
from typing import Any
from uuid import UUID

from django.db import transaction

from athlete_svc.core.infrastructure.persistence.models import EventOutbox


@dataclass
class _Envelope:
    event: Any
    metadata: dict[str, Any] = field(default_factory=dict)


def _serialize(value: Any) -> Any:
    if isinstance(value, datetime):
        return value.isoformat()
    if isinstance(value, date):
        return value.isoformat()
    if isinstance(value, UUID):
        return str(value)
    return value


def _to_payload(event: Any) -> dict[str, Any]:
    if not is_dataclass(event):
        raise TypeError(f"event {type(event)!r} is not a dataclass")
    return {k: _serialize(v) for k, v in asdict(event).items()}


class TransactionalOutboxPublisher:
    """Use as a context manager — flush on __exit__ writes the outbox rows
    inside a single atomic transaction.

        with TransactionalOutboxPublisher() as pub:
            athlete = Athlete.objects.create(...)
            pub.publish(AthleteCreatedEvent(id=str(athlete.id), ...))
    """

    def __init__(self, default_metadata: dict[str, Any] | None = None) -> None:
        self.default_metadata = default_metadata or {}
        self._envelopes: list[_Envelope] = []

    def publish(self, event: Any, metadata: dict[str, Any] | None = None) -> None:
        merged = {**self.default_metadata, **(metadata or {})}
        self._envelopes.append(_Envelope(event=event, metadata=merged))

    def flush(self) -> None:
        try:
            with transaction.atomic():
                for envelope in self._envelopes:
                    cls = type(envelope.event)
                    EventOutbox.objects.create(
                        event_class=f"{cls.__module__}.{cls.__name__}",
                        payload=_to_payload(envelope.event),
                        metadata=envelope.metadata,
                    )
        finally:
            self._envelopes.clear()

    def __enter__(self) -> "TransactionalOutboxPublisher":
        return self

    def __exit__(self, exc_type, exc, tb) -> None:
        # Only flush if no exception escaped the with-block. On exception
        # we discard envelopes so a failed write doesn't leak events.
        if exc_type is None:
            self.flush()
        else:
            self._envelopes.clear()
