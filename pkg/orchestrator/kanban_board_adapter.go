// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package orchestrator

import (
	"context"
	"time"

	"github.com/guild-framework/guild-core/pkg/gerror"
	"github.com/guild-framework/guild-core/pkg/storage"
)

// kanbanBoardRepoAdapter adapts storage.BoardRepository to kanban.BoardRepository
type kanbanBoardRepoAdapter struct {
	repo storage.BoardRepository
}

func (k *kanbanBoardRepoAdapter) CreateBoard(ctx context.Context, board interface{}) error {
	if k.repo != nil {
		if boardMap, ok := board.(map[string]interface{}); ok {
			storageBoard := &storage.Board{
				ID:           boardMap["ID"].(string),
				CommissionID: boardMap["CommissionID"].(string),
				Name:         boardMap["Name"].(string),
				Status:       boardMap["Status"].(string),
				CreatedAt:    boardMap["CreatedAt"].(time.Time),
				UpdatedAt:    boardMap["UpdatedAt"].(time.Time),
			}
			if desc, exists := boardMap["Description"]; exists && desc != nil {
				if descStr, ok := desc.(*string); ok {
					storageBoard.Description = descStr
				}
			}
			return k.repo.CreateBoard(ctx, storageBoard)
		}
	}
	return gerror.New(gerror.ErrCodeStorage, "board repository not available", nil).
		WithComponent("orchestrator").
		WithOperation("CreateBoard")
}

func (k *kanbanBoardRepoAdapter) GetBoard(ctx context.Context, id string) (interface{}, error) {
	if k.repo != nil {
		return k.repo.GetBoard(ctx, id)
	}
	return nil, gerror.New(gerror.ErrCodeStorage, "board repository not available", nil).
		WithComponent("orchestrator").
		WithOperation("GetBoard")
}

func (k *kanbanBoardRepoAdapter) UpdateBoard(ctx context.Context, board interface{}) error {
	if k.repo != nil {
		if boardMap, ok := board.(map[string]interface{}); ok {
			storageBoard := &storage.Board{
				ID:           boardMap["ID"].(string),
				CommissionID: boardMap["CommissionID"].(string),
				Name:         boardMap["Name"].(string),
				Status:       boardMap["Status"].(string),
				CreatedAt:    boardMap["CreatedAt"].(time.Time),
				UpdatedAt:    boardMap["UpdatedAt"].(time.Time),
			}
			if desc, exists := boardMap["Description"]; exists && desc != nil {
				if descStr, ok := desc.(*string); ok {
					storageBoard.Description = descStr
				}
			}
			return k.repo.UpdateBoard(ctx, storageBoard)
		}
	}
	return gerror.New(gerror.ErrCodeStorage, "board repository not available", nil).
		WithComponent("orchestrator").
		WithOperation("UpdateBoard")
}

func (k *kanbanBoardRepoAdapter) DeleteBoard(ctx context.Context, id string) error {
	if k.repo != nil {
		return k.repo.DeleteBoard(ctx, id)
	}
	return gerror.New(gerror.ErrCodeStorage, "board repository not available", nil).
		WithComponent("orchestrator").
		WithOperation("DeleteBoard")
}

func (k *kanbanBoardRepoAdapter) ListBoards(ctx context.Context) ([]interface{}, error) {
	if k.repo != nil {
		boards, err := k.repo.ListBoards(ctx)
		if err != nil {
			return nil, err
		}
		// Convert to []interface{}
		result := make([]interface{}, len(boards))
		for i, board := range boards {
			result[i] = board
		}
		return result, nil
	}
	return nil, gerror.New(gerror.ErrCodeStorage, "board repository not available", nil).
		WithComponent("orchestrator").
		WithOperation("ListBoards")
}
