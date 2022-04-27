package config

import "time"

type Config struct {
	NetworkName   string
	BucketName    string
	ContainerName string
	MinioUser     string
	MinioPwd      string
	MaxPayloadSize int64
	RefreshInterval time.Duration
}
