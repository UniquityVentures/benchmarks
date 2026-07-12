import json
from django.http import HttpResponse, JsonResponse
from django.views import View
from django.views.decorators.csrf import csrf_exempt
from django.utils.decorators import method_decorator
from django.shortcuts import get_object_or_404
from .models import Article

@method_decorator(csrf_exempt, name='dispatch')
class ArticleListCreateView(View):
    def get(self, request, *args, **kwargs):
        articles = list(Article.objects.values('id', 'title', 'content', 'created_at', 'updated_at'))
        return JsonResponse(articles, safe=False)
    
    def post(self, request, *args, **kwargs):
        try:
            data = json.loads(request.body)
            article = Article.objects.create(
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
    def get(self, request, pk, *args, **kwargs):
        article = get_object_or_404(Article, pk=pk)
        return JsonResponse({
            'id': article.id,
            'title': article.title,
            'content': article.content,
            'created_at': article.created_at.isoformat(),
            'updated_at': article.updated_at.isoformat()
        })
        
    def put(self, request, pk, *args, **kwargs):
        article = get_object_or_404(Article, pk=pk)
        try:
            data = json.loads(request.body)
            article.title = data.get('title', article.title)
            article.content = data.get('content', article.content)
            article.save()
            return JsonResponse({
                'id': article.id,
                'title': article.title,
                'content': article.content,
                'created_at': article.created_at.isoformat(),
                'updated_at': article.updated_at.isoformat()
            })
        except Exception as e:
            return JsonResponse({'error': str(e)}, status=400)
            
    def delete(self, request, pk, *args, **kwargs):
        article = get_object_or_404(Article, pk=pk)
        article.delete()
        return HttpResponse(status=204)
