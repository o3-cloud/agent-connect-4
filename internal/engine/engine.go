// Package engine implements the pure Connect Four rules (RULE-001..009).
//
// It has no knowledge of players, games, transports or storage (NFR-007), so
// every interface shares exactly the same rules (NFR-004).
package engine

import "errors"

// Board dimensions (RULE-001). Row 0 is the top row; row 5 is the bottom row.
const (
	Rows    = 6
	Columns = 7
	connect = 4
)

// Disc is the content of a cell: a player's color, or Empty.
type Disc string

// Disc values. Red always moves first.
const (
	Empty  Disc = ""
	Red    Disc = "red"
	Yellow Disc = "yellow"
)

// Opponent returns the other player's disc, or Empty for Empty.
func (d Disc) Opponent() Disc {
	switch d {
	case Red:
		return Yellow
	case Yellow:
		return Red
	default:
		return Empty
	}
}

// Reason explains how a game ended.
type Reason string

// Result reasons.
const (
	ReasonConnectFour Reason = "connect_four"
	ReasonDraw        Reason = "draw"
)

// Result describes the board after a move. Winner is Empty for a draw.
type Result struct {
	Over   bool
	Winner Disc
	Reason Reason
}

// Errors returned by Drop. The board is unchanged when any of them is returned.
var (
	ErrInvalidColumn = errors.New("column must be an integer from 0 to 6")
	ErrColumnFull    = errors.New("column is full")
	ErrInvalidDisc   = errors.New("disc must be red or yellow")
)

// Board is a 6 × 7 Connect Four grid. The zero value is not usable; call NewBoard.
type Board struct {
	cells [Rows][Columns]Disc
	moves int
}

// NewBoard returns an empty board.
func NewBoard() *Board {
	return &Board{}
}

// Cells returns a copy of the grid, indexed [row][column].
func (b *Board) Cells() [Rows][Columns]Disc {
	return b.cells
}

// MoveCount returns the number of discs on the board.
func (b *Board) MoveCount() int {
	return b.moves
}

// LegalMoves returns the columns that still have an empty cell (RULE-009).
func (b *Board) LegalMoves() []int {
	legal := make([]int, 0, Columns)
	for column := 0; column < Columns; column++ {
		if b.cells[0][column] == Empty {
			legal = append(legal, column)
		}
	}
	return legal
}

// Drop places disc in the lowest empty cell of column (RULE-002) and reports
// the row it landed in and whether the move ended the game (RULE-005..008).
func (b *Board) Drop(column int, disc Disc) (int, Result, error) {
	if disc != Red && disc != Yellow {
		return 0, Result{}, ErrInvalidDisc
	}
	if column < 0 || column >= Columns {
		return 0, Result{}, ErrInvalidColumn
	}
	row := -1
	for r := Rows - 1; r >= 0; r-- {
		if b.cells[r][column] == Empty {
			row = r
			break
		}
	}
	if row < 0 {
		return 0, Result{}, ErrColumnFull
	}

	b.cells[row][column] = disc
	b.moves++

	switch {
	case b.connectsFour(row, column, disc):
		return row, Result{Over: true, Winner: disc, Reason: ReasonConnectFour}, nil
	case b.moves == Rows*Columns:
		return row, Result{Over: true, Winner: Empty, Reason: ReasonDraw}, nil
	default:
		return row, Result{}, nil
	}
}

// connectsFour reports whether the disc at (row, column) completes a line of
// four or more in any direction.
func (b *Board) connectsFour(row, column int, disc Disc) bool {
	directions := [][2]int{{0, 1}, {1, 0}, {1, 1}, {1, -1}}
	for _, d := range directions {
		count := 1 + b.run(row, column, d[0], d[1], disc) + b.run(row, column, -d[0], -d[1], disc)
		if count >= connect {
			return true
		}
	}
	return false
}

// run counts consecutive discs matching disc, starting next to (row, column)
// and stepping by (dRow, dColumn).
func (b *Board) run(row, column, dRow, dColumn int, disc Disc) int {
	count := 0
	for r, c := row+dRow, column+dColumn; r >= 0 && r < Rows && c >= 0 && c < Columns && b.cells[r][c] == disc; r, c = r+dRow, c+dColumn {
		count++
	}
	return count
}
