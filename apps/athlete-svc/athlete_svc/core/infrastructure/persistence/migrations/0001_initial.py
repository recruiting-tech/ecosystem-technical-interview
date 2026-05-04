import uuid

import django.db.models.deletion
import django.utils.timezone
from django.db import migrations, models


class Migration(migrations.Migration):

    initial = True

    dependencies = []

    operations = [
        migrations.CreateModel(
            name="Athlete",
            fields=[
                ("id", models.UUIDField(default=uuid.uuid4, editable=False, primary_key=True, serialize=False)),
                ("first_name", models.CharField(max_length=100)),
                ("last_name", models.CharField(max_length=100)),
                ("sport", models.CharField(max_length=50)),
                ("email", models.EmailField(max_length=254)),
                ("date_of_birth", models.DateField()),
                ("created_at", models.DateTimeField(default=django.utils.timezone.now)),
                ("updated_at", models.DateTimeField(auto_now=True)),
            ],
            options={
                "db_table": "athletes",
                "indexes": [models.Index(fields=["sport"], name="athletes_sport_idx")],
            },
        ),
        migrations.CreateModel(
            name="EventOutbox",
            fields=[
                ("id", models.UUIDField(default=uuid.uuid4, editable=False, primary_key=True, serialize=False)),
                ("event_class", models.CharField(max_length=255)),
                ("payload", models.JSONField(blank=True, default=dict)),
                ("metadata", models.JSONField(blank=True, default=dict)),
                ("created_at", models.DateTimeField(default=django.utils.timezone.now)),
                ("processed_at", models.DateTimeField(blank=True, null=True)),
            ],
            options={
                "db_table": "event_outbox",
                "indexes": [models.Index(fields=["processed_at", "created_at"], name="outbox_unprocessed_idx")],
            },
        ),
    ]
