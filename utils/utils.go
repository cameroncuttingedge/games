// utils/utils.go

package utils

import (
	"github.com/cameroncuttingedge/tic_tac_toe/game"
	"github.com/google/uuid"
)

func GenerateUUIDString() string {
	id := uuid.New()
	return id.String()
	//return "1"
}

func ConvertBoardToStrings(board [3][3]game.Player) [3][3]string {
	var stringBoard [3][3]string
	for i, row := range board {
		for j, cell := range row {
			stringBoard[i][j] = string(cell)
		}
	}
	return stringBoard
}
