package counseling

import (
	"context"
	"strings"

	"limen/internal/domain"
	academicdomain "limen/internal/domain/academic"
	counselingdomain "limen/internal/domain/counseling"

	"github.com/sirupsen/logrus"
)

type UseCase struct {
	notes     counselingdomain.CounselingNoteRepository
	followups counselingdomain.CounselingFollowupRepository
	students  academicdomain.StudentRepository
	logger    *logrus.Logger
}

func NewUseCase(notes counselingdomain.CounselingNoteRepository, followups counselingdomain.CounselingFollowupRepository, students academicdomain.StudentRepository, logger *logrus.Logger) *UseCase {
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	return &UseCase{notes: notes, followups: followups, students: students, logger: logger}
}

func (uc *UseCase) ListNotes(ctx context.Context, filter counselingdomain.CounselingNoteFilter, page, pageSize int) ([]counselingdomain.CounselingNote, int64, error) {
	items, total, err := uc.notes.List(ctx, filter, pageSize, (page-1)*pageSize)
	if err != nil {
		uc.logError(err, "counseling_list_notes", nil)
	}
	return items, total, err
}

func (uc *UseCase) CreateNote(ctx context.Context, studentUserID, authorID, note string) (counselingdomain.CounselingNote, error) {
	note = strings.TrimSpace(note)
	if note == "" {
		return counselingdomain.CounselingNote{}, domain.ErrInvalid
	}
	if _, err := uc.students.FindByUserID(ctx, studentUserID); err != nil {
		uc.logError(err, "counseling_validate_student", logrus.Fields{"student_id": studentUserID})
		return counselingdomain.CounselingNote{}, err
	}
	id, err := uc.notes.Create(ctx, counselingdomain.CounselingNote{StudentUserID: studentUserID, AuthorID: authorID, Note: note})
	if err != nil {
		uc.logError(err, "counseling_create_note", logrus.Fields{"student_id": studentUserID})
		return counselingdomain.CounselingNote{}, err
	}
	item, err := uc.notes.FindByID(ctx, id)
	if err != nil {
		uc.logError(err, "counseling_get_created_note", logrus.Fields{"note_id": id})
	}
	return item, err
}

func (uc *UseCase) UpdateNote(ctx context.Context, noteID, note string) (counselingdomain.CounselingNote, error) {
	note = strings.TrimSpace(note)
	if note == "" {
		return counselingdomain.CounselingNote{}, domain.ErrInvalid
	}
	item, err := uc.notes.FindByID(ctx, noteID)
	if err != nil {
		uc.logError(err, "counseling_get_note", logrus.Fields{"note_id": noteID})
		return counselingdomain.CounselingNote{}, err
	}
	item.Note = note
	if err := uc.notes.Update(ctx, item); err != nil {
		uc.logError(err, "counseling_update_note", logrus.Fields{"note_id": noteID})
		return counselingdomain.CounselingNote{}, err
	}
	item, err = uc.notes.FindByID(ctx, noteID)
	if err != nil {
		uc.logError(err, "counseling_get_updated_note", logrus.Fields{"note_id": noteID})
	}
	return item, err
}

func (uc *UseCase) AddFollowup(ctx context.Context, noteID, createdBy, note string) (counselingdomain.CounselingFollowup, error) {
	note = strings.TrimSpace(note)
	if note == "" {
		return counselingdomain.CounselingFollowup{}, domain.ErrInvalid
	}
	if _, err := uc.notes.FindByID(ctx, noteID); err != nil {
		uc.logError(err, "counseling_get_note_for_followup", logrus.Fields{"note_id": noteID})
		return counselingdomain.CounselingFollowup{}, err
	}
	id, err := uc.followups.Create(ctx, counselingdomain.CounselingFollowup{NoteID: noteID, Note: note, CreatedBy: createdBy})
	if err != nil {
		uc.logError(err, "counseling_create_followup", logrus.Fields{"note_id": noteID})
		return counselingdomain.CounselingFollowup{}, err
	}
	item, err := uc.followups.FindByID(ctx, id)
	if err != nil {
		uc.logError(err, "counseling_get_created_followup", logrus.Fields{"followup_id": id})
	}
	return item, err
}

func (uc *UseCase) logError(err error, operation string, fields logrus.Fields) {
	if err == nil || err == domain.ErrInvalid || err == domain.ErrNotFound || err == domain.ErrAlreadyExists {
		return
	}
	entry := uc.logger.WithError(err).WithField("operation", operation)
	if fields != nil {
		entry = entry.WithFields(fields)
	}
	entry.Error("usecase failed")
}
