# Quiz API - Frontend Integration Guide

Base URL: `/api/v1`

## Authentication

Semua endpoint butuh cookie `session` valid. Endpoint write (POST/PUT/PATCH/DELETE) butuh cookie `csrf` dan header `X-CSRF-Token` sama dengan nilai CSRF token.

**Error responses:**
- `401 UNAUTHENTICATED` - session hilang/expired
- `403 CSRF_INVALID` - CSRF token salah/hilang di write endpoint

## Question Bank Endpoints

### List Question Banks
```http
GET /api/v1/question-banks
```

**Query params:**
- `scope` - `latihan` atau `ujian`
- `subjectId` - UUID subject
- `ownerId` - UUID owner
- `page` - default 1
- `pageSize` - default 20, max 100

**Response 200:**
```json
{
  "items": [
    {
      "id": "uuid",
      "subjectId": "uuid",
      "scope": "latihan",
      "ownerId": "uuid",
      "questionCount": 5
    }
  ],
  "total": 1,
  "page": 1,
  "pageSize": 20
}
```

### Create Question Bank
```http
POST /api/v1/question-banks
```

**Request body:**
```json
{
  "subjectId": "uuid",
  "scope": "latihan"
}
```

**Response 201:** object `QuestionBank`

### List Questions in Bank
```http
GET /api/v1/question-banks/:bankId/questions
```

**Query params:**
- `page`, `pageSize`

**Response 200:**
```json
{
  "items": [
    {
      "id": "uuid",
      "bankId": "uuid",
      "type": "pilihan_ganda",
      "body": "Pertanyaan text",
      "options": {"choices": [{"id": "A", "text": "..."}]},
      "answerKey": {"correct": "A"}
    }
  ],
  "total": 5,
  "page": 1,
  "pageSize": 20
}
```

⚠️ **Warning:** Endpoint ini expose `answerKey` ke semua user authenticated.

### Create Question
```http
POST /api/v1/question-banks/:bankId/questions
```

**Request body:**
```json
{
  "type": "pilihan_ganda",
  "body": "Pertanyaan text (required)",
  "options": {},
  "answerKey": {}
}
```

**Question types:**
- `pilihan_ganda`
- `pilihan_ganda_kompleks`
- `benar_salah`
- `rentang_nilai`
- `essay`

**Format `options` dan `answerKey` per type:**

```json
// pilihan_ganda
{
  "options": {"choices": [{"id": "A", "text": "Jawaban A"}]},
  "answerKey": {"correct": "A"}
}

// pilihan_ganda_kompleks
{
  "options": {"choices": [{"id": "A", "text": "..."}, {"id": "B", "text": "..."}]},
  "answerKey": {"correct": ["A", "B"]}
}

// benar_salah
{
  "answerKey": {"correct": true}
}

// rentang_nilai
{
  "options": {"min": 0, "max": 10},
  "answerKey": {"min": 7, "max": 10}
}

// essay
{
  "answerKey": {"rubric": "Kriteria penilaian..."}
}
```

**Response 201:** object `Question`

### Update Question
```http
PUT /api/v1/questions/:questionId
```

**Request body:** sama dengan create question

**Response 200:** object `Question`

### Delete Question
```http
DELETE /api/v1/questions/:questionId
```

**Response 204** - no content

**Error 409 CONFLICT** - question masih dipakai di quiz

## Quiz Endpoints

### List Quizzes
```http
GET /api/v1/quizzes
```

**Query params:**
- `subjectId` - UUID
- `classId` - UUID (tidak di-filter di backend saat ini)
- `status` - `draft`, `published`, `active`, `ended`
- `page`, `pageSize`

**Response 200:**
```json
{
  "items": [
    {
      "id": "uuid",
      "title": "Quiz Title",
      "subjectId": "uuid",
      "description": "..."
    }
  ],
  "total": 1,
  "page": 1,
  "pageSize": 20
}
```

### Create Quiz
```http
POST /api/v1/quizzes
```

**Request body:**
```json
{
  "subjectId": "uuid",
  "title": "Quiz Title (required)",
  "description": "Optional description",
  "publishedAt": "2026-11-02T10:00:00Z",
  "startTime": "2026-11-02T10:00:00Z",
  "endTime": "2026-11-02T11:00:00Z",
  "questionIds": ["uuid-q1", "uuid-q2", "uuid-q3"]
}
```

**Validasi:**
- `title` tidak boleh kosong
- `startTime < endTime`
- Timestamps format RFC3339
- `questionIds` boleh array kosong

**Response 201:** object `QuizDetail`

### Get Quiz Detail
```http
GET /api/v1/quizzes/:quizId
```

**Query params (optional):**
- `attemptId` - UUID attempt untuk verifikasi

**Response 200:**
```json
{
  "id": "uuid",
  "title": "Quiz Title",
  "startTime": "2026-11-02T10:00:00Z",
  "endTime": "2026-11-02T11:00:00Z",
  "questions": [
    {
      "id": 1,
      "questionNumber": 1,
      "type": "pilihan_ganda",
      "question": "Pertanyaan text",
      "options": [{"id": 0, "label": "Jawaban A"}]
    }
  ]
}
```

⚠️ **Warning:** 
- Response tidak include `answerKey`
- `questions[].id` adalah integer (question number), bukan UUID
- Endpoint terbuka untuk semua authenticated user

## Quiz Attempt Endpoints (Student Flow)

### Start Quiz Attempt
```http
POST /api/v1/quizzes/:quizId/attempts
```

**Request body:** kosong

**Response 201:**
```json
{
  "attemptId": "uuid",
  "quizId": "uuid",
  "quizTitle": "Quiz Title",
  "startedAt": "2026-11-02T10:15:00Z",
  "submittedAt": null,
  "status": "in_progress",
  "score": 0
}
```

**Error 409 CONFLICT** - sudah ada attempt `in_progress` untuk quiz ini

⚠️ **Warning:** Tidak ada validasi `publishedAt`, `startTime`, `endTime`. FE harus cek sendiri.

### List Attempts
```http
GET /api/v1/quiz-attempts
```

**Query params:**
- `quizId` - UUID
- `studentUserId` - UUID
- `status` - `in_progress`, `submitted`, `graded`
- `page`, `pageSize`

**Response 200:**
```json
{
  "items": [
    {
      "attemptId": "uuid",
      "quizId": "uuid",
      "quizTitle": "Quiz Title",
      "startedAt": "2026-11-02T10:15:00Z",
      "submittedAt": "2026-11-02T10:45:00Z",
      "status": "graded",
      "score": 85.5
    }
  ],
  "total": 1,
  "page": 1,
  "pageSize": 20
}
```

### Get Attempt Result
```http
GET /api/v1/quiz-attempts/:attemptId
```

**Response 200:**
```json
{
  "attemptId": "uuid",
  "quizId": "uuid",
  "quizTitle": "Quiz Title",
  "startedAt": "2026-11-02T10:15:00Z",
  "submittedAt": "2026-11-02T10:45:00Z",
  "status": "graded",
  "score": 85.5,
  "essayPending": false,
  "answers": [
    {
      "answerId": "uuid",
      "questionId": 1,
      "userAnswer": "A",
      "isCorrect": true
    }
  ]
}
```

**Field notes:**
- `essayPending: true` - ada essay belum di-grade
- `userAnswer` - bisa JSON scalar, array, object, atau null
- `isCorrect` - null kalau belum di-grade

### Save Answer (Autosave)
```http
PATCH /api/v1/quiz-attempts/:attemptId/answers/:questionId
```

⚠️ **Important:** `:questionId` adalah **question number** (integer 1, 2, 3...), bukan UUID question.

**Request body:**
```json
{
  "userAnswer": "A"
}
```

**Response 200:**
```json
{
  "questionId": 1,
  "userAnswer": "A",
  "isCompleted": false,
  "updatedAt": "2026-11-02T10:20:00Z"
}
```

**Field notes:**
- `isCompleted: true` - jumlah answer records >= jumlah soal quiz. Tidak validasi value.
- `userAnswer` terima any valid JSON

**Error 409 CONFLICT** - attempt tidak `in_progress`

### Submit Quiz
```http
POST /api/v1/quiz-attempts/:attemptId/submit
```

**Request body:** kosong

**Response 200:** sama dengan GET attempt result

**Behavior:**
- Auto-grading objective questions (`pilihan_ganda`, `pilihan_ganda_kompleks`, `benar_salah`, `rentang_nilai`)
- Essay tetap `isCorrect: null`, `score: null`
- Status jadi `submitted` kalau ada essay pending
- Status jadi `graded` kalau semua question sudah ada score

**Error 409 CONFLICT** - attempt tidak `in_progress`

## Grading Endpoint (Teacher Flow)

### Grade Answer
```http
PUT /api/v1/quiz-attempts/:attemptId/answers/:answerId/grade
```

⚠️ **Important:** `:answerId` adalah **UUID answer**, bukan question number.

**Request body:**
```json
{
  "isCorrect": true,
  "score": 100,
  "feedback": "Bagus"
}
```

**Validasi:**
- Objective question: butuh `isCorrect` atau `score`. Jika `isCorrect: true` maka `score` forced 100, else 0.
- Essay: butuh `score` atau `isCorrect`. Range `score` 0-100.
- `feedback` opsional, tersimpan tapi tidak muncul di response

**Response 200:**
```json
{
  "answerId": "uuid",
  "isCorrect": true,
  "score": 100,
  "gradedBy": "uuid-user",
  "gradedAt": "2026-11-02T11:00:00Z"
}
```

**Status transition:**
- Attempt jadi `graded` kalau submitted dan semua answer punya score non-null

## Student Quiz Flow

```
1. GET /quizzes → pilih quiz
2. POST /quizzes/:quizId/attempts → mulai attempt
3. GET /quizzes/:quizId?attemptId=:attemptId → load soal
4. Loop: PATCH /quiz-attempts/:attemptId/answers/:questionNumber → autosave
5. Optional: GET /quiz-attempts/:attemptId → cek progress/resume
6. POST /quiz-attempts/:attemptId/submit → submit
7. GET /quiz-attempts/:attemptId → lihat hasil
   - Jika essayPending: true, tunggu teacher grading
   - Polling/refresh untuk cek status jadi graded
```

## Teacher Quiz Flow

```
1. POST /question-banks → buat bank
2. POST /question-banks/:bankId/questions (multiple) → buat soal
3. GET /question-banks → list bank
4. GET /question-banks/:bankId/questions → pilih soal
5. POST /quizzes → buat quiz dengan questionIds
6. GET /quiz-attempts?status=submitted → cek submission
7. GET /quiz-attempts/:attemptId → detail jawaban student
8. Loop essay: PUT /quiz-attempts/:attemptId/answers/:answerId/grade
9. GET /quiz-attempts/:attemptId → verify status jadi graded
```

## Error Response Format

```json
{
  "code": "VALIDATION_ERROR",
  "message": "validation failed",
  "details": {
    "fields": [
      {
        "field": "title",
        "message": "title is required"
      }
    ]
  }
}
```

**Common error codes:**
- `401 UNAUTHENTICATED` - auth gagal
- `403 CSRF_INVALID` - CSRF token salah
- `404 NOT_FOUND` - resource tidak ada
- `409 CONFLICT` - constraint violation (active attempt, referenced question, non-in-progress save/submit)
- `422 VALIDATION_ERROR` - input tidak valid
- `500 INTERNAL_ERROR` - server error

## Security Warnings

⚠️ **Backend tidak enforce authorization berikut - FE harus implement sendiri:**

1. **No role checks** - semua authenticated user bisa create/edit/delete bank, question, quiz, attempt, dan grading
2. **No ownership checks** - user bisa akses/edit/submit attempt user lain
3. **No schedule enforcement** - student bisa start attempt kapan saja, ignore `publishedAt`/`startTime`/`endTime`
4. **Answer key exposure** - GET question bank endpoints expose `answerKey` ke semua user
5. **Quiz question exposure** - GET quiz detail terbuka tanpa butuh attempt
6. **Race condition** - concurrent POST attempts bisa buat multiple `in_progress` attempt

**Rekomendasi FE:**
- Check user role sebelum show create/edit/grade UI
- Filter quiz list by schedule sebelum allow start attempt
- Validate attempt ownership sebelum save/submit
- Jangan fetch question bank dengan answer key di student UI
- Implement client-side attempt lock/warning

## Data Seeder

Run test seeder:
```bash
make seed-quiz
```

Seeder buat:
- User `guru.quiz@limen.test` (role `guru_mapel`)
- User `siswa.quiz@limen.test` (role `siswa`)
- Subject `QZTEST`
- Sample question bank dengan 5 question types
- Sample quiz dengan attempt dan grading

Seeder idempotent untuk user/subject, non-idempotent untuk bank/quiz/attempt (re-run buat data baru).
