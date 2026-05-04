"""Domain event publisher protocol. Production wires this to the
TransactionalOutboxPublisher (writes to outbox in the same DB transaction);
the poll_event_outbox management command later flushes outbox rows to Kafka.

This indirection is what makes our event delivery at-least-once: the HTTP
request can never observe a state where the row exists but the event was
"lost" before the broker acknowledged it.
"""

from typing import Any, Protocol


class DomainEventPublisher(Protocol):
    def publish(self, event: Any, metadata: dict | None = None) -> None: ...
    def flush(self) -> None: ...
