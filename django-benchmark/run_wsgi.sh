#!/bin/bash
# Run Django application using WSGI with Gunicorn
echo "Starting Django using WSGI (Gunicorn)..."
exec uv run gunicorn config.wsgi:application \
  --bind 0.0.0.0:8125 \
  --workers 4 \
  --worker-class gevent \
  --worker-connections 1000
