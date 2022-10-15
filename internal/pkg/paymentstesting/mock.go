package paymentstesting

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/ory/dockertest"
	"github.com/ory/dockertest/docker"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func RunWithMock(t *testing.T, fn func(t *mtest.T)) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		panic(err)
	}

	// pulls an image, creates a container based on it and runs it
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "bitnami/mongodb",
		Cmd: []string{
			"--storageEngine",
			"inMemory",
		},
		Tag: "4.4",
		Env: []string{
			"MONGODB_REPLICA_SET_MODE=primary",
			"MONGODB_REPLICA_SET_KEY=abcdef",
			"MONGODB_ADVERTISED_HOSTNAME=localhost",
			"MONGODB_ROOT_PASSWORD=root",
		},
	}, func(config *docker.HostConfig) {
		config.NetworkMode = "host"
	})
	if err != nil {
		panic(err)
	}

	uri := "mongodb://localhost:" + resource.GetPort("27017/tcp")
	client, err := mongo.NewClient(options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(2*time.Second))
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		panic(err)
	}

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	if err := pool.Retry(func() error {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(2*time.Second))
		defer cancel()
		return client.Ping(ctx, readpref.Primary())
	}); err != nil {
		panic("could not connect to database, last error: " + err.Error())
	}

	err = mtest.Setup(mtest.NewSetupOptions().SetURI(uri))
	if err != nil {
		panic(err)
	}

	mtest.New(t).Run("Default", fn)

	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}
}
