package models

import (
	"context"
)

type BankingBridgePlugin interface {
	// User Creation & Link
	CreateUser(context.Context, CreateUserRequest) (CreateUserResponse, error)
	CreateUserLink(context.Context, CreateUserLinkRequest) (CreateUserLinkResponse, error)
	CompleteUserLink(context.Context, CompleteUserLinkRequest) (CompleteUserLinkResponse, error)

	// User Deletion: Consent & User
	DeleteUserConnection(context.Context, DeleteUserConnectionRequest) (DeleteUserConnectionResponse, error)
	DeleteUser(context.Context, DeleteUserRequest) (DeleteUserResponse, error)
}

type CreateUserRequest struct {
	PaymentServiceUser *PSPPaymentServiceUser
}

type CreateUserResponse struct {
	// Optional permanent token linked to the user above.
	// Some Banking Bridges connectors have the permanent token created at user
	// creation, so we need to pass it back to the core if it's the case.
	// Other connectors have it when the user finished the authentication flow,
	// so this is optional and will be added later on thanks to webhooks.
	PermanentToken *string
	// Metadata linked to the user above.
	Metadata map[string]string
}

type CreateUserLinkRequest struct {
	PaymentServiceUser  *PSPPaymentServiceUser
	PSUBankBridge       *PSUBankBridge
	ClientRedirectURI   *string
	FormanceRedirectURI *string
	CallBackState       string
	WebhookBaseURL      string
}

type CallbackState struct {
	// Used for both Tink and Powens, in order to prevent CSRF attacks, we
	// add a random string to the redirect URI's state, and when receiving the
	// callback, we check that the state is the same as the one we sent.
	Randomized string `json:"randomized"`
}

type CreateUserLinkResponse struct {
	// Link created to forward to the user for authentication
	Link string

	// Optional temporary token linked to the link above.
	// This token is only used to create the link and will be invalidated as
	// soon as the user finishes the authentication flow or the link expires.
	TemporaryLinkToken *Token
}

type CompleteUserLinkRequest struct {
	QueryValues map[string][]string
	Headers     map[string][]string
	Body        []byte

	RelatedAttempt *PSUBankBridgeConnectionAttempt
}

type CompleteUserLinkResponse struct {
	Success *CompleteUserLinkSuccessResponse
	Error   *CompleteUserLinkErrorResponse
}

type CompleteUserLinkSuccessResponse struct {
	Connections []PSUBankBridgeConnection
}

type CompleteUserLinkErrorResponse struct {
	Error string `json:"error"`
}

type DeleteUserConnectionRequest struct {
	PaymentServiceUser *PSPPaymentServiceUser
	PSUBankBridge      *PSUBankBridge
	Connection         *PSUBankBridgeConnection
}
type DeleteUserConnectionResponse struct{}

type DeleteUserRequest struct {
	PaymentServiceUser *PSPPaymentServiceUser
	PSUBankBridge      *PSUBankBridge
}
type DeleteUserResponse struct{}
