import asyncio
import json
from django.http import HttpResponse, JsonResponse
from django.views import View
from django.views.decorators.csrf import csrf_exempt
from django.utils.decorators import method_decorator
from django.shortcuts import aget_object_or_404
from .models import Article

@method_decorator(csrf_exempt, name='dispatch')
class ArticleListCreateView(View):
    async def get(self, request, *args, **kwargs):
        articles = []
        async for article in Article.objects.values('id', 'title', 'content', 'created_at', 'updated_at'):
            articles.append(article)
        return JsonResponse(articles, safe=False)
    
    async def post(self, request, *args, **kwargs):
        try:
            data = json.loads(request.body)
            article = await Article.objects.acreate(
                title=data.get('title', ''),
                content=data.get('content', '')
            )
            return JsonResponse({
                'id': article.id,
                'title': article.title,
                'content': article.content,
                'created_at': article.created_at.isoformat(),
                'updated_at': article.updated_at.isoformat()
            }, status=201)
        except Exception as e:
            return JsonResponse({'error': str(e)}, status=400)

@method_decorator(csrf_exempt, name='dispatch')
class ArticleDetailUpdateDeleteView(View):
    async def get(self, request, pk, *args, **kwargs):
        article = await aget_object_or_404(Article, pk=pk)
        return JsonResponse({
            'id': article.id,
            'title': article.title,
            'content': article.content,
            'created_at': article.created_at.isoformat(),
            'updated_at': article.updated_at.isoformat()
        })
        
    async def put(self, request, pk, *args, **kwargs):
        article = await aget_object_or_404(Article, pk=pk)
        try:
            data = json.loads(request.body)
            article.title = data.get('title', article.title)
            article.content = data.get('content', article.content)
            await article.asave()
            return JsonResponse({
                'id': article.id,
                'title': article.title,
                'content': article.content,
                'created_at': article.created_at.isoformat(),
                'updated_at': article.updated_at.isoformat()
            })
        except Exception as e:
            return JsonResponse({'error': str(e)}, status=400)
            
    async def delete(self, request, pk, *args, **kwargs):
        article = await aget_object_or_404(Article, pk=pk)
        await article.adelete()
        return HttpResponse(status=204)

@method_decorator(csrf_exempt, name='dispatch')
class ArticleTruncateView(View):
    async def post(self, request, *args, **kwargs):
        from asgiref.sync import sync_to_async
        from django.db import connection
        
        def do_truncate():
            with connection.cursor() as cursor:
                cursor.execute("TRUNCATE TABLE articles RESTART IDENTITY CASCADE;")
                
        await sync_to_async(do_truncate)()
        _task_store.clear()
        return HttpResponse(status=204)

@method_decorator(csrf_exempt, name='dispatch')
class CounterView(View):
    async def post(self, request, *args, **kwargs):
        try:
            data = json.loads(request.body)
            counter = data.get('counter')
            if counter is None:
                return JsonResponse({'error': 'counter field is required'}, status=400)
            return JsonResponse({'counter': int(counter) + 1})
        except Exception as e:
            return JsonResponse({'error': str(e)}, status=400)

import os
from celery.result import AsyncResult
from .tasks import increment_task

_task_seq = 0
_task_store = {}

async def _process_task(task_id: str, val: int):
    _task_store[task_id] = {
        'status': 'completed',
        'result': val + 1
    }

@method_decorator(csrf_exempt, name='dispatch')
class TaskSubmitView(View):
    async def post(self, request, *args, **kwargs):
        try:
            val = int(request.body.decode('utf-8').strip())
        except ValueError:
            return HttpResponse("invalid integer payload", status=400)

        if os.environ.get("USE_CELERY") == "1":
            async_res = increment_task.delay(val)
            return HttpResponse(async_res.id, content_type="text/plain")

        global _task_seq
        _task_seq += 1
        task_id = str(_task_seq)
        _task_store[task_id] = {'status': 'pending'}

        asyncio.create_task(_process_task(task_id, val))

        return HttpResponse(task_id, content_type="text/plain")

@method_decorator(csrf_exempt, name='dispatch')
class TaskStatusView(View):
    async def get(self, request, task_id, *args, **kwargs):
        if os.environ.get("USE_CELERY") == "1":
            res = AsyncResult(str(task_id))
            if res.ready():
                return JsonResponse({'status': 'completed', 'result': res.result})
            return JsonResponse({'status': 'pending'})

        res = _task_store.get(str(task_id))
        if res is None:
            return HttpResponse("task not found", status=404)
        return JsonResponse(res)

from django.http import HttpResponseBadRequest

def websocket_info_view(request):
    return HttpResponseBadRequest("Please connect via WebSockets.")
