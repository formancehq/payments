package payment_test

import (
	"context"
	payment "github.com/numary/payment/pkg"
	"github.com/ory/dockertest"
	_ "github.com/ory/dockertest"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"log"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		panic(err)
	}

	// pulls an image, creates a container based on it and runs it
	resource, err := pool.Run("mongo", "4.4", []string{})
	if err != nil {
		panic(err)
	}

	uri := "mongodb://localhost:" + resource.GetPort("27017/tcp")
	client, err := mongo.NewClient(options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}

	ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(2*time.Second))
	err = client.Connect(ctx)
	if err != nil {
		panic(err)
	}

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	if err := pool.Retry(func() error {
		ctx, _ := context.WithDeadline(context.Background(), time.Now().Add(2*time.Second))
		return client.Ping(ctx, readpref.Primary())
	}); err != nil {
		panic("could not connect to database, last error: " + err.Error())
	}

	err = mtest.Setup(mtest.NewSetupOptions().SetURI(uri))
	if err != nil {
		panic(err)
	}

	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

func runWithMock(t *testing.T, fn func(t *mtest.T)) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Default))
	defer mt.Close()

	mt.RunOpts("Default", mtest.NewOptions().CollectionName(payment.PaymentCollection), fn)
}
