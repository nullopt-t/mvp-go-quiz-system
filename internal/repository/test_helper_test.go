package repository_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// setupTestDB connects to MongoDB for integration testing.
// Returns an isolated test database and a cleanup function.
// If MongoDB is not reachable, the calling test is cleanly skipped.
func setupTestDB(t *testing.T) (*mongo.Database, func()) {
	t.Helper()

	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Skipf("Skipping MongoDB integration test: failed to connect: %v", err)
		return nil, func() {}
	}

	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		t.Skipf("Skipping MongoDB integration test: ping failed (%s): %v", uri, err)
		return nil, func() {}
	}

	dbName := fmt.Sprintf("quiz_test_%d", time.Now().UnixNano())
	db := client.Database(dbName)

	cleanup := func() {
		dropCtx, dropCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer dropCancel()
		_ = db.Drop(dropCtx)
		_ = client.Disconnect(dropCtx)
	}

	return db, cleanup
}
