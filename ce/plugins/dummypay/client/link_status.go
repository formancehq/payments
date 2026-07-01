package client

const (
	StatusQueryParamID = "status"
	UserIDQueryParamID = "user_id"
)

type LinkStatus string

const (
	LinkStatusSuccess LinkStatus = "success"
	LinkStatusError   LinkStatus = "error"
)
