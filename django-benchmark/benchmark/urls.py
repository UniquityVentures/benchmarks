from django.urls import path
from .views import ArticleListCreateView, ArticleDetailUpdateDeleteView

urlpatterns = [
    path('articles/', ArticleListCreateView.as_view(), name='article_list_create'),
    path('articles/<int:pk>/', ArticleDetailUpdateDeleteView.as_view(), name='article_detail_update_delete'),
]
