"""
Django settings for athlete-svc. Minimal — keeps the surface area readable
for an interview.
"""

import os
from pathlib import Path

BASE_DIR = Path(__file__).resolve().parent.parent
REPO_ROOT = BASE_DIR.parent.parent  # ecosystem-technical-interview/

SECRET_KEY = os.environ.get(
    "DJANGO_SECRET_KEY",
    "dev-only-secret-do-not-use-in-prod",
)
DEBUG = os.environ.get("DEBUG", "1") == "1"
ALLOWED_HOSTS = ["*"]

INSTALLED_APPS = [
    "django.contrib.contenttypes",
    "django.contrib.auth",
    "rest_framework",
    "athlete_svc.core",
]

MIDDLEWARE = [
    "django.middleware.common.CommonMiddleware",
]

ROOT_URLCONF = "athlete_svc.urls"

WSGI_APPLICATION = "athlete_svc.wsgi.application"

# Migrations live alongside the persistence adapter rather than at the
# default location so the package layout reflects the architecture.
MIGRATION_MODULES = {
    "core": "athlete_svc.core.infrastructure.persistence.migrations",
}

DATABASES = {
    "default": {
        "ENGINE": "django.db.backends.postgresql",
        "NAME": os.environ.get("DB_NAME", "athlete_svc"),
        "USER": os.environ.get("DB_USER", "emc"),
        "PASSWORD": os.environ.get("DB_PASSWORD", "emc"),
        "HOST": os.environ.get("DB_HOST", "localhost"),
        "PORT": os.environ.get("DB_PORT", "5532"),
    }
}

LANGUAGE_CODE = "en-us"
TIME_ZONE = "UTC"
USE_I18N = False
USE_TZ = True
DEFAULT_AUTO_FIELD = "django.db.models.BigAutoField"

# Kafka / Schema Registry config used by the publisher and the outbox poller.
KAFKA_PRODUCER_CONFIG = {
    "bootstrap.servers": os.environ.get("KAFKA_BROKERS", "localhost:19092"),
    "client.id": "athlete-svc",
    "enable.idempotence": True,
    "acks": "all",
}
SCHEMA_REGISTRY_CONFIG = {
    "url": os.environ.get("SCHEMA_REGISTRY_URL", "http://localhost:18081"),
}

# Where the canonical Avro schemas live (shared across services).
SCHEMAS_DIR = REPO_ROOT / "schemas"

LOGGING = {
    "version": 1,
    "disable_existing_loggers": False,
    "formatters": {
        "simple": {"format": "%(asctime)s %(levelname)s %(name)s :: %(message)s"},
    },
    "handlers": {
        "console": {"class": "logging.StreamHandler", "formatter": "simple"},
    },
    "root": {"handlers": ["console"], "level": "INFO"},
}
