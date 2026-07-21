from django.urls import path
from .views import (
    ArticleListCreateView,
    ArticleDetailUpdateDeleteView,
    ArticleTruncateView,
    CounterView,
    TaskSubmitView,
    TaskStatusView,
    websocket_info_view
)

urlpatterns = [
    path('articles/', ArticleListCreateView.as_view(), name='article_list_create'),
    path('articles/<int:pk>/', ArticleDetailUpdateDeleteView.as_view(), name='article_detail_update_delete'),
    path('truncate/', ArticleTruncateView.as_view(), name='article_truncate'),
    path('counter/', CounterView.as_view(), name='counter'),
    path('task/', TaskSubmitView.as_view(), name='task_submit'),
    path('task/<str:task_id>/', TaskStatusView.as_view(), name='task_status'),
    path('ws/', websocket_info_view, name='websocket_info'),
]
