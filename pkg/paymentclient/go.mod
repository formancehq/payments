module github.com/numary/payments-cloud/pkg/paymentclient

go 1.13

replace github.com/numary/payment => ../../

require (
	github.com/numary/payment v0.0.0-00010101000000-000000000000
	github.com/ory/dockertest v3.3.5+incompatible
	github.com/pborman/uuid v1.2.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/stretchr/testify v1.7.0
	go.mongodb.org/mongo-driver v1.8.3
	golang.org/x/oauth2 v0.0.0-20211104180415-d3ed0bb246c8
)
