package main

import (
	"os"

	"github.com/cameroncuttingedge/tic_tac_toe/api"
	"github.com/cameroncuttingedge/tic_tac_toe/websocket"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	InitializeLogger()
	websocket.StartEventListening()
	log.Info().Msg("Starting App")
	api.StartAPI()

}

func InitializeLogger() {
	loggingEnabled := os.Getenv("LOGGING")
	if loggingEnabled != "true" {
		log.Logger = log.Output(os.Stdout)
	} else {
		runLogFile, err := os.OpenFile(
			"myapp.log",
			os.O_APPEND|os.O_CREATE|os.O_WRONLY,
			0664,
		)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to open log file")
		}
		multi := zerolog.MultiLevelWriter(runLogFile, os.Stdout)
		log.Logger = zerolog.New(multi).With().Timestamp().Logger()
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}
