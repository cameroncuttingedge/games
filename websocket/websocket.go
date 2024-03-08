package websocket

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/cameroncuttingedge/tic_tac_toe/events"
	"github.com/cameroncuttingedge/tic_tac_toe/utils"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

var gameConnections = make(map[string][]*websocket.Conn)
var lock sync.Mutex

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true }, // Allow connections from any origin
}

func GameWebSocketHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID, ok := vars["gameID"]
	if !ok {
		http.Error(w, "Game ID is required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Str("gameID", gameID).Msg("WebSocket upgrade error")
		return
	}
	defer conn.Close()

	registerConnection(gameID, conn)
	log.Info().Str("gameID", gameID).Int("connectionsCount", len(gameConnections[gameID])).Msg("WebSocket connection established and registered")

	sendGameState(conn, gameID)

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			log.Error().Err(err).Str("gameID", gameID).Msg("WebSocket closed unexpectedly")
			break
		}
	}
}

func sendGameState(conn *websocket.Conn, gameID string) {
	lock.Lock()
	game, exists := utils.Games[gameID]
	lock.Unlock()

	if !exists || game == nil {
		log.Error().Str("gameID", gameID).Msg("Game not found or is nil, cannot send game state")
		return
	}

	gameState := game.GetState()

	if err := conn.WriteJSON(gameState); err != nil {
		log.Error().Err(err).Str("gameID", gameID).Msg("Error sending game state")
	}
}

func BroadcastGameStateUpdate(gameID string, gameState events.GameState) {
	lock.Lock()
	logAllConnections()
	connections, exists := gameConnections[gameID]
	lock.Unlock()

	if !exists {
		log.Info().Str("gameID", gameID).Msg("No connections to broadcast")
		return
	}

	log.Info().Str("gameID", gameID).Int("connectionsCount", len(connections)).Msg("Broadcasting game state update")
	gameStateJSON, _ := json.Marshal(gameState)
	log.Info().Str("gameID", gameID).RawJSON("gameStateJSON", gameStateJSON).Msg("Attempting to broadcast game state")
	for i, conn := range connections {
		if err := conn.WriteJSON(gameState); err != nil {
			log.Error().Err(err).Str("gameID", gameID).Msgf("Failed to broadcast game state update to connection %d", i)
		} else {
			// Log successful broadcast
			log.Info().Str("gameID", gameID).Msgf("Successfully broadcasted game state update to connection %d", i)
		}
	}
}

func registerConnection(gameID string, conn *websocket.Conn) {
	lock.Lock()
	defer lock.Unlock()
	gameConnections[gameID] = append(gameConnections[gameID], conn)
	log.Info().Str("gameID", gameID).Msg("New WebSocket connection registered")
}

func deregisterConnection(gameID string, conn *websocket.Conn) {
	lock.Lock()
	defer lock.Unlock()
	connections := gameConnections[gameID]
	for i, c := range connections {
		if c == conn {
			gameConnections[gameID] = append(connections[:i], connections[i+1:]...)
			log.Info().Str("gameID", gameID).Int("remainingConnections", len(gameConnections[gameID])).Msg("WebSocket connection deregistered")
			break
		}
	}
}

func StartEventListening() {
	log.Info().Msg("Event listener starting...")
	go func() {
		for gameEvent := range events.EventChannel {
			gameEventData, err := json.Marshal(gameEvent.Data)
			if err != nil {
				// Handle error, maybe log that marshaling failed
				log.Error().Err(err).Msg("Failed to marshal game event data to JSON")
				continue
			}

			log.Info().
				Str("gameID", gameEvent.Data.ID).
				RawJSON("gameEvent", gameEventData).
				Msg("Received game event, broadcasting update")
			BroadcastGameStateUpdate(gameEvent.Data.ID, gameEvent.Data)
		}
		log.Info().Msg("Event listener goroutine exited.")
	}()
}

func logAllConnections() {
	if len(gameConnections) == 0 {
		log.Info().Msg("No active WebSocket connections for any game")
		return
	}
	for gameID, conns := range gameConnections {
		connIDs := make([]string, len(conns))
		for i, conn := range conns {
			connIDs[i] = conn.RemoteAddr().String()
		}
		log.Info().Str("gameID", gameID).Strs("connections", connIDs).Msg("Current WebSocket connections")
	}
}
