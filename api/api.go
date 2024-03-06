package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/cameroncuttingedge/tic_tac_toe/game"
	"github.com/cameroncuttingedge/tic_tac_toe/utils"
	"github.com/cameroncuttingedge/tic_tac_toe/websocket"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

var (
	lock sync.Mutex
)

type Move struct {
	Username string `json:"username"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
}

func StartAPI() {
	r := mux.NewRouter() // Create a new router

	r.HandleFunc("/game/create", createGameHandler).Methods("POST")
	r.HandleFunc("/game/{gameID}/join", joinGameHandler).Methods("POST")
	r.HandleFunc("/game/{gameID}/move", makeMoveHandler).Methods("POST")
	r.HandleFunc("/game/{gameID}/state/", GetGameStateHandler).Methods("GET")
	r.HandleFunc("/game/{gameID}/restart", RequestRestart).Methods("POST")
	r.HandleFunc("/ws/game/state/{gameID}", websocket.GameWebSocketHandler)

	fmt.Println("Server started on http://localhost:8080")
	http.ListenAndServe(":8080", r)
}

func RequestRestart(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID, ok := vars["gameID"]
	if !ok {
		http.Error(w, "Game ID is required", http.StatusBadRequest)
		return
	}

	playerID := r.URL.Query().Get("playerID")
	if playerID == "" {
		http.Error(w, "Player ID is required", http.StatusBadRequest)
		return
	}

	lock.Lock()
	defer lock.Unlock()

	game, exists := utils.Games[gameID]
	if !exists {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	if !game.IsValidPlayer(playerID) {
		http.Error(w, "Invalid player", http.StatusBadRequest)
		return
	}

	game.RequestRestart(playerID)

	response := map[string]string{"message": "Restart request acknowledged."}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func GetGameStateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID, ok := vars["gameID"]
	if !ok {
		http.Error(w, "Game ID is required", http.StatusBadRequest)
		return
	}

	lock.Lock()
	defer lock.Unlock()

	game, exists := utils.Games[gameID]
	if !exists {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}

	gameState := game.GetState()

	jsonData, err := json.Marshal(gameState)
	if err != nil {
		http.Error(w, "Failed to marshal game state to JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func createGameHandler(w http.ResponseWriter, r *http.Request) {
	lock.Lock()
	defer lock.Unlock()

	log.Info().Msg("Attempting to create New game")

	playerID := r.URL.Query().Get("playerID")
	if playerID == "" {
		http.Error(w, "Player ID is required", http.StatusBadRequest)
		return
	}

	gameID := utils.GenerateUUIDString()
	newGame := game.NewGame(gameID, playerID)
	utils.Games[gameID] = newGame
	newGame.PublishState()

	response := map[string]string{"gameID": gameID}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func joinGameHandler(w http.ResponseWriter, r *http.Request) {
	lock.Lock()
	defer lock.Unlock()

	vars := mux.Vars(r) // Get URL parameters
	gameID := vars["gameID"]
	playerID := r.URL.Query().Get("playerID")
	if playerID == "" {
		http.Error(w, "Player ID is required", http.StatusBadRequest)
		return
	}

	game, exists := utils.Games[gameID]
	if !exists || game.Status != "waiting" || game.Players[0] == playerID {
		http.Error(w, "Invalid game ID, game not waiting, or same player", http.StatusBadRequest)
		return
	}

	game.AddSecondPlayer(playerID)

	response := map[string]string{"message": fmt.Sprintf("Player %s successfully joined the game!", playerID)}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func makeMoveHandler(w http.ResponseWriter, r *http.Request) {
	lock.Lock()
	defer lock.Unlock()

	move, err := validateAndExtractMove(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	game, err := validateGameState(mux.Vars(r)["gameID"], move.Username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = executeMoveAndUpdateState(game, move)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := struct {
		Board  [3][3]string `json:"board"` // Convert game.Player to string for JSON encoding
		Turn   string       `json:"turn"`
		Over   bool         `json:"over"`
		Winner string       `json:"winner"`
	}{
		Board:  utils.ConvertBoardToStrings(game.Board),
		Turn:   string(game.Turn),
		Over:   game.Over,
		Winner: string(game.Winner),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func validateGameState(gameID string, username string) (*game.Game, error) {
	game, exists := utils.Games[gameID]
	if !exists {
		return nil, fmt.Errorf("game not found")
	}

	if game.Status != "active" {
		return nil, fmt.Errorf("game is not active")
	}

	if !game.IsValidPlayer(username) {
		return nil, fmt.Errorf("invalid player: %s", username)
	}

	if game.Turn != username {
		return nil, fmt.Errorf("it's not your turn")
	}

	return game, nil
}

func validateAndExtractMove(r *http.Request) (*Move, error) {
	vars := mux.Vars(r)
	_, ok := vars["gameID"]
	if !ok {
		return nil, fmt.Errorf("game ID is required")
	}

	var move Move
	if err := json.NewDecoder(r.Body).Decode(&move); err != nil {
		return nil, fmt.Errorf("error decoding JSON: %v", err)
	}

	// Validate required fields
	if move.Username == "" {
		return nil, fmt.Errorf("no username in JSON")
	}
	if move.X < 0 || move.Y < 0 || move.X > 2 || move.Y > 2 {
		return nil, fmt.Errorf("coordinates are out of bounds")
	}

	return &move, nil
}

func executeMoveAndUpdateState(game *game.Game, move *Move) error {
	if !game.MakeMove(move.Username, move.X, move.Y) {
		return fmt.Errorf("invalid move")
	}

	if game.CheckWin() {
		game.Over = true
		game.Winner = game.Turn
	} else if game.CheckDraw() {
		game.Over = true
	}
	return nil
}
