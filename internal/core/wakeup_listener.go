package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
)

// webhookWakeupListener 实现WakeupListener接口，通过HTTP Webhook接收唤醒信号
type webhookWakeupListener struct {
	port    int
	path    string
	logger  *slog.Logger
	handler func(WakeupEvent) error
	server  *http.Server
	running bool
	mutex   sync.RWMutex
}

// Start 启动监听器
func (w *webhookWakeupListener) Start(ctx context.Context) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.running {
		return fmt.Errorf("监听器已在运行")
	}

	mux := http.NewServeMux()
	mux.HandleFunc(w.path, w.handleWebhook)

	w.server = &http.Server{
		Addr:    ":" + strconv.Itoa(w.port),
		Handler: mux,
	}

	go func() {
		err := w.server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			w.logger.Error("HTTP服务器启动失败", "error", err)
		}
	}()

	w.running = true
	w.logger.Info("Webhook监听器启动成功", "port", w.port, "path", w.path)
	return nil
}

// Stop 停止监听器
func (w *webhookWakeupListener) Stop(ctx context.Context) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if !w.running {
		return nil
	}

	if w.server != nil {
		err := w.server.Shutdown(ctx)
		if err != nil {
			return fmt.Errorf("停止HTTP服务器失败: %w", err)
		}
	}

	w.running = false
	w.logger.Info("Webhook监听器已停止")
	return nil
}

// SetHandler 设置唤醒事件处理器
func (w *webhookWakeupListener) SetHandler(handler func(WakeupEvent) error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.handler = handler
}

// IsListening 检查是否正在监听
func (w *webhookWakeupListener) IsListening() bool {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	return w.running
}

// GetListenAddress 获取监听地址
func (w *webhookWakeupListener) GetListenAddress() string {
	return fmt.Sprintf("http://localhost:%d%s", w.port, w.path)
}

// handleWebhook 处理Webhook请求
func (w *webhookWakeupListener) handleWebhook(rw http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(rw, "只支持POST方法", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求体
	var wakeupEvent WakeupEvent
	err := json.NewDecoder(req.Body).Decode(&wakeupEvent)
	if err != nil {
		w.logger.Error("解析唤醒事件失败", "error", err)
		http.Error(rw, "无效的请求体", http.StatusBadRequest)
		return
	}

	w.logger.Info("收到唤醒事件", "task_id", wakeupEvent.MonitoringTaskID, "reason", wakeupEvent.Reason)

	// 调用处理器
	w.mutex.RLock()
	handler := w.handler
	w.mutex.RUnlock()

	if handler != nil {
		err = handler(wakeupEvent)
		if err != nil {
			w.logger.Error("处理唤醒事件失败", "error", err)
			http.Error(rw, "处理事件失败", http.StatusInternalServerError)
			return
		}
	}

	// 返回成功响应
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte(`{"status":"ok"}`))
} 