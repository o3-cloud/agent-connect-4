package tests

import (
	"errors"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	"connect4arena/internal/engine"
)

// play drops discs into the given columns, alternating red then yellow.
func play(t *testing.T, columns ...int) (*engine.Board, engine.Result) {
	t.Helper()
	board := engine.NewBoard()
	disc := engine.Red
	var result engine.Result
	for i, column := range columns {
		if result.Over {
			t.Fatalf("move %d (column %d) played after the game ended", i+1, column)
		}
		var err error
		_, result, err = board.Drop(column, disc)
		if err != nil {
			t.Fatalf("move %d (column %d): %v", i+1, column, err)
		}
		disc = disc.Opponent()
	}
	return board, result
}

func TestNewBoardIsEmptySixBySeven(t *testing.T) {
	board := engine.NewBoard()
	cells := board.Cells()
	if len(cells) != engine.Rows || engine.Rows != 6 {
		t.Fatalf("rows = %d, want 6", len(cells))
	}
	for r, row := range cells {
		if len(row) != engine.Columns || engine.Columns != 7 {
			t.Fatalf("row %d has %d columns, want 7", r, len(row))
		}
		for c, cell := range row {
			if cell != engine.Empty {
				t.Fatalf("cell [%d][%d] = %q, want empty", r, c, cell)
			}
		}
	}
	if got, want := board.LegalMoves(), []int{0, 1, 2, 3, 4, 5, 6}; !reflect.DeepEqual(got, want) {
		t.Fatalf("LegalMoves() = %v, want %v", got, want)
	}
}

func TestDropStacksFromTheBottomRow(t *testing.T) {
	board := engine.NewBoard()
	row, _, err := board.Drop(3, engine.Red)
	if err != nil || row != 5 {
		t.Fatalf("first drop: row=%d err=%v, want row 5", row, err)
	}
	row, _, err = board.Drop(3, engine.Yellow)
	if err != nil || row != 4 {
		t.Fatalf("second drop: row=%d err=%v, want row 4", row, err)
	}
	cells := board.Cells()
	if cells[5][3] != engine.Red || cells[4][3] != engine.Yellow {
		t.Fatalf("column 3 bottom-up = %q,%q, want red,yellow", cells[5][3], cells[4][3])
	}
}

func TestDropRejectsInvalidColumns(t *testing.T) {
	for _, column := range []int{-1, 7, 100} {
		t.Run(strconv.Itoa(column), func(t *testing.T) {
			board := engine.NewBoard()
			if _, _, err := board.Drop(column, engine.Red); !errors.Is(err, engine.ErrInvalidColumn) {
				t.Fatalf("Drop(%d) error = %v, want ErrInvalidColumn", column, err)
			}
			if board.MoveCount() != 0 {
				t.Fatalf("board changed after rejected drop")
			}
		})
	}
}

func TestDropRejectsFullColumn(t *testing.T) {
	board, _ := play(t, 0, 0, 0, 0, 0, 0)
	before := board.Cells()
	if _, _, err := board.Drop(0, engine.Red); !errors.Is(err, engine.ErrColumnFull) {
		t.Fatalf("Drop into full column error = %v, want ErrColumnFull", err)
	}
	if !reflect.DeepEqual(board.Cells(), before) {
		t.Fatalf("board changed after rejected drop")
	}
	if got, want := board.LegalMoves(), []int{1, 2, 3, 4, 5, 6}; !reflect.DeepEqual(got, want) {
		t.Fatalf("LegalMoves() = %v, want %v", got, want)
	}
}

func TestDropRejectsEmptyDisc(t *testing.T) {
	board := engine.NewBoard()
	if _, _, err := board.Drop(0, engine.Empty); !errors.Is(err, engine.ErrInvalidDisc) {
		t.Fatalf("Drop with empty disc error = %v, want ErrInvalidDisc", err)
	}
}

func TestWinsAreDetected(t *testing.T) {
	tests := []struct {
		name    string
		columns []int
		winner  engine.Disc
	}{
		{name: "horizontal (BDR-008 scenario 1)", columns: []int{0, 0, 1, 1, 2, 2, 3}, winner: engine.Red},
		{name: "vertical (BDR-008 scenario 2)", columns: []int{0, 1, 0, 1, 0, 1, 0}, winner: engine.Red},
		{name: "rising diagonal (BDR-008 scenario 3)", columns: []int{0, 1, 1, 2, 2, 3, 2, 3, 3, 6, 3}, winner: engine.Red},
		{name: "falling diagonal (BDR-008 scenario 3)", columns: []int{6, 5, 5, 4, 4, 3, 4, 3, 3, 0, 3}, winner: engine.Red},
		{name: "yellow horizontal", columns: []int{0, 1, 0, 2, 6, 3, 6, 4}, winner: engine.Yellow},
		{name: "horizontal completed in the middle", columns: []int{0, 0, 1, 1, 3, 3, 2}, winner: engine.Red},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			board, result := play(t, test.columns...)
			if !result.Over || result.Winner != test.winner || result.Reason != engine.ReasonConnectFour {
				t.Fatalf("result = %+v, want %s win by connect_four", result, test.winner)
			}
			if board.MoveCount() != len(test.columns) {
				t.Fatalf("MoveCount() = %d, want %d", board.MoveCount(), len(test.columns))
			}
		})
	}
}

func TestNoWinBeforeFourInARow(t *testing.T) {
	_, result := play(t, 0, 0, 1, 1, 2, 2)
	if result.Over {
		t.Fatalf("result = %+v, want game still in progress", result)
	}
}

func TestFullBoardWithoutLineIsDraw(t *testing.T) {
	// BDR-009 scenario 1.
	board, result := play(t, 0, 1, 0, 1, 0, 1, 2, 3, 2, 3, 2, 3, 4, 5, 4, 5, 4, 5, 6, 0, 6, 0, 6, 0,
		1, 2, 1, 2, 1, 2, 3, 4, 3, 4, 3, 4, 5, 6, 5, 6, 5, 6)
	if !result.Over || result.Winner != engine.Empty || result.Reason != engine.ReasonDraw {
		t.Fatalf("result = %+v, want draw", result)
	}
	if board.MoveCount() != 42 || len(board.LegalMoves()) != 0 {
		t.Fatalf("MoveCount()=%d LegalMoves()=%v, want 42 and none", board.MoveCount(), board.LegalMoves())
	}
}

func TestDiscOpponent(t *testing.T) {
	if engine.Red.Opponent() != engine.Yellow || engine.Yellow.Opponent() != engine.Red || engine.Empty.Opponent() != engine.Empty {
		t.Fatal("Opponent() mapping is wrong")
	}
}

func TestCellsReturnsACopy(t *testing.T) {
	board := engine.NewBoard()
	cells := board.Cells()
	cells[5][0] = engine.Red
	if board.Cells()[5][0] != engine.Empty {
		t.Fatal("mutating Cells() changed the board")
	}
}

// NFR-007: the engine is pure and imports nothing from transport, storage or the game service.
func TestEngineImportsOnlyPureStandardLibrary(t *testing.T) {
	allowed := map[string]bool{"errors": true}
	files, err := filepath.Glob(filepath.Join("..", "internal", "engine", "*.go"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no engine sources found: %v", err)
	}
	for _, file := range files {
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, spec := range parsed.Imports {
			path, _ := strconv.Unquote(spec.Path.Value)
			if !allowed[path] {
				t.Errorf("%s imports %q; the engine may only import %v", filepath.Base(file), path, allowed)
			}
		}
	}
}
