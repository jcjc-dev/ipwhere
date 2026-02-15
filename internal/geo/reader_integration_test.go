//go:build integration

package geo

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

// Integration tests that run against real MMDB database files.
// These tests verify that the geo reader works correctly with actual databases.
//
// Run with: go test -tags=integration ./...
//
// Environment variables:
//   - CITY_DB_PATH: Path to city MMDB file (default: looks in data/ directory)
//   - ASN_DB_PATH: Path to ASN MMDB file (default: looks in data/ directory)

func findTestDatabases() (cityPath, asnPath string, err error) {
	// Check environment variables first
	cityPath = os.Getenv("CITY_DB_PATH")
	asnPath = os.Getenv("ASN_DB_PATH")

	if cityPath != "" && asnPath != "" {
		return cityPath, asnPath, nil
	}

	// Look in common locations
	searchPaths := []string{
		".",
		"data",
		"../../data",
		"../../../data",
	}

	cityNames := []string{"dbip-city-lite.mmdb", "GeoLite2-City.mmdb"}
	asnNames := []string{"dbip-asn-lite.mmdb", "GeoLite2-ASN.mmdb"}

	for _, base := range searchPaths {
		if cityPath == "" {
			for _, name := range cityNames {
				p := filepath.Join(base, name)
				if _, err := os.Stat(p); err == nil {
					cityPath = p
					break
				}
			}
		}
		if asnPath == "" {
			for _, name := range asnNames {
				p := filepath.Join(base, name)
				if _, err := os.Stat(p); err == nil {
					asnPath = p
					break
				}
			}
		}
	}

	if cityPath == "" || asnPath == "" {
		return "", "", os.ErrNotExist
	}

	return cityPath, asnPath, nil
}

func setupIntegrationReader(t *testing.T) *Reader {
	t.Helper()

	cityPath, asnPath, err := findTestDatabases()
	if err != nil {
		t.Skipf("Skipping integration test: database files not found. Set CITY_DB_PATH and ASN_DB_PATH environment variables.")
	}

	reader, err := NewReader(cityPath, asnPath, false)
	if err != nil {
		t.Fatalf("Failed to create reader: %v", err)
	}

	t.Cleanup(func() {
		reader.Close()
	})

	return reader
}

// TestKnownPublicIPs tests lookups against well-known public IP addresses
// to verify the database returns sensible results.
func TestKnownPublicIPs(t *testing.T) {
	reader := setupIntegrationReader(t)

	tests := []struct {
		name            string
		ip              string
		expectedCountry string // Expected country (may vary slightly with DB updates)
		expectASN       bool   // Whether we expect ASN data
	}{
		{
			name:            "Google DNS (8.8.8.8)",
			ip:              "8.8.8.8",
			expectedCountry: "United States",
			expectASN:       true,
		},
		{
			name:            "Cloudflare DNS (1.1.1.1)",
			ip:              "1.1.1.1",
			expectedCountry: "Australia", // Cloudflare's APNIC research address
			expectASN:       true,
		},
		{
			name:            "Google DNS IPv6",
			ip:              "2001:4860:4860::8888",
			expectedCountry: "United States",
			expectASN:       true,
		},
		{
			name:            "Quad9 DNS (9.9.9.9)",
			ip:              "9.9.9.9",
			expectedCountry: "United States",
			expectASN:       true,
		},
		{
			name:            "OpenDNS (208.67.222.222)",
			ip:              "208.67.222.222",
			expectedCountry: "United States",
			expectASN:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("Invalid test IP: %s", tt.ip)
			}

			info, err := reader.Lookup(ip)
			if err != nil {
				t.Fatalf("Lookup failed: %v", err)
			}

			// Verify basic fields are populated
			if info.IP != tt.ip {
				t.Errorf("IP mismatch: got %s, want %s", info.IP, tt.ip)
			}

			if info.Attribution == "" {
				t.Error("Attribution should not be empty")
			}

			// Verify country (this is the most stable field)
			if info.Country != tt.expectedCountry {
				t.Logf("Warning: Country mismatch for %s: got %q, expected %q (may be due to DB update)", tt.ip, info.Country, tt.expectedCountry)
			}

			// Verify we got a country code
			if info.ISOCode == "" {
				t.Error("ISO code should not be empty for public IP")
			}

			// Verify ASN data if expected
			if tt.expectASN {
				if info.ASN == nil {
					t.Error("Expected ASN data but got nil")
				}
				if info.Organization == "" {
					t.Error("Expected organization name but got empty string")
				}
			}
		})
	}
}

// TestIPv4AndIPv6Parity verifies that IPv4 and IPv6 lookups work consistently
func TestIPv4AndIPv6Parity(t *testing.T) {
	reader := setupIntegrationReader(t)

	// Google's IPv4 and IPv6 DNS should return similar results
	ipv4 := net.ParseIP("8.8.8.8")
	ipv6 := net.ParseIP("2001:4860:4860::8888")

	info4, err := reader.Lookup(ipv4)
	if err != nil {
		t.Fatalf("IPv4 lookup failed: %v", err)
	}

	info6, err := reader.Lookup(ipv6)
	if err != nil {
		t.Fatalf("IPv6 lookup failed: %v", err)
	}

	// Both should be in the same country (Google infrastructure)
	if info4.Country != info6.Country {
		t.Logf("Note: IPv4 (%s) and IPv6 (%s) returned different countries: %s vs %s",
			ipv4, ipv6, info4.Country, info6.Country)
	}

	// Both should have ASN data
	if info4.ASN == nil || info6.ASN == nil {
		t.Error("Both IPv4 and IPv6 should have ASN data")
	}
}

// TestPrivateIPRanges tests that private IPs return empty/minimal results
func TestPrivateIPRanges(t *testing.T) {
	reader := setupIntegrationReader(t)

	privateIPs := []string{
		"10.0.0.1",
		"172.16.0.1",
		"192.168.1.1",
		"127.0.0.1",
		"fc00::1", // IPv6 private
	}

	for _, ipStr := range privateIPs {
		t.Run(ipStr, func(t *testing.T) {
			ip := net.ParseIP(ipStr)
			info, err := reader.Lookup(ip)
			if err != nil {
				t.Fatalf("Lookup failed for private IP %s: %v", ipStr, err)
			}

			// Private IPs should still return a valid response with IP field
			if info.IP != ipStr {
				t.Errorf("IP mismatch: got %s, want %s", info.IP, ipStr)
			}

			// Private IPs typically have no country data
			if info.Country != "" {
				t.Logf("Note: Private IP %s returned country: %s", ipStr, info.Country)
			}
		})
	}
}

// TestDatabaseConsistency verifies the database returns consistent results
func TestDatabaseConsistency(t *testing.T) {
	reader := setupIntegrationReader(t)

	ip := net.ParseIP("8.8.8.8")

	// Run multiple lookups and verify consistency
	var firstResult *IPInfo
	for i := 0; i < 10; i++ {
		info, err := reader.Lookup(ip)
		if err != nil {
			t.Fatalf("Lookup %d failed: %v", i, err)
		}

		if firstResult == nil {
			firstResult = info
			continue
		}

		// Verify results are identical
		if info.Country != firstResult.Country {
			t.Errorf("Inconsistent country on lookup %d: got %s, want %s", i, info.Country, firstResult.Country)
		}
		if info.City != firstResult.City {
			t.Errorf("Inconsistent city on lookup %d: got %s, want %s", i, info.City, firstResult.City)
		}
	}
}

// TestAllFieldsPopulated verifies that a lookup returns all expected fields
func TestAllFieldsPopulated(t *testing.T) {
	reader := setupIntegrationReader(t)

	// Use Google DNS as it should have complete data
	ip := net.ParseIP("8.8.8.8")
	info, err := reader.Lookup(ip)
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}

	// Check required fields
	if info.IP == "" {
		t.Error("IP should not be empty")
	}
	if info.Attribution == "" {
		t.Error("Attribution should not be empty")
	}

	// Check geo fields (should be populated for public IPs)
	fields := map[string]interface{}{
		"Country":      info.Country,
		"ISOCode":      info.ISOCode,
		"ASN":          info.ASN,
		"Organization": info.Organization,
	}

	for name, value := range fields {
		if value == nil || value == "" {
			t.Errorf("Field %s should be populated for Google DNS", name)
		}
	}

	// Log all fields for visibility
	t.Logf("Lookup result for %s:", ip)
	t.Logf("  Country: %s (%s)", info.Country, info.ISOCode)
	t.Logf("  City: %s, Region: %s", info.City, info.Region)
	if info.Latitude != nil && info.Longitude != nil {
		t.Logf("  Location: %.4f, %.4f", *info.Latitude, *info.Longitude)
	}
	t.Logf("  Timezone: %s", info.Timezone)
	if info.ASN != nil {
		t.Logf("  ASN: AS%d (%s)", *info.ASN, info.Organization)
	}
}
