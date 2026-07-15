import json
from channels.generic.websocket import AsyncJsonWebsocketConsumer

class BenchmarkConsumer(AsyncJsonWebsocketConsumer):
    small_response = {
        "type": "ping",
        "timestamp": 1783993200123,
        "client_id": "client_8b31a",
        "seq": 1024,
        "payload": "hello"
    }

    medium_response = {
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

    large_response = {
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

    async def connect(self):
        await self.accept()

    async def disconnect(self, close_code):
        pass

    async def receive_json(self, content, **kwargs):
        query = content.get("query", "small")
        if query == "medium":
            await self.send_json(self.medium_response)
        elif query == "large":
            await self.send_json(self.large_response)
        else:
            await self.send_json(self.small_response)
