package service

import (
	"context"

	"github.com/flutapp/chat-service/internal/repository"
	"github.com/flutapp/chat-service/internal/utils"
)

type BlockService interface {
	BlockUser(ctx context.Context, userID, targetID string) error
	UnblockUser(ctx context.Context, userID, targetID string) error
	IsBlocked(ctx context.Context, userID, targetID string) (bool, error)
	GetBlockedList(ctx context.Context, userID string) ([]string, error)
}

type blockService struct {
	repo repository.BlockRepository
}

func NewBlockService(repo repository.BlockRepository) BlockService {
	return &blockService{repo}
}

func (s *blockService) BlockUser(ctx context.Context, userID, targetID string) error {
	if userID == targetID {
		return utils.ErrInvalidPayload
	}
	_, err := s.repo.Create(ctx, userID, targetID)
	return err
}

func (s *blockService) UnblockUser(ctx context.Context, userID, targetID string) error {
	return s.repo.Delete(ctx, userID, targetID)
}

func (s *blockService) IsBlocked(ctx context.Context, userID, targetID string) (bool, error) {
	return s.repo.IsBlocked(ctx, userID, targetID)
}

func (s *blockService) GetBlockedList(ctx context.Context, userID string) ([]string, error) {
	return s.repo.GetBlockedList(ctx, userID)
}
