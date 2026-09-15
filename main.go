package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

const port = "8080"

func main() {
	db, err := InitDB()
	if err != nil {
		log.Fatalf("db init error: %v", err)
	}
	defer db.Close()

	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	r.Static("/static", "./frontend")
	r.Static("/uploads", "./uploads")

	r.GET("/", func(c *gin.Context) {
		c.File("./frontend/login.html")
	})
	r.GET("/applications", func(c *gin.Context) {
		c.File("./frontend/application.html")
	})
	r.GET("/approval", func(c *gin.Context) {
		c.File("./frontend/approval.html")
	})

	api := r.Group("/api")
	{
		api.POST("/login", LoginHandler)
	}

	auth := api.Group("")
	auth.Use(AuthMiddleware())
	{
		auth.GET("/vehicles", GetVehiclesHandler)
		auth.GET("/applications", GetApplicationsHandler)
		auth.GET("/applications/:id", GetApplicationByIDHandler)

		sales := auth.Group("")
		sales.Use(AuthMiddleware(RoleSales, RoleAdmin))
		{
			sales.POST("/applications", CreateApplicationHandler)
			sales.POST("/applications/:id/submit", SubmitApplicationHandler)
			sales.POST("/applications/:id/documents", UploadDocumentHandler)
		}

		approver := auth.Group("")
		approver.Use(AuthMiddleware(RoleApprover, RoleAdmin))
		{
			approver.POST("/applications/:id/approve", ApproveApplicationHandler)
			approver.POST("/applications/:id/reject", RejectApplicationHandler)
		}
	}

	log.Printf("server running on http://localhost:%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
