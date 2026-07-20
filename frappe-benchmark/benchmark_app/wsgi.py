"""WSGI middleware that rewrites clean benchmark API paths to Frappe's
``/api/method/<dotted.path>`` convention **and** provides a WebSocket
echo endpoint at ``/api/ws/``.

Frappe's API router only handles ``/api/method/...`` and ``/api/resource/...``.
The benchmark harness calls clean paths like ``/api/articles/``, so this
middleware rewrites them at the WSGI level before Frappe ever sees them.

For WebSockets, gunicorn must be started with the GeventWebSocketWorker
(``--worker-class geventwebsocket.gunicorn.workers.GeventWebSocketWorker``).
When a WebSocket upgrade hits ``/api/ws/``, we handle it directly here
without involving Frappe at all.
"""

import json
import re

import frappe.app

from benchmark_app.api import LARGE_RESPONSE, MEDIUM_RESPONSE, SMALL_RESPONSE

# Pre-serialise the echo payloads once at import time.
_WS_RESPONSES = {
    "small":  json.dumps(SMALL_RESPONSE),
    "medium": json.dumps(MEDIUM_RESPONSE),
    "large":  json.dumps(LARGE_RESPONSE),
}

# Ordered list of (regex, rewritten_path) pairs.
# The regex matches PATH_INFO; named groups become query-string params.
_ROUTES = [
    (re.compile(r"^/api/articles/(?P<article_id>\d+)/?$"), "/api/method/benchmark_app.api.article_detail_update_delete"),
    (re.compile(r"^/api/articles/?$"),                     "/api/method/benchmark_app.api.articles_list_create"),
    (re.compile(r"^/api/counter/?$"),                      "/api/method/benchmark_app.api.counter"),
    (re.compile(r"^/api/truncate/?$"),                     "/api/method/benchmark_app.api.truncate"),
]


def _handle_websocket(ws):
    """Read JSON messages and echo back the appropriately-sized payload."""
    while not ws.closed:
        message = ws.receive()
        if message is None:
            break

        # Determine which response size to send back
        try:
            data = json.loads(message)
            query = data.get("query", "small")
        except Exception:
            query = "small"

        ws.send(_WS_RESPONSES.get(query, _WS_RESPONSES["small"]))


def application(environ, start_response):
    path = environ.get("PATH_INFO", "")

    # --- WebSocket upgrade at /api/ws/ ---
    ws = environ.get("wsgi.websocket")
    if ws and re.match(r"^/api/ws/?$", path):
        _handle_websocket(ws)
        return []

    # --- HTTP route rewriting ---
    for pattern, target in _ROUTES:
        m = pattern.match(path)
        if m:
            environ["PATH_INFO"] = target

            # Append captured groups (e.g. article_id) to the query string
            extras = "&".join(f"{k}={v}" for k, v in m.groupdict().items())
            if extras:
                qs = environ.get("QUERY_STRING", "")
                environ["QUERY_STRING"] = f"{qs}&{extras}" if qs else extras
            break

    return frappe.app.application(environ, start_response)
