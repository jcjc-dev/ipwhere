package api

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Shoyu-Dev/ipwhere/internal/geo"
	"github.com/go-chi/chi/v5"
)

// MockGeoReader implements geo.ReaderInterface for testing
type MockGeoReader struct{}

func (m *MockGeoReader) Lookup(ip net.IP) (*geo.IPInfo, error) {
	lat := 37.4056
	lon := -122.0775
	asn := uint(15169)

	return &geo.IPInfo{
		IP:           ip.String(),
		Country:      "United States",
		ISOCode:      "US",
		InEU:         false,
		City:         "Mountain View",
		Region:       "California",
		Latitude:     &lat,
		Longitude:    &lon,
		Timezone:     "America/Los_Angeles",
		ASN:          &asn,
		Organization: "Google LLC",
		Attribution:  geo.Attribution,
	}, nil
}

func (m *MockGeoReader) Close() error {
	return nil
}

func (m *MockGeoReader) OnlineFeaturesEnabled() bool {
	return false
}

func setupTestRouter() *chi.Mux {
	r := chi.NewRouter()
	handler := NewHandler(&MockGeoReader{}, false)
	handler.SetupRoutes(r)
	return r
}

func TestIPLookup(t *testing.T) {
	r := setupTestRouter()

	tests := []struct {
		name           string
		url            string
		expectedStatus int
		checkResponse  func(*testing.T, map[string]interface{})
	}{
		{
			name:           "basic lookup",
			url:            "/api/ip?ip=8.8.8.8",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if resp["ip"] != "8.8.8.8" {
					t.Errorf("expected ip to be 8.8.8.8, got %v", resp["ip"])
				}
				if resp["country"] != "United States" {
					t.Errorf("expected country to be United States, got %v", resp["country"])
				}
				if resp["attribution"] == nil {
					t.Error("expected attribution to be present")
				}
			},
		},
		{
			name:           "filter single field",
			url:            "/api/ip?ip=8.8.8.8&return=country",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if resp["country"] != "United States" {
					t.Errorf("expected country to be United States, got %v", resp["country"])
				}
				if resp["city"] != nil {
					t.Error("expected city to not be present when not requested")
				}
				// ip and attribution should always be present
				if resp["ip"] == nil {
					t.Error("expected ip to be present")
				}
				if resp["attribution"] == nil {
					t.Error("expected attribution to be present")
				}
			},
		},
		{
			name:           "filter multiple fields",
			url:            "/api/ip?ip=8.8.8.8&return=country&return=city&return=asn",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if resp["country"] != "United States" {
					t.Errorf("expected country to be United States, got %v", resp["country"])
				}
				if resp["city"] != "Mountain View" {
					t.Errorf("expected city to be Mountain View, got %v", resp["city"])
				}
				if resp["asn"] == nil {
					t.Error("expected asn to be present")
				}
				if resp["region"] != nil {
					t.Error("expected region to not be present when not requested")
				}
			},
		},
		{
			name:           "invalid IP",
			url:            "/api/ip?ip=invalid",
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, resp map[string]interface{}) {
				if resp["error"] == nil {
					t.Error("expected error message")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

func TestHealth(t *testing.T) {
	r := setupTestRouter()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("expected status to be ok, got %s", resp["status"])
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expected   string
	}{
		{
			name:       "from RemoteAddr",
			remoteAddr: "192.168.1.1:12345",
			expected:   "192.168.1.1",
		},
		{
			name: "from X-Forwarded-For",
			headers: map[string]string{
				"X-Forwarded-For": "10.0.0.1, 192.168.1.1",
			},
			remoteAddr: "127.0.0.1:12345",
			expected:   "10.0.0.1",
		},
		{
			name: "from X-Real-IP",
			headers: map[string]string{
				"X-Real-IP": "10.0.0.2",
			},
			remoteAddr: "127.0.0.1:12345",
			expected:   "10.0.0.2",
		},
		{
			name: "X-Forwarded-For takes precedence",
			headers: map[string]string{
				"X-Forwarded-For": "10.0.0.1",
				"X-Real-IP":       "10.0.0.2",
			},
			remoteAddr: "127.0.0.1:12345",
			expected:   "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr

			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			result := getClientIP(req)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
