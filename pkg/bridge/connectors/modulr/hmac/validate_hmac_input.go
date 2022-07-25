package hmac

type ValidationError struct {
	Message string
}

func validateInput(apiKey string, apiSecret string) *ValidationError {
	if apiKey == "" {
		return &ValidationError{"api_key cannot be empty"}
	}

	if apiSecret == "" {
		return &ValidationError{"api_secret cannot be empty"}
	}

	return nil
}
