#!/usr/bin/env python3
"""
Script untuk migrasi format response Postman collection
dari format lama ke format baru berdasarkan response.go dan error.go.

Format Baru:
- Success (non-list):   {"data": {...}, "meta": {"requestId": "req-example-xxx"}}
- Success (list):       {"data": [...], "meta": {"requestId": "req-example-xxx", "page": 1, "pageSize": 20, "total": N, "totalPages": T}}
- Error:                {"error": {"code": "...", "message": "...", "details": {"fields": [...]}}, "meta": {"requestId": "req-example-xxx"}}
"""

import json
import math

SAMPLE_REQUEST_ID = "req-example-abc123"


def is_error_body(body: dict) -> bool:
    """Cek apakah body adalah error format lama: {code, message, details}."""
    return (
        isinstance(body, dict)
        and "code" in body
        and "message" in body
        and not ("data" in body or "error" in body or "meta" in body)
    )


def is_list_body(body: dict) -> bool:
    """Cek apakah body adalah list response lama: {items, total, page, pageSize}."""
    return (
        isinstance(body, dict)
        and "items" in body
        and "total" in body
        and "page" in body
        and "pageSize" in body
        and "data" not in body
        and "meta" not in body
    )


def calc_total_pages(total: int, page_size: int) -> int:
    if page_size <= 0 or total <= 0:
        return 0
    return math.ceil(total / page_size)


def migrate_body(body_str: str, status_code: int) -> str:
    """Migrasi satu body string ke format baru."""
    if not body_str or body_str.strip() == "":
        return body_str  # 204 No Content tetap kosong

    try:
        body = json.loads(body_str)
    except (json.JSONDecodeError, TypeError):
        return body_str  # Bukan JSON, biarkan apa adanya

    # Sudah dimigrasi (punya 'data' atau 'error' + 'meta')
    if isinstance(body, dict) and "meta" in body and ("data" in body or "error" in body):
        return body_str

    if status_code >= 400:
        # Format error baru
        if is_error_body(body):
            new_body = {
                "error": {
                    "code": body.get("code", ""),
                    "message": body.get("message", ""),
                },
                "meta": {"requestId": SAMPLE_REQUEST_ID},
            }
            # Sertakan details hanya jika ada field-nya
            details = body.get("details")
            if details is not None:
                new_body["error"]["details"] = details
            return json.dumps(new_body, indent=2, ensure_ascii=False)
        else:
            return body_str

    else:
        # Format sukses
        if is_list_body(body):
            # List response: data = array items, meta = PageMeta
            items = body.get("items", [])
            total = body.get("total", 0)
            page = body.get("page", 1)
            page_size = body.get("pageSize", 20)
            new_body = {
                "data": items,
                "meta": {
                    "requestId": SAMPLE_REQUEST_ID,
                    "page": page,
                    "pageSize": page_size,
                    "total": total,
                    "totalPages": calc_total_pages(total, page_size),
                },
            }
            return json.dumps(new_body, indent=2, ensure_ascii=False)
        elif isinstance(body, dict) and "data" not in body and "meta" not in body:
            # Objek biasa → bungkus dengan data + meta
            new_body = {
                "data": body,
                "meta": {"requestId": SAMPLE_REQUEST_ID},
            }
            return json.dumps(new_body, indent=2, ensure_ascii=False)
        else:
            return body_str


def migrate_responses(responses: list) -> list:
    """Migrasi semua response dalam satu endpoint."""
    for resp in responses:
        code = resp.get("code", 200)
        body = resp.get("body", "")
        resp["body"] = migrate_body(body, code)
    return responses


def traverse_and_migrate(items: list) -> list:
    """Rekursif traverse seluruh collection dan migrasi responses."""
    for item in items:
        if "response" in item and item["response"]:
            item["response"] = migrate_responses(item["response"])
        if "item" in item:
            traverse_and_migrate(item["item"])
    return items


def main():
    collection_path = "postman/limen_api_collection.json"

    print(f"Membaca {collection_path}...")
    with open(collection_path, "r", encoding="utf-8") as f:
        collection = json.load(f)

    print("Migrasi format response...")
    collection["item"] = traverse_and_migrate(collection["item"])

    print(f"Menulis ulang {collection_path}...")
    with open(collection_path, "w", encoding="utf-8") as f:
        json.dump(collection, f, indent=2, ensure_ascii=False)

    print("✓ Selesai! Semua response example telah diperbarui ke format baru.")
    print()
    print("Format baru yang diterapkan:")
    print("  Success (single): { \"data\": {...}, \"meta\": { \"requestId\": \"...\" } }")
    print("  Success (list):   { \"data\": [...], \"meta\": { \"requestId\": \"...\", \"page\": 1, \"pageSize\": 20, \"total\": N, \"totalPages\": T } }")
    print("  Error:            { \"error\": { \"code\": \"...\", \"message\": \"...\", \"details\": {...} }, \"meta\": { \"requestId\": \"...\" } }")


if __name__ == "__main__":
    import os
    # Jalankan dari root project
    root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    os.chdir(root)
    main()
