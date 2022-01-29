package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/config"
	"github.com/gochik/chik/handlers/heartbeat"
	"github.com/gochik/chik/handlers/router"
	"github.com/rs/zerolog/log"
)

var peers sync.Map

// Version executable version
var Version = "dev"

func main() {
	log.Info().Msgf("Version: %v", Version)

	config.SetConfigFileName("server.conf")
	config.AddSearchPath("/etc/chik")
	config.AddSearchPath(".")
	err := config.ParseConfig()
	if err != nil {
		log.Warn().Msgf("Error parsing config file: %v", err)
	}
	ok := true

	var token string
	config.GetStruct("connection.token", &token)
	if token == "" {
		config.Set("connection.token", "")
		log.Warn().Msg("Cannot get CA token from config file")
		ok = false
	}

	var port uint16
	config.GetStruct("connection.port", &port)
	if port == 0 {
		config.Set("connection.port", uint16(6767))
		log.Warn().Msg("Cannot get port from config file")
		ok = false
	}

	if !ok {
		config.Sync()
		log.Fatal().Msg("Config file contains errors, check the logfile.")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conf, err := config.TLSConfig(ctx, token)
	if err != nil {
		log.Fatal().Msgf("Failed to get TLS config: %v", err)
	}

	srv, err := tls.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port), conf)
	if err != nil {
		log.Fatal().Msgf("Error starting server: %v", err)
	}

	for {
		connection, err := srv.Accept()
		if err != nil {
			log.Error().Msgf("Connection error: %v", err)
			continue
		}

		// Creating the controller that is handling the newly connected client
		go func() {
			log.Info().Msg("New connection: creating a new controller")
			controller := chik.NewController()
			innerctx, innercancel := context.WithCancel(context.Background())
			go controller.Start(innerctx, []chik.Handler{
				router.New(&peers),
				heartbeat.New(2 * time.Minute),
			})
			ctx, remoteCancel := chik.StartRemote(controller, connection, chik.MaxIdleTime)
			<-ctx.Done()
			innercancel()
			remoteCancel()
		}()
	}
}
