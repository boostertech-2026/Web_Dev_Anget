package routes

import (
	"langgraph-ops-server/handlers"
	"langgraph-ops-server/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	api := r.Group("/api")
	{
		api.POST("/login", handlers.Login)

		// Protected routes
		auth := api.Group("")
		auth.Use(middleware.AuthRequired())
		{
			auth.POST("/agent/chat", handlers.AgentChat)

			// Dashboard
			auth.GET("/dashboard/summary", handlers.GetDashboardSummary)

			hosts := auth.Group("/host")
			{
				hosts.GET("", handlers.GetHosts)
				hosts.POST("", handlers.AddHost)
				hosts.PUT("/:id", handlers.UpdateHost)
				hosts.DELETE("/:id", handlers.DeleteHost)
				hosts.PUT("/:id/ipmi", handlers.UpdateHostIpmi)
				hosts.POST("/:id/ipmi/check", handlers.CheckIpmiConnectivity)
			}

			// Metrics
			metrics := auth.Group("/metrics")
			{
				metrics.GET("/latest", handlers.GetLatestMetrics)
				metrics.GET("/history", handlers.GetMetricsHistory)
				metrics.GET("/traffic", handlers.GetTrafficMetrics)
			}

			// Alerts
			alerts := auth.Group("/alerts")
			{
				alerts.GET("", handlers.GetAlerts)
				alerts.POST("/:id/ack", handlers.AckAlert)
				alerts.POST("/:id/resolve", handlers.ResolveAlert)
				alerts.GET("/rules", handlers.GetAlertRules)
				alerts.PUT("/rules/:id", handlers.UpdateAlertRule)
			}

			// Error history
			errors := auth.Group("/errors")
			{
				errors.GET("", handlers.GetErrorHistory)
				errors.GET("/stats", handlers.GetErrorStats)
				errors.POST("/:id/resolve", handlers.ResolveError)
			}

			task := auth.Group("/task")
			{
				task.GET("/list", handlers.GetTasks)
				task.POST("/create", handlers.CreateTask)
			}

			client := auth.Group("/client")
			{
				client.GET("/list", handlers.GetClients)
				client.POST("", handlers.AddClient)
				client.PUT("/:id", handlers.UpdateClient)
				client.DELETE("/:id", handlers.DeleteClient)
				client.POST("/heartbeat", handlers.ClientHeartbeat)
				client.POST("/send", handlers.SendClientCmd)
			}

			ssh := auth.Group("/ssh")
			{
				ssh.POST("/connect", handlers.SshConnect)
				ssh.POST("/execute", handlers.SshExecute)
			}
		}
	}

	r.GET("/ws/log", handlers.WSLog)
	r.GET("/ws/client", handlers.ClientWS)
	r.GET("/ws/cmd", handlers.ClientWS)

	return r
}
