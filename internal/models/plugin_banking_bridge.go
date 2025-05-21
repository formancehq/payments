package models

import (
	"context"
)

type BankingBridgePlugin interface {
	// User Creation & Link
	CreateUserLink(context.Context, CreateUserLinkRequest) (CreateUserLinkResponse, error)

	// User Deletion: Consent & User
	DeleteUserConnection(context.Context, DeleteUserConnectionRequest) (DeleteUserConnectionResponse, error)
	DeleteUser(context.Context, DeleteUserRequest) (DeleteUserResponse, error)
}

type CreateUserLinkRequest struct {
	PaymentServiceUser *PSPPaymentServiceUser
	RedirectURI        string
	WebhookBaseURL     string
}

type CreateUserLinkResponse struct {
	// Link created to forward to the user for authentication
	Link string

	// Temporary token linked to the link above.
	// This token is only used to create the link and will be invalidated as
	// soon as the user finishes the authentication flow or the link expires.
	TemporaryLinkToken *Token
	// Permanent token: this is the token that will be used to fetch the
	// banking transactions.
	// Some Banking Bridges connectors have the permanent token created at link
	// generation, so we need to pass it back to the core if it's the case.
	// Other connectors have it when the user finished the authentication flow,
	// so this is optional and will be added later on thanks to webhooks.
	PermanentToken *Token
}

type DeleteUserConnectionRequest struct {
	PaymentServiceUser *PSPPaymentServiceUser
	BankBridgeConsent  *PSUBankBridgeConsent
	Connection         *PSUBankBridgeConnection
}
type DeleteUserConnectionResponse struct{}

type DeleteUserRequest struct {
	PaymentServiceUser *PSPPaymentServiceUser
	BankBridgeConsent  *PSUBankBridgeConsent
}
type DeleteUserResponse struct{}
