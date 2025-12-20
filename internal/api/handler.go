package api

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"github.com/Shoyu-Dev/ipwhere/internal/geo"
	"github.com/go-chi/chi/v5"
)

// Handler holds the dependencies for HTTP handlers
type Handler struct {
	geoReader            geo.ReaderInterface
	enableOnlineFeatures bool
}

// NewHandler creates a new Handler with the given geo reader
func NewHandler(geoReader geo.ReaderInterface, enableOnlineFeatures bool) *Handler {
	return &Handler{
		geoReader:            geoReader,
		enableOnlineFeatures: enableOnlineFeatures,
	}
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error       string `json:"error"`
	Attribution string `json:"attribution"`
}

// writeJSON writes a JSON response with the given status code
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{
		Error:       message,
		Attribution: geo.Attribution,
	})
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if parsedIP := net.ParseIP(ip); parsedIP != nil {
				return ip
			}
		}
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		if parsedIP := net.ParseIP(xri); parsedIP != nil {
			return xri
		}
	}

	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// IPLookup godoc
// @Summary      Look up IP geolocation
// @Description  Returns geolocation data for the requesting IP or specified IP address
// @Tags         lookup
// @Accept       json
// @Produce      json
// @Param        ip      query     string  false  "IP address to lookup (defaults to client IP)"
// @Param        return  query     []string  false  "Fields to return (can be repeated). Valid values: hostname, country, iso_code, in_eu, city, region, latitude, longitude, timezone, asn, organization"
// @Success      200     {object}  geo.IPInfo
// @Failure      400     {object}  ErrorResponse
// @Failure      500     {object}  ErrorResponse
// @Router       /api/ip [get]
func (h *Handler) IPLookup(w http.ResponseWriter, r *http.Request) {
	// Get IP to lookup
	ipStr := r.URL.Query().Get("ip")
	if ipStr == "" {
		ipStr = getClientIP(r)
	}

	// Parse IP
	ip := net.ParseIP(ipStr)
	if ip == nil {
		writeError(w, http.StatusBadRequest, "Invalid IP address")
		return
	}

	// Lookup IP
	info, err := h.geoReader.Lookup(ip)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to lookup IP")
		return
	}

	// Check for field filtering
	returnFields := r.URL.Query()["return"]
	if len(returnFields) > 0 {
		// Normalize field names (convert to lowercase)
		normalizedFields := make([]string, len(returnFields))
		for i, f := range returnFields {
			normalizedFields[i] = strings.ToLower(f)
		}
		filtered := info.FilterFields(normalizedFields)
		writeJSON(w, http.StatusOK, filtered)
		return
	}

	writeJSON(w, http.StatusOK, info)
}

// Health godoc
// @Summary      Health check
// @Description  Returns health status of the service
// @Tags         health
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /health [get]
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// Debug godoc
// @Summary      Debug request headers
// @Description  Returns all request headers and connection info for debugging
// @Tags         debug
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Router       /api/debug [get]
func (h *Handler) Debug(w http.ResponseWriter, r *http.Request) {
	headers := make(map[string]string)
	for name, values := range r.Header {
		headers[name] = values[0]
	}

	debugInfo := map[string]interface{}{
		"remoteAddr":       r.RemoteAddr,
		"host":             r.Host,
		"requestURI":       r.RequestURI,
		"headers":          headers,
		"xForwardedFor":    r.Header.Get("X-Forwarded-For"),
		"xRealIP":          r.Header.Get("X-Real-IP"),
		"xAzureClientIP":   r.Header.Get("X-Azure-ClientIP"),
		"xOriginalHost":    r.Header.Get("X-Original-Host"),
		"xClientIP":        r.Header.Get("X-Client-IP"),
		"cfConnectingIP":   r.Header.Get("CF-Connecting-IP"),
		"trueClientIP":     r.Header.Get("True-Client-IP"),
		"forwardedHeader":  r.Header.Get("Forwarded"),
		"detectedClientIP": getClientIP(r),
	}

	writeJSON(w, http.StatusOK, debugInfo)
}

// FeaturesResponse represents the feature flags response
type FeaturesResponse struct {
	OnlineFeatures bool `json:"onlineFeatures"`
}

// Features godoc
// @Summary      Get feature flags
// @Description  Returns the enabled feature flags for the service
// @Tags         features
// @Produce      json
// @Success      200  {object}  FeaturesResponse
// @Router       /api/features [get]
func (h *Handler) Features(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, FeaturesResponse{
		OnlineFeatures: h.enableOnlineFeatures,
	})
}

// SetupRoutes configures the API routes
func (h *Handler) SetupRoutes(r chi.Router) {
	r.Get("/api/ip", h.IPLookup)
	r.Get("/api/debug", h.Debug)
	r.Get("/api/features", h.Features)
	r.Get("/health", h.Health)
}
