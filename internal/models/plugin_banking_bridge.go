package models

import (
	"context"
	"encoding/base64"

	"github.com/gibson042/canonicaljson-go"
	"github.com/google/uuid"
)

type BankingBridgePlugin interface {
	// User Creation & Link
	// Create the user on the banking bridge aggregator
	CreateUser(context.Context, CreateUserRequest) (CreateUserResponse, error)
	// Create the link for a specific user to open the authentication flow in
	// the aggregator webview
	CreateUserLink(context.Context, CreateUserLinkRequest) (CreateUserLinkResponse, error)
	// Complete the user link flow by fetching the data from the banking bridge
	// aggregator (mostly the permanent access token for the user)
	CompleteUserLink(context.Context, CompleteUserLinkRequest) (CompleteUserLinkResponse, error)
	// Create the link to update a specific connection of a user when the user
	// if disconnected from the banking bridge aggregator
	UpdateUserLink(context.Context, UpdateUserLinkRequest) (UpdateUserLinkResponse, error)
	// Complete the user link update flow by fetching the data from the banking
	// bridge aggregator (mostly the new permanent access token for the user)
	CompleteUpdateUserLink(context.Context, CompleteUpdateUserLinkRequest) (CompleteUpdateUserLinkResponse, error)

	// User Deletion: Consent & User
	// Delete a specific connection of a user on the aggregator
	DeleteUserConnection(context.Context, DeleteUserConnectionRequest) (DeleteUserConnectionResponse, error)
	// Delete a specific user on the aggregator
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
	PermanentToken *Token
	// ID of the user on the banking bridge
	PSPUserID *string
	// Metadata linked to the user above.
	Metadata map[string]string
}

type CreateUserLinkRequest struct {
	AttemptID           string
	PaymentServiceUser  *PSPPaymentServiceUser
	PSUBankBridge       *PSUBankBridge
	ApplicationName     string
	ClientRedirectURL   *string
	FormanceRedirectURL *string
	CallBackState       string
	WebhookBaseURL      string
}

type CreateUserLinkResponse struct {
	// Link created to forward to the user for authentication
	Link string

	// Optional temporary token linked to the link above.
	// This token is only used to create the link and will be invalidated as
	// soon as the user finishes the authentication flow or the link expires.
	TemporaryLinkToken *Token
}

type UpdateUserLinkRequest struct {
	AttemptID           string
	PaymentServiceUser  *PSPPaymentServiceUser
	PSUBankBridge       *PSUBankBridge
	Connection          *PSUBankBridgeConnection
	ApplicationName     string
	ClientRedirectURL   *string
	FormanceRedirectURL *string
	CallBackState       string
	WebhookBaseURL      string
}

type UpdateUserLinkResponse struct {
	Link string

	// Optional temporary token linked to the link above.
	// This token is only used to create the link and will be invalidated as
	// soon as the user finishes the authentication flow or the link expires.
	TemporaryLinkToken *Token
}

type CompleteUpdateUserLinkRequest struct {
	HTTPCallInformation HTTPCallInformation
	RelatedAttempt      *PSUBankBridgeConnectionAttempt
}

type CompleteUpdateUserLinkResponse struct {
	Success *UserLinkSuccessResponse
	Error   *UserLinkErrorResponse
}

type CompleteUserLinkRequest struct {
	HTTPCallInformation HTTPCallInformation
	RelatedAttempt      *PSUBankBridgeConnectionAttempt
}

type CompleteUserLinkResponse struct {
	Success *UserLinkSuccessResponse
	Error   *UserLinkErrorResponse
}

type UserLinkSuccessResponse struct {
	Connections []PSPPsuBankBridgeConnection
}

type UserLinkErrorResponse struct {
	Error string `json:"error"`
}

type DeleteUserConnectionRequest struct {
	PaymentServiceUser *PSPPaymentServiceUser
	PSUBankBridge      *PSUBankBridge
	Connection         *PSPPsuBankBridgeConnection
}
type DeleteUserConnectionResponse struct{}

type DeleteUserRequest struct {
	PaymentServiceUser *PSPPaymentServiceUser
	PSUBankBridge      *PSUBankBridge
}
type DeleteUserResponse struct{}

type HTTPCallInformation struct {
	QueryValues map[string][]string
	Headers     map[string][]string
	Body        []byte
}

type CallbackState struct {
	// Used for both Tink and Powens, in order to prevent CSRF attacks, we
	// add a random string to the redirect URI's state, and when receiving the
	// callback, we check that the state is the same as the one we sent.
	Randomized string `json:"randomized"`
	// ID of the attempt, used to get the client redirect URL
	AttemptID uuid.UUID `json:"attemptID"`
}

func (pid CallbackState) String() string {
	data, err := canonicaljson.Marshal(pid)
	if err != nil {
		panic(err)
	}

	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(data)
}

func CallbackStateFromString(value string) (CallbackState, error) {
	ret := CallbackState{}
	data, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(value)
	if err != nil {
		return ret, err
	}
	err = canonicaljson.Unmarshal(data, &ret)
	if err != nil {
		return ret, err
	}

	return ret, nil
}

type PSPBankBridgeAccount struct {
	PSPAccount
	BankBridgeUserID       *string
	BankBridgeConnectionID *string
}

type PSPBankBridgePayment struct {
	PSPPayment
	BankBridgeUserID       *string
	BankBridgeConnectionID *string
}
