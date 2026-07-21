import json
from odoo import http
from odoo.http import request, Response
from odoo.service import model as service_model
from odoo.addons.bus.websocket import WebsocketConnectionHandler, Websocket

SMALL_RESPONSE = {
    "type": "ping",
    "timestamp": 1783993200123,
    "client_id": "client_8b31a",
    "seq": 1024,
    "payload": "hello"
}

MEDIUM_RESPONSE = {
    "type": "dashboard_update",
    "timestamp": 1783993200123,
    "meta": {
        "session_id": "sess_812da1823abf",
        "user_role": "editor",
        "version": "1.4.0"
    },
    "metrics": [
        {
            "id": i + 1,
            "name": f"metric_name_indicator_{i}",
            "value": float(i * 1.5),
            "status": "ok"
        } for i in range(50)
    ]
}

LARGE_RESPONSE = {
    "type": "bulk_sync",
    "sync_id": "sync_91238ba18",
    "records_count": 500,
    "records": [
        {
            "id": f"rec_{i+1:04d}",
            "title": f"Random Article Title {i+1:04d}",
            "body": "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.",
            "tags": ["performance", "benchmark", "websocket", "go"],
            "author": {
                "name": "Jane Doe",
                "email": "jane.doe@example.com"
            }
        } for i in range(500)
    ]
}


def format_iso_datetime(dt):
    if not dt:
        return ""
    if isinstance(dt, str):
        return dt
    return dt.isoformat()


def serialize_article(article):
    return {
        'id': article.id,
        'title': article.title or '',
        'content': article.content or '',
        'created_at': format_iso_datetime(article.create_date),
        'updated_at': format_iso_datetime(article.write_date),
    }


class BenchmarkWebsocket(Websocket):
    def _limit_rate(self):
        pass


class BenchmarkController(http.Controller):


    @http.route(['/api/counter', '/api/counter/'], type='http', auth='none', methods=['POST'], csrf=False, cors='*')
    def counter(self, **kwargs):
        try:
            data = json.loads(request.httprequest.data)
            counter_val = data.get('counter')
            if counter_val is None:
                return Response(json.dumps({'error': 'counter field is required'}), status=400, mimetype='application/json')
            res = json.dumps({'counter': int(counter_val) + 1})
            return Response(res, status=200, mimetype='application/json')
        except Exception as e:
            return Response(json.dumps({'error': str(e)}), status=400, mimetype='application/json')

    @http.route(['/api/articles', '/api/articles/'], type='http', auth='none', methods=['GET', 'POST'], csrf=False, cors='*')
    def articles_list_create(self, **kwargs):
        method = request.httprequest.method
        if method == 'GET':
            title_filter = request.params.get('title')
            domain = []
            if title_filter:
                domain.append(('title', 'like', title_filter))
            articles = request.env['benchmark.article'].sudo().search(domain)
            res = [serialize_article(a) for a in articles]
            return Response(json.dumps(res), status=200, mimetype='application/json')
        elif method == 'POST':
            def _create_article():
                data = json.loads(request.httprequest.data)
                article = request.env['benchmark.article'].sudo().create({
                    'title': data.get('title', ''),
                    'content': data.get('content', ''),
                })
                res = serialize_article(article)
                return Response(json.dumps(res), status=201, mimetype='application/json')

            try:
                return service_model.retrying(_create_article, env=request.env)
            except Exception as e:
                return Response(json.dumps({'error': str(e)}), status=400, mimetype='application/json')
        return Response(status=405)

    @http.route(['/api/articles/<int:article_id>', '/api/articles/<int:article_id>/'], type='http', auth='none', methods=['GET', 'PUT', 'DELETE'], csrf=False, cors='*')
    def article_detail_update_delete(self, article_id, **kwargs):
        method = request.httprequest.method
        if method == 'GET':
            article = request.env['benchmark.article'].sudo().browse(article_id)
            if not article.exists():
                return Response(json.dumps({'error': 'Not found'}), status=404, mimetype='application/json')
            res = serialize_article(article)
            return Response(json.dumps(res), status=200, mimetype='application/json')

        elif method == 'PUT':
            def _update_article():
                article = request.env['benchmark.article'].sudo().browse(article_id)
                if not article.exists():
                    return Response(json.dumps({'error': 'Not found'}), status=404, mimetype='application/json')
                data = json.loads(request.httprequest.data)
                vals = {}
                if 'title' in data:
                    vals['title'] = data['title']
                if 'content' in data:
                    vals['content'] = data['content']
                article.write(vals)
                res = serialize_article(article)
                return Response(json.dumps(res), status=200, mimetype='application/json')

            try:
                return service_model.retrying(_update_article, env=request.env)
            except Exception as e:
                return Response(json.dumps({'error': str(e)}), status=400, mimetype='application/json')

        elif method == 'DELETE':
            def _delete_article():
                article = request.env['benchmark.article'].sudo().browse(article_id)
                if not article.exists():
                    return Response(json.dumps({'error': 'Not found'}), status=404, mimetype='application/json')
                article.unlink()
                return Response(status=204)

            try:
                return service_model.retrying(_delete_article, env=request.env)
            except Exception as e:
                return Response(json.dumps({'error': str(e)}), status=400, mimetype='application/json')

        return Response(status=405)

    @http.route(['/api/truncate', '/api/truncate/'], type='http', auth='none', methods=['POST'], csrf=False, cors='*')
    def truncate_articles(self, **kwargs):
        def _truncate():
            request.env.cr.execute("TRUNCATE TABLE benchmark_article, benchmark_task RESTART IDENTITY CASCADE;")
            request.env.cr.commit()
            return Response(status=204)

        try:
            return service_model.retrying(_truncate, env=request.env)
        except Exception as e:
            return Response(json.dumps({'error': str(e)}), status=500, mimetype='application/json')

    @http.route(['/api/task', '/api/task/'], type='http', auth='none', methods=['POST'], csrf=False, cors='*')
    def task_submit(self, **kwargs):
        try:
            raw_body = request.httprequest.data
            val = int(raw_body.decode('utf-8').strip())
        except Exception:
            return Response(json.dumps({'error': 'invalid integer payload'}), status=400, mimetype='application/json')

        def _create_task():
            task = request.env['benchmark.task'].sudo().create({'val': val, 'state': 'pending'})
            task.with_delay().run_increment_job()
            return Response(str(task.id), status=200, mimetype='text/plain')

        try:
            return service_model.retrying(_create_task, env=request.env)
        except Exception as e:
            request.env.cr.rollback()
            return Response(json.dumps({'error': str(e)}), status=500, mimetype='application/json')

    @http.route(['/api/task/<int:task_id>', '/api/task/<int:task_id>/'], type='http', auth='none', methods=['GET'], csrf=False, cors='*')
    def task_status(self, task_id, **kwargs):
        task = request.env['benchmark.task'].sudo().browse(task_id)
        if not task.exists():
            return Response(json.dumps({'error': 'Not found'}), status=404, mimetype='application/json')

        if task.state == 'completed':
            res = json.dumps({'status': 'completed', 'result': task.result})
        else:
            res = json.dumps({'status': 'pending'})
        return Response(res, status=200, mimetype='application/json')


    @http.route(['/api/ws', '/api/ws/'], type='http', auth='none', cors='*', websocket=True)
    def api_ws(self, **kwargs):
        sock = request.httprequest._HTTPRequest__environ.get('socket')
        if not sock:
            return Response("Websockets require Gevent evented server (--workers 0)", status=400)

        response = WebsocketConnectionHandler._get_handshake_response(request.httprequest.headers)
        session = request.session
        httprequest = request.httprequest

        def handle_ws():
            ws = BenchmarkWebsocket(sock, session, httprequest.cookies)
            try:
                for message in ws.get_messages():
                    if message == b'\x00':
                        continue
                    try:
                        data = json.loads(message)
                        query = data.get("query", "small")
                    except Exception:
                        query = "small"

                    if query == "medium":
                        ws._send(json.dumps(MEDIUM_RESPONSE))
                    elif query == "large":
                        ws._send(json.dumps(LARGE_RESPONSE))
                    else:
                        ws._send(json.dumps(SMALL_RESPONSE))
            except Exception:
                pass

        response.call_on_close(handle_ws)
        return response
