app_name = "benchmark_app"
app_title = "Benchmark App"
app_publisher = "Benchmark Team"
app_description = "Frappe app for benchmarking performance"
app_email = "benchmark@example.com"
app_license = "MIT"

# API routing is handled by the WSGI middleware in benchmark_app/wsgi.py
# which rewrites clean paths (/api/articles/) to /api/method/... before
# they reach Frappe's API router.

