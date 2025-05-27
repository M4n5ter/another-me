package common

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	json "github.com/json-iterator/go"

	. "github.com/m4n5ter/another-me/pkg/option"
)

// HTTPClient 是 HTTP 客户端
type HTTPClient struct {
	client *http.Client
}

// NewHTTPClient 创建一个 HTTP 客户端
func NewHTTPClient(client *http.Client) *HTTPClient {
	return &HTTPClient{
		client: client,
	}
}

// HTTPPost 发送 POST 请求
//
//	endpoint: 请求的 URL
//	headers: 请求的 headers
//	requestBody: 请求的 body
//	responseBody: 响应的 body
//	operation: 操作的名称
//
// 返回错误信息
//
//nolint:dupl // 与 HTTPPut 的实现类似
func (h *HTTPClient) HTTPPost(ctx context.Context, endpoint string, headers Option[http.Header], requestBody, responseBody any, operation string) error {
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal %s request: %w", operation, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if headers.IsSome() {
		for key, values := range headers.Unwrap() {
			for _, value := range values {
				req.Header.Set(key, value)
			}
		}
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to %s: %s", operation, resp.Status)
	}

	if responseBody != nil {
		if err := json.NewDecoder(resp.Body).Decode(responseBody); err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// HTTPPostWithForm 发送 POST 请求
//
//	endpoint: 请求的 URL
//	headers: 请求的 headers
//	requestForm: 请求的表单数据
//	responseBody: 响应的 body
//	operation: 操作的名称
//
// 返回错误信息
func (h *HTTPClient) HTTPPostWithForm(ctx context.Context, endpoint string, headers Option[http.Header], requestForm url.Values, responseBody any, operation string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(requestForm.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if headers.IsSome() {
		for key, values := range headers.Unwrap() {
			for _, value := range values {
				req.Header.Set(key, value)
			}
		}
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to %s: %s", operation, resp.Status)
	}

	if responseBody != nil {
		if err := json.NewDecoder(resp.Body).Decode(responseBody); err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// HTTPGet 发送 GET 请求
//
//	endpoint: 请求的 URL
//	headers: 请求的 headers
//	queryParams: 请求的查询参数
//	responseBody: 响应的 body
//	operation: 操作的名称
//
// 返回错误信息
func (h *HTTPClient) HTTPGet(ctx context.Context, endpoint string, headers Option[http.Header], queryParams Option[url.Values], responseBody any, operation string) error {
	fullURL := endpoint
	if queryParams.IsSome() {
		fullURL += "?" + queryParams.Unwrap().Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if headers.IsSome() {
		for key, values := range headers.Unwrap() {
			for _, value := range values {
				req.Header.Set(key, value)
			}
		}
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to %s: %s", operation, resp.Status)
	}

	if responseBody != nil {
		if err := json.NewDecoder(resp.Body).Decode(responseBody); err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// HTTPDelete 发送 DELETE 请求
//
//	endpoint: 请求的 URL
//	headers: 请求的 headers
//	responseBody: 响应的 body
//	operation: 操作的名称
//
// 返回错误信息
//
//nolint:dupl // 与 HTTPGet 的实现类似
func (h *HTTPClient) HTTPDelete(ctx context.Context, endpoint string, headers Option[http.Header], responseBody any, operation string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if headers.IsSome() {
		for key, values := range headers.Unwrap() {
			for _, value := range values {
				req.Header.Set(key, value)
			}
		}
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to %s: %s", operation, resp.Status)
	}

	if responseBody != nil {
		if err := json.NewDecoder(resp.Body).Decode(responseBody); err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// HTTPPut 发送 PUT 请求
//
//	endpoint: 请求的 URL
//	headers: 请求的 headers
//	requestBody: 请求的 body
//	responseBody: 响应的 body
//	operation: 操作的名称
//
// 返回错误信息
//
//nolint:dupl // 与 HTTPPost 的实现类似
func (h *HTTPClient) HTTPPut(ctx context.Context, endpoint string, headers Option[http.Header], requestBody, responseBody any, operation string) error {
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal %s request: %w", operation, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if headers.IsSome() {
		for key, values := range headers.Unwrap() {
			for _, value := range values {
				req.Header.Set(key, value)
			}
		}
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to %s: %s", operation, resp.Status)
	}

	if responseBody != nil {
		if err := json.NewDecoder(resp.Body).Decode(responseBody); err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// HTTPPatch 发送 PATCH 请求
//
//	endpoint: 请求的 URL
//	headers: 请求的 headers
//	requestBody: 请求的 body
//	responseBody: 响应的 body
//	operation: 操作的名称
//
// 返回错误信息
//
//nolint:dupl // 与 HTTPPut 的实现类似
func (h *HTTPClient) HTTPPatch(ctx context.Context, endpoint string, headers Option[http.Header], requestBody, responseBody any, operation string) error {
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal %s request: %w", operation, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if headers.IsSome() {
		for key, values := range headers.Unwrap() {
			for _, value := range values {
				req.Header.Set(key, value)
			}
		}
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to %s: %s", operation, resp.Status)
	}

	if responseBody != nil {
		if err := json.NewDecoder(resp.Body).Decode(responseBody); err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// HTTPOptions 发送 OPTIONS 请求
//
//	endpoint: 请求的 URL
//	headers: 请求的 headers
//	responseBody: 响应的 body
//	operation: 操作的名称
//
// 返回错误信息
//
//nolint:dupl // 与 HTTPGet 的实现类似
func (h *HTTPClient) HTTPOptions(ctx context.Context, endpoint string, headers Option[http.Header], responseBody any, operation string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodOptions, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if headers.IsSome() {
		for key, values := range headers.Unwrap() {
			for _, value := range values {
				req.Header.Set(key, value)
			}
		}
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to %s: %s", operation, resp.Status)
	}

	if responseBody != nil {
		if err := json.NewDecoder(resp.Body).Decode(responseBody); err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}
