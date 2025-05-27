package memory

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	memoryDTO "github.com/m4n5ter/mindscape/memory/api/dto"

	"github.com/m4n5ter/another-me/pkg/common"
	. "github.com/m4n5ter/another-me/pkg/option"
)

// Memory 是 mindscape 的 memory 客户端
type Memory struct {
	client     *common.HTTPClient
	baseURL    string
	pathPrefix string
}

// NewMemory 创建一个 mindscape 的 memory 客户端
func NewMemory(client Option[http.Client], baseURL string) *Memory {
	if client.IsNone() {
		client = Some(http.Client{
			Timeout: 30 * time.Second,
		})
	}

	return &Memory{client: common.NewHTTPClient(client.UnwrapAsPtr()), baseURL: baseURL, pathPrefix: "/api/v1/memory"}
}

// StoreMemoryFragment 存储记忆片段
func (m *Memory) StoreMemoryFragment(ctx context.Context, storeReq memoryDTO.StoreMemoryRequest) (*memoryDTO.MemoryFragmentResponse, error) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	var fragmentResp memoryDTO.MemoryFragmentResponse
	if err := m.client.HTTPPost(ctx, fmt.Sprintf("%s%s/fragments", m.baseURL, m.pathPrefix), Some(headers), storeReq, &fragmentResp, "store memory fragment"); err != nil {
		return nil, fmt.Errorf("failed to store memory fragment: %w", err)
	}
	return &fragmentResp, nil
}

// GetMemoryFragment 获取记忆片段
func (m *Memory) GetMemoryFragment(ctx context.Context, fragmentID string) (*memoryDTO.MemoryFragmentResponse, error) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	var fragmentResp memoryDTO.MemoryFragmentResponse
	if err := m.client.HTTPGet(ctx, fmt.Sprintf("%s%s/fragments/%s", m.baseURL, m.pathPrefix, fragmentID), Some(headers), None[url.Values](), &fragmentResp, "get memory fragment"); err != nil {
		return nil, fmt.Errorf("failed to get memory fragment: %w", err)
	}
	return &fragmentResp, nil
}

// ListMemoryFragments 获取记忆片段列表
func (m *Memory) ListMemoryFragments(ctx context.Context, listReq memoryDTO.ListMemoryFragmentsRequest) (*memoryDTO.ListMemoryFragmentsResponse, error) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	var listResp memoryDTO.ListMemoryFragmentsResponse
	if err := m.client.HTTPGet(ctx, fmt.Sprintf("%s%s/fragments", m.baseURL, m.pathPrefix), Some(headers), None[url.Values](), &listResp, "list memory fragments"); err != nil {
		return nil, fmt.Errorf("failed to list memory fragments: %w", err)
	}
	return &listResp, nil
}

// RecallMemory 回忆记忆
func (m *Memory) RecallMemory(ctx context.Context, recallReq memoryDTO.RecallMemoriesRequest) (*memoryDTO.RecallMemoriesResponse, error) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	var recallResp memoryDTO.RecallMemoriesResponse
	if err := m.client.HTTPPost(ctx, fmt.Sprintf("%s%s/recall", m.baseURL, m.pathPrefix), Some(headers), recallReq, &recallResp, "recall memories"); err != nil {
		return nil, fmt.Errorf("failed to recall memories: %w", err)
	}
	return &recallResp, nil
}

// GetUserProfile 获取用户画像
func (m *Memory) GetUserProfile(ctx context.Context, userID string) (*memoryDTO.UserProfileResponse, error) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	var userProfileResp memoryDTO.UserProfileResponse
	if err := m.client.HTTPGet(ctx, fmt.Sprintf("%s%s/users/%s/profile", m.baseURL, m.pathPrefix, userID), Some(headers), None[url.Values](), &userProfileResp, "get user profile"); err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}
	return &userProfileResp, nil
}

// UpdateUserProfile 更新用户画像
func (m *Memory) UpdateUserProfile(ctx context.Context, userID string, updateReq memoryDTO.UpdateUserProfileRequest) (*memoryDTO.UserProfileResponse, error) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	var userProfileResp memoryDTO.UserProfileResponse
	if err := m.client.HTTPPut(ctx, fmt.Sprintf("%s%s/users/%s/profile", m.baseURL, m.pathPrefix, userID), Some(headers), updateReq, &userProfileResp, "update user profile"); err != nil {
		return nil, fmt.Errorf("failed to update user profile: %w", err)
	}
	return &userProfileResp, nil
}
