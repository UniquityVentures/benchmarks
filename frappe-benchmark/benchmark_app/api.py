import json
import frappe
from frappe.utils import now_datetime

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


def serialize_article(doc):
    name_val = doc.name
    try:
        id_val = int(name_val)
    except (ValueError, TypeError):
        id_val = name_val

    creation = doc.creation
    modified = doc.modified

    return {
        "id": id_val,
        "title": doc.title or "",
        "content": doc.content or "",
        "created_at": creation.isoformat() if hasattr(creation, "isoformat") else str(creation or ""),
        "updated_at": modified.isoformat() if hasattr(modified, "isoformat") else str(modified or ""),
    }


@frappe.whitelist(allow_guest=True, methods=["POST"])
def counter():
    try:
        data = json.loads(frappe.request.data or "{}")
        val = data.get("counter")
        if val is None:
            frappe.response.status_code = 400
            return {"error": "counter field is required"}
        return {"counter": int(val) + 1}
    except Exception as e:
        frappe.response.status_code = 400
        return {"error": str(e)}


@frappe.whitelist(allow_guest=True, methods=["GET", "POST"])
def articles_list_create():
    method = frappe.request.method
    if method == "GET":
        title = frappe.request.args.get("title")
        filters = {}
        if title:
            filters["title"] = ["like", f"%{title}%"]
        docs = frappe.get_all("Benchmark Article", filters=filters, fields=["name", "title", "content", "creation", "modified"])
        res = [serialize_article(d) for d in docs]
        return res
    elif method == "POST":
        try:
            data = json.loads(frappe.request.data or "{}")
            doc = frappe.get_doc({
                "doctype": "Benchmark Article",
                "title": data.get("title", ""),
                "content": data.get("content", ""),
            })
            doc.insert(ignore_permissions=True)
            frappe.db.commit()
            frappe.response.status_code = 201
            return serialize_article(doc)
        except Exception as e:
            frappe.response.status_code = 400
            return {"error": str(e)}


@frappe.whitelist(allow_guest=True, methods=["GET", "PUT", "DELETE"])
def article_detail_update_delete(article_id=None):
    if not article_id:
        article_id = frappe.request.args.get("article_id") or frappe.form_dict.get("article_id")

    if not article_id or not frappe.db.exists("Benchmark Article", article_id):
        frappe.response.status_code = 404
        return {"error": "Not found"}

    method = frappe.request.method
    if method == "GET":
        doc = frappe.get_doc("Benchmark Article", article_id)
        return serialize_article(doc)
    elif method == "PUT":
        try:
            data = json.loads(frappe.request.data or "{}")
            doc = frappe.get_doc("Benchmark Article", article_id)
            if "title" in data:
                doc.title = data["title"]
            if "content" in data:
                doc.content = data["content"]
            doc.save(ignore_permissions=True)
            frappe.db.commit()
            return serialize_article(doc)
        except Exception as e:
            frappe.response.status_code = 400
            return {"error": str(e)}
    elif method == "DELETE":
        try:
            frappe.delete_doc("Benchmark Article", article_id, ignore_permissions=True)
            frappe.db.commit()
            frappe.response.status_code = 204
            return
        except Exception as e:
            frappe.response.status_code = 400
            return {"error": str(e)}


@frappe.whitelist(allow_guest=True, methods=["POST"])
def truncate():
    try:
        frappe.db.sql("TRUNCATE TABLE `tabBenchmark Article`")
        frappe.db.commit()
        frappe.response.status_code = 204
        return
    except Exception as e:
        frappe.response.status_code = 500
        return {"error": str(e)}


def run_task_job(task_id, val):
    res = {"status": "completed", "result": int(val) + 1}
    frappe.cache().set_value(f"task_{task_id}", json.dumps(res))


@frappe.whitelist(allow_guest=True, methods=["POST"])
def task_submit():
    try:
        raw = (frappe.request.data or b"").decode("utf-8").strip()
        val = int(raw)
    except Exception:
        frappe.response.status_code = 400
        return {"error": "invalid integer payload"}

    task_id = frappe.generate_hash(length=12)
    frappe.cache().set_value(f"task_{task_id}", json.dumps({"status": "pending"}))
    frappe.enqueue("benchmark_app.api.run_task_job", queue="default", task_id=task_id, val=val)

    frappe.response["type"] = "download"
    frappe.response["filename"] = "task.txt"
    frappe.response["filecontent"] = task_id
    frappe.response["content_type"] = "text/plain"
    frappe.response["display_content_as"] = "inline"
    return task_id


@frappe.whitelist(allow_guest=True, methods=["GET"])
def task_status(task_id=None):
    if not task_id:
        task_id = frappe.request.args.get("task_id") or frappe.form_dict.get("task_id")
    if not task_id:
        frappe.response.status_code = 400
        return {"error": "task_id required"}
    raw = frappe.cache().get_value(f"task_{task_id}")
    if not raw:
        frappe.response.status_code = 404
        return {"error": "task not found"}
    data = json.loads(raw)
    frappe.response.update(data)
    return
