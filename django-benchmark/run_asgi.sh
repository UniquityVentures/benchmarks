#!/bin/bash
# Run Django application using ASGI with Uvicorn
echo "Starting Django using ASGI (Uvicorn)..."
exec uv run uvicorn config.asgi:application --host 0.0.0.0 --port 8124 --workers 4 --lifespan off --log-level warning
