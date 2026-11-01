package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"limen/internal/adapter/persistence/postgres/academic"
	"limen/internal/adapter/persistence/postgres/quiz"
	"limen/internal/platform/config"
	"limen/internal/platform/database"
	"limen/internal/usecase/auth"
	questionbankuc "limen/internal/usecase/questionbank"
	quizuc "limen/internal/usecase/quiz"

	"limen/internal/domain"
	academicdomain "limen/internal/domain/academic"
	quizdomain "limen/internal/domain/quiz"

	"github.com/sirupsen/logrus"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	db, err := database.NewPostgresConnection(ctx, cfg)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get postgres pool: %w", err)
	}
	defer sqlDB.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	subjectRepo := academic.NewSubjectRepository(db)
	bankRepo := quiz.NewQuestionBankRepository(db)
	questionRepo := quiz.NewQuestionRepository(db)
	quizRepo := quiz.NewQuizRepository(db)
	quizQuestionRepo := quiz.NewQuizQuestionRepository(db)
	attemptRepo := quiz.NewQuizAttemptRepository(db)
	answerRepo := quiz.NewQuizAnswerRepository(db)
	queryRepo := quiz.NewQueryRepository(db)

	bankUC := questionbankuc.NewUseCase(bankRepo, questionRepo, subjectRepo, logger)
	quizUC := quizuc.NewUseCase(quizRepo, questionRepo, quizQuestionRepo, attemptRepo, answerRepo, queryRepo, subjectRepo, logger)

	teacherID, err := seedUser(ctx, sqlDB, "guru.quiz@limen.test", "Guru Quiz", "GuruQuiz123", "guru_mapel")
	if err != nil {
		return err
	}
	studentID, err := seedUser(ctx, sqlDB, "siswa.quiz@limen.test", "Siswa Quiz", "SiswaQuiz123", "siswa")
	if err != nil {
		return err
	}
	subjectID, err := seedSubject(ctx, subjectRepo, "QZTEST", "Quiz Testing")
	if err != nil {
		return err
	}
	fmt.Printf("seeded teacher=%s student=%s subject=%s\n", teacherID, studentID, subjectID)

	bank, err := bankUC.CreateBank(ctx, subjectID, teacherID, quizdomain.QuestionBankScopeExam)
	if err != nil {
		return fmt.Errorf("create bank: %w", err)
	}
	fmt.Printf("created bank=%s scope=%s\n", bank.ID, bank.Scope)

	questions := []struct {
		input   questionbankuc.QuestionInput
		answer  json.RawMessage
		correct bool
	}{
		{
			input: questionbankuc.QuestionInput{
				Type:      quizdomain.QuestionTypeMultipleChoice,
				Body:      "Ibu kota Indonesia adalah?",
				Options:   raw(`{"choices":[{"id":"A","text":"Bandung"},{"id":"B","text":"Jakarta"},{"id":"C","text":"Surabaya"}]}`),
				AnswerKey: raw(`{"correct":"B"}`),
			},
			answer:  raw(`"B"`),
			correct: true,
		},
		{
			input: questionbankuc.QuestionInput{
				Type:      quizdomain.QuestionTypeMultipleSelect,
				Body:      "Pilih bilangan prima.",
				Options:   raw(`{"choices":[{"id":"A","text":"2"},{"id":"B","text":"3"},{"id":"C","text":"4"},{"id":"D","text":"5"}]}`),
				AnswerKey: raw(`{"correct":["A","B","D"]}`),
			},
			answer:  raw(`["A","B","D"]`),
			correct: true,
		},
		{
			input: questionbankuc.QuestionInput{
				Type:      quizdomain.QuestionTypeTrueFalse,
				Body:      "Air mendidih pada 100°C di tekanan 1 atm.",
				AnswerKey: raw(`{"correct":true}`),
			},
			answer:  raw(`true`),
			correct: true,
		},
		{
			input: questionbankuc.QuestionInput{
				Type:      quizdomain.QuestionTypeRange,
				Body:      "Isi skor kepuasan 0-10.",
				Options:   raw(`{"min":0,"max":10}`),
				AnswerKey: raw(`{"min":7,"max":10}`),
			},
			answer:  raw(`8`),
			correct: true,
		},
		{
			input: questionbankuc.QuestionInput{
				Type:      quizdomain.QuestionTypeEssay,
				Body:      "Jelaskan proses fotosintesis.",
				AnswerKey: raw(`{"rubric":"klorofil, cahaya, CO2, H2O, glukosa"}`),
			},
			answer:  raw(`"Fotosintesis mengubah cahaya menjadi energi kimia."`),
			correct: true,
		},
	}

	questionIDs := make([]string, len(questions))
	for i, q := range questions {
		created, err := bankUC.CreateQuestion(ctx, bank.ID, q.input)
		if err != nil {
			return fmt.Errorf("create question %d: %w", i+1, err)
		}
		questionIDs[i] = created.ID
		fmt.Printf("created question #%d id=%s type=%s\n", i+1, created.ID, created.Type)
	}

	quizDetail, err := quizUC.CreateQuiz(ctx, quizuc.CreateQuizInput{
		SubjectID:   subjectID,
		OwnerID:     teacherID,
		Title:       "Quiz Testing Seeder",
		Description: ptr("Quiz otomatis untuk menguji mekanisme pengerjaan dan penilaian."),
		QuestionIDs: questionIDs,
	})
	if err != nil {
		return fmt.Errorf("create quiz: %w", err)
	}
	fmt.Printf("created quiz=%s title=%q questions=%d\n", quizDetail.ID, quizDetail.Title, len(quizDetail.Questions))

	attempt, err := quizUC.StartAttempt(ctx, quizDetail.ID, studentID)
	if err != nil {
		return fmt.Errorf("start attempt: %w", err)
	}
	fmt.Printf("started attempt=%s status=%s\n", attempt.AttemptID, attempt.Status)

	for i, q := range questions {
		saved, err := quizUC.SaveAnswer(ctx, attempt.AttemptID, i+1, q.answer)
		if err != nil {
			return fmt.Errorf("save answer #%d: %w", i+1, err)
		}
		fmt.Printf("saved answer #%d userAnswer=%s isCompleted=%t\n", saved.QuestionID, string(saved.UserAnswer), saved.IsCompleted)
	}

	submitted, err := quizUC.SubmitAttempt(ctx, attempt.AttemptID)
	if err != nil {
		return fmt.Errorf("submit attempt: %w", err)
	}
	printResult("after submit", submitted)

	var essayAnswerID string
	for _, answer := range submitted.Answers {
		if answer.QuestionID == len(questions) {
			essayAnswerID = answer.AnswerID
		}
	}
	if essayAnswerID == "" {
		return errors.New("essay answer not found in submit result")
	}
	correct := true
	score := 100.0
	if _, err := quizUC.GradeAnswer(ctx, attempt.AttemptID, essayAnswerID, teacherID, quizuc.GradeInput{
		IsCorrect: &correct,
		Score:     &score,
		Feedback:  ptr("Jawaban benar, proses sudah disebutkan."),
	}); err != nil {
		return fmt.Errorf("grade essay answer: %w", err)
	}

	final, err := quizUC.GetAttempt(ctx, attempt.AttemptID)
	if err != nil {
		return fmt.Errorf("get final attempt: %w", err)
	}
	printResult("after grading essay", final)
	fmt.Println("quiz testing seeder completed successfully")
	return nil
}

func printResult(label string, result quizdomain.AttemptResult) {
	score := "null"
	if result.Score != nil {
		score = fmt.Sprintf("%.2f", *result.Score)
	}
	fmt.Printf("[%s] attempt=%s quiz=%q status=%s score=%s essayPending=%t answers=%d\n",
		label, result.AttemptID, result.QuizTitle, result.Status, score, result.EssayPending, len(result.Answers))
	for _, answer := range result.Answers {
		isCorrect := "null"
		if answer.IsCorrect != nil {
			isCorrect = fmt.Sprintf("%t", *answer.IsCorrect)
		}
		fmt.Printf("  - q#%d answerId=%s isCorrect=%s userAnswer=%s\n", answer.QuestionID, answer.AnswerID, isCorrect, string(answer.UserAnswer))
	}
}

func seedUser(ctx context.Context, sqlDB *sql.DB, email, fullName, password, roleCode string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	hash, err := auth.HashArgon2idPassword(password)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO roles (code, name) VALUES ($1, $2)
		ON CONFLICT (code) DO NOTHING
	`, roleCode, roleCode); err != nil {
		return "", fmt.Errorf("seed role %s: %w", roleCode, err)
	}

	var userID string
	if err := tx.QueryRowContext(ctx, `
		INSERT INTO users (email, password_hash, full_name, status)
		VALUES ($1, $2, $3, 'active')
		ON CONFLICT (email) DO UPDATE
		SET password_hash = EXCLUDED.password_hash,
			full_name = EXCLUDED.full_name,
			status = 'active',
			updated_at = now(),
			deleted_at = NULL
		RETURNING id
	`, email, hash, fullName).Scan(&userID); err != nil {
		return "", fmt.Errorf("seed user %s: %w", email, err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO user_roles (user_id, role_code) VALUES ($1, $2)
		ON CONFLICT (user_id, role_code) DO NOTHING
	`, userID, roleCode); err != nil {
		return "", fmt.Errorf("assign role %s: %w", roleCode, err)
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return userID, nil
}

func seedSubject(ctx context.Context, repo *academic.SubjectRepository, code, name string) (string, error) {
	existing, err := repo.FindByCode(ctx, code)
	if err == nil {
		return existing.ID, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return "", fmt.Errorf("find subject %s: %w", code, err)
	}
	id, err := repo.Create(ctx, academicdomain.Subject{Code: code, Name: name})
	if err != nil {
		return "", fmt.Errorf("create subject %s: %w", code, err)
	}
	return id, nil
}

func raw(value string) json.RawMessage {
	return json.RawMessage(value)
}

func ptr(value string) *string {
	return &value
}
