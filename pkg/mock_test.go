package payment_test

import (
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
	"testing"
)

func runWithMock(t *testing.T, name string, fn func(t *mtest.T)) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run(name, fn)
}
