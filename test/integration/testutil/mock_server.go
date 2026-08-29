// Copyright(c) 2026 The Rainway AI Gateway (壬远AI网关) Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testutil

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// MockConfigServer 模拟 ai-gateway-api 的 InnerAPI 配置导出接口
type MockConfigServer struct {
	server  *httptest.Server
	mu      sync.RWMutex
	version string
	data    map[string]interface{}
}

// NewMockConfigServer 创建模拟配置服务器
func NewMockConfigServer(t *testing.T, module string) *MockConfigServer {
	m := &MockConfigServer{
		version: "20260101120000",
		data: map[string]interface{}{
			"Version": "20260101120000",
			"Rules":   []string{"rule1"},
		},
	}

	mux := http.NewServeMux()
	mux.HandleFunc(fmt.Sprintf("/configs/%s", module), m.handleConfig)
	m.server = httptest.NewServer(mux)

	t.Cleanup(func() {
		m.server.Close()
	})

	return m
}

func (m *MockConfigServer) handleConfig(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	version := m.version
	data := m.data
	m.mu.RUnlock()

	clientVersion := r.URL.Query().Get("version")
	if clientVersion == version {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ErrNum":200,"Data":null}`))
		return
	}

	resp := map[string]interface{}{
		"ErrNum": 200,
		"Data":   data,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// SetVersion 更新配置版本，触发 conf-agent 拉取新配置
func (m *MockConfigServer) SetVersion(version string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.version = version
	m.data = map[string]interface{}{
		"Version": version,
		"Rules":   []string{"rule_" + version},
	}
}

// URL 返回模拟服务器地址
func (m *MockConfigServer) URL() string {
	return m.server.URL
}

// MockBFEServer 模拟 BFE 的 reload 接口
type MockBFEServer struct {
	server     *httptest.Server
	mu         sync.RWMutex
	reloads    []string
	shouldFail bool
}

// NewMockBFEServer 创建模拟 BFE 服务器
func NewMockBFEServer(t *testing.T, module string) *MockBFEServer {
	m := &MockBFEServer{}

	mux := http.NewServeMux()
	mux.HandleFunc(fmt.Sprintf("/reload/%s", module), m.handleReload)
	m.server = httptest.NewServer(mux)

	t.Cleanup(func() {
		m.server.Close()
	})

	return m
}

func (m *MockBFEServer) handleReload(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")

	m.mu.Lock()
	m.reloads = append(m.reloads, path)
	fail := m.shouldFail
	m.mu.Unlock()

	if fail {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"reload failed"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"error":null}`))
}

// ReloadCount 返回收到的 reload 请求数量
func (m *MockBFEServer) ReloadCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.reloads)
}

// LastReloadPath 返回最近一次 reload 请求的 path
func (m *MockBFEServer) LastReloadPath() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.reloads) == 0 {
		return ""
	}
	return m.reloads[len(m.reloads)-1]
}

// SetFail 设置下一次 reload 是否失败
func (m *MockBFEServer) SetFail(fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.shouldFail = fail
}

// URL 返回模拟服务器地址
func (m *MockBFEServer) URL() string {
	return m.server.URL
}
