// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package storage

import (
	"context"
	"database/sql"

	"github.com/guild-ventures/guild-core/pkg/gerror"
	"github.com/guild-ventures/guild-core/pkg/storage/db"
)

// SQLiteBoardRepository implements BoardRepository using SQLite
type SQLiteBoardRepository struct {
	database *Database
}

// newSQLiteBoardRepository creates a new SQLite board repository (private constructor)
func newSQLiteBoardRepository(database *Database) BoardRepository {
	return &SQLiteBoardRepository{
		database: database,
	}
}

// DefaultBoardRepositoryFactory creates a board repository for registry use
func DefaultBoardRepositoryFactory(database *Database) BoardRepository {
	return newSQLiteBoardRepository(database)
}

// CreateBoard creates a new board
func (r *SQLiteBoardRepository) CreateBoard(ctx context.Context, board *Board) error {
	if err := r.database.Queries().CreateBoard(ctx, db.CreateBoardParams{
		ID:           board.ID,
		CommissionID: board.CommissionID,
		Name:         board.Name,
		Description:  board.Description,
		Status:       board.Status,
	}); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeStorage, "failed to create board").
			WithComponent("storage").
			WithOperation("create_board")
	}
	return nil
}

// GetBoard retrieves a board by ID
func (r *SQLiteBoardRepository) GetBoard(ctx context.Context, id string) (*Board, error) {
	dbBoard, err := r.database.Queries().GetBoard(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, gerror.Newf(gerror.ErrCodeNotFound, "board not found: %s", id).
				WithComponent("storage").
				WithOperation("get_board")
		}
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get board").
			WithComponent("storage").
			WithOperation("get_board")
	}

	board := &Board{
		ID:           dbBoard.ID,
		CommissionID: dbBoard.CommissionID,
		Name:         dbBoard.Name,
		Description:  dbBoard.Description,
		Status:       dbBoard.Status,
		CreatedAt:    *dbBoard.CreatedAt,
		UpdatedAt:    *dbBoard.UpdatedAt,
	}

	return board, nil
}

// GetBoardByCommission retrieves the board for a specific commission
func (r *SQLiteBoardRepository) GetBoardByCommission(ctx context.Context, commissionID string) (*Board, error) {
	dbBoard, err := r.database.Queries().GetBoardByCommission(ctx, commissionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, gerror.Newf(gerror.ErrCodeNotFound, "board not found for commission: %s", commissionID).
				WithComponent("storage").
				WithOperation("get_board_by_commission")
		}
		return nil, gerror.Wrap(err, gerror.ErrCodeStorage, "failed to get board by commission").
			WithComponent("storage").
			WithOperation("get_board_by_commission")
	}

	board := &Board{
		ID:           dbBoard.ID,
		CommissionID: dbBoard.CommissionID,
		Name:         dbBoard.Name,
		Description:  dbBoard.Description,
		Status:       dbBoard.Status,
		CreatedAt:    *dbBoard.CreatedAt,
		UpdatedAt:    *dbBoard.UpdatedAt,
	}

	return board, nil
}

// UpdateBoard updates an existing board
func (r *SQLiteBoardRepository) UpdateBoard(ctx context.Context, board *Board) error {
	if err := r.database.Queries().UpdateBoard(ctx, db.UpdateBoardParams{
		Name:        board.Name,
		Description: board.Description,
		Status:      board.Status,
		ID:          board.ID,
	}); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to update board").
			WithComponent("storage").
			WithOperation("update_board")
	}
	return nil
}

// DeleteBoard removes a board by ID
func (r *SQLiteBoardRepository) DeleteBoard(ctx context.Context, id string) error {
	if err := r.database.Queries().DeleteBoard(ctx, id); err != nil {
		return gerror.Wrap(err, gerror.ErrCodeInternal, "failed to delete board").
			WithComponent("storage").
			WithOperation("delete_board")
	}
	return nil
}

// ListBoards returns all boards
func (r *SQLiteBoardRepository) ListBoards(ctx context.Context) ([]*Board, error) {
	dbBoards, err := r.database.Queries().ListBoards(ctx)
	if err != nil {
		return nil, gerror.Wrap(err, gerror.ErrCodeInternal, "failed to list boards").
			WithComponent("storage").
			WithOperation("list_boards")
	}

	boards := make([]*Board, len(dbBoards))
	for i, dbBoard := range dbBoards {
		boards[i] = &Board{
			ID:           dbBoard.ID,
			CommissionID: dbBoard.CommissionID,
			Name:         dbBoard.Name,
			Description:  dbBoard.Description,
			Status:       dbBoard.Status,
			CreatedAt:    *dbBoard.CreatedAt,
			UpdatedAt:    *dbBoard.UpdatedAt,
		}
	}

	return boards, nil
}
