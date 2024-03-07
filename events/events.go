package events

type GameState struct {
	ID             string
	Board          [3][3]string
	Turn           string
	Winner         string
	Over           bool
	Players        [2]string
	CurrentX       string
	Status         string
	RestartRequest map[string]bool
}

type GameEvent struct {
	Data GameState
}

var EventChannel = make(chan GameEvent, 100)
