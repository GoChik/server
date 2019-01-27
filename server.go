package main

import (
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"github.com/gochik/chik"
	"github.com/gochik/chik/config"
	"github.com/gochik/chik/handlers"
	"github.com/sirupsen/logrus"
)

var peers = sync.Map{}

// Version executable version
var Version = "dev"

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.Debug("Version: ", Version)

	config.SetConfigFileName("server.conf")
	config.AddSearchPath("/etc/chik")
	config.AddSearchPath(".")
	err := config.ParseConfig()
	if err != nil {
		logrus.Warn("Error parsing config file: ", err)
	}
	ok := true

	var publicKeyPath string
	config.GetStruct("connection.public_key_path", &publicKeyPath)
	if publicKeyPath == "" {
		config.Set("connection.public_key_path", "")
		logrus.Warn("Cannot get public key path from config file")
		ok = false
	}

	var privateKeyPath string
	config.GetStruct("connection.private_key_path", &privateKeyPath)
	if privateKeyPath == "" {
		config.Set("connection.private_key_path", "")
		logrus.Warn("Cannot get private key path from config file")
		ok = false
	}

	var port uint16
	config.GetStruct("connection.port", &port)
	if port == 0 {
		config.Set("connection.port", uint16(6767))
		logrus.Warn("Cannot get port from config file")
		ok = false
	}

	if !ok {
		config.Sync()
		logrus.Fatal("Config file contains errors, check the logfile.")
	}

	cert, err := tls.LoadX509KeyPair(publicKeyPath, privateKeyPath)
	if err != nil {
		logrus.Fatal("Error loading tls certificate", err)
	}

	config := tls.Config{Certificates: []tls.Certificate{cert}}
	config.Rand = rand.Reader

	listener, err := tls.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port), &config)
	if err != nil {
		logrus.Fatal("Error listening", err)
	}

	for {
		connection, err := listener.Accept()
		if err != nil {
			logrus.Debug("Connection error", err)
			continue
		}

		// Creating the controller that is handling the newly connected client
		logrus.Debug("Creating a new controller")
		go func() {
			controller := chik.NewController()
			controller.Start(handlers.NewForwardingHandler(&peers))
			controller.Start(handlers.NewHeartBeatHandler(2 * time.Minute))
			<-controller.Connect(connection)
			controller.Shutdown()
		}()
	}
}
