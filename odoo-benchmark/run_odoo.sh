#!/bin/bash
# Run Odoo application with benchmark_plugin
echo "Starting Odoo with benchmark plugin on port 8126..."



SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCHMARKS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

ulimit -n 65536 2>/dev/null || ulimit -n 4096 2>/dev/null

export ODOO_HTTP_SOCKET_TIMEOUT=60
export ODOO_MAX_HTTP_THREADS=1000


exec uv run python3 "$BENCHMARKS_DIR/odoo/odoo-bin" \
    --addons-path="$SCRIPT_DIR,$SCRIPT_DIR/queue,$BENCHMARKS_DIR/odoo/addons" \
    -d odoo_benchmark \
    -i benchmark_plugin,queue_job \
    -u benchmark_plugin,queue_job \
    --load=base,web,queue_job \
    -p 8126 \
    --workers 0 \
    --db_maxconn 2048 \
    --limit-memory-soft 8589934592 \
    --limit-memory-hard 10737418240 \
    --max-cron-threads 0 \
    --limit-time-cpu 600 \
    --limit-time-real 600 \
    --log-level warn


