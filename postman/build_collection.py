#!/usr/bin/env python3
import json
import os

def make_url(path, query=None, variables=None):
    segments = [p for p in path.strip("/").split("/") if p]
    raw = "{{baseUrl}}/" + "/".join(segments)
    
    url_obj = {
        "raw": raw,
        "host": ["{{baseUrl}}"],
        "path": segments
    }
    
    if query:
        url_obj["query"] = []
        q_parts = []
        for q in query:
            url_obj["query"].append({
                "key": q.get("key"),
                "value": str(q.get("value", "")),
                "description": q.get("description", ""),
                "disabled": q.get("disabled", False)
            })
            if not q.get("disabled", False):
                q_parts.append(f"{q.get('key')}={q.get('value')}")
        if q_parts:
            url_obj["raw"] += "?" + "&".join(q_parts)
            
    if variables:
        url_obj["variable"] = []
        for v in variables:
            url_obj["variable"].append({
                "key": v.get("key"),
                "value": str(v.get("value", "")),
                "description": v.get("description", "")
            })
            
    return url_obj

def make_request(name, method, path, description="", body=None, query=None, variables=None, csrf=False, content_type="application/json", test_script=None, is_binary=False, examples=None):
    headers = []
    if content_type:
        headers.append({"key": "Content-Type", "value": content_type, "type": "text"})
    headers.append({"key": "Accept", "value": "application/json" if content_type != "text/event-stream" else "text/event-stream", "type": "text"})
    
    if csrf:
        headers.append({"key": "X-CSRF-Token", "value": "{{csrf_token}}", "description": "Derived CSRF token dari session", "type": "text"})

    req_obj = {
        "method": method,
        "header": headers,
        "url": make_url(path, query, variables),
        "description": description
    }

    if body is not None:
        if is_binary:
            req_obj["body"] = {
                "mode": "file",
                "file": {"src": ""}
            }
        elif isinstance(body, (dict, list)):
            req_obj["body"] = {
                "mode": "raw",
                "raw": json.dumps(body, indent=2),
                "options": {
                    "raw": {
                        "language": "json"
                    }
                }
            }
        elif isinstance(body, str):
            req_obj["body"] = {
                "mode": "raw",
                "raw": body,
                "options": {
                    "raw": {
                        "language": "text"
                    }
                }
            }

    item = {
        "name": name,
        "request": req_obj,
        "response": []
    }

    if test_script:
        item["event"] = [
            {
                "listen": "test",
                "script": {
                    "exec": test_script.strip().split("\n"),
                    "type": "text/javascript"
                }
            }
        ]

    # Process saved response examples
    if examples:
        for ex in examples:
            ex_status_code = ex.get("code", 200)
            ex_status_text = ex.get("status", "OK" if ex_status_code == 200 else ("Created" if ex_status_code == 201 else "No Content"))
            ex_name = ex.get("name", f"{ex_status_code} {ex_status_text}")
            ex_body = ex.get("body", "")
            ex_ct = ex.get("contentType", content_type or "application/json")
            
            resp_body_str = ""
            if isinstance(ex_body, (dict, list)):
                resp_body_str = json.dumps(ex_body, indent=2)
            elif isinstance(ex_body, str):
                resp_body_str = ex_body
                
            resp_headers = []
            if ex_ct:
                resp_headers.append({"key": "Content-Type", "value": ex_ct})
            for h in ex.get("headers", []):
                resp_headers.append(h)

            item["response"].append({
                "name": ex_name,
                "originalRequest": req_obj,
                "status": ex_status_text,
                "code": ex_status_code,
                "_postman_previewlanguage": "json" if "json" in ex_ct else "text",
                "header": resp_headers,
                "cookie": ex.get("cookie", []),
                "body": resp_body_str
            })

    return item

def build_collection():
    collection = {
        "info": {
            "_postman_id": "c1f7a28e-5d1b-4f6c-8a9d-0123456789ab",
            "name": "LIMEN API Documentation",
            "description": "Dokumentasi REST API LIMEN (Sistem Absensi Terintegrasi & Akademik) berdasarkan router `routes.go`.\nDilengkapi dengan seluruh Example Request & Response untuk setiap endpoint.\n\n### Panduan Autentikasi & CSRF:\n1. Jalankan request **Login** di folder `01 - Authentikasi & Profil`.\n2. Script Postman secara otomatis akan mengambil cookie `csrf` dan menyimpannya ke collection variable `csrf_token`.\n3. Request mutasi (POST, PUT, PATCH, DELETE) yang dilindungi CSRF akan otomatis menyertakan header `X-CSRF-Token: {{csrf_token}}` dan cookie `session` dikelola otomatis oleh Postman Cookie Jar.",
            "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
        },
        "variable": [
            {"key": "baseUrl", "value": "http://localhost:8080", "type": "string", "description": "Base URL API LIMEN"},
            {"key": "csrf_token", "value": "", "type": "string", "description": "Token CSRF otomatis dari cookie saat login"},
            {"key": "sample_userId", "value": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d", "type": "string", "description": "Contoh UUID User"},
            {"key": "sample_deviceId", "value": "b2c3d4e5-f6a7-8b9c-0d1e-2f3a4b5c6d7e", "type": "string", "description": "Contoh UUID Device"},
            {"key": "sample_classId", "value": "c3d4e5f6-a7b8-9c0d-1e2f-3a4b5c6d7e8f", "type": "string", "description": "Contoh UUID Class"},
            {"key": "sample_subjectId", "value": "d4e5f6a7-b89c-0d1e-2f3a-4b5c6d7e8f90", "type": "string", "description": "Contoh UUID Subject"},
            {"key": "sample_scheduleId", "value": "e5f6a7b8-9c0d-1e2f-3a4b-5c6d7e8f901a", "type": "string", "description": "Contoh UUID Schedule"},
            {"key": "sample_assessmentId", "value": "f6a7b89c-0d1e-2f3a-4b5c-6d7e8f901a2b", "type": "string", "description": "Contoh UUID Assessment"},
            {"key": "sample_assignmentId", "value": "a7b89c0d-1e2f-3a4b-5c6d-7e8f901a2b3c", "type": "string", "description": "Contoh UUID Assignment"},
            {"key": "sample_bankId", "value": "b89c0d1e-2f3a-4b5c-6d7e-8f901a2b3c4d", "type": "string", "description": "Contoh UUID Question Bank"},
            {"key": "sample_questionId", "value": "c90d1e2f-3a4b-5c6d-7e8f-901a2b3c4d5e", "type": "string", "description": "Contoh UUID Question"},
            {"key": "sample_quizId", "value": "d01e2f3a-4b5c-6d7e-8f90-1a2b3c4d5e6f", "type": "string", "description": "Contoh UUID Quiz"},
            {"key": "sample_attemptId", "value": "e12f3a4b-5c6d-7e8f-901a-2b3c4d5e6f70", "type": "string", "description": "Contoh UUID Quiz Attempt"},
            {"key": "sample_noteId", "value": "f23a4b5c-6d7e-8f90-1a2b-3c4d5e6f7081", "type": "string", "description": "Contoh UUID Counseling Note"},
            {"key": "sample_notificationId", "value": "a34b5c6d-7e8f-901a-2b3c-4d5e6f708192", "type": "string", "description": "Contoh UUID Notification"}
        ],
        "item": []
    }

    # ==========================================
    # 01 - Authentikasi & Profil (Auth)
    # ==========================================
    auth_tests = """
pm.test("Status code is 200", function () {
    pm.response.to.have.status(200);
});

pm.cookies.jar().get(pm.variables.get("baseUrl"), "csrf", function (error, cookie) {
    if (cookie) {
        pm.collectionVariables.set("csrf_token", cookie);
        console.log("CSRF token automatically set:", cookie);
    }
});
"""

    auth_folder = {
        "name": "01 - Authentikasi & Profil",
        "description": "Endpoint autentikasi sesi pengguna, profil login, logout, dan reset password.",
        "item": [
            make_request(
                name="Login",
                method="POST",
                path="/api/v1/auth/login",
                description="Autentikasi akun. Mengembalikan data profil user serta menyetel HttpOnly cookie `session` dan readable cookie `csrf`.",
                body={"email": "admin@sekolah.sch.id", "password": "AdminPassword123"},
                test_script=auth_tests,
                examples=[
                    {
                        "name": "200 OK - Login Success",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "user": {
                                "id": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                                "email": "admin@sekolah.sch.id",
                                "fullName": "Administrator Sekolah",
                                "roles": ["admin"],
                                "status": "active"
                            }
                        },
                        "headers": [
                            {"key": "Set-Cookie", "value": "session=dGhpcy1pcy1hLXNlc3Npb24tdG9rZW4=; Path=/; HttpOnly; SameSite=Strict"},
                            {"key": "Set-Cookie", "value": "csrf=Y3NyZi10b2tlbi1leGFtcGxl; Path=/; SameSite=Strict"}
                        ]
                    },
                    {
                        "name": "401 Unauthorized - Invalid Credentials",
                        "code": 401,
                        "status": "Unauthorized",
                        "body": {
                            "code": "INVALID_CREDENTIALS",
                            "message": "invalid email or password",
                            "details": {"fields": []}
                        }
                    },
                    {
                        "name": "422 Unprocessable Entity - Validation Error",
                        "code": 422,
                        "status": "Unprocessable Entity",
                        "body": {
                            "code": "VALIDATION_ERROR",
                            "message": "validation failed",
                            "details": {
                                "fields": [
                                    {"field": "email", "message": "must be a valid email"}
                                ]
                            }
                        }
                    }
                ]
            ),
            make_request(
                name="Get Current User (Me)",
                method="GET",
                path="/api/v1/auth/me",
                description="Mengambil detail identitas pengguna yang sedang login berdasarkan cookie sesi aktif.",
                examples=[
                    {
                        "name": "200 OK - Current User Profile",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "id": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                            "email": "admin@sekolah.sch.id",
                            "fullName": "Administrator Sekolah",
                            "status": "active",
                            "roles": ["admin"]
                        }
                    },
                    {
                        "name": "401 Unauthorized - No Session",
                        "code": 401,
                        "status": "Unauthorized",
                        "body": {
                            "code": "UNAUTHENTICATED",
                            "message": "session token required",
                            "details": {"fields": []}
                        }
                    }
                ]
            ),
            make_request(
                name="Logout",
                method="POST",
                path="/api/v1/auth/logout",
                description="Menghapus sesi login dari database dan membersihkan cookie session.",
                csrf=True,
                examples=[
                    {
                        "name": "204 No Content - Successfully Logged Out",
                        "code": 204,
                        "status": "No Content",
                        "body": "",
                        "headers": [
                            {"key": "Set-Cookie", "value": "session=; Path=/; Max-Age=0; HttpOnly"},
                            {"key": "Set-Cookie", "value": "csrf=; Path=/; Max-Age=0"}
                        ]
                    }
                ]
            ),
            make_request(
                name="Reset Password (Admin Only)",
                method="POST",
                path="/api/v1/users/:userId/password-reset",
                description="Admin mereset password akun pengguna tertentu.",
                variables=[{"key": "userId", "value": "{{sample_userId}}", "description": "Target User ID (UUID)"}],
                body={"newPassword": "NewSecurePassword123"},
                csrf=True,
                examples=[
                    {
                        "name": "204 No Content - Password Reset Success",
                        "code": 204,
                        "status": "No Content",
                        "body": ""
                    },
                    {
                        "name": "404 Not Found - User Not Found",
                        "code": 404,
                        "status": "Not Found",
                        "body": {
                            "code": "NOT_FOUND",
                            "message": "user not found",
                            "details": {"fields": []}
                        }
                    }
                ]
            )
        ]
    }
    collection["item"].append(auth_folder)

    # ==========================================
    # 02 - Pengaturan Sekolah (Settings)
    # ==========================================
    settings_folder = {
        "name": "02 - Pengaturan Sekolah",
        "description": "Konfigurasi jam absensi sekolah dan batas toleransi keterlambatan.",
        "item": [
            make_request(
                name="Get School Settings",
                method="GET",
                path="/api/v1/settings/school",
                description="Mengambil data konfigurasi absensi sekolah saat ini.",
                examples=[
                    {
                        "name": "200 OK - School Settings",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "id": "s1e2t3t4-i5n6-7g8s-9c0d-1e2f3a4b5c6d",
                            "firstEntryTime": "07:00",
                            "timezone": "Asia/Jakarta",
                            "lateToleranceMinutes": 15,
                            "updatedBy": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                            "updatedAt": "2026-11-15T08:00:00Z"
                        }
                    }
                ]
            ),
            make_request(
                name="Update School Settings",
                method="PUT",
                path="/api/v1/settings/school",
                description="Memperbarui jam masuk pertama dan toleransi menit keterlambatan.",
                body={"firstEntryTime": "07:00", "lateToleranceMinutes": 15},
                csrf=True,
                examples=[
                    {
                        "name": "200 OK - Settings Updated",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "id": "s1e2t3t4-i5n6-7g8s-9c0d-1e2f3a4b5c6d",
                            "firstEntryTime": "07:00",
                            "timezone": "Asia/Jakarta",
                            "lateToleranceMinutes": 15,
                            "updatedBy": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                            "updatedAt": "2026-11-15T08:30:00Z"
                        }
                    }
                ]
            )
        ]
    }
    collection["item"].append(settings_folder)

    # ==========================================
    # 03 - Manajemen Perangkat (Devices)
    # ==========================================
    device_folder = {
        "name": "03 - Manajemen Perangkat",
        "description": "Pendaftaran dan pemantauan perangkat absensi fisik / IoT.",
        "item": [
            make_request(
                name="List Devices",
                method="GET",
                path="/api/v1/devices",
                description="Mendapatkan daftar seluruh perangkat absensi terdaftar dengan pagination.",
                query=[
                    {"key": "page", "value": "1"},
                    {"key": "pageSize", "value": "20"},
                    {"key": "isActive", "value": "true", "disabled": True}
                ],
                examples=[
                    {
                        "name": "200 OK - Device List",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "items": [
                                {
                                    "id": "b2c3d4e5-f6a7-8b9c-0d1e-2f3a4b5c6d7e",
                                    "name": "Mesin Fingerprint Gerbang Depan",
                                    "location": "Lobi Gedung Utama",
                                    "serial": "FP-GATE-01",
                                    "isActive": True,
                                    "lastSeenAt": "2026-11-15T07:15:20Z",
                                    "createdAt": "2026-10-01T08:00:00Z",
                                    "updatedAt": "2026-11-15T07:15:20Z"
                                }
                            ],
                            "total": 1,
                            "page": 1,
                            "pageSize": 20
                        }
                    }
                ]
            ),
            make_request(
                name="Create Device",
                method="POST",
                path="/api/v1/devices",
                description="Mendaftarkan perangkat absensi baru. Mengembalikan token `deviceKey` untuk mesin.",
                body={
                    "name": "Mesin Fingerprint Gerbang Depan",
                    "location": "Lobi Gedung Utama",
                    "serial": "FP-GATE-01",
                    "isActive": True
                },
                csrf=True,
                examples=[
                    {
                        "name": "201 Created - Device Created",
                        "code": 201,
                        "status": "Created",
                        "body": {
                            "id": "b2c3d4e5-f6a7-8b9c-0d1e-2f3a4b5c6d7e",
                            "name": "Mesin Fingerprint Gerbang Depan",
                            "location": "Lobi Gedung Utama",
                            "serial": "FP-GATE-01",
                            "isActive": True,
                            "lastSeenAt": None,
                            "createdAt": "2026-11-15T08:00:00Z",
                            "updatedAt": "2026-11-15T08:00:00Z",
                            "deviceKey": "dev_key_sec_994a3bc76d1e4e5"
                        }
                    }
                ]
            ),
            make_request(
                name="Update Device",
                method="PUT",
                path="/api/v1/devices/:deviceId",
                description="Memperbarui nama, lokasi, serial, atau status aktif perangkat.",
                variables=[{"key": "deviceId", "value": "{{sample_deviceId}}", "description": "UUID Perangkat"}],
                body={
                    "name": "Mesin Fingerprint Gerbang Timur",
                    "location": "Pintu Gerbang Timur",
                    "serial": "FP-GATE-01-REV",
                    "isActive": True
                },
                csrf=True,
                examples=[
                    {
                        "name": "200 OK - Device Updated",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "id": "b2c3d4e5-f6a7-8b9c-0d1e-2f3a4b5c6d7e",
                            "name": "Mesin Fingerprint Gerbang Timur",
                            "location": "Pintu Gerbang Timur",
                            "serial": "FP-GATE-01-REV",
                            "isActive": True,
                            "lastSeenAt": "2026-11-15T07:15:20Z",
                            "createdAt": "2026-10-01T08:00:00Z",
                            "updatedAt": "2026-11-15T08:30:00Z"
                        }
                    }
                ]
            )
        ]
    }
    collection["item"].append(device_folder)

    # ==========================================
    # 04 - Manajemen Pengguna (Users)
    # ==========================================
    user_folder = {
        "name": "04 - Manajemen Pengguna",
        "description": "Pengelolaan data akun guru, staf, wali murid, dan siswa.",
        "item": [
            make_request(
                name="List Users",
                method="GET",
                path="/api/v1/users",
                description="Daftar akun pengguna dengan filter role, status, dan pencarian nama/email.",
                query=[
                    {"key": "page", "value": "1"},
                    {"key": "pageSize", "value": "20"},
                    {"key": "role", "value": "guru_mapel", "disabled": True},
                    {"key": "status", "value": "active", "disabled": True},
                    {"key": "q", "value": "Budi", "disabled": True}
                ],
                examples=[
                    {
                        "name": "200 OK - Users List",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "items": [
                                {
                                    "id": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                                    "email": "guru.matematika@sekolah.sch.id",
                                    "fullName": "Drs. Ahmad Fauzi, M.Pd",
                                    "status": "active",
                                    "roles": ["guru_mapel", "guru_wali"],
                                    "profile": None,
                                    "lastLoginAt": "2026-11-15T06:55:00Z",
                                    "createdAt": "2026-10-01T00:00:00Z",
                                    "updatedAt": "2026-11-15T06:55:00Z"
                                }
                            ],
                            "total": 1,
                            "page": 1,
                            "pageSize": 20
                        }
                    }
                ]
            ),
            make_request(
                name="Create User",
                method="POST",
                path="/api/v1/users",
                description="Admin membuat pengguna baru dengan password aman dan roles.",
                body={
                    "email": "guru.matematika@sekolah.sch.id",
                    "password": "Password123",
                    "fullName": "Drs. Ahmad Fauzi, M.Pd",
                    "roles": ["guru_mapel", "guru_wali"]
                },
                csrf=True,
                examples=[
                    {
                        "name": "201 Created - User Created",
                        "code": 201,
                        "status": "Created",
                        "body": {
                            "id": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                            "email": "guru.matematika@sekolah.sch.id",
                            "fullName": "Drs. Ahmad Fauzi, M.Pd",
                            "status": "active",
                            "roles": ["guru_mapel", "guru_wali"],
                            "profile": None,
                            "lastLoginAt": None,
                            "createdAt": "2026-11-15T08:00:00Z",
                            "updatedAt": "2026-11-15T08:00:00Z"
                        }
                    }
                ]
            ),
            make_request(
                name="Get User by ID",
                method="GET",
                path="/api/v1/users/:userId",
                description="Mengambil detail lengkap satu pengguna.",
                variables=[{"key": "userId", "value": "{{sample_userId}}"}],
                examples=[
                    {
                        "name": "200 OK - User Detail",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "id": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                            "email": "guru.matematika@sekolah.sch.id",
                            "fullName": "Drs. Ahmad Fauzi, M.Pd",
                            "status": "active",
                            "roles": ["guru_mapel", "guru_wali"],
                            "profile": None,
                            "lastLoginAt": "2026-11-15T06:55:00Z",
                            "createdAt": "2026-10-01T00:00:00Z",
                            "updatedAt": "2026-11-15T06:55:00Z"
                        }
                    }
                ]
            ),
            make_request(
                name="Update User",
                method="PUT",
                path="/api/v1/users/:userId",
                description="Memperbarui data pengguna, nama lengkap, status, atau roles.",
                variables=[{"key": "userId", "value": "{{sample_userId}}"}],
                body={
                    "email": "guru.matematika@sekolah.sch.id",
                    "fullName": "Dr. Ahmad Fauzi, M.Pd",
                    "status": "active",
                    "roles": ["guru_mapel", "guru_wali"]
                },
                csrf=True,
                examples=[
                    {
                        "name": "200 OK - User Updated",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "id": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                            "email": "guru.matematika@sekolah.sch.id",
                            "fullName": "Dr. Ahmad Fauzi, M.Pd",
                            "status": "active",
                            "roles": ["guru_mapel", "guru_wali"],
                            "profile": None,
                            "lastLoginAt": "2026-11-15T06:55:00Z",
                            "createdAt": "2026-10-01T00:00:00Z",
                            "updatedAt": "2026-11-15T08:30:00Z"
                        }
                    }
                ]
            )
        ]
    }
    collection["item"].append(user_folder)

    # ==========================================
    # 05 - Akademik (Academic)
    # ==========================================
    academic_folder = {
        "name": "05 - Akademik",
        "description": "Pengelolaan data master kelas, mata pelajaran, profil siswa, dan penugasan guru.",
        "item": [
            # Classes
            make_request(
                name="List Classes",
                method="GET",
                path="/api/v1/classes",
                description="Daftar kelas beserta wali kelas dan jumlah siswa.",
                query=[
                    {"key": "page", "value": "1"},
                    {"key": "pageSize", "value": "20"},
                    {"key": "level", "value": "10", "disabled": True},
                    {"key": "q", "value": "MIPA", "disabled": True}
                ],
                examples=[
                    {
                        "name": "200 OK - Classes List",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "items": [
                                {
                                    "id": "c3d4e5f6-a7b8-9c0d-1e2f-3a4b5c6d7e8f",
                                    "name": "X MIPA 1",
                                    "level": "10",
                                    "homeroomTeacherUserId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                                    "studentCount": 36
                                }
                            ],
                            "total": 1,
                            "page": 1,
                            "pageSize": 20
                        }
                    }
                ]
            ),
            make_request(
                name="Create Class",
                method="POST",
                path="/api/v1/classes",
                description="Membuat ruang kelas baru.",
                body={"name": "X MIPA 1", "level": "10", "homeroomTeacherUserId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d"},
                csrf=True,
                examples=[
                    {
                        "name": "201 Created - Class Created",
                        "code": 201,
                        "status": "Created",
                        "body": {
                            "id": "c3d4e5f6-a7b8-9c0d-1e2f-3a4b5c6d7e8f",
                            "name": "X MIPA 1",
                            "level": "10",
                            "homeroomTeacherUserId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                            "studentCount": 0
                        }
                    }
                ]
            ),
            make_request(
                name="Update Class",
                method="PUT",
                path="/api/v1/classes/:classId",
                description="Memperbarui nama kelas, tingkat, atau wali kelas.",
                variables=[{"key": "classId", "value": "{{sample_classId}}"}],
                body={"name": "X MIPA 1 Unggulan", "level": "10", "homeroomTeacherUserId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d"},
                csrf=True,
                examples=[
                    {
                        "name": "200 OK - Class Updated",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "id": "c3d4e5f6-a7b8-9c0d-1e2f-3a4b5c6d7e8f",
                            "name": "X MIPA 1 Unggulan",
                            "level": "10",
                            "homeroomTeacherUserId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                            "studentCount": 36
                        }
                    }
                ]
            ),
            # Subjects
            make_request(
                name="List Subjects",
                method="GET",
                path="/api/v1/subjects",
                description="Daftar mata pelajaran aktif.",
                query=[
                    {"key": "page", "value": "1"},
                    {"key": "pageSize", "value": "20"},
                    {"key": "q", "value": "Biologi", "disabled": True}
                ],
                examples=[
                    {
                        "name": "200 OK - Subjects List",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "items": [
                                {
                                    "id": "d4e5f6a7-b89c-0d1e-2f3a-4b5c6d7e8f90",
                                    "code": "BIO-10",
                                    "name": "Biologi Peminatan Kelas X"
                                }
                            ],
                            "total": 1,
                            "page": 1,
                            "pageSize": 20
                        }
                    }
                ]
            ),
            make_request(
                name="Create Subject",
                method="POST",
                path="/api/v1/subjects",
                description="Menambahkan mata pelajaran baru dengan kode unik.",
                body={"code": "BIO-10", "name": "Biologi Peminatan Kelas X"},
                csrf=True,
                examples=[
                    {
                        "name": "201 Created - Subject Created",
                        "code": 201,
                        "status": "Created",
                        "body": {
                            "id": "d4e5f6a7-b89c-0d1e-2f3a-4b5c6d7e8f90",
                            "code": "BIO-10",
                            "name": "Biologi Peminatan Kelas X"
                        }
                    }
                ]
            ),
            # Students
            make_request(
                name="List Students",
                method="GET",
                path="/api/v1/students",
                description="Daftar profil siswa dan kelasnya.",
                query=[
                    {"key": "page", "value": "1"},
                    {"key": "pageSize", "value": "20"},
                    {"key": "classId", "value": "{{sample_classId}}", "disabled": True},
                    {"key": "sort", "value": "name", "disabled": True},
                    {"key": "q", "value": "Rizky", "disabled": True}
                ],
                examples=[
                    {
                        "name": "200 OK - Students List",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "items": [
                                {
                                    "userId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                                    "fullName": "Muhammad Rizky Pratama",
                                    "nis": "2024101001",
                                    "classId": "c3d4e5f6-a7b8-9c0d-1e2f-3a4b5c6d7e8f",
                                    "className": "X MIPA 1",
                                    "status": "active"
                                }
                            ],
                            "total": 1,
                            "page": 1,
                            "pageSize": 20
                        }
                    }
                ]
            ),
            make_request(
                name="Get Student by ID",
                method="GET",
                path="/api/v1/students/:studentId",
                description="Detail data akademik satu orang siswa.",
                variables=[{"key": "studentId", "value": "{{sample_userId}}"}],
                examples=[
                    {
                        "name": "200 OK - Student Detail",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "userId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                            "fullName": "Muhammad Rizky Pratama",
                            "nis": "2024101001",
                            "classId": "c3d4e5f6-a7b8-9c0d-1e2f-3a4b5c6d7e8f",
                            "className": "X MIPA 1",
                            "status": "active"
                        }
                    }
                ]
            ),
            make_request(
                name="Update Student",
                method="PATCH",
                path="/api/v1/students/:studentId",
                description="Memperbarui NIS atau memindahkan siswa ke kelas lain.",
                variables=[{"key": "studentId", "value": "{{sample_userId}}"}],
                body={"nis": "2024101001", "classId": "{{sample_classId}}"},
                csrf=True,
                examples=[
                    {
                        "name": "200 OK - Student Updated",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "userId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                            "fullName": "Muhammad Rizky Pratama",
                            "nis": "2024101001",
                            "classId": "c3d4e5f6-a7b8-9c0d-1e2f-3a4b5c6d7e8f",
                            "className": "X MIPA 1",
                            "status": "active"
                        }
                    }
                ]
            ),
            # Teaching Assignments
            make_request(
                name="List Teaching Assignments",
                method="GET",
                path="/api/v1/teaching-assignments",
                description="Melihat penugasan guru yang mengajar mata pelajaran tertentu di kelas tertentu.",
                query=[
                    {"key": "page", "value": "1"},
                    {"key": "pageSize", "value": "20"},
                    {"key": "teacherUserId", "value": "{{sample_userId}}", "disabled": True},
                    {"key": "classId", "value": "{{sample_classId}}", "disabled": True},
                    {"key": "subjectId", "value": "{{sample_subjectId}}", "disabled": True}
                ],
                examples=[
                    {
                        "name": "200 OK - Teaching Assignments List",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "items": [
                                {
                                    "id": "e5f6a7b8-9c0d-1e2f-3a4b-5c6d7e8f901a",
                                    "teacherUserId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                                    "subjectId": "d4e5f6a7-b89c-0d1e-2f3a-4b5c6d7e8f90",
                                    "classId": "c3d4e5f6-a7b8-9c0d-1e2f-3a4b5c6d7e8f"
                                }
                            ],
                            "total": 1,
                            "page": 1,
                            "pageSize": 20
                        }
                    }
                ]
            ),
            make_request(
                name="Create Teaching Assignment",
                method="POST",
                path="/api/v1/teaching-assignments",
                description="Menugaskan guru mengajar mata pelajaran tertentu di suatu kelas.",
                body={
                    "teacherUserId": "{{sample_userId}}",
                    "subjectId": "{{sample_subjectId}}",
                    "classId": "{{sample_classId}}"
                },
                csrf=True,
                examples=[
                    {
                        "name": "201 Created - Assignment Created",
                        "code": 201,
                        "status": "Created",
                        "body": {
                            "id": "e5f6a7b8-9c0d-1e2f-3a4b-5c6d7e8f901a",
                            "teacherUserId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                            "subjectId": "d4e5f6a7-b89c-0d1e-2f3a-4b5c6d7e8f90",
                            "classId": "c3d4e5f6-a7b8-9c0d-1e2f-3a4b5c6d7e8f"
                        }
                    }
                ]
            )
        ]
    }
    collection["item"].append(academic_folder)

    # ==========================================
    # 06 - Jadwal Pelajaran (Schedules)
    # ==========================================
    schedule_folder = {
        "name": "06 - Jadwal Pelajaran",
        "description": "Pengelolaan jadwal kegiatan belajar mengajar dan ujian.",
        "item": [
            make_request(
                name="List Schedules",
                method="GET",
                path="/api/v1/schedules",
                description="Melihat jadwal pelajaran berdasarkan kelas, mapel, hari (1-7), atau rentang tanggal.",
                query=[
                    {"key": "page", "value": "1"},
                    {"key": "pageSize", "value": "20"},
                    {"key": "classId", "value": "{{sample_classId}}", "disabled": True},
                    {"key": "subjectId", "value": "{{sample_subjectId}}", "disabled": True},
                    {"key": "kind", "value": "lesson", "disabled": True},
                    {"key": "dayOfWeek", "value": "1", "disabled": True},
                    {"key": "dateFrom", "value": "2026-11-16", "disabled": True},
                    {"key": "dateTo", "value": "2026-11-20", "disabled": True}
                ],
                examples=[
                    {
                        "name": "200 OK - Schedules List",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "items": [
                                {
                                    "id": "e5f6a7b8-9c0d-1e2f-3a4b-5c6d7e8f901a",
                                    "classId": "c3d4e5f6-a7b8-9c0d-1e2f-3a4b5c6d7e8f",
                                    "subjectId": "d4e5f6a7-b89c-0d1e-2f3a-4b5c6d7e8f90",
                                    "teacherUserId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                                    "dayOfWeek": 1,
                                    "startTime": "07:30",
                                    "endTime": "09:00",
                                    "kind": "lesson"
                                }
                            ],
                            "total": 1,
                            "page": 1,
                            "pageSize": 20
                        }
                    }
                ]
            ),
            make_request(
                name="Create Schedule",
                method="POST",
                path="/api/v1/schedules",
                description="Menambahkan jadwal pelajaran baru.",
                body={
                    "classId": "{{sample_classId}}",
                    "subjectId": "{{sample_subjectId}}",
                    "teacherUserId": "{{sample_userId}}",
                    "dayOfWeek": 1,
                    "startTime": "07:30",
                    "endTime": "09:00",
                    "kind": "lesson"
                },
                csrf=True,
                examples=[
                    {
                        "name": "201 Created - Schedule Created",
                        "code": 201,
                        "status": "Created",
                        "body": {
                            "id": "e5f6a7b8-9c0d-1e2f-3a4b-5c6d7e8f901a",
                            "classId": "c3d4e5f6-a7b8-9c0d-1e2f-3a4b5c6d7e8f",
                            "subjectId": "d4e5f6a7-b89c-0d1e-2f3a-4b5c6d7e8f90",
                            "teacherUserId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                            "dayOfWeek": 1,
                            "startTime": "07:30",
                            "endTime": "09:00",
                            "kind": "lesson"
                        }
                    }
                ]
            ),
            make_request(
                name="Update Schedule",
                method="PUT",
                path="/api/v1/schedules/:scheduleId",
                description="Memperbarui jam, hari, atau guru pengajar jadwal.",
                variables=[{"key": "scheduleId", "value": "{{sample_scheduleId}}"}],
                body={
                    "startTime": "08:00",
                    "endTime": "09:30",
                    "dayOfWeek": 1,
                    "kind": "lesson"
                },
                csrf=True,
                examples=[
                    {
                        "name": "200 OK - Schedule Updated",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "id": "e5f6a7b8-9c0d-1e2f-3a4b-5c6d7e8f901a",
                            "classId": "c3d4e5f6-a7b8-9c0d-1e2f-3a4b5c6d7e8f",
                            "subjectId": "d4e5f6a7-b89c-0d1e-2f3a-4b5c6d7e8f90",
                            "teacherUserId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                            "dayOfWeek": 1,
                            "startTime": "08:00",
                            "endTime": "09:30",
                            "kind": "lesson"
                        }
                    }
                ]
            ),
            make_request(
                name="Delete Schedule",
                method="DELETE",
                path="/api/v1/schedules/:scheduleId",
                description="Menghapus jadwal pelajaran.",
                variables=[{"key": "scheduleId", "value": "{{sample_scheduleId}}"}],
                csrf=True,
                examples=[
                    {
                        "name": "204 No Content - Schedule Deleted",
                        "code": 204,
                        "status": "No Content",
                        "body": ""
                    }
                ]
            )
        ]
    }
    collection["item"].append(schedule_folder)

    # ==========================================
    # 07 - Penilaian (Assessments)
    # ==========================================
    assessment_folder = {
        "name": "07 - Penilaian",
        "description": "Pengelolaan master penilaian (UH, UTS, UAS) dan input skor siswa.",
        "item": [
            make_request(
                name="List Assessments",
                method="GET",
                path="/api/v1/assessments",
                description="Daftar penilaian untuk kelas dan mata pelajaran tertentu.",
                query=[
                    {"key": "page", "value": "1"},
                    {"key": "pageSize", "value": "20"},
                    {"key": "classId", "value": "{{sample_classId}}", "disabled": True},
                    {"key": "subjectId", "value": "{{sample_subjectId}}", "disabled": True},
                    {"key": "kind", "value": "UH1", "disabled": True},
                    {"key": "period", "value": "2024/2025 Genap", "disabled": True}
                ],
                examples=[
                    {
                        "name": "200 OK - Assessments List",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "items": [
                                {
                                    "id": "f6a7b89c-0d1e-2f3a-4b5c-6d7e8f901a2b",
                                    "classId": "c3d4e5f6-a7b8-9c0d-1e2f-3a4b5c6d7e8f",
                                    "subjectId": "d4e5f6a7-b89c-0d1e-2f3a-4b5c6d7e8f90",
                                    "teacherUserId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                                    "kind": "UH1",
                                    "kkm": 75.0,
                                    "period": "2024/2025 Genap"
                                }
                            ],
                            "total": 1,
                            "page": 1,
                            "pageSize": 20
                        }
                    }
                ]
            ),
            make_request(
                name="Create Assessment",
                method="POST",
                path="/api/v1/assessments",
                description="Membuat jenis penilaian baru beserta target KKM.",
                body={
                    "classId": "{{sample_classId}}",
                    "subjectId": "{{sample_subjectId}}",
                    "kind": "UH1",
                    "period": "2024/2025 Genap",
                    "kkm": 75.0
                },
                csrf=True,
                examples=[
                    {
                        "name": "201 Created - Assessment Created",
                        "code": 201,
                        "status": "Created",
                        "body": {
                            "id": "f6a7b89c-0d1e-2f3a-4b5c-6d7e8f901a2b",
                            "classId": "c3d4e5f6-a7b8-9c0d-1e2f-3a4b5c6d7e8f",
                            "subjectId": "d4e5f6a7-b89c-0d1e-2f3a-4b5c6d7e8f90",
                            "teacherUserId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                            "kind": "UH1",
                            "kkm": 75.0,
                            "period": "2024/2025 Genap"
                        }
                    }
                ]
            ),
            make_request(
                name="List Assessment Scores",
                method="GET",
                path="/api/v1/assessments/:assessmentId/scores",
                description="Melihat perolehan nilai seluruh siswa pada penilaian tertentu.",
                variables=[{"key": "assessmentId", "value": "{{sample_assessmentId}}"}],
                query=[{"key": "page", "value": "1"}, {"key": "pageSize", "value": "50"}],
                examples=[
                    {
                        "name": "200 OK - Scores List",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "items": [
                                {
                                    "assessmentId": "f6a7b89c-0d1e-2f3a-4b5c-6d7e8f901a2b",
                                    "studentUserId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                                    "studentName": "Muhammad Rizky Pratama",
                                    "score": 88.5,
                                    "updatedBy": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                                    "updatedAt": "2026-11-15T09:00:00Z"
                                }
                            ],
                            "total": 1,
                            "page": 1,
                            "pageSize": 50
                        }
                    }
                ]
            ),
            make_request(
                name="Upsert Student Score",
                method="PUT",
                path="/api/v1/assessments/:assessmentId/scores/:studentUserId",
                description="Menginput atau merevisi nilai seorang siswa.",
                variables=[
                    {"key": "assessmentId", "value": "{{sample_assessmentId}}"},
                    {"key": "studentUserId", "value": "{{sample_userId}}"}
                ],
                body={"score": 88.5},
                csrf=True,
                examples=[
                    {
                        "name": "200 OK - Score Saved",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "assessmentId": "f6a7b89c-0d1e-2f3a-4b5c-6d7e8f901a2b",
                            "studentUserId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                            "studentName": "Muhammad Rizky Pratama",
                            "score": 88.5,
                            "updatedBy": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                            "updatedAt": "2026-11-15T09:15:00Z"
                        }
                    }
                ]
            )
        ]
    }
    collection["item"].append(assessment_folder)

    # ==========================================
    # 08 - Minat Bakat & Passing Grade (Aptitude)
    # ==========================================
    aptitude_folder = {
        "name": "08 - Minat Bakat & Passing Grade",
        "description": "Hasil tes minat bakat siswa, pilihan jurusan target, dan data referensi passing grade.",
        "item": [
            make_request(
                name="List Student Aptitude Results",
                method="GET",
                path="/api/v1/students/:studentId/aptitude-results",
                description="Melihat hasil tes minat bakat siswa.",
                variables=[{"key": "studentId", "value": "{{sample_userId}}"}],
                examples=[
                    {
                        "name": "200 OK - Aptitude Results",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "studentUserId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                            "results": [
                                {
                                    "id": "a7b89c0d-1e2f-3a4b-5c6d-7e8f901a2b3c",
                                    "result": "Dominan Minat Realistik & Investigatif (Sains & Teknik Komputer)",
                                    "assessedAt": "2026-10-15T10:00:00Z"
                                }
                            ]
                        }
                    }
                ]
            ),
            make_request(
                name="Set Target Major",
                method="POST",
                path="/api/v1/students/:studentId/target-majors",
                description="Siswa menetapkan program studi/universitas target.",
                variables=[{"key": "studentId", "value": "{{sample_userId}}"}],
                body={"major": "Teknik Informatika - Institut Teknologi Bandung"},
                csrf=True,
                examples=[
                    {
                        "name": "201 Created - Target Major Saved",
                        "code": 201,
                        "status": "Created",
                        "body": {
                            "id": "b89c0d1e-2f3a-4b5c-6d7e-8f901a2b3c4d",
                            "studentUserId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                            "major": "Teknik Informatika - Institut Teknologi Bandung",
                            "createdAt": "2026-11-15T09:30:00Z"
                        }
                    }
                ]
            ),
            make_request(
                name="List Passing Grades",
                method="GET",
                path="/api/v1/passing-grades",
                description="Mencari referensi skor passing grade universitas.",
                query=[
                    {"key": "page", "value": "1"},
                    {"key": "pageSize", "value": "20"},
                    {"key": "university", "value": "Institut Teknologi Bandung", "disabled": True},
                    {"key": "major", "value": "Teknik Informatika", "disabled": True},
                    {"key": "year", "value": "2025", "disabled": True}
                ],
                examples=[
                    {
                        "name": "200 OK - Passing Grades List",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "items": [
                                {
                                    "id": "c90d1e2f-3a4b-5c6d-7e8f-901a2b3c4d5e",
                                    "university": "Institut Teknologi Bandung",
                                    "major": "Teknik Informatika",
                                    "score": 675.5,
                                    "year": 2025
                                }
                            ],
                            "total": 1,
                            "page": 1,
                            "pageSize": 20
                        }
                    }
                ]
            ),
            make_request(
                name="Get Passing Grade by ID",
                method="GET",
                path="/api/v1/passing-grades/:passingGradeId",
                description="Mengambil detail informasi passing grade.",
                variables=[{"key": "passingGradeId", "value": "{{sample_assessmentId}}"}],
                examples=[
                    {
                        "name": "200 OK - Passing Grade Detail",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "id": "c90d1e2f-3a4b-5c6d-7e8f-901a2b3c4d5e",
                            "university": "Institut Teknologi Bandung",
                            "major": "Teknik Informatika",
                            "score": 675.5,
                            "year": 2025
                        }
                    }
                ]
            )
        ]
    }
    collection["item"].append(aptitude_folder)

    # ==========================================
    # 09 - Bimbingan Konseling (Counseling)
    # ==========================================
    counseling_folder = {
        "name": "09 - Bimbingan Konseling",
        "description": "Pencatatan sesi bimbingan konseling dan tindak lanjut perkembangan siswa.",
        "item": [
            make_request(
                name="List Counseling Notes",
                method="GET",
                path="/api/v1/counseling-notes",
                description="Melihat catatan konseling siswa.",
                query=[
                    {"key": "page", "value": "1"},
                    {"key": "pageSize", "value": "20"},
                    {"key": "studentUserId", "value": "{{sample_userId}}", "disabled": True},
                    {"key": "classId", "value": "{{sample_classId}}", "disabled": True},
                    {"key": "sort", "value": "-createdAt"}
                ],
                examples=[
                    {
                        "name": "200 OK - Counseling Notes List",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "items": [
                                {
                                    "id": "f23a4b5c-6d7e-8f90-1a2b-3c4d5e6f7081",
                                    "studentUserId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                                    "authorId": "d01e2f3a-4b5c-6d7e-8f90-1a2b3c4d5e6f",
                                    "note": "Siswa berkonsultasi mengenai strategi pemilihan jurusan SNBP.",
                                    "createdAt": "2026-11-15T10:00:00Z"
                                }
                            ],
                            "total": 1,
                            "page": 1,
                            "pageSize": 20
                        }
                    }
                ]
            ),
            make_request(
                name="Create Counseling Note",
                method="POST",
                path="/api/v1/counseling-notes",
                description="Guru BK / Wali kelas membuat catatan konseling baru.",
                body={
                    "studentUserId": "{{sample_userId}}",
                    "note": "Siswa berkonsultasi mengenai persiapan ujian masuk perguruan tinggi dan kendala fokus belajar di rumah."
                },
                csrf=True,
                examples=[
                    {
                        "name": "201 Created - Note Created",
                        "code": 201,
                        "status": "Created",
                        "body": {
                            "id": "f23a4b5c-6d7e-8f90-1a2b-3c4d5e6f7081",
                            "studentUserId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                            "authorId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                            "note": "Siswa berkonsultasi mengenai persiapan ujian masuk perguruan tinggi dan kendala fokus belajar di rumah.",
                            "createdAt": "2026-11-15T10:15:00Z"
                        }
                    }
                ]
            ),
            make_request(
                name="Update Counseling Note",
                method="PUT",
                path="/api/v1/counseling-notes/:noteId",
                description="Memperbarui isi catatan konseling.",
                variables=[{"key": "noteId", "value": "{{sample_noteId}}"}],
                body={"note": "Catatan konseling diperbarui: Siswa diberikan rekomendasi rencana belajar harian dan modul pendampingan."},
                csrf=True,
                examples=[
                    {
                        "name": "200 OK - Note Updated",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "id": "f23a4b5c-6d7e-8f90-1a2b-3c4d5e6f7081",
                            "studentUserId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                            "authorId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                            "note": "Catatan konseling diperbarui: Siswa diberikan rekomendasi rencana belajar harian dan modul pendampingan.",
                            "createdAt": "2026-11-15T10:15:00Z"
                        }
                    }
                ]
            ),
            make_request(
                name="Add Followup to Note",
                method="POST",
                path="/api/v1/counseling-notes/:noteId/followups",
                description="Menambahkan riwayat tindak lanjut (follow-up) terhadap sesi konseling.",
                variables=[{"key": "noteId", "value": "{{sample_noteId}}"}],
                body={"note": "Telah dilakukan koordinasi dengan orang tua siswa untuk memantau jam belajar malam."},
                csrf=True,
                examples=[
                    {
                        "name": "201 Created - Followup Added",
                        "code": 201,
                        "status": "Created",
                        "body": {
                            "id": "e12f3a4b-5c6d-7e8f-901a-2b3c4d5e6f70",
                            "noteId": "f23a4b5c-6d7e-8f90-1a2b-3c4d5e6f7081",
                            "note": "Telah dilakukan koordinasi dengan orang tua siswa untuk memantau jam belajar malam.",
                            "createdBy": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                            "createdAt": "2026-11-15T11:00:00Z"
                        }
                    }
                ]
            )
        ]
    }
    collection["item"].append(counseling_folder)

    # ==========================================
    # 10 - Laporan & Rapor (Reports)
    # ==========================================
    report_folder = {
        "name": "10 - Laporan & Rapor",
        "description": "Rekapitulasi absensi harian/kelas, rekap nilai mata pelajaran, dan rapor akademik siswa.",
        "item": [
            make_request(
                name="Attendance Report",
                method="GET",
                path="/api/v1/reports/attendance",
                description="Rekapitulasi kehadiran (hadir, terlambat, izin, sakit, alpa) berdasarkan group (student, class, atau day).",
                query=[
                    {"key": "groupBy", "value": "student"},
                    {"key": "classId", "value": "{{sample_classId}}", "disabled": True},
                    {"key": "studentUserId", "value": "{{sample_userId}}", "disabled": True},
                    {"key": "subjectType", "value": "student", "disabled": True},
                    {"key": "dateFrom", "value": "2026-11-01", "disabled": True},
                    {"key": "dateTo", "value": "2026-11-15", "disabled": True}
                ],
                examples=[
                    {
                        "name": "200 OK - Attendance Summary by Student",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "groupBy": "student",
                            "items": [
                                {
                                    "present": 20,
                                    "late": 2,
                                    "excused": 1,
                                    "sick": 1,
                                    "absent": 0,
                                    "total": 24,
                                    "studentUserId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                                    "studentName": "Muhammad Rizky Pratama"
                                }
                            ]
                        }
                    }
                ]
            ),
            make_request(
                name="Grades Report",
                method="GET",
                path="/api/v1/reports/grades",
                description="Rekapitulasi nilai siswa dalam suatu kelas dan mata pelajaran pada periode tertentu.",
                query=[
                    {"key": "page", "value": "1"},
                    {"key": "pageSize", "value": "50"},
                    {"key": "classId", "value": "{{sample_classId}}", "disabled": True},
                    {"key": "subjectId", "value": "{{sample_subjectId}}", "disabled": True},
                    {"key": "period", "value": "2024/2025 Genap", "disabled": True}
                ],
                examples=[
                    {
                        "name": "200 OK - Grades Summary",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "items": [
                                {
                                    "studentUserId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                                    "studentName": "Muhammad Rizky Pratama",
                                    "subjectId": "d4e5f6a7-b89c-0d1e-2f3a-4b5c6d7e8f90",
                                    "subjectName": "Biologi Peminatan Kelas X",
                                    "averageScore": 87.5
                                }
                            ],
                            "total": 1,
                            "page": 1,
                            "pageSize": 50
                        }
                    }
                ]
            ),
            make_request(
                name="Student Report Card",
                method="GET",
                path="/api/v1/students/:studentId/report-card",
                description="Melihat lembar rapor nilai komprehensif siswa untuk seluruh mata pelajaran pada periode tertentu.",
                variables=[{"key": "studentId", "value": "{{sample_userId}}"}],
                query=[{"key": "period", "value": "2024/2025 Genap"}],
                examples=[
                    {
                        "name": "200 OK - Student Report Card",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "studentUserId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                            "period": "2024/2025 Genap",
                            "subjects": [
                                {
                                    "subjectId": "d4e5f6a7-b89c-0d1e-2f3a-4b5c6d7e8f90",
                                    "subjectName": "Biologi Peminatan Kelas X",
                                    "score": 88.0,
                                    "kkm": 75.0
                                },
                                {
                                    "subjectId": "e5f6a7b8-9c0d-1e2f-3a4b-5c6d7e8f901a",
                                    "subjectName": "Matematika Wajib Kelas X",
                                    "score": 90.0,
                                    "kkm": 75.0
                                }
                            ]
                        }
                    }
                ]
            )
        ]
    }
    collection["item"].append(report_folder)

    # ==========================================
    # 11 - Tugas Siswa (Assignments)
    # ==========================================
    assignment_folder = {
        "name": "11 - Penugasan (Assignments)",
        "description": "Pembuatan tugas rumah/proyek dan pemberian nilai pengumpulan tugas siswa.",
        "item": [
            make_request(
                name="List Assignments",
                method="GET",
                path="/api/v1/assignments",
                description="Mengambil daftar penugasan dengan filter kelas, mapel, atau guru pembuat.",
                query=[
                    {"key": "page", "value": "1"},
                    {"key": "pageSize", "value": "20"},
                    {"key": "classId", "value": "{{sample_classId}}", "disabled": True},
                    {"key": "subjectId", "value": "{{sample_subjectId}}", "disabled": True},
                    {"key": "teacherUserId", "value": "{{sample_userId}}", "disabled": True},
                    {"key": "sort", "value": "-createdAt"}
                ],
                examples=[
                    {
                        "name": "200 OK - Assignments List",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "items": [
                                {
                                    "id": "a7b89c0d-1e2f-3a4b-5c6d-7e8f901a2b3c",
                                    "classId": "c3d4e5f6-a7b8-9c0d-1e2f-3a4b5c6d7e8f",
                                    "subjectId": "d4e5f6a7-b89c-0d1e-2f3a-4b5c6d7e8f90",
                                    "teacherUserId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                                    "title": "Tugas Bab 3: Hukum Gerak Newton",
                                    "description": "Kerjakan soal evaluasi halaman 45-47 pada buku paket.",
                                    "dueAt": "2026-11-25T23:59:59Z",
                                    "createdAt": "2026-11-15T08:00:00Z"
                                }
                            ],
                            "total": 1,
                            "page": 1,
                            "pageSize": 20
                        }
                    }
                ]
            ),
            make_request(
                name="Create Assignment",
                method="POST",
                path="/api/v1/assignments",
                description="Guru membuat tugas baru dengan batas waktu pengumpulan (format RFC3339).",
                body={
                    "classId": "{{sample_classId}}",
                    "subjectId": "{{sample_subjectId}}",
                    "title": "Tugas Bab 3: Hukum Gerak Newton",
                    "description": "Kerjakan soal evaluasi halaman 45-47 pada buku pegangan dan kumpulkan rangkuman langkahnya.",
                    "dueAt": "2026-11-25T23:59:59Z"
                },
                csrf=True,
                examples=[
                    {
                        "name": "201 Created - Assignment Created",
                        "code": 201,
                        "status": "Created",
                        "body": {
                            "id": "a7b89c0d-1e2f-3a4b-5c6d-7e8f901a2b3c",
                            "classId": "c3d4e5f6-a7b8-9c0d-1e2f-3a4b5c6d7e8f",
                            "subjectId": "d4e5f6a7-b89c-0d1e-2f3a-4b5c6d7e8f90",
                            "teacherUserId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                            "title": "Tugas Bab 3: Hukum Gerak Newton",
                            "description": "Kerjakan soal evaluasi halaman 45-47 pada buku pegangan dan kumpulkan rangkuman langkahnya.",
                            "dueAt": "2026-11-25T23:59:59Z",
                            "createdAt": "2026-11-15T08:00:00Z"
                        }
                    }
                ]
            ),
            make_request(
                name="List Submissions",
                method="GET",
                path="/api/v1/assignments/:assignmentId/submissions",
                description="Melihat rekap pengumpulan jawaban tugas oleh siswa.",
                variables=[{"key": "assignmentId", "value": "{{sample_assignmentId}}"}],
                query=[{"key": "page", "value": "1"}, {"key": "pageSize", "value": "50"}],
                examples=[
                    {
                        "name": "200 OK - Submissions List",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "items": [
                                {
                                    "assignmentId": "a7b89c0d-1e2f-3a4b-5c6d-7e8f901a2b3c",
                                    "studentUserId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                                    "studentName": "Muhammad Rizky Pratama",
                                    "score": 92.5,
                                    "note": "Analisis konsep fisika sangat lengkap dan sistematis.",
                                    "submittedAt": "2026-11-20T14:30:00Z",
                                    "gradedBy": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d"
                                }
                            ],
                            "total": 1,
                            "page": 1,
                            "pageSize": 50
                        }
                    }
                ]
            ),
            make_request(
                name="Grade Submission",
                method="PUT",
                path="/api/v1/assignments/:assignmentId/submissions/:studentUserId",
                description="Guru menilai dan memberi catatan umpan balik pada pengumpulan tugas siswa.",
                variables=[
                    {"key": "assignmentId", "value": "{{sample_assignmentId}}"},
                    {"key": "studentUserId", "value": "{{sample_userId}}"}
                ],
                body={
                    "score": 92.5,
                    "note": "Analisis konsep fisika sangat lengkap dan sistematis."
                },
                csrf=True,
                examples=[
                    {
                        "name": "200 OK - Submission Graded",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "assignmentId": "a7b89c0d-1e2f-3a4b-5c6d-7e8f901a2b3c",
                            "studentUserId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                            "studentName": "Muhammad Rizky Pratama",
                            "score": 92.5,
                            "note": "Analisis konsep fisika sangat lengkap dan sistematis.",
                            "submittedAt": "2026-11-20T14:30:00Z",
                            "gradedBy": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d"
                        }
                    }
                ]
            )
        ]
    }
    collection["item"].append(assignment_folder)

    # ==========================================
    # 12 - Bank Soal (Question Bank)
    # ==========================================
    bank_folder = {
        "name": "12 - Bank Soal",
        "description": "Penyimpanan dan butir-butir soal ujian / kuis (pilihan ganda, essay, benar salah, dsb).",
        "item": [
            make_request(
                name="List Question Banks",
                method="GET",
                path="/api/v1/question-banks",
                description="Melihat daftar bank soal milik guru.",
                query=[
                    {"key": "page", "value": "1"},
                    {"key": "pageSize", "value": "20"},
                    {"key": "subjectId", "value": "{{sample_subjectId}}", "disabled": True},
                    {"key": "ownerId", "value": "{{sample_userId}}", "disabled": True},
                    {"key": "scope", "value": "latihan", "disabled": True}
                ],
                examples=[
                    {
                        "name": "200 OK - Question Banks List",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "items": [
                                {
                                    "id": "b89c0d1e-2f3a-4b5c-6d7e-8f901a2b3c4d",
                                    "subjectId": "d4e5f6a7-b89c-0d1e-2f3a-4b5c6d7e8f90",
                                    "scope": "latihan",
                                    "ownerId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                                    "questionCount": 25
                                }
                            ],
                            "total": 1,
                            "page": 1,
                            "pageSize": 20
                        }
                    }
                ]
            ),
            make_request(
                name="Create Question Bank",
                method="POST",
                path="/api/v1/question-banks",
                description="Membuat ruang bank soal baru (scope: latihan atau ujian).",
                body={"subjectId": "{{sample_subjectId}}", "scope": "latihan"},
                csrf=True,
                examples=[
                    {
                        "name": "201 Created - Bank Created",
                        "code": 201,
                        "status": "Created",
                        "body": {
                            "id": "b89c0d1e-2f3a-4b5c-6d7e-8f901a2b3c4d",
                            "subjectId": "d4e5f6a7-b89c-0d1e-2f3a-4b5c6d7e8f90",
                            "scope": "latihan",
                            "ownerId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                            "questionCount": 0
                        }
                    }
                ]
            ),
            make_request(
                name="List Questions in Bank",
                method="GET",
                path="/api/v1/question-banks/:bankId/questions",
                description="Mengambil butir-butir pertanyaan di dalam bank soal.",
                variables=[{"key": "bankId", "value": "{{sample_bankId}}"}],
                query=[{"key": "page", "value": "1"}, {"key": "pageSize", "value": "50"}],
                examples=[
                    {
                        "name": "200 OK - Questions List",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "items": [
                                {
                                    "id": "c90d1e2f-3a4b-5c6d-7e8f-901a2b3c4d5e",
                                    "bankId": "b89c0d1e-2f3a-4b5c-6d7e-8f901a2b3c4d",
                                    "type": "pilihan_ganda",
                                    "body": "Sebuah benda bermassa 2 kg bergerak dengan percepatan 3 m/s2. Berapakah gaya yang bekerja pada benda tersebut?",
                                    "options": [
                                        {"id": "A", "label": "5 N"},
                                        {"id": "B", "label": "6 N"},
                                        {"id": "C", "label": "8 N"},
                                        {"id": "D", "label": "10 N"}
                                    ],
                                    "answerKey": {"correctOption": "B"}
                                }
                            ],
                            "total": 1,
                            "page": 1,
                            "pageSize": 50
                        }
                    }
                ]
            ),
            make_request(
                name="Create Question",
                method="POST",
                path="/api/v1/question-banks/:bankId/questions",
                description="Menambahkan butir soal ke dalam bank soal.",
                variables=[{"key": "bankId", "value": "{{sample_bankId}}"}],
                body={
                    "type": "pilihan_ganda",
                    "body": "Sebuah benda bermassa 2 kg bergerak dengan percepatan 3 m/s2. Berapakah gaya yang bekerja pada benda tersebut?",
                    "options": [
                        {"id": "A", "label": "5 N"},
                        {"id": "B", "label": "6 N"},
                        {"id": "C", "label": "8 N"},
                        {"id": "D", "label": "10 N"}
                    ],
                    "answerKey": {"correctOption": "B"}
                },
                csrf=True,
                examples=[
                    {
                        "name": "201 Created - Question Created",
                        "code": 201,
                        "status": "Created",
                        "body": {
                            "id": "c90d1e2f-3a4b-5c6d-7e8f-901a2b3c4d5e",
                            "bankId": "b89c0d1e-2f3a-4b5c-6d7e-8f901a2b3c4d",
                            "type": "pilihan_ganda",
                            "body": "Sebuah benda bermassa 2 kg bergerak dengan percepatan 3 m/s2. Berapakah gaya yang bekerja pada benda tersebut?",
                            "options": [
                                {"id": "A", "label": "5 N"},
                                {"id": "B", "label": "6 N"},
                                {"id": "C", "label": "8 N"},
                                {"id": "D", "label": "10 N"}
                            ],
                            "answerKey": {"correctOption": "B"}
                        }
                    }
                ]
            ),
            make_request(
                name="Update Question",
                method="PUT",
                path="/api/v1/questions/:questionId",
                description="Memperbarui butir pertanyaan atau kunci jawaban.",
                variables=[{"key": "questionId", "value": "{{sample_questionId}}"}],
                body={
                    "type": "pilihan_ganda",
                    "body": "Sebuah benda bermassa 4 kg bergerak dengan percepatan 2 m/s2. Berapakah gaya yang bekerja pada benda tersebut?",
                    "options": [
                        {"id": "A", "label": "6 N"},
                        {"id": "B", "label": "8 N"},
                        {"id": "C", "label": "12 N"},
                        {"id": "D", "label": "16 N"}
                    ],
                    "answerKey": {"correctOption": "B"}
                },
                csrf=True,
                examples=[
                    {
                        "name": "200 OK - Question Updated",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "id": "c90d1e2f-3a4b-5c6d-7e8f-901a2b3c4d5e",
                            "bankId": "b89c0d1e-2f3a-4b5c-6d7e-8f901a2b3c4d",
                            "type": "pilihan_ganda",
                            "body": "Sebuah benda bermassa 4 kg bergerak dengan percepatan 2 m/s2. Berapakah gaya yang bekerja pada benda tersebut?",
                            "options": [
                                {"id": "A", "label": "6 N"},
                                {"id": "B", "label": "8 N"},
                                {"id": "C", "label": "12 N"},
                                {"id": "D", "label": "16 N"}
                            ],
                            "answerKey": {"correctOption": "B"}
                        }
                    }
                ]
            ),
            make_request(
                name="Delete Question",
                method="DELETE",
                path="/api/v1/questions/:questionId",
                description="Menghapus butir soal.",
                variables=[{"key": "questionId", "value": "{{sample_questionId}}"}],
                csrf=True,
                examples=[
                    {
                        "name": "204 No Content - Question Deleted",
                        "code": 204,
                        "status": "No Content",
                        "body": ""
                    }
                ]
            )
        ]
    }
    collection["item"].append(bank_folder)

    # ==========================================
    # 13 - Kuis & Ujian (Quiz & Attempts)
    # ==========================================
    quiz_folder = {
        "name": "13 - Kuis & Ujian",
        "description": "Publikasi paket kuis/ujian, pengerjaan kuis siswa, dan penilaian otomatis maupun manual.",
        "item": [
            make_request(
                name="List Quizzes",
                method="GET",
                path="/api/v1/quizzes",
                description="Melihat daftar kuis yang tersedia.",
                query=[
                    {"key": "page", "value": "1"},
                    {"key": "pageSize", "value": "20"},
                    {"key": "subjectId", "value": "{{sample_subjectId}}", "disabled": True},
                    {"key": "classId", "value": "{{sample_classId}}", "disabled": True},
                    {"key": "status", "value": "published", "disabled": True}
                ],
                examples=[
                    {
                        "name": "200 OK - Quizzes List",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "items": [
                                {
                                    "id": "d01e2f3a-4b5c-6d7e-8f90-1a2b3c4d5e6f",
                                    "title": "Kuis Fisika Bab 2 & 3",
                                    "subjectId": "d4e5f6a7-b89c-0d1e-2f3a-4b5c6d7e8f90",
                                    "description": "Durasi pengerjaan 60 menit. Dilarang bekerja sama."
                                }
                            ],
                            "total": 1,
                            "page": 1,
                            "pageSize": 20
                        }
                    }
                ]
            ),
            make_request(
                name="Create Quiz",
                method="POST",
                path="/api/v1/quizzes",
                description="Guru merakit kuis baru dari butir-butir soal yang ada di bank soal.",
                body={
                    "subjectId": "{{sample_subjectId}}",
                    "title": "Kuis Fisika Bab 2 & 3",
                    "description": "Durasi pengerjaan 60 menit. Dilarang bekerja sama.",
                    "publishedAt": "2026-11-20T08:00:00Z",
                    "startTime": "2026-11-20T08:00:00Z",
                    "endTime": "2026-11-20T09:30:00Z",
                    "questionIds": ["{{sample_questionId}}"]
                },
                csrf=True,
                examples=[
                    {
                        "name": "201 Created - Quiz Created",
                        "code": 201,
                        "status": "Created",
                        "body": {
                            "id": "d01e2f3a-4b5c-6d7e-8f90-1a2b3c4d5e6f",
                            "title": "Kuis Fisika Bab 2 & 3",
                            "startTime": "2026-11-20T08:00:00Z",
                            "endTime": "2026-11-20T09:30:00Z",
                            "questions": [
                                {
                                    "id": "c90d1e2f-3a4b-5c6d-7e8f-901a2b3c4d5e",
                                    "questionNumber": 1,
                                    "type": "pilihan_ganda",
                                    "question": "Sebuah benda bermassa 2 kg bergerak dengan percepatan 3 m/s2...",
                                    "options": [
                                        {"id": "A", "label": "5 N"},
                                        {"id": "B", "label": "6 N"}
                                    ]
                                }
                            ]
                        }
                    }
                ]
            ),
            make_request(
                name="Get Quiz Detail",
                method="GET",
                path="/api/v1/quizzes/:quizId",
                description="Mengambil lembar soal kuis untuk pengerjaan siswa.",
                variables=[{"key": "quizId", "value": "{{sample_quizId}}"}],
                query=[{"key": "attemptId", "value": "{{sample_attemptId}}", "disabled": True}],
                examples=[
                    {
                        "name": "200 OK - Quiz Detail",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "id": "d01e2f3a-4b5c-6d7e-8f90-1a2b3c4d5e6f",
                            "title": "Kuis Fisika Bab 2 & 3",
                            "startTime": "2026-11-20T08:00:00Z",
                            "endTime": "2026-11-20T09:30:00Z",
                            "questions": [
                                {
                                    "id": "c90d1e2f-3a4b-5c6d-7e8f-901a2b3c4d5e",
                                    "questionNumber": 1,
                                    "type": "pilihan_ganda",
                                    "question": "Sebuah benda bermassa 2 kg bergerak dengan percepatan 3 m/s2...",
                                    "options": [
                                        {"id": "A", "label": "5 N"},
                                        {"id": "B", "label": "6 N"}
                                    ]
                                }
                            ]
                        }
                    }
                ]
            ),
            make_request(
                name="Start Quiz Attempt",
                method="POST",
                path="/api/v1/quizzes/:quizId/attempts",
                description="Siswa memulai pengerjaan kuis (membuat attempt baru).",
                variables=[{"key": "quizId", "value": "{{sample_quizId}}"}],
                csrf=True,
                examples=[
                    {
                        "name": "201 Created - Attempt Started",
                        "code": 201,
                        "status": "Created",
                        "body": {
                            "attemptId": "e12f3a4b-5c6d-7e8f-901a-2b3c4d5e6f70",
                            "quizId": "d01e2f3a-4b5c-6d7e-8f90-1a2b3c4d5e6f",
                            "quizTitle": "Kuis Fisika Bab 2 & 3",
                            "startedAt": "2026-11-20T08:05:00Z",
                            "submittedAt": None,
                            "status": "in_progress",
                            "score": None
                        }
                    }
                ]
            ),
            make_request(
                name="List Quiz Attempts",
                method="GET",
                path="/api/v1/quiz-attempts",
                description="Melihat daftar riwayat sesi pengerjaan kuis.",
                query=[
                    {"key": "page", "value": "1"},
                    {"key": "pageSize", "value": "20"},
                    {"key": "quizId", "value": "{{sample_quizId}}", "disabled": True},
                    {"key": "studentUserId", "value": "{{sample_userId}}", "disabled": True},
                    {"key": "status", "value": "submitted", "disabled": True}
                ],
                examples=[
                    {
                        "name": "200 OK - Attempts List",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "items": [
                                {
                                    "attemptId": "e12f3a4b-5c6d-7e8f-901a-2b3c4d5e6f70",
                                    "quizId": "d01e2f3a-4b5c-6d7e-8f90-1a2b3c4d5e6f",
                                    "quizTitle": "Kuis Fisika Bab 2 & 3",
                                    "startedAt": "2026-11-20T08:05:00Z",
                                    "submittedAt": "2026-11-20T08:45:00Z",
                                    "status": "submitted",
                                    "score": 90.0
                                }
                            ],
                            "total": 1,
                            "page": 1,
                            "pageSize": 20
                        }
                    }
                ]
            ),
            make_request(
                name="Get Quiz Attempt by ID",
                method="GET",
                path="/api/v1/quiz-attempts/:attemptId",
                description="Melihat lembar hasil pengerjaan kuis beserta rincian jawaban.",
                variables=[{"key": "attemptId", "value": "{{sample_attemptId}}"}],
                examples=[
                    {
                        "name": "200 OK - Attempt Details",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "attemptId": "e12f3a4b-5c6d-7e8f-901a-2b3c4d5e6f70",
                            "quizId": "d01e2f3a-4b5c-6d7e-8f90-1a2b3c4d5e6f",
                            "quizTitle": "Kuis Fisika Bab 2 & 3",
                            "startedAt": "2026-11-20T08:05:00Z",
                            "submittedAt": "2026-11-20T08:45:00Z",
                            "status": "submitted",
                            "score": 90.0,
                            "essayPending": False,
                            "answers": [
                                {
                                    "answerId": "11223344-5566-7788-99aa-bbccddeeff00",
                                    "questionId": 1,
                                    "userAnswer": "B",
                                    "isCorrect": True
                                }
                            ]
                        }
                    }
                ]
            ),
            make_request(
                name="Save Answer",
                method="PATCH",
                path="/api/v1/quiz-attempts/:attemptId/answers/:questionId",
                description="Menyimpan jawaban sementara siswa untuk nomor soal tertentu.",
                variables=[
                    {"key": "attemptId", "value": "{{sample_attemptId}}"},
                    {"key": "questionId", "value": "1", "description": "Nomor urut soal"}
                ],
                body={"userAnswer": "B"},
                csrf=True,
                examples=[
                    {
                        "name": "200 OK - Answer Saved",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "questionId": 1,
                            "userAnswer": "B",
                            "isCompleted": True,
                            "updatedAt": "2026-11-20T08:15:00Z"
                        }
                    }
                ]
            ),
            make_request(
                name="Submit Quiz Attempt",
                method="POST",
                path="/api/v1/quiz-attempts/:attemptId/submit",
                description="Siswa mengakhiri dan mengumpulkan lembar jawaban kuis.",
                variables=[{"key": "attemptId", "value": "{{sample_attemptId}}"}],
                csrf=True,
                examples=[
                    {
                        "name": "200 OK - Quiz Submitted",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "attemptId": "e12f3a4b-5c6d-7e8f-901a-2b3c4d5e6f70",
                            "quizId": "d01e2f3a-4b5c-6d7e-8f90-1a2b3c4d5e6f",
                            "quizTitle": "Kuis Fisika Bab 2 & 3",
                            "startedAt": "2026-11-20T08:05:00Z",
                            "submittedAt": "2026-11-20T08:45:00Z",
                            "status": "submitted",
                            "score": 90.0,
                            "essayPending": False,
                            "answers": [
                                {
                                    "answerId": "11223344-5566-7788-99aa-bbccddeeff00",
                                    "questionId": 1,
                                    "userAnswer": "B",
                                    "isCorrect": True
                                }
                            ]
                        }
                    }
                ]
            ),
            make_request(
                name="Grade Answer (Essay)",
                method="PUT",
                path="/api/v1/quiz-attempts/:attemptId/answers/:answerId/grade",
                description="Guru menilai jawaban soal essay siswa secara manual.",
                variables=[
                    {"key": "attemptId", "value": "{{sample_attemptId}}"},
                    {"key": "answerId", "value": "{{sample_questionId}}"}
                ],
                body={
                    "isCorrect": True,
                    "score": 10.0,
                    "feedback": "Penjelasan sangat runtut dan menggunakan istilah ilmiah yang tepat."
                },
                csrf=True,
                examples=[
                    {
                        "name": "200 OK - Answer Graded",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "answerId": "11223344-5566-7788-99aa-bbccddeeff00",
                            "isCorrect": True,
                            "score": 10.0,
                            "gradedBy": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                            "gradedAt": "2026-11-20T10:00:00Z"
                        }
                    }
                ]
            )
        ]
    }
    collection["item"].append(quiz_folder)

    # ==========================================
    # 14 - Notifikasi (Notifications)
    # ==========================================
    notif_folder = {
        "name": "14 - Notifikasi",
        "description": "Pengiriman dan penerimaan pesan pemberitahuan in-app bagi pengguna.",
        "item": [
            make_request(
                name="List Notifications",
                method="GET",
                path="/api/v1/notifications",
                description="Mengambil daftar notifikasi milik pengguna login.",
                query=[
                    {"key": "page", "value": "1"},
                    {"key": "pageSize", "value": "20"},
                    {"key": "unreadOnly", "value": "false", "disabled": True}
                ],
                examples=[
                    {
                        "name": "200 OK - Notifications List",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "items": [
                                {
                                    "id": "a34b5c6d-7e8f-901a-2b3c-4d5e6f708192",
                                    "userId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                                    "type": "announcement",
                                    "payload": {
                                        "title": "Jadwal Ujian Tengah Semester",
                                        "message": "Jadwal UTS telah diunggah ke portal akademik. Harap persiapkan diri dengan baik."
                                    },
                                    "isRead": False,
                                    "createdAt": "2026-11-15T08:00:00Z"
                                }
                            ],
                            "total": 1,
                            "page": 1,
                            "pageSize": 20
                        }
                    }
                ]
            ),
            make_request(
                name="Create Notification",
                method="POST",
                path="/api/v1/notifications",
                description="Membuat dan mengirimkan notifikasi baru ke sistem.",
                body={
                    "type": "announcement",
                    "payload": {
                        "title": "Jadwal Ujian Tengah Semester",
                        "message": "Jadwal UTS telah diunggah ke portal akademik. Harap persiapkan diri dengan baik."
                    }
                },
                csrf=True,
                examples=[
                    {
                        "name": "201 Created - Notification Created",
                        "code": 201,
                        "status": "Created",
                        "body": {
                            "id": "a34b5c6d-7e8f-901a-2b3c-4d5e6f708192",
                            "userId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                            "type": "announcement",
                            "payload": {
                                "title": "Jadwal Ujian Tengah Semester",
                                "message": "Jadwal UTS telah diunggah ke portal akademik. Harap persiapkan diri dengan baik."
                            },
                            "isRead": False,
                            "createdAt": "2026-11-15T08:00:00Z"
                        }
                    }
                ]
            ),
            make_request(
                name="Mark Notification as Read",
                method="POST",
                path="/api/v1/notifications/:notificationId/read",
                description="Menandai notifikasi telah dibaca.",
                variables=[{"key": "notificationId", "value": "{{sample_notificationId}}"}],
                csrf=True,
                examples=[
                    {
                        "name": "200 OK - Marked as Read",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "id": "a34b5c6d-7e8f-901a-2b3c-4d5e6f708192",
                            "userId": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
                            "type": "announcement",
                            "payload": {
                                "title": "Jadwal Ujian Tengah Semester",
                                "message": "Jadwal UTS telah diunggah ke portal akademik. Harap persiapkan diri dengan baik."
                            },
                            "isRead": True,
                            "createdAt": "2026-11-15T08:00:00Z"
                        }
                    }
                ]
            )
        ]
    }
    collection["item"].append(notif_folder)

    # ==========================================
    # 15 - Biometrik Fingerprint (Fingerprint Gateway)
    # ==========================================
    fingerprint_folder = {
        "name": "15 - Biometrik Fingerprint",
        "description": "Gateway biometrik sidik jari untuk identifikasi presensi, pendaftaran (enrollment), dan server-sent events stream.",
        "item": [
            make_request(
                name="Identify Fingerprint (Raw Binary)",
                method="POST",
                path="/api/v1/fingerprint/identify",
                description="Identifikasi sidik jari dengan mengirimkan data binary raw template (Content-Type: application/octet-stream).",
                content_type="application/octet-stream",
                is_binary=True,
                examples=[
                    {
                        "name": "200 OK - Match Found",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "success": True,
                            "matched": True,
                            "score": 98.4,
                            "externalUserId": "siswa-2024101001",
                            "message": "Identifikasi berhasil"
                        }
                    },
                    {
                        "name": "200 OK - No Match",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "success": True,
                            "matched": False,
                            "score": 42.1,
                            "externalUserId": "",
                            "message": "Sidik jari tidak dikenali"
                        }
                    }
                ]
            ),
            make_request(
                name="Enroll Fingerprint (JSON Base64)",
                method="POST",
                path="/api/v1/fingerprint/enroll",
                description="Mendaftarkan template sidik jari dengan format base64 yang dikaitkan ke externalUserId (NIS / NIP).",
                body={
                    "externalUserId": "siswa-2024101001",
                    "template": "AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyAhIiMkJSYnKCkqKywtLi8w"
                },
                examples=[
                    {
                        "name": "200 OK - Enrollment Success",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "success": True,
                            "message": "Pendaftaran sidik jari berhasil",
                            "externalUserId": "siswa-2024101001"
                        }
                    }
                ]
            ),
            make_request(
                name="Enroll Fingerprint (Raw Binary)",
                method="POST",
                path="/api/v1/fingerprint/enroll/raw",
                description="Mendaftarkan template sidik jari binary mentah dengan query ?externalUserId.",
                query=[{"key": "externalUserId", "value": "siswa-2024101001"}],
                content_type="application/octet-stream",
                is_binary=True,
                examples=[
                    {
                        "name": "200 OK - Enrollment Success",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "success": True,
                            "message": "Pendaftaran sidik jari binary berhasil",
                            "externalUserId": "siswa-2024101001"
                        }
                    }
                ]
            ),
            make_request(
                name="Get Latest Identification",
                method="GET",
                path="/api/v1/fingerprint/latest",
                description="Mengambil data scan presensi sidik jari terakhir yang berhasil dipindai oleh mesin.",
                examples=[
                    {
                        "name": "200 OK - Latest Identification",
                        "code": 200,
                        "status": "OK",
                        "body": {
                            "success": True,
                            "data": {
                                "externalUserId": "siswa-2024101001",
                                "score": 95.2,
                                "timestamp": "2026-11-15T07:05:12Z"
                            }
                        }
                    }
                ]
            ),
            make_request(
                name="Stream Realtime Identifications (SSE)",
                method="GET",
                path="/api/v1/fingerprint/stream",
                description="Koneksi Server-Sent Events (SSE) realtime untuk menerima hasil pemindaian presensi sidik jari secara langsung.",
                content_type="text/event-stream",
                examples=[
                    {
                        "name": "200 OK - Realtime Event Stream",
                        "code": 200,
                        "status": "OK",
                        "contentType": "text/event-stream",
                        "body": ": connected\n\ndata: {\"externalUserId\":\"siswa-2024101001\",\"score\":95.2,\"timestamp\":\"2026-11-15T07:05:12Z\"}\n\n: keepalive\n\n"
                    }
                ]
            )
        ]
    }
    collection["item"].append(fingerprint_folder)

    return collection

if __name__ == "__main__":
    col = build_collection()
    out_path = os.path.join(os.path.dirname(__file__), "limen_api_collection.json")
    with open(out_path, "w", encoding="utf-8") as f:
        json.dump(col, f, indent=2, ensure_ascii=False)
    print(f"Postman collection successfully generated at: {out_path}")
