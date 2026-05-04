"""AthleteCreatedEvent.

PII contract: this event MUST NOT carry email, phone, DOB, or any other
PII. Downstream consumers fetch sensitive data on demand from athlete-svc
under a separate auth boundary. Keeping events PII-free lets us replay
topics across environments and into analytics without an extra scrub step.
"""

from dataclasses import dataclass


@dataclass(frozen=True)
class AthleteCreatedEvent:
    id: str
    first_name: str
    last_name: str
    sport: str
    occurred_at: int  # milliseconds since unix epoch (matches schema's long/timestamp-millis)

    @property
    def key(self) -> str:
        return self.id
