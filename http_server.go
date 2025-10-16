// server.go (your file)
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	jobQueue *Queue
)

type Attachment struct {
	Name string `json:"name" binding:"required"`
	URL  string `json:"url" binding:"required"`
}

type UserRequest struct {
	Email         string       `json:"email" binding:"required,email"`
	Secret        string       `json:"secret" binding:"required"`
	Task          string       `json:"task" binding:"required"`
	Round         uint         `json:"round" binding:"required"`
	Nonce         string       `json:"nonce" binding:"required"`
	Brief         string       `json:"brief" binding:"required"`
	Checks        []string     `json:"checks"`
	EvaluationURL string       `json:"evaluation_url" binding:"required,url"`
	Attachments   []Attachment `json:"attachments"`
}

func StartServer(addr string) error {
	gin.SetMode(gin.ReleaseMode)

	jobQueue = NewQueue(100, 3)
	rootCtx, rootCancel := context.WithCancel(context.Background())

	defer rootCancel()
	jobQueue.Start(rootCtx)

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "running",
			"time":   time.Now().Format("02-01-2006 15:04:05"),
			"queue": gin.H{
				"capacity": cap(jobQueue.ch),
				"len":      len(jobQueue.ch),
				"workers":  jobQueue.workers,
			},
		})
	})

	r.POST("/ingest", func(c *gin.Context) {
		var req UserRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if data, err := json.MarshalIndent(req, "", "  "); err == nil {
			log.Println("Incoming request:\n", string(data))
		}

		if req.Secret != os.Getenv("API_SECRET") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_secret"})
			return
		}

		if err := jobQueue.TryEnqueue(Job{Req: req}, 200*time.Millisecond); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "queue_busy",
				"error":  err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "queued"})
	})

	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	idleConnsClosed := make(chan struct{})
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt)
		<-ch
		log.Println("Shutting down...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)

		rootCancel()
		close(idleConnsClosed)
	}()

	log.Printf("HTTP Server listening on %s\n", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	<-idleConnsClosed
	return nil
}
