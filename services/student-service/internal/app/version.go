package app

// Service metadata
const ServiceName = "student-service"

// Build-time injection variables
// These are set via -ldflags during build:
//
//	go build -ldflags="-X 'student-service/internal/app.Version=1.0.0'"
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)
