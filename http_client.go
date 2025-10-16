package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type EvaluatorRequest struct {
	Email     string `json:"email"`
	Task      string `json:"task"`
	Round     uint   `json:"round"`
	Nonce     string `json:"nonce"`
	RepoURL   string `json:"repo_url"`
	CommitSHA string `json:"commit_sha"`
	PagesURL  string `json:"pages_url"`
}

func HTTPPostPutClient(url string, headers map[string]string, data any, method string) ([]byte, error) {
	var body io.Reader

	if data != nil {
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		body = bytes.NewBuffer(jsonBytes)
	} else {
		body = nil
	}

	client := &http.Client{Timeout: 10 * time.Second}

	var m string
	switch method {
	case "POST":
		m = http.MethodPost
	case "PUT":
		m = http.MethodPut
	default:
		return nil, errors.New("invalid method: " + method)
	}

	req, err := http.NewRequest(m, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, errors.New("non-2xx response: " + resp.Status + " - " + string(b))
	}

	return io.ReadAll(resp.Body)
}

func HTTPGetClient(url string, headers map[string]string) ([]byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("non-2xx response: %s - %s", resp.Status, string(b))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func HTTPDeleteClient(url string, headers map[string]string) error {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return errors.New("non-2xx response: " + resp.Status + " - " + string(b))
	}

	return nil
}

func GetWithBackoff(url string, headers map[string]string, retries int, delay time.Duration) ([]byte, error) {
	var lastErr error

	for i := 0; i < retries; i++ {
		body, err := HTTPGetClient(url, headers)
		if err == nil {
			return body, nil
		}

		lastErr = err
		if i < retries-1 {
			time.Sleep(delay)
			delay *= 2
		}
	}

	return nil, fmt.Errorf("all retries failed after %d attempts: %w", retries, lastErr)
}

func PostPutWithBackoff(url string, headers map[string]string, data any, method string, retries int, delay time.Duration) ([]byte, error) {
	var lastErr error

	for i := 0; i < retries; i++ {
		body, err := HTTPPostPutClient(url, headers, data, method)
		if err == nil {
			return body, nil
		}

		lastErr = err
		if i < retries-1 {
			time.Sleep(delay)
			delay *= 2
		}
	}

	return nil, fmt.Errorf("all retries failed after %d attempts: %w", retries, lastErr)
}

func SatisfyEvaluator(req EvaluatorRequest, url string) error {
	_, err := PostPutWithBackoff(url, Headers(), req, "POST", 5, 2*time.Second)
	if err != nil {
		return err
	}

	jsonBytes, _ := json.Marshal(req)
	fmt.Println("Sent evaluator request for task:", req.Task, "to", url, "with body:", string(jsonBytes))

	return nil
}
