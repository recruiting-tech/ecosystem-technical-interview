"""poll_event_outbox: drain the EventOutbox table to Kafka.

Runs as a Django management command:

    python manage.py poll_event_outbox

It uses a Postgres advisory lock to make sure only one instance is draining
at a time across multiple replicas. Each row is published to Kafka and
marked processed *atomically* (inside a transaction) so a crash mid-drain
just resumes — never duplicates.

In production you'd run this on a schedule (cron / k8s CronJob) every few
seconds, or as a long-running worker with a small sleep between iterations.
"""

from __future__ import annotations

import importlib
import logging
from datetime import datetime, timezone

from django.core.management.base import BaseCommand
from django.db import connection, transaction

from athlete_svc.core.infrastructure.events.publishers.kafka_publisher import KafkaPublisher
from athlete_svc.core.infrastructure.persistence.models import EventOutbox

logger = logging.getLogger(__name__)

ADVISORY_LOCK_KEY = 0xE0_0B_07  # arbitrary 32-bit constant; "EOBOX"


class Command(BaseCommand):
    help = "Drain the event outbox to Kafka."

    def add_arguments(self, parser) -> None:
        parser.add_argument(
            "--batch-size", type=int, default=500,
            help="Max rows to process per run (default: 500).",
        )

    def handle(self, *args, batch_size: int, **options) -> None:
        # Try to acquire the advisory lock. If another instance holds it, exit.
        with connection.cursor() as cur:
            cur.execute("SELECT pg_try_advisory_lock(%s)", [ADVISORY_LOCK_KEY])
            got = cur.fetchone()[0]
        if not got:
            logger.info("another poller holds the advisory lock; exiting")
            return

        try:
            self._drain(batch_size)
        finally:
            with connection.cursor() as cur:
                cur.execute("SELECT pg_advisory_unlock(%s)", [ADVISORY_LOCK_KEY])

    def _drain(self, batch_size: int) -> None:
        publisher = KafkaPublisher()
        try:
            rows = list(
                EventOutbox.objects.filter(processed_at__isnull=True)
                .order_by("created_at")[:batch_size]
            )
            for row in rows:
                with transaction.atomic():
                    event = self._rehydrate(row)
                    publisher.publish_one(event, metadata=row.metadata or {})
                    publisher.flush()
                    EventOutbox.objects.filter(id=row.id).update(
                        processed_at=datetime.now(timezone.utc)
                    )
                logger.info("dispatched event_class=%s id=%s", row.event_class, row.id)
            logger.info("drained %d outbox rows", len(rows))
        finally:
            publisher.flush()

    @staticmethod
    def _rehydrate(row: EventOutbox):
        module_path, class_name = row.event_class.rsplit(".", 1)
        cls = getattr(importlib.import_module(module_path), class_name)
        return cls(**row.payload)
