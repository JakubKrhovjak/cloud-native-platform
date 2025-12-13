package testnats

import (
	"context"
	"sync"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	sharedContainer *NATSContainer
	sharedOnce      sync.Once
	sharedMu        sync.Mutex
)

type NATSContainer struct {
	Container testcontainers.Container
	URL       string
}

// SetupSharedNATS creates a single NATS container shared across all tests
// This is the RECOMMENDED approach for all tests - much faster than isolated containers
//
// IMPORTANT: Tests using shared container CANNOT run in parallel!
//
// Usage:
//
//	func TestMyService(t *testing.T) {
//	    natsContainer := testnats.SetupSharedNATS(t)
//	    defer natsContainer.Cleanup(t)  // ← Only call once at top level
//
//	    t.Run("Test1", func(t *testing.T) {
//	        // ... test using natsContainer.URL
//	    })
//	}
func SetupSharedNATS(t *testing.T) *NATSContainer {
	t.Helper()

	sharedOnce.Do(func() {
		ctx := context.Background()

		req := testcontainers.ContainerRequest{
			Image:        "nats:2.10-alpine",
			ExposedPorts: []string{"4222/tcp"},
			WaitingFor:   wait.ForListeningPort("4222/tcp"),
		}

		natsContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		require.NoError(t, err)

		host, err := natsContainer.Host(ctx)
		require.NoError(t, err)

		port, err := natsContainer.MappedPort(ctx, "4222")
		require.NoError(t, err)

		natsURL := "nats://" + host + ":" + port.Port()

		sharedContainer = &NATSContainer{
			Container: natsContainer,
			URL:       natsURL,
		}
	})

	return sharedContainer
}

func (nc *NATSContainer) Cleanup(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	if nc.Container != nil {
		if err := nc.Container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %s", err)
		}
	}
}

func (nc *NATSContainer) Connect(t *testing.T) *nats.Conn {
	t.Helper()

	conn, err := nats.Connect(nc.URL)
	require.NoError(t, err)

	t.Cleanup(func() { conn.Close() })

	return conn
}
