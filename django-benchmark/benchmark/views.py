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
