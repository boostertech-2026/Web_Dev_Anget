package main

import (
	"log"
	"time"

	"eino-ops-server/agent"
	"eino-ops-server/models"
	"eino-ops-server/routes"
)

func main() {
	models.InitDB()
	agent.InitGraph()

	// Start periodic host health check (every 60 seconds)
	agent.StartHealthCheck(60 * time.Second)

	r := routes.SetupRouter()
	log.Println("Server starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
