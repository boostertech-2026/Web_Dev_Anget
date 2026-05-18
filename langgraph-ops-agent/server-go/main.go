package main

import (
	"log"
	"langgraph-ops-server/agent"
	"langgraph-ops-server/models"
	"langgraph-ops-server/routes"
)

func main() {
	models.InitDB()
	agent.InitGraph()

	r := routes.SetupRouter()
	log.Println("Server starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
