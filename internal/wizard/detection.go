package wizard

import "strings"

// ServiceType represents the detected type of a compose service.
type ServiceType string

const (
	ServiceTypeWebServer ServiceType = "web-server"
	ServiceTypeDatabase  ServiceType = "database"
	ServiceTypeCache     ServiceType = "cache"
	ServiceTypeAppServer ServiceType = "app-server"
	ServiceTypeGeneric   ServiceType = "generic"
)

// DetectServiceType determines the service type from its Docker image name.
func DetectServiceType(image string) ServiceType {
	img := strings.ToLower(image)
	// Strip tag
	if idx := strings.LastIndex(img, ":"); idx != -1 {
		img = img[:idx]
	}
	// Strip registry prefix
	if idx := strings.LastIndex(img, "/"); idx != -1 {
		img = img[idx+1:]
	}

	webServers := []string{"nginx", "httpd", "apache", "traefik", "caddy", "haproxy", "envoy"}
	for _, ws := range webServers {
		if strings.Contains(img, ws) {
			return ServiceTypeWebServer
		}
	}

	databases := []string{"postgres", "mysql", "mariadb", "mongo", "cockroach", "cassandra", "elasticsearch", "opensearch", "couchdb", "neo4j", "influxdb"}
	for _, db := range databases {
		if strings.Contains(img, db) {
			return ServiceTypeDatabase
		}
	}

	caches := []string{"redis", "memcached", "valkey"}
	for _, c := range caches {
		if strings.Contains(img, c) {
			return ServiceTypeCache
		}
	}

	appServers := []string{"node", "python", "golang", "java", "ruby", "php", "dotnet", "flask", "django", "express", "spring", "laravel", "rails"}
	for _, as := range appServers {
		if strings.Contains(img, as) {
			return ServiceTypeAppServer
		}
	}

	return ServiceTypeGeneric
}

// ShouldSuggestIngress returns true if the service type typically needs an Ingress.
func ShouldSuggestIngress(st ServiceType) bool {
	return st == ServiceTypeWebServer
}

// ShouldSuggestStatefulSet returns true if the service type typically needs a StatefulSet.
func ShouldSuggestStatefulSet(st ServiceType) bool {
	return st == ServiceTypeDatabase
}

// ShouldSuggestHPA returns true if the service type can be horizontally scaled.
func ShouldSuggestHPA(st ServiceType) bool {
	return st == ServiceTypeWebServer || st == ServiceTypeAppServer
}

// ShouldSuggestPDB returns true if the service type typically needs a PDB.
func ShouldSuggestPDB(st ServiceType) bool {
	return st == ServiceTypeDatabase || st == ServiceTypeCache
}
