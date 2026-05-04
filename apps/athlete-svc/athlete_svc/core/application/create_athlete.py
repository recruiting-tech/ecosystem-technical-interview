"""CreateAthleteUseCase. The Django Athlete model itself is the repo —
no abstract repository layer. The use case is responsible for pairing the
DB write with an outbox event in a single atomic transaction.
"""

from __future__ import annotations

import time
from dataclasses import dataclass
from datetime import date

from django.db import transaction

from athlete_svc.core.domain.events import AthleteCreatedEvent
from athlete_svc.core.domain.events.publisher import DomainEventPublisher
from athlete_svc.core.infrastructure.persistence.models import Athlete


@dataclass(frozen=True)
class CreateAthleteParams:
    first_name: str
    last_name: str
    sport: str
    email: str
    date_of_birth: date


class CreateAthleteUseCase:
    """Creates an athlete and queues an AthleteCreatedEvent in the outbox.
    Both writes are wrapped in a single DB transaction — either both land
    or neither does.
    """

    def __init__(self, athlete_model: type[Athlete], publisher: DomainEventPublisher) -> None:
        self.athlete_model = athlete_model
        self.publisher = publisher

    def execute(self, params: CreateAthleteParams) -> Athlete:
        with transaction.atomic():
            athlete = self.athlete_model.objects.create(
                first_name=params.first_name,
                last_name=params.last_name,
                sport=params.sport,
                email=params.email,
                date_of_birth=params.date_of_birth,
            )
            self.publisher.publish(
                AthleteCreatedEvent(
                    id=str(athlete.id),
                    first_name=athlete.first_name,
                    last_name=athlete.last_name,
                    sport=athlete.sport,
                    occurred_at=int(time.time() * 1000),
                )
            )
        return athlete
