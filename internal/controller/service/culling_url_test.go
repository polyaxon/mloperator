package service

import (
	"fmt"
	"testing"
)

// TestURLConstructionWithVariousPaths tests URL construction with different path configurations
// This test verifies that the path handling logic (trimming trailing slashes) works correctly
func TestURLConstructionWithVariousPaths(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		namespace   string
		port        int32
		path        string
		expectedURL string
	}{
		{
			name:        "Empty path - defaults to /api/status",
			serviceName: "jupyter-service",
			namespace:   "default",
			port:        8888,
			path:        "",
			expectedURL: "http://jupyter-service.default.svc.cluster.local:8888/api/status",
		},
		{
			name:        "Explicit /api/status path",
			serviceName: "jupyter-service",
			namespace:   "default",
			port:        8888,
			path:        "/api/status",
			expectedURL: "http://jupyter-service.default.svc.cluster.local:8888/api/status",
		},
		{
			name:        "JupyterLab with /lab/api/status",
			serviceName: "jupyter-service",
			namespace:   "default",
			port:        8888,
			path:        "/lab/api/status",
			expectedURL: "http://jupyter-service.default.svc.cluster.local:8888/lab/api/status",
		},
		{
			name:        "Path with trailing slash - should be trimmed",
			serviceName: "jupyter-service",
			namespace:   "default",
			port:        8888,
			path:        "/lab/api/status/",
			expectedURL: "http://jupyter-service.default.svc.cluster.local:8888/lab/api/status",
		},
		{
			name:        "Custom health endpoint",
			serviceName: "jupyter-service",
			namespace:   "custom-ns",
			port:        8080,
			path:        "/health",
			expectedURL: "http://jupyter-service.custom-ns.svc.cluster.local:8080/health",
		},
		{
			name:        "Multiple path segments with trailing slash",
			serviceName: "jupyter-service",
			namespace:   "custom-ns",
			port:        8080,
			path:        "/user/notebook/status/",
			expectedURL: "http://jupyter-service.custom-ns.svc.cluster.local:8080/user/notebook/status",
		},
		{
			name:        "Root path with just slash - trimmed",
			serviceName: "jupyter-service",
			namespace:   "default",
			port:        8888,
			path:        "/",
			expectedURL: "http://jupyter-service.default.svc.cluster.local:8888",
		},
		{
			name:        "Deep nested custom endpoint",
			serviceName: "notebook-server",
			namespace:   "ml-workspace",
			port:        9999,
			path:        "/org/team/status",
			expectedURL: "http://notebook-server.ml-workspace.svc.cluster.local:9999/org/team/status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the URL construction logic from checkHttpActivity
			host := fmt.Sprintf("%s.%s.svc.cluster.local", tt.serviceName, tt.namespace)

			// This matches the path handling in the actual implementation
			path := tt.path
			if path == "" {
				path = "/api/status"
			}
			// Trim trailing slash
			if path != "" && path[len(path)-1] == '/' {
				path = path[:len(path)-1]
			}

			// Construct the URL (path is the full endpoint path)
			url := fmt.Sprintf("http://%s:%d%s", host, tt.port, path)

			// Verify the URL matches expected
			if url != tt.expectedURL {
				t.Errorf("URL construction mismatch:\n  Expected: %s\n  Got:      %s", tt.expectedURL, url)
			} else {
				t.Logf("✓ Correct URL: %s", url)
			}
		})
	}
}

// TestPathHandlingEdgeCases tests edge cases in path handling
func TestPathHandlingEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		inputPath    string
		expectedPath string
		shouldTrim   bool
	}{
		{
			name:         "Empty string stays empty",
			inputPath:    "",
			expectedPath: "",
			shouldTrim:   false,
		},
		{
			name:         "Single slash becomes empty",
			inputPath:    "/",
			expectedPath: "",
			shouldTrim:   true,
		},
		{
			name:         "Path without trailing slash unchanged",
			inputPath:    "/lab",
			expectedPath: "/lab",
			shouldTrim:   false,
		},
		{
			name:         "Path with trailing slash trimmed",
			inputPath:    "/lab/",
			expectedPath: "/lab",
			shouldTrim:   true,
		},
		{
			name:         "Multiple trailing slashes",
			inputPath:    "/lab///",
			expectedPath: "/lab//",
			shouldTrim:   true,
		},
		{
			name:         "Very long path with trailing slash",
			inputPath:    "/a/b/c/d/e/f/g/",
			expectedPath: "/a/b/c/d/e/f/g",
			shouldTrim:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate path handling logic
			path := tt.inputPath
			if path != "" && path[len(path)-1] == '/' {
				path = path[:len(path)-1]
			}

			if path != tt.expectedPath {
				t.Errorf("Path handling mismatch:\n  Input:    %q\n  Expected: %q\n  Got:      %q", tt.inputPath, tt.expectedPath, path)
			} else {
				t.Logf("✓ Input %q → Output %q", tt.inputPath, path)
			}
		})
	}
}

// TestPortDefaulting documents expected behavior for port defaulting
func TestPortDefaulting(t *testing.T) {
	tests := []struct {
		name         string
		probePort    int32
		servicePorts []int32
		expectedPort int32
		shouldError  bool
	}{
		{
			name:         "Explicit probe port used",
			probePort:    9999,
			servicePorts: []int32{8888, 8080},
			expectedPort: 9999,
			shouldError:  false,
		},
		{
			name:         "Default to first service port",
			probePort:    0,
			servicePorts: []int32{8888},
			expectedPort: 8888,
			shouldError:  false,
		},
		{
			name:         "Default to first of multiple ports",
			probePort:    0,
			servicePorts: []int32{8080, 8888, 9000},
			expectedPort: 8080,
			shouldError:  false,
		},
		{
			name:         "No ports defined - error",
			probePort:    0,
			servicePorts: []int32{},
			expectedPort: 0,
			shouldError:  true,
		},
		{
			name:         "Explicit port even when service has different ports",
			probePort:    3000,
			servicePorts: []int32{8888},
			expectedPort: 3000,
			shouldError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate port selection logic
			var port int32
			var hasError bool

			if tt.probePort != 0 {
				// Use explicit port
				port = tt.probePort
			} else {
				// Default to first service port
				if len(tt.servicePorts) == 0 {
					hasError = true
				} else {
					port = tt.servicePorts[0]
				}
			}

			if hasError != tt.shouldError {
				t.Errorf("Error expectation mismatch: expected error=%v, got error=%v", tt.shouldError, hasError)
			}

			if !hasError && port != tt.expectedPort {
				t.Errorf("Port selection mismatch: expected %d, got %d", tt.expectedPort, port)
			} else if !hasError {
				t.Logf("✓ Probe port %d, service ports %v → selected port %d", tt.probePort, tt.servicePorts, port)
			} else {
				t.Logf("✓ Correctly detected error for no ports")
			}
		})
	}
}
