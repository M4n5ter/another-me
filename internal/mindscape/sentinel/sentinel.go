package sentinel

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
	sentinelDTO "github.com/m4n5ter/mindscape/sentinel/api/dto"

	"github.com/m4n5ter/another-me/pkg/common"
	. "github.com/m4n5ter/another-me/pkg/option"
)

// Sentinel 客户端
type Sentinel struct {
	client     *common.HTTPClient
	baseURL    string
	pathPrefix string
}

// NewSentinel 创建一个 Sentinel 客户端
func NewSentinel(client Option[http.Client], baseURL string) *Sentinel {
	if client.IsNone() {
		client = Some(http.Client{
			Timeout: 30 * time.Second,
		})
	}

	return &Sentinel{client: common.NewHTTPClient(client.UnwrapAsPtr()), baseURL: baseURL, pathPrefix: "/api/v1/sentinel"}
}

// CreateTask 创建一个任务
func (s *Sentinel) CreateTask(ctx context.Context, createReq sentinelDTO.CreateTaskRequest) (*sentinelDTO.TaskResponse, error) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	var taskResp sentinelDTO.TaskResponse
	if err := s.client.HTTPPost(ctx, fmt.Sprintf("%s%s/tasks", s.baseURL, s.pathPrefix), Some(headers), createReq, &taskResp, "create task"); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	return &taskResp, nil
}

// GetTask 获取一个任务
func (s *Sentinel) GetTask(ctx context.Context, taskID uuid.UUID) (*sentinelDTO.TaskResponse, error) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	var taskResp sentinelDTO.TaskResponse
	if err := s.client.HTTPGet(ctx, fmt.Sprintf("%s%s/tasks/%s", s.baseURL, s.pathPrefix, taskID), Some(headers), None[url.Values](), &taskResp, "get task"); err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	return &taskResp, nil
}

// DeleteTask 删除一个任务
func (s *Sentinel) DeleteTask(ctx context.Context, taskID uuid.UUID) error {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	if err := s.client.HTTPDelete(ctx, fmt.Sprintf("%s%s/tasks/%s", s.baseURL, s.pathPrefix, taskID), Some(headers), nil, "delete task"); err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	return nil
}

// HealthCheck 检查 Sentinel 是否健康
func (s *Sentinel) HealthCheck(ctx context.Context) error {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	if err := s.client.HTTPGet(ctx, fmt.Sprintf("%s%s/health", s.baseURL, s.pathPrefix), Some(headers), None[url.Values](), nil, "health check"); err != nil {
		return fmt.Errorf("failed to health check: %w", err)
	}

	return nil
}
