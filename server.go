package main

import (
	"context"
	"crypto/rand"
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

var peers = sync.Map{}

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

	var publicKeyPath string
	config.GetStruct("connection.public_key_path", &publicKeyPath)
	if publicKeyPath == "" {
		config.Set("connection.public_key_path", "")
		log.Warn().Msg("Cannot get public key path from config file")
		ok = false
	}

	var privateKeyPath string
	config.GetStruct("connection.private_key_path", &privateKeyPath)
	if privateKeyPath == "" {
		config.Set("connection.private_key_path", "")
		log.Warn().Msg("Cannot get private key path from config file")
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

	cert, err := tls.LoadX509KeyPair(publicKeyPath, privateKeyPath)
	if err != nil {
		log.Fatal().Msgf("Error loading tls certificate: %v", err)
	}

	config := tls.Config{Certificates: []tls.Certificate{cert}}
	config.Rand = rand.Reader

	listener, err := tls.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port), &config)
	if err != nil {
		log.Fatal().Msgf("Error listening: %v", err)
	}

	for {
		connection, err := listener.Accept()
		if err != nil {
			log.Error().Msgf("Connection error: %v", err)
			continue
		}

		// Creating the controller that is handling the newly connected client
		go func() {
			log.Info().Msg("New connection: creating a new controller")
			controller := chik.NewController()
			ctx, cancel := context.WithCancel(context.Background())
			go controller.Start(ctx, []chik.Handler{
				router.New(&peers),
				heartbeat.New(2 * time.Minute),
			})
			<-controller.Connect(connection)
			cancel()
		}()
	}
}
