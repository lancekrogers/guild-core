package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/guild-ventures/guild-core/pkg/storage/db"
)

// SQLiteBoardRepository implements BoardRepository using SQLite
type SQLiteBoardRepository struct {
	database *Database
}

// NewSQLiteBoardRepository creates a new SQLite board repository
func NewSQLiteBoardRepository(database *Database) BoardRepository {
	return &SQLiteBoardRepository{
		database: database,
	}
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
		return fmt.Errorf("failed to create board: %w", err)
	}
	return nil
}

// GetBoard retrieves a board by ID
func (r *SQLiteBoardRepository) GetBoard(ctx context.Context, id string) (*Board, error) {
	dbBoard, err := r.database.Queries().GetBoard(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("board not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get board: %w", err)
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
			return nil, fmt.Errorf("board not found for commission: %s", commissionID)
		}
		return nil, fmt.Errorf("failed to get board by commission: %w", err)
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
		return fmt.Errorf("failed to update board: %w", err)
	}
	return nil
}

// DeleteBoard removes a board by ID
func (r *SQLiteBoardRepository) DeleteBoard(ctx context.Context, id string) error {
	if err := r.database.Queries().DeleteBoard(ctx, id); err != nil {
		return fmt.Errorf("failed to delete board: %w", err)
	}
	return nil
}

// ListBoards returns all boards
func (r *SQLiteBoardRepository) ListBoards(ctx context.Context) ([]*Board, error) {
	dbBoards, err := r.database.Queries().ListBoards(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list boards: %w", err)
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