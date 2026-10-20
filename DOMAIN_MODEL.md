# Domain Model

## User

- `users`
- `roles`
- `user_roles`
- `sessions`

## Academic

- `students`
- `teachers`
- `classes`
- `subjects`
- `teaching_assignments`
- `schedules`

## Attendance

- `school_settings`
- `devices`
- `biometric_identities`
- `attendance_events`
- `attendance_records`
- `attendance_corrections`

## Audit

- `audit_logs`
- `notifications`

## Assessment

- `assessments`
- `assessment_scores`

## Assignment

- `assignments`
- `assignment_submissions`

## Quiz

- `question_banks`
- `questions`
- `quizzes`
- `quiz_questions`
- `quiz_attempts`
- `quiz_answers`

## Counseling

- `counseling_notes`
- `counseling_followups`
- `aptitude_results`
- `target_majors`
- `passing_grades`

## Cross-Domain Relations

- `academic` depends on `user` through student and teacher user IDs.
- `attendance` depends on `user` for attendance subjects and actors.
- `assessment`, `assignment`, and `quiz` depend on `user` and `academic`.
- `counseling` depends on student user IDs.
- `audit` records activity across all domains.

## Deferred

Business rules and role-based access rules are not finalized.
