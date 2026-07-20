#!/bin/bash
# Run Odoo application with benchmark_plugin
echo "Starting Odoo with benchmark plugin on port 8126..."

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BENCHMARKS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

exec uv run python3 "$BENCHMARKS_DIR/odoo/odoo-bin" \
    --addons-path="$SCRIPT_DIR,$BENCHMARKS_DIR/odoo/addons" \
    -d odoo_benchmark \
    -i benchmark_plugin \
    -p 8126 \
    --workers 4 \
    --max-cron-threads 0 \
    --limit-time-cpu 600 \
    --limit-time-real 600 \
    --log-level warn
