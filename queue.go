// queue.go
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"
)

type Job struct {
	Req UserRequest
	// You can add metadata here (receivedAt, attempt, etc.)
}

type Queue struct {
	ch      chan Job
	workers int
}

func NewQueue(size, workers int) *Queue {
	return &Queue{
		ch:      make(chan Job, size),
		workers: workers,
	}
}

func (q *Queue) Start(ctx context.Context) {
	for i := 0; i < q.workers; i++ {
		go func(id int) {
			log.Printf("[worker %d] started", id)
			for {
				select {
				case <-ctx.Done():
					log.Printf("[worker %d] stopping", id)
					return
				case job := <-q.ch:
					processJob(ctx, job)
				}
			}
		}(i + 1)
	}
}

func (q *Queue) TryEnqueue(job Job, timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case q.ch <- job:
		return nil
	case <-timer.C:
		return errors.New("queue_full_or_slow")
	}
}

func ProcessRequest(req UserRequest) error {
	switch req.Round {
	case 1:
		return Round1(req)
	case 2:
		return Round2(req)
	default:
		return fmt.Errorf("unsupported_round:%d", req.Round)
	}
}

func processJob(ctx context.Context, job Job) {
	_, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	err := ProcessRequest(job.Req)
	if err != nil {
		log.Printf("job_failed: %v", err)
		return
	}
}
