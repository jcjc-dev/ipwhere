package geo

import (
	"fmt"
	"net"
	"sync"

	"github.com/oschwald/geoip2-golang"
)

// IPInfo represents the complete IP geolocation information
type IPInfo struct {
	IP           string   `json:"ip"`
	Hostname     string   `json:"hostname,omitempty"`
	Country      string   `json:"country,omitempty"`
	ISOCode      string   `json:"iso_code,omitempty"`
	InEU         bool     `json:"in_eu,omitempty"`
	City         string   `json:"city,omitempty"`
	Region       string   `json:"region,omitempty"`
	Latitude     *float64 `json:"latitude,omitempty"`
	Longitude    *float64 `json:"longitude,omitempty"`
	Timezone     string   `json:"timezone,omitempty"`
	ASN          *uint    `json:"asn,omitempty"`
	Organization string   `json:"organization,omitempty"`
	Attribution  string   `json:"attribution"`
}

// Attribution is the required attribution for DB-IP
const Attribution = "IP Geolocation by DB-IP (https://db-ip.com)"

// Reader wraps the geoip2 database readers
type Reader struct {
	cityDB               *geoip2.Reader
	asnDB                *geoip2.Reader
	enableOnlineFeatures bool
	mu                   sync.RWMutex
}

// ReaderInterface defines the interface for geo lookups (useful for testing)
type ReaderInterface interface {
	Lookup(ip net.IP) (*IPInfo, error)
	Close() error
	OnlineFeaturesEnabled() bool
}

// NewReader creates a new geo reader from the given database paths
func NewReader(cityDBPath, asnDBPath string, enableOnlineFeatures bool) (*Reader, error) {
	cityDB, err := geoip2.Open(cityDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open city database: %w", err)
	}

	asnDB, err := geoip2.Open(asnDBPath)
	if err != nil {
		cityDB.Close()
		return nil, fmt.Errorf("failed to open ASN database: %w", err)
	}

	return &Reader{
		cityDB:               cityDB,
		asnDB:                asnDB,
		enableOnlineFeatures: enableOnlineFeatures,
	}, nil
}

// Lookup retrieves IP information for the given IP address
func (r *Reader) Lookup(ip net.IP) (*IPInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	info := &IPInfo{
		IP:          ip.String(),
		Attribution: Attribution,
	}

	// City/Country lookup
	city, err := r.cityDB.City(ip)
	if err == nil {
		info.Country = city.Country.Names["en"]
		info.ISOCode = city.Country.IsoCode
		info.InEU = city.Country.IsInEuropeanUnion
		info.City = city.City.Names["en"]

		if len(city.Subdivisions) > 0 {
			info.Region = city.Subdivisions[0].Names["en"]
		}

		if city.Location.Latitude != 0 || city.Location.Longitude != 0 {
			lat := city.Location.Latitude
			lon := city.Location.Longitude
			info.Latitude = &lat
			info.Longitude = &lon
		}

		info.Timezone = city.Location.TimeZone
	}

	// ASN lookup
	asn, err := r.asnDB.ASN(ip)
	if err == nil {
		asnNum := asn.AutonomousSystemNumber
		info.ASN = &asnNum
		info.Organization = asn.AutonomousSystemOrganization
	}

	// Reverse DNS lookup for hostname (only if online features are enabled)
	if r.enableOnlineFeatures {
		names, err := net.LookupAddr(ip.String())
		if err == nil && len(names) > 0 {
			// Remove trailing dot from hostname if present
			hostname := names[0]
			if len(hostname) > 0 && hostname[len(hostname)-1] == '.' {
				hostname = hostname[:len(hostname)-1]
			}
			info.Hostname = hostname
		}
	}

	return info, nil
}

// Close closes both database readers
func (r *Reader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errs []error
	if r.cityDB != nil {
		if err := r.cityDB.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if r.asnDB != nil {
		if err := r.asnDB.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing databases: %v", errs)
	}
	return nil
}

// OnlineFeaturesEnabled returns whether online features are enabled
func (r *Reader) OnlineFeaturesEnabled() bool {
	return r.enableOnlineFeatures
}

// FilterFields returns a new IPInfo with only the requested fields
func (info *IPInfo) FilterFields(fields []string) map[string]interface{} {
	result := make(map[string]interface{})
	result["ip"] = info.IP
	result["attribution"] = info.Attribution

	fieldMap := map[string]interface{}{
		"hostname":     info.Hostname,
		"country":      info.Country,
		"iso_code":     info.ISOCode,
		"in_eu":        info.InEU,
		"city":         info.City,
		"region":       info.Region,
		"latitude":     info.Latitude,
		"longitude":    info.Longitude,
		"timezone":     info.Timezone,
		"asn":          info.ASN,
		"organization": info.Organization,
	}

	for _, field := range fields {
		if val, ok := fieldMap[field]; ok {
			result[field] = val
		}
	}

	return result
}
