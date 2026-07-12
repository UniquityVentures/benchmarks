import json
from django.test import TestCase
from django.urls import reverse
from .models import Article

class ArticleCRUDTestCase(TestCase):
    def setUp(self):
        self.article = Article.objects.create(
            title="Initial Title",
            content="Initial Content"
        )
        self.list_url = reverse('article_list_create')
        self.detail_url = reverse('article_detail_update_delete', kwargs={'pk': self.article.pk})

    def test_list_articles(self):
        response = self.client.get(self.list_url)
        self.assertEqual(response.status_code, 200)
        data = response.json()
        self.assertEqual(len(data), 1)
        self.assertEqual(data[0]['title'], "Initial Title")

    def test_create_article(self):
        payload = {
            "title": "New Article",
            "content": "New Content"
        }
        response = self.client.post(
            self.list_url,
            data=json.dumps(payload),
            content_type="application/json"
        )
        self.assertEqual(response.status_code, 201)
        data = response.json()
        self.assertIn("id", data)
        self.assertEqual(data["title"], "New Article")
        self.assertEqual(data["content"], "New Content")

    def test_retrieve_article(self):
        response = self.client.get(self.detail_url)
        self.assertEqual(response.status_code, 200)
        data = response.json()
        self.assertEqual(data["title"], "Initial Title")
        self.assertEqual(data["content"], "Initial Content")

    def test_update_article(self):
        payload = {
            "title": "Updated Title",
            "content": "Updated Content"
        }
        response = self.client.put(
            self.detail_url,
            data=json.dumps(payload),
            content_type="application/json"
        )
        self.assertEqual(response.status_code, 200)
        data = response.json()
        self.assertEqual(data["title"], "Updated Title")
        self.assertEqual(data["content"], "Updated Content")

    def test_delete_article(self):
        response = self.client.delete(self.detail_url)
        self.assertEqual(response.status_code, 204)
        self.assertFalse(Article.objects.filter(pk=self.article.pk).exists())
