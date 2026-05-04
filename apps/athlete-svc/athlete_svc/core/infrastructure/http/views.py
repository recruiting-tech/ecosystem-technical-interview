"""HTTP views. Thin — they parse input, call a use case, format output.

We keep the publisher wiring in the view rather than a service container
because the wiring is small enough that visibility wins over indirection.
"""

from __future__ import annotations

from datetime import date

from rest_framework import status
from rest_framework.exceptions import ValidationError
from rest_framework.response import Response
from rest_framework.views import APIView

from athlete_svc.core.application.create_athlete import (
    CreateAthleteParams,
    CreateAthleteUseCase,
)
from athlete_svc.core.infrastructure.events.publishers.transactional_outbox_publisher import (
    TransactionalOutboxPublisher,
)
from athlete_svc.core.infrastructure.persistence.models import Athlete


def _serialize(a: Athlete) -> dict:
    return {
        "id": str(a.id),
        "first_name": a.first_name,
        "last_name": a.last_name,
        "sport": a.sport,
        "email": a.email,
        "date_of_birth": a.date_of_birth.isoformat(),
        "created_at": a.created_at.isoformat(),
    }


class AthletesView(APIView):
    def post(self, request) -> Response:
        body = request.data or {}
        required = ("first_name", "last_name", "sport", "email", "date_of_birth")
        missing = [f for f in required if not body.get(f)]
        if missing:
            raise ValidationError({"missing_fields": missing})

        try:
            dob = date.fromisoformat(body["date_of_birth"])
        except (TypeError, ValueError) as e:
            raise ValidationError({"date_of_birth": "expected YYYY-MM-DD"}) from e

        params = CreateAthleteParams(
            first_name=body["first_name"],
            last_name=body["last_name"],
            sport=body["sport"],
            email=body["email"],
            date_of_birth=dob,
        )
        with TransactionalOutboxPublisher() as pub:
            uc = CreateAthleteUseCase(athlete_model=Athlete, publisher=pub)
            athlete = uc.execute(params)
        return Response(_serialize(athlete), status=status.HTTP_201_CREATED)


class AthleteDetailView(APIView):
    def get(self, request, athlete_id) -> Response:
        try:
            athlete = Athlete.objects.get(pk=athlete_id)
        except Athlete.DoesNotExist:
            return Response(
                {"code": "not_found", "message": "athlete not found"},
                status=status.HTTP_404_NOT_FOUND,
            )
        return Response(_serialize(athlete))
