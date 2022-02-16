package paymentclient_test

import (
	"context"
	payment "github.com/numary/payment/pkg"
	"github.com/numary/payments-cloud/pkg/paymentclient"
	"github.com/ory/dockertest"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"log"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"
)

var (
	Service       payment.Service
	Configuration *paymentclient.Configuration
	Server        *httptest.Server
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
	defer pool.Purge(resource)

	err = mtest.Setup(mtest.NewSetupOptions().SetURI(uri))
	if err != nil {
		panic(err)
	}

	Service = payment.NewDefaultService(client.Database("testing"))
	router := payment.NewMux(Service)
	Server = httptest.NewServer(router)
	defer Server.Close()

	url, _ := url.Parse(Server.URL)
	Configuration = paymentclient.NewConfiguration()
	Configuration.Host = url.Host
	Configuration.Scheme = "http"
	Configuration.Debug = true

	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

func TestCreatePayment(t *testing.T) {
	client := paymentclient.NewAPIClient(Configuration)
	_, _, err := client.PaymentsApi.
		CreatePayment(context.Background(), "foo").
		PaymentData(paymentclient.PaymentData{}).
		Execute()
	assert.NoError(t, err)
}

func TestUpdatePayment(t *testing.T) {

	orgId := uuid.New()

	payment, err := Service.CreatePayment(context.Background(), orgId, payment.Data{})
	assert.NoError(t, err)

	client := paymentclient.NewAPIClient(Configuration)
	_, err = client.PaymentsApi.
		UpdatePayment(context.Background(), orgId, payment.ID).
		PaymentData(paymentclient.PaymentData{}).
		Execute()
	assert.NoError(t, err)
}

func TestUpsertPayment(t *testing.T) {
	orgId := uuid.New()

	client := paymentclient.NewAPIClient(Configuration)
	_, err := client.PaymentsApi.
		UpdatePayment(context.Background(), orgId, "1").
		PaymentData(paymentclient.PaymentData{}).
		Upsert(true).
		Execute()
	assert.NoError(t, err)
}

func TestListPayments(t *testing.T) {
	orgId := uuid.New()

	_, err := Service.CreatePayment(context.Background(), orgId, payment.Data{})
	assert.NoError(t, err)

	_, err = Service.CreatePayment(context.Background(), orgId, payment.Data{})
	assert.NoError(t, err)

	client := paymentclient.NewAPIClient(Configuration)
	payments, _, err := client.PaymentsApi.
		ListPayments(context.Background(), orgId).
		Execute()
	assert.NoError(t, err)
	assert.Len(t, payments, 2)
}
