package sentinel

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	json "github.com/json-iterator/go"
	sentinelDTO "github.com/m4n5ter/mindscape/sentinel/api/dto"
)

// Sentinel 客户端
type Sentinel struct {
	client     *http.Client
	baseURL    string
	pathPrefix string
}

// NewSentinel 创建一个 Sentinel 客户端
func NewSentinel(client *http.Client, baseURL string) *Sentinel {
	return &Sentinel{client: client, baseURL: baseURL, pathPrefix: "/api/v1/sentinel"}
}

// CreateTask 创建一个任务
func (s *Sentinel) CreateTask(ctx context.Context, createReq sentinelDTO.CreateTaskRequest) (*sentinelDTO.TaskResponse, error) {
	fullURL := fmt.Sprintf("%s%s/tasks", s.baseURL, s.pathPrefix)
	jsonBody, err := json.Marshal(createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	var taskResp sentinelDTO.TaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&taskResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &taskResp, nil
}

// GetTask 获取一个任务
func (s *Sentinel) GetTask(ctx context.Context, taskID uuid.UUID) (*sentinelDTO.TaskResponse, error) {
	fullURL := fmt.Sprintf("%s%s/tasks/%s", s.baseURL, s.pathPrefix, taskID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	var taskResp sentinelDTO.TaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&taskResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &taskResp, nil
}

// DeleteTask 删除一个任务
func (s *Sentinel) DeleteTask(ctx context.Context, taskID uuid.UUID) error {
	fullURL := fmt.Sprintf("%s%s/tasks/%s", s.baseURL, s.pathPrefix, taskID)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, fullURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete task: %s", resp.Status)
	}

	return nil
}

// HealthCheck 检查 Sentinel 是否健康
func (s *Sentinel) HealthCheck(ctx context.Context) error {
	fullURL := fmt.Sprintf("%s%s/health", s.baseURL, s.pathPrefix)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to health check: %s", resp.Status)
	}

	return nil
}
