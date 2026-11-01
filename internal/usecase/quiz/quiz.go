package quiz

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	"limen/internal/domain"
	academicdomain "limen/internal/domain/academic"
	quizdomain "limen/internal/domain/quiz"

	"github.com/sirupsen/logrus"
)

const fullScore = 100.0

type UseCase struct {
	quizzes   quizdomain.QuizRepository
	questions quizdomain.QuestionRepository
	quizQs    quizdomain.QuizQuestionRepository
	attempts  quizdomain.QuizAttemptRepository
	answers   quizdomain.QuizAnswerRepository
	query     quizdomain.QueryRepository
	subjects  academicdomain.SubjectRepository
	logger    *logrus.Logger
}

func NewUseCase(
	quizzes quizdomain.QuizRepository,
	questions quizdomain.QuestionRepository,
	quizQs quizdomain.QuizQuestionRepository,
	attempts quizdomain.QuizAttemptRepository,
	answers quizdomain.QuizAnswerRepository,
	query quizdomain.QueryRepository,
	subjects academicdomain.SubjectRepository,
	logger *logrus.Logger,
) *UseCase {
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	return &UseCase{quizzes: quizzes, questions: questions, quizQs: quizQs, attempts: attempts, answers: answers, query: query, subjects: subjects, logger: logger}
}

type CreateQuizInput struct {
	SubjectID   string
	OwnerID     string
	Title       string
	Description *string
	PublishedAt *time.Time
	StartTime   *time.Time
	EndTime     *time.Time
	QuestionIDs []string
}

type SaveAnswerResult struct {
	QuestionID  int
	UserAnswer  json.RawMessage
	IsCompleted bool
	UpdatedAt   time.Time
}

type GradeInput struct {
	IsCorrect *bool
	Score     *float64
	Feedback  *string
}

type GradeResult struct {
	AnswerID  string
	IsCorrect *bool
	Score     *float64
	GradedBy  string
	GradedAt  time.Time
}

func (uc *UseCase) CreateQuiz(ctx context.Context, input CreateQuizInput) (quizdomain.QuizDetail, error) {
	if input.SubjectID == "" || input.OwnerID == "" || strings.TrimSpace(input.Title) == "" {
		return quizdomain.QuizDetail{}, domain.ErrInvalid
	}
	if input.StartTime != nil && input.EndTime != nil && !input.StartTime.Before(*input.EndTime) {
		return quizdomain.QuizDetail{}, domain.ErrInvalid
	}
	if _, err := uc.subjects.FindByID(ctx, input.SubjectID); err != nil {
		uc.logError(err, "quiz_create_validate_subject", logrus.Fields{"subject_id": input.SubjectID})
		return quizdomain.QuizDetail{}, err
	}
	for _, questionID := range input.QuestionIDs {
		if _, err := uc.questions.FindByID(ctx, questionID); err != nil {
			uc.logError(err, "quiz_create_validate_question", logrus.Fields{"question_id": questionID})
			return quizdomain.QuizDetail{}, err
		}
	}
	id, err := uc.quizzes.Create(ctx, quizdomain.Quiz{
		SubjectID:   input.SubjectID,
		OwnerID:     input.OwnerID,
		Title:       strings.TrimSpace(input.Title),
		Description: input.Description,
		PublishedAt: input.PublishedAt,
		StartTime:   input.StartTime,
		EndTime:     input.EndTime,
	})
	if err != nil {
		uc.logError(err, "quiz_create", nil)
		return quizdomain.QuizDetail{}, err
	}
	for i, questionID := range input.QuestionIDs {
		if err := uc.quizQs.Add(ctx, quizdomain.QuizQuestion{QuizID: id, QuestionID: questionID, QuestionNumber: i + 1}); err != nil {
			uc.logError(err, "quiz_create_add_question", logrus.Fields{"quiz_id": id, "question_id": questionID})
			return quizdomain.QuizDetail{}, err
		}
	}
	detail, err := uc.query.GetQuizDetail(ctx, id)
	if err != nil {
		uc.logError(err, "quiz_create_get_detail", logrus.Fields{"quiz_id": id})
	}
	return detail, err
}

func (uc *UseCase) ListQuizzes(ctx context.Context, filter quizdomain.QuizFilter, page, pageSize int) ([]quizdomain.QuizSummary, int64, error) {
	items, total, err := uc.query.ListQuizzes(ctx, filter, pageSize, (page-1)*pageSize)
	if err != nil {
		uc.logError(err, "quiz_list", nil)
	}
	return items, total, err
}

func (uc *UseCase) GetQuizDetail(ctx context.Context, quizID, attemptID string) (quizdomain.QuizDetail, error) {
	detail, err := uc.query.GetQuizDetail(ctx, quizID)
	if err != nil {
		uc.logError(err, "quiz_detail", logrus.Fields{"quiz_id": quizID})
		return quizdomain.QuizDetail{}, err
	}
	if attemptID != "" {
		attempt, err := uc.attempts.FindByID(ctx, attemptID)
		if err != nil {
			uc.logError(err, "quiz_detail_attempt", logrus.Fields{"attempt_id": attemptID})
			return quizdomain.QuizDetail{}, err
		}
		if attempt.QuizID != quizID {
			return quizdomain.QuizDetail{}, domain.ErrNotFound
		}
	}
	return detail, nil
}

func (uc *UseCase) StartAttempt(ctx context.Context, quizID, studentUserID string) (quizdomain.Attempt, error) {
	quiz, err := uc.quizzes.FindByID(ctx, quizID)
	if err != nil {
		uc.logError(err, "quiz_start_get", logrus.Fields{"quiz_id": quizID})
		return quizdomain.Attempt{}, err
	}
	if _, err := uc.attempts.FindActive(ctx, quizID, studentUserID); err == nil {
		return quizdomain.Attempt{}, domain.ErrConflict
	} else if !errors.Is(err, domain.ErrNotFound) {
		uc.logError(err, "quiz_start_active", logrus.Fields{"quiz_id": quizID})
		return quizdomain.Attempt{}, err
	}
	now := time.Now()
	id, err := uc.attempts.Create(ctx, quizdomain.QuizAttempt{
		QuizID:        quizID,
		StudentUserID: studentUserID,
		StartedAt:     now,
		Status:        quizdomain.QuizAttemptStatusInProgress,
	})
	if err != nil {
		uc.logError(err, "quiz_start_create", logrus.Fields{"quiz_id": quizID})
		return quizdomain.Attempt{}, err
	}
	attempt, err := uc.attempts.FindByID(ctx, id)
	if err != nil {
		uc.logError(err, "quiz_start_fetch", logrus.Fields{"attempt_id": id})
		return quizdomain.Attempt{}, err
	}
	return attemptView(attempt, quiz.Title), nil
}

func (uc *UseCase) SaveAnswer(ctx context.Context, attemptID string, questionNumber int, userAnswer json.RawMessage) (SaveAnswerResult, error) {
	attempt, err := uc.attempts.FindByID(ctx, attemptID)
	if err != nil {
		uc.logError(err, "quiz_answer_get_attempt", logrus.Fields{"attempt_id": attemptID})
		return SaveAnswerResult{}, err
	}
	if attempt.Status != quizdomain.QuizAttemptStatusInProgress {
		return SaveAnswerResult{}, domain.ErrConflict
	}
	qq, err := uc.quizQs.FindByNumber(ctx, attempt.QuizID, questionNumber)
	if err != nil {
		uc.logError(err, "quiz_answer_find_question", logrus.Fields{"quiz_id": attempt.QuizID, "question_number": questionNumber})
		return SaveAnswerResult{}, err
	}
	if err := uc.answers.Upsert(ctx, quizdomain.QuizAnswer{AttemptID: attemptID, QuestionID: qq.QuestionID, Answer: userAnswer}); err != nil {
		uc.logError(err, "quiz_answer_upsert", logrus.Fields{"attempt_id": attemptID})
		return SaveAnswerResult{}, err
	}
	saved, err := uc.answers.FindByAttemptAndQuestion(ctx, attemptID, qq.QuestionID)
	if err != nil {
		uc.logError(err, "quiz_answer_fetch", logrus.Fields{"attempt_id": attemptID})
		return SaveAnswerResult{}, err
	}
	total, answered, err := uc.progress(ctx, attempt.QuizID, attemptID)
	if err != nil {
		uc.logError(err, "quiz_answer_progress", logrus.Fields{"attempt_id": attemptID})
		return SaveAnswerResult{}, err
	}
	return SaveAnswerResult{QuestionID: questionNumber, UserAnswer: saved.Answer, IsCompleted: total > 0 && answered >= total, UpdatedAt: saved.UpdatedAt}, nil
}

func (uc *UseCase) SubmitAttempt(ctx context.Context, attemptID string) (quizdomain.AttemptResult, error) {
	attempt, err := uc.attempts.FindByID(ctx, attemptID)
	if err != nil {
		uc.logError(err, "quiz_submit_get_attempt", logrus.Fields{"attempt_id": attemptID})
		return quizdomain.AttemptResult{}, err
	}
	if attempt.Status != quizdomain.QuizAttemptStatusInProgress {
		return quizdomain.AttemptResult{}, domain.ErrConflict
	}
	qqList, err := uc.quizQs.ListByQuizID(ctx, attempt.QuizID)
	if err != nil {
		uc.logError(err, "quiz_submit_list_questions", logrus.Fields{"quiz_id": attempt.QuizID})
		return quizdomain.AttemptResult{}, err
	}
	existing, err := uc.answers.ListByAttemptID(ctx, attemptID)
	if err != nil {
		uc.logError(err, "quiz_submit_list_answers", logrus.Fields{"attempt_id": attemptID})
		return quizdomain.AttemptResult{}, err
	}
	byQuestion := make(map[string]quizdomain.QuizAnswer, len(existing))
	for _, a := range existing {
		byQuestion[a.QuestionID] = a
	}
	var sum float64
	essayPending := false
	for _, qq := range qqList {
		question, err := uc.questions.FindByID(ctx, qq.QuestionID)
		if err != nil {
			uc.logError(err, "quiz_submit_get_question", logrus.Fields{"question_id": qq.QuestionID})
			return quizdomain.AttemptResult{}, err
		}
		answer, ok := byQuestion[qq.QuestionID]
		if !ok {
			answer = quizdomain.QuizAnswer{AttemptID: attemptID, QuestionID: qq.QuestionID}
		}
		if question.Type == quizdomain.QuestionTypeEssay {
			answer.IsCorrect = nil
			answer.Score = nil
			essayPending = true
		} else {
			correct := gradeObjective(question.Type, question.AnswerKey, answer.Answer)
			score := 0.0
			if correct {
				score = fullScore
			}
			answer.IsCorrect = &correct
			answer.Score = &score
			sum += score
		}
		if err := uc.answers.Upsert(ctx, answer); err != nil {
			uc.logError(err, "quiz_submit_upsert_answer", logrus.Fields{"attempt_id": attemptID})
			return quizdomain.AttemptResult{}, err
		}
	}
	score := percentage(sum, len(qqList))
	now := time.Now()
	attempt.SubmittedAt = &now
	attempt.Score = &score
	if essayPending {
		attempt.Status = quizdomain.QuizAttemptStatusSubmitted
	} else {
		attempt.Status = quizdomain.QuizAttemptStatusGraded
	}
	if err := uc.attempts.Update(ctx, attempt); err != nil {
		uc.logError(err, "quiz_submit_update_attempt", logrus.Fields{"attempt_id": attemptID})
		return quizdomain.AttemptResult{}, err
	}
	return uc.buildResult(ctx, attempt)
}

func (uc *UseCase) ListAttempts(ctx context.Context, filter quizdomain.AttemptFilter, page, pageSize int) ([]quizdomain.Attempt, int64, error) {
	items, total, err := uc.query.ListAttempts(ctx, filter, pageSize, (page-1)*pageSize)
	if err != nil {
		uc.logError(err, "quiz_attempt_list", nil)
	}
	return items, total, err
}

func (uc *UseCase) GetAttempt(ctx context.Context, attemptID string) (quizdomain.AttemptResult, error) {
	attempt, err := uc.attempts.FindByID(ctx, attemptID)
	if err != nil {
		uc.logError(err, "quiz_attempt_get", logrus.Fields{"attempt_id": attemptID})
		return quizdomain.AttemptResult{}, err
	}
	return uc.buildResult(ctx, attempt)
}

func (uc *UseCase) GradeAnswer(ctx context.Context, attemptID, answerID, graderID string, input GradeInput) (GradeResult, error) {
	attempt, err := uc.attempts.FindByID(ctx, attemptID)
	if err != nil {
		uc.logError(err, "quiz_grade_get_attempt", logrus.Fields{"attempt_id": attemptID})
		return GradeResult{}, err
	}
	answer, err := uc.answers.FindByID(ctx, answerID)
	if err != nil {
		uc.logError(err, "quiz_grade_get_answer", logrus.Fields{"answer_id": answerID})
		return GradeResult{}, err
	}
	if answer.AttemptID != attemptID {
		return GradeResult{}, domain.ErrNotFound
	}
	question, err := uc.questions.FindByID(ctx, answer.QuestionID)
	if err != nil {
		uc.logError(err, "quiz_grade_get_question", logrus.Fields{"question_id": answer.QuestionID})
		return GradeResult{}, err
	}
	isCorrect, score, err := resolveGrade(question.Type, input)
	if err != nil {
		return GradeResult{}, err
	}
	now := time.Now()
	answer.IsCorrect = isCorrect
	answer.Score = score
	answer.Feedback = input.Feedback
	answer.GradedBy = &graderID
	answer.GradedAt = &now
	if err := uc.answers.Update(ctx, answer); err != nil {
		uc.logError(err, "quiz_grade_update_answer", logrus.Fields{"answer_id": answerID})
		return GradeResult{}, err
	}
	if err := uc.recompute(ctx, attempt); err != nil {
		uc.logError(err, "quiz_grade_recompute", logrus.Fields{"attempt_id": attemptID})
		return GradeResult{}, err
	}
	return GradeResult{AnswerID: answer.ID, IsCorrect: answer.IsCorrect, Score: answer.Score, GradedBy: graderID, GradedAt: now}, nil
}

func (uc *UseCase) buildResult(ctx context.Context, attempt quizdomain.QuizAttempt) (quizdomain.AttemptResult, error) {
	quiz, err := uc.quizzes.FindByID(ctx, attempt.QuizID)
	if err != nil {
		uc.logError(err, "quiz_result_get_quiz", logrus.Fields{"quiz_id": attempt.QuizID})
		return quizdomain.AttemptResult{}, err
	}
	qqList, err := uc.quizQs.ListByQuizID(ctx, attempt.QuizID)
	if err != nil {
		uc.logError(err, "quiz_result_list_questions", logrus.Fields{"quiz_id": attempt.QuizID})
		return quizdomain.AttemptResult{}, err
	}
	numberByQuestion := make(map[string]int, len(qqList))
	typeByQuestion := make(map[string]quizdomain.QuestionType, len(qqList))
	for _, qq := range qqList {
		numberByQuestion[qq.QuestionID] = qq.QuestionNumber
		question, err := uc.questions.FindByID(ctx, qq.QuestionID)
		if err != nil {
			uc.logError(err, "quiz_result_get_question", logrus.Fields{"question_id": qq.QuestionID})
			return quizdomain.AttemptResult{}, err
		}
		typeByQuestion[qq.QuestionID] = question.Type
	}
	answerList, err := uc.answers.ListByAttemptID(ctx, attempt.ID)
	if err != nil {
		uc.logError(err, "quiz_result_list_answers", logrus.Fields{"attempt_id": attempt.ID})
		return quizdomain.AttemptResult{}, err
	}
	answers := make([]quizdomain.AttemptAnswer, 0, len(answerList))
	essayPending := false
	for _, a := range answerList {
		if typeByQuestion[a.QuestionID] == quizdomain.QuestionTypeEssay && a.Score == nil {
			essayPending = true
		}
		answers = append(answers, quizdomain.AttemptAnswer{
			AnswerID:   a.ID,
			QuestionID: numberByQuestion[a.QuestionID],
			UserAnswer: a.Answer,
			IsCorrect:  a.IsCorrect,
		})
	}
	sort.Slice(answers, func(i, j int) bool { return answers[i].QuestionID < answers[j].QuestionID })
	return quizdomain.AttemptResult{Attempt: attemptView(attempt, quiz.Title), EssayPending: essayPending, Answers: answers}, nil
}

func (uc *UseCase) recompute(ctx context.Context, attempt quizdomain.QuizAttempt) error {
	qqList, err := uc.quizQs.ListByQuizID(ctx, attempt.QuizID)
	if err != nil {
		return err
	}
	answerList, err := uc.answers.ListByAttemptID(ctx, attempt.ID)
	if err != nil {
		return err
	}
	var sum float64
	essayPending := false
	for _, a := range answerList {
		if a.Score != nil {
			sum += *a.Score
		} else {
			essayPending = true
		}
	}
	score := percentage(sum, len(qqList))
	attempt.Score = &score
	if essayPending || attempt.SubmittedAt == nil {
		attempt.Status = quizdomain.QuizAttemptStatusSubmitted
	} else {
		attempt.Status = quizdomain.QuizAttemptStatusGraded
	}
	return uc.attempts.Update(ctx, attempt)
}

func (uc *UseCase) progress(ctx context.Context, quizID, attemptID string) (int, int, error) {
	qqList, err := uc.quizQs.ListByQuizID(ctx, quizID)
	if err != nil {
		return 0, 0, err
	}
	answerList, err := uc.answers.ListByAttemptID(ctx, attemptID)
	if err != nil {
		return 0, 0, err
	}
	return len(qqList), len(answerList), nil
}

func attemptView(attempt quizdomain.QuizAttempt, quizTitle string) quizdomain.Attempt {
	return quizdomain.Attempt{
		AttemptID:   attempt.ID,
		QuizID:      attempt.QuizID,
		QuizTitle:   quizTitle,
		StartedAt:   attempt.StartedAt,
		SubmittedAt: attempt.SubmittedAt,
		Status:      attempt.Status,
		Score:       attempt.Score,
	}
}

func percentage(sum float64, total int) float64 {
	if total == 0 {
		return 0
	}
	return math.Round(sum/float64(total)*100) / 100
}

func resolveGrade(questionType quizdomain.QuestionType, input GradeInput) (*bool, *float64, error) {
	if questionType != quizdomain.QuestionTypeEssay {
		isCorrect := input.IsCorrect
		if isCorrect == nil {
			if input.Score == nil {
				return nil, nil, domain.ErrInvalid
			}
			correct := *input.Score > 0
			isCorrect = &correct
		}
		score := 0.0
		if *isCorrect {
			score = fullScore
		}
		return isCorrect, &score, nil
	}
	score := input.Score
	if score == nil {
		if input.IsCorrect == nil {
			return nil, nil, domain.ErrInvalid
		}
		value := 0.0
		if *input.IsCorrect {
			value = fullScore
		}
		score = &value
	} else if *score < 0 || *score > fullScore {
		return nil, nil, domain.ErrInvalid
	}
	isCorrect := input.IsCorrect
	if isCorrect == nil {
		correct := *score > 0
		isCorrect = &correct
	}
	return isCorrect, score, nil
}

func gradeObjective(questionType quizdomain.QuestionType, answerKey, userAnswer json.RawMessage) bool {
	if len(userAnswer) == 0 || string(userAnswer) == "null" {
		return false
	}
	switch questionType {
	case quizdomain.QuestionTypeMultipleChoice:
		var key struct {
			Correct string `json:"correct"`
		}
		var value string
		if json.Unmarshal(answerKey, &key) != nil || json.Unmarshal(userAnswer, &value) != nil {
			return false
		}
		return key.Correct != "" && key.Correct == value
	case quizdomain.QuestionTypeMultipleSelect:
		var key struct {
			Correct []string `json:"correct"`
		}
		var value []string
		if json.Unmarshal(answerKey, &key) != nil || json.Unmarshal(userAnswer, &value) != nil {
			return false
		}
		return sameSet(key.Correct, value)
	case quizdomain.QuestionTypeTrueFalse:
		var key struct {
			Correct bool `json:"correct"`
		}
		var value bool
		if json.Unmarshal(answerKey, &key) != nil || json.Unmarshal(userAnswer, &value) != nil {
			return false
		}
		return key.Correct == value
	case quizdomain.QuestionTypeRange:
		var key struct {
			Min *float64 `json:"min"`
			Max *float64 `json:"max"`
		}
		var value float64
		if json.Unmarshal(answerKey, &key) != nil || json.Unmarshal(userAnswer, &value) != nil {
			return false
		}
		if key.Min != nil && value < *key.Min {
			return false
		}
		if key.Max != nil && value > *key.Max {
			return false
		}
		return key.Min != nil || key.Max != nil
	default:
		return false
	}
}

func sameSet(expected, actual []string) bool {
	if len(expected) != len(actual) {
		return false
	}
	counts := make(map[string]int, len(expected))
	for _, v := range expected {
		counts[v]++
	}
	for _, v := range actual {
		counts[v]--
		if counts[v] < 0 {
			return false
		}
	}
	return true
}

func (uc *UseCase) logError(err error, operation string, fields logrus.Fields) {
	if err == nil || err == domain.ErrInvalid || err == domain.ErrNotFound || err == domain.ErrAlreadyExists || err == domain.ErrConflict {
		return
	}
	entry := uc.logger.WithError(err).WithField("operation", operation)
	if fields != nil {
		entry = entry.WithFields(fields)
	}
	entry.Error("usecase failed")
}
