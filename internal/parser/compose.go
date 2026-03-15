package parser

import (
	"fmt"
	"os"
)

// ParseComposeFile reads and parses a docker-compose.yml file from the given path.
func ParseComposeFile(path string) (*ComposeFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading compose file %q: %w", path, err)
	}
	return ParseComposeBytes(data)
}

// ParseComposeBytes parses docker-compose YAML from raw bytes.
func ParseComposeBytes(data []byte) (*ComposeFile, error) {
	// TODO: implement full parsing with normalization
	_ = data
	return nil, fmt.Errorf("parser not yet implemented")
}
