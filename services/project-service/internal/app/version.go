package app

// Service metadata
const ServiceName = "project-service"

// Build-time injection variables
// These are set via -ldflags during build:
//
//	go build -ldflags="-X 'project-service/internal/app.Version=1.0.0'"
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)
