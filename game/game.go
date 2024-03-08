package game

import (
	"fmt"
	"strings"

	"github.com/cameroncuttingedge/tic_tac_toe/events"
	"github.com/rs/zerolog/log"
)

type Player string

const (
	PlayerX Player = "X"
	PlayerO Player = "O"
	None    Player = " "
)

type Game struct {
	ID             string
	Board          [3][3]Player
	Turn           string
	Winner         string
	Over           bool
	Players        [2]string
	CurrentX       string
	Status         string
	RestartRequest map[string]bool
}

// NewGame initializes a new game with the first player
func NewGame(gameID string, player1ID string) *Game {

	game := &Game{
		ID:             gameID,
		Board:          [3][3]Player{{None, None, None}, {None, None, None}, {None, None, None}},
		Turn:           player1ID, // First player's turn by default
		Players:        [2]string{player1ID, ""},
		CurrentX:       player1ID,
		Status:         "waiting",
		RestartRequest: make(map[string]bool),
	}

	return game
}

func (g *Game) AddSecondPlayer(player2ID string) {
	if g.Players[1] == "" && g.Status == "waiting" {
		g.Players[1] = player2ID
		g.Status = "active"
		g.PublishState()
	}
}

// SwapPlayers swaps the symbols for the players
func (g *Game) SwapPlayers() {
	if g.CurrentX == g.Players[0] {
		g.CurrentX = g.Players[1]
	} else {
		g.CurrentX = g.Players[0]
	}
}

// MakeMove updates the game board with the player's move, switches the turn, and checks for win/draw
// Adjusted MakeMove to use the username for turn management
func (g *Game) MakeMove(username string, x, y int) bool {
	if !g.IsValidPlayer(username) || g.Status != "active" || g.Turn != username {
		return false
	}

	playerSymbol := g.getPlayerSymbol(username)
	if x < 0 || y < 0 || x > 2 || y > 2 || g.Board[x][y] != None {
		return false
	}

	g.Board[x][y] = playerSymbol
	win := g.CheckWin()
	draw := g.CheckDraw()

	if win || draw {
		g.Over = true
		if win {
			g.Winner = username
		}
	}

	g.SwitchTurn()
	g.PublishState() // Ensure this is called to update all clients immediately
	return true
}

func (g *Game) RequestRestart(playerID string) bool {
	g.RestartRequest[playerID] = true
	log.Info().Str("playerID", playerID).Str("gameID", g.ID).Msg("Restart requested")
	// Check if both players requested a restart
	if len(g.RestartRequest) == 2 {
		// Reset game state for a new game, but keep players
		g.Board = [3][3]Player{{None, None, None}, {None, None, None}, {None, None, None}}
		g.Over = false
		g.Winner = ""
		g.RestartRequest = make(map[string]bool)
		log.Info().Str("gameID", g.ID).Msg("Both players requested restart. Game state reset.")
		// if true, update
		g.PublishState()
		return true
	}
	log.Info().Str("gameID", g.ID).Int("restartRequests", len(g.RestartRequest)).Msg("Waiting for the other player to request restart")
	g.PublishState()
	return false
}

func (g *Game) SwitchTurn() {
	if g.Turn == g.Players[0] {
		g.Turn = g.Players[1]
	} else {
		g.Turn = g.Players[0]
	}
}

func (g *Game) CheckWin() bool {
	// Define win conditions
	lines := [][3][2]int{
		{{0, 0}, {0, 1}, {0, 2}}, {{1, 0}, {1, 1}, {1, 2}}, {{2, 0}, {2, 1}, {2, 2}},
		{{0, 0}, {1, 0}, {2, 0}}, {{0, 1}, {1, 1}, {2, 1}}, {{0, 2}, {1, 2}, {2, 2}},
		{{0, 0}, {1, 1}, {2, 2}}, {{2, 0}, {1, 1}, {0, 2}},
	}
	for _, line := range lines {
		if g.Board[line[0][0]][line[0][1]] != None &&
			g.Board[line[0][0]][line[0][1]] == g.Board[line[1][0]][line[1][1]] &&
			g.Board[line[1][0]][line[1][1]] == g.Board[line[2][0]][line[2][1]] {
			return true
		}
	}
	return false
}

func (g *Game) CheckDraw() bool {
	for _, row := range g.Board {
		for _, cell := range row {
			if cell == None {
				return false
			}
		}
	}
	return true
}

func (g *Game) getPlayerSymbol(username string) Player {
	if username == g.CurrentX {
		return PlayerX
	}
	return PlayerO
}

func (g Game) PrintBoard() {
	for _, row := range g.Board {
		for _, cell := range row {
			if cell == None {
				fmt.Print("- ")
			} else {
				fmt.Printf("%s ", cell)
			}
		}
		fmt.Println()
	}
}

func (g *Game) IsValidPlayer(username string) bool {
	for _, player := range g.Players {
		if player == username {
			return true
		}
	}
	return false
}

func (g *Game) PublishState() {
	gameState := events.GameState{
		ID:             g.ID,
		Board:          convertBoard(g.Board),
		Turn:           g.Turn,
		Winner:         g.Winner,
		Over:           g.Over,
		Players:        g.Players,
		CurrentX:       g.CurrentX,
		Status:         g.Status,
		RestartRequest: g.RestartRequest,
	}
	if gameState.RestartRequest == nil {
		gameState.RestartRequest = make(map[string]bool)
	}

	log.Info().
		Str("gameID", g.ID).
		Msg("Publishing game state")
	events.EventChannel <- events.GameEvent{Data: gameState}
	log.Info().
		Str("gameID", g.ID).
		Msg("Published game state")
}

// GameState returns a string representation of the current game state
func (g *Game) GameState() string {
	var sb strings.Builder

	// Print the Board
	sb.WriteString("Current Board:\n")
	for _, row := range g.Board {
		for _, cell := range row {
			if cell == None {
				sb.WriteString("- ")
			} else {
				sb.WriteString(fmt.Sprintf("%s ", cell))
			}
		}
		sb.WriteString("\n")
	}

	// Print current turn, status, and players
	sb.WriteString(fmt.Sprintf("Turn: %s\n", g.Turn))
	sb.WriteString(fmt.Sprintf("Status: %s\n", g.Status))
	sb.WriteString(fmt.Sprintf("Player X: %s\n", g.CurrentX))
	otherPlayer := g.Players[0]
	if otherPlayer == g.CurrentX {
		otherPlayer = g.Players[1]
	}
	sb.WriteString(fmt.Sprintf("Player O: %s\n", otherPlayer))
	if g.Over {
		sb.WriteString(fmt.Sprintf("Game Over: Yes\n"))
		if g.Winner != "" {
			sb.WriteString(fmt.Sprintf("Winner: %s\n", g.Winner))
		} else {
			sb.WriteString("Winner: None (Draw)\n")
		}
	} else {
		sb.WriteString("Game Over: No\n")
	}

	return sb.String()
}

func (g *Game) GetState() map[string]interface{} {
	state := make(map[string]interface{})
	state["ID"] = g.ID
	state["Board"] = g.Board
	state["Turn"] = g.Turn
	state["Winner"] = g.Winner
	state["Over"] = g.Over
	state["Players"] = g.Players
	state["CurrentX"] = g.CurrentX
	state["Status"] = g.Status
	state["RestartRequest"] = g.RestartRequest
	return state
}

func convertBoard(board [3][3]Player) [3][3]string {
	var converted [3][3]string
	for i, row := range board {
		for j, cell := range row {
			converted[i][j] = string(cell)
		}
	}
	return converted
}
