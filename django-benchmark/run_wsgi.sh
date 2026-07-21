#!/bin/bash
# Run Django application using WSGI with Gunicorn and Celery background workers

export USE_CELERY=1

echo "Starting Celery worker in background..."
uv run celery -A config worker --loglevel=warning --concurrency=16 &
CELERY_PID=$!

trap "kill $CELERY_PID 2>/dev/null" EXIT

echo "Starting Django using WSGI (Gunicorn)..."
exec uv run python -m gunicorn config.wsgi:application \
  --bind 0.0.0.0:8125 \
  --workers 4 \
  --worker-class gthread \
  --threads 4
