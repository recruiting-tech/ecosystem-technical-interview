"""Django models. These ARE the repositories — use cases call
Athlete.objects.create / .get etc. directly. No abstract base classes."""

import uuid

from django.db import models
from django.utils import timezone


class Athlete(models.Model):
    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    first_name = models.CharField(max_length=100)
    last_name = models.CharField(max_length=100)
    sport = models.CharField(max_length=50)
    email = models.EmailField()
    date_of_birth = models.DateField()
    created_at = models.DateTimeField(default=timezone.now)
    updated_at = models.DateTimeField(auto_now=True)

    class Meta:
        db_table = "athletes"
        indexes = [models.Index(fields=["sport"])]


class EventOutbox(models.Model):
    """Transactional outbox row. Written in the same DB transaction as the
    business write so the event is durable before any HTTP response is sent.
    The poll_event_outbox management command drains rows to Kafka and marks
    them processed.
    """

    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    event_class = models.CharField(max_length=255, help_text="Fully qualified Python path of the event dataclass.")
    payload = models.JSONField(default=dict, blank=True)
    metadata = models.JSONField(default=dict, blank=True)
    created_at = models.DateTimeField(default=timezone.now)
    processed_at = models.DateTimeField(null=True, blank=True)

    class Meta:
        db_table = "event_outbox"
        indexes = [models.Index(fields=["processed_at", "created_at"])]
