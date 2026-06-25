package idcard

import (
	"context"
	"fmt"

	apperr "github.com/ajaypatel01/CampusDesk/internal/platform/errors"
	"github.com/ajaypatel01/CampusDesk/internal/platform/storage"
	"github.com/google/uuid"
)

type Service struct {
	repo    *Repository
	storage *storage.Client
}

func NewService(repo *Repository, s *storage.Client) *Service {
	return &Service{repo: repo, storage: s}
}

func (s *Service) GenerateStudentCards(ctx context.Context, studentIDs []uuid.UUID, yearID uuid.UUID) ([]byte, string, error) {
	if len(studentIDs) == 0 || yearID == uuid.Nil {
		return nil, "", apperr.ErrInvalidInput
	}
	var cards []StudentCardData
	for _, id := range studentIDs {
		d, err := s.repo.GetStudentCardData(ctx, id, yearID)
		if err != nil {
			continue
		}
		cards = append(cards, *d)
	}
	if len(cards) == 0 {
		return nil, "", apperr.ErrNotFound
	}
	photos := s.downloadPhotos(ctx, cards)
	pdf, err := generateStudentCards(cards, photos)
	if err != nil {
		return nil, "", fmt.Errorf("generate student cards: %w", err)
	}
	return pdf, "student_id_cards.pdf", nil
}

func (s *Service) GenerateTeacherCards(ctx context.Context, userIDs []uuid.UUID) ([]byte, string, error) {
	if len(userIDs) == 0 {
		return nil, "", apperr.ErrInvalidInput
	}
	var cards []TeacherCardData
	for _, id := range userIDs {
		d, err := s.repo.GetTeacherCardData(ctx, id)
		if err != nil {
			continue
		}
		cards = append(cards, *d)
	}
	if len(cards) == 0 {
		return nil, "", apperr.ErrNotFound
	}
	photos := s.downloadTeacherPhotos(ctx, cards)
	pdf, err := generateTeacherCards(cards, photos)
	if err != nil {
		return nil, "", fmt.Errorf("generate teacher cards: %w", err)
	}
	return pdf, "teacher_id_cards.pdf", nil
}

func (s *Service) downloadPhotos(ctx context.Context, cards []StudentCardData) map[string][]byte {
	result := map[string][]byte{}
	if !s.storage.Enabled() {
		return result
	}
	for _, c := range cards {
		if c.PhotoKey == "" {
			continue
		}
		if _, ok := result[c.PhotoKey]; ok {
			continue
		}
		data, err := s.storage.Download(c.PhotoKey)
		if err == nil {
			result[c.PhotoKey] = data
		}
	}
	return result
}

func (s *Service) downloadTeacherPhotos(ctx context.Context, cards []TeacherCardData) map[string][]byte {
	result := map[string][]byte{}
	if !s.storage.Enabled() {
		return result
	}
	for _, c := range cards {
		if c.PhotoKey == "" {
			continue
		}
		if _, ok := result[c.PhotoKey]; ok {
			continue
		}
		data, err := s.storage.Download(c.PhotoKey)
		if err == nil {
			result[c.PhotoKey] = data
		}
	}
	return result
}
