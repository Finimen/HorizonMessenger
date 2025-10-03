package main

// PROPRIETARY AND CONFIDENTIAL
// This code contains trade secrets and confidential material of Finimen Sniper / FSC.
// Any unauthorized use, disclosure, or duplication is strictly prohibited.
// © 2025 Finimen Sniper / FSC. All rights reserved.

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "massager/docs"
)

func InitializeServer() (*Container, error) {
	container, err := NewContainer()
	if err != nil {
		return nil, err
	}
	return container, nil
}

func GetContainer() (*Container, error) {
	return NewContainer()
}

func Stop(container *Container) {
	container.Close()
}

// @title Massager API
// @version 1.0
// @description Real-time messaging web service
// @termsOfService https://github.com/Finimen/SafeMassager

// @contact.name FinimenSniper
// @contact.url https://github.com/Finimen/SafeMassager
// @contact.email finimensniper@gmail.com

// @license.name MIT
// @license.url https://github.com/Finimen/SafeMassager

// @host localhost:8080
// @BasePath /api

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT token in the format: Bearer <token>

// @schemes http

// main initializes and starts the messenger server
// @Summary Information about API
// @Description Returns basic information about the API
// @Tags info
// @Produce json
// @Success 200 {object} map[string]string
// @Router / [get]
func main() {
	container, err := InitializeServer()
	if err != nil {
		log.Fatal("Failed to initialize server:", err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Server is starting on http://localhost%s\n", container.Server.Addr)
		if err := container.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: http://localhost%s\n", err)
		}
	}()

	<-quit
	log.Println("Shutting down server...")

	Stop(container)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := container.Server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}
