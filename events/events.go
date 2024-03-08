package events

type GameState struct {
	ID             string          `json:"id"`
	Board          [3][3]string    `json:"board"`
	Turn           string          `json:"turn"`
	Winner         string          `json:"winner"`
	Over           bool            `json:"over"`
	Players        [2]string       `json:"players"`
	CurrentX       string          `json:"currentX"`
	Status         string          `json:"status"`
	RestartRequest map[string]bool `json:"restartRequest"`
}

type GameEvent struct {
	Data GameState
}

var EventChannel = make(chan GameEvent, 100)
