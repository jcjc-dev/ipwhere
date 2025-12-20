package geo

import (
	"net"
	"testing"
)

// MockReader is a mock implementation of ReaderInterface for testing
type MockReader struct {
	MockLookup func(ip net.IP) (*IPInfo, error)
}

func (m *MockReader) Lookup(ip net.IP) (*IPInfo, error) {
	if m.MockLookup != nil {
		return m.MockLookup(ip)
	}
	return &IPInfo{
		IP:           ip.String(),
		Country:      "United States",
		ISOCode:      "US",
		InEU:         false,
		City:         "Mountain View",
		Region:       "California",
		Latitude:     floatPtr(37.4056),
		Longitude:    floatPtr(-122.0775),
		Timezone:     "America/Los_Angeles",
		ASN:          uintPtr(15169),
		Organization: "Google LLC",
		Attribution:  Attribution,
	}, nil
}

func (m *MockReader) Close() error {
	return nil
}

func (m *MockReader) OnlineFeaturesEnabled() bool {
	return false
}

func floatPtr(f float64) *float64 {
	return &f
}

func uintPtr(u uint) *uint {
	return &u
}

func TestIPInfoFilterFields(t *testing.T) {
	info := &IPInfo{
		IP:           "8.8.8.8",
		Country:      "United States",
		ISOCode:      "US",
		InEU:         false,
		City:         "Mountain View",
		Region:       "California",
		Latitude:     floatPtr(37.4056),
		Longitude:    floatPtr(-122.0775),
		Timezone:     "America/Los_Angeles",
		ASN:          uintPtr(15169),
		Organization: "Google LLC",
		Attribution:  Attribution,
	}

	tests := []struct {
		name     string
		fields   []string
		expected []string
	}{
		{
			name:     "single field",
			fields:   []string{"country"},
			expected: []string{"ip", "attribution", "country"},
		},
		{
			name:     "multiple fields",
			fields:   []string{"country", "city", "asn"},
			expected: []string{"ip", "attribution", "country", "city", "asn"},
		},
		{
			name:     "case insensitive",
			fields:   []string{"country", "iso_code"},
			expected: []string{"ip", "attribution", "country", "iso_code"},
		},
		{
			name:     "empty fields",
			fields:   []string{},
			expected: []string{"ip", "attribution"},
		},
		{
			name:     "invalid field ignored",
			fields:   []string{"invalid", "country"},
			expected: []string{"ip", "attribution", "country"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := info.FilterFields(tt.fields)

			// Check that expected fields are present
			for _, field := range tt.expected {
				if _, ok := result[field]; !ok {
					t.Errorf("expected field %s to be present", field)
				}
			}

			// Check that unexpected fields are not present
			for key := range result {
				found := false
				for _, exp := range tt.expected {
					if key == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("unexpected field %s in result", key)
				}
			}
		})
	}
}

func TestAttribution(t *testing.T) {
	if Attribution != "IP Geolocation by DB-IP (https://db-ip.com)" {
		t.Errorf("Attribution constant is incorrect: %s", Attribution)
	}
}
