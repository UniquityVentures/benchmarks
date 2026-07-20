#!/bin/bash
# Run Frappe application with benchmark_app
echo "Starting Frappe with benchmark app on port 8127..."

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

export FRAPPE_SITE="localhost"
export FRAPPE_STREAM_LOGGING=1
ulimit -n 65536 2>/dev/null || ulimit -n 4096 2>/dev/null

# Ensure asset manifest exists (skip full yarn build for benchmark-only use)
mkdir -p "$SCRIPT_DIR/sites/assets"
[ -f "$SCRIPT_DIR/sites/assets/assets.json" ] || echo '{}' > "$SCRIPT_DIR/sites/assets/assets.json"

# 1. Bootstrap site if the database hasn't been initialized yet
#    Check for tabDocType — if missing, the framework SQL was never imported.
NEEDS_INIT=$(uv run python3 -c "
import frappe
frappe.init('localhost', sites_path='$SCRIPT_DIR/sites')
frappe.connect()
tables = frappe.db.get_tables(cached=False)
print('yes' if 'tabDocType' not in tables else 'no')
" 2>/dev/null)

if [ "$NEEDS_INIT" = "yes" ]; then
    echo "Database not initialized — running new-site to bootstrap Frappe tables..."
    uv run python3 -m frappe.utils.bench_helper frappe \
        new-site localhost \
        --no-setup-db \
        --admin-password admin \
        --install-app benchmark_app \
        --force
fi

# 2. Sync database schema (runs patches, syncs doctypes)
uv run python3 -m frappe.utils.bench_helper frappe --site localhost migrate

# 3. Start web server
exec uv run gunicorn benchmark_app.wsgi:application \
    --bind 0.0.0.0:8127 \
    --workers 4 \
    --worker-class geventwebsocket.gunicorn.workers.GeventWebSocketWorker \
    --timeout 600 \
    --log-level warn
