package types

type HTTPError struct {
	Error string `json:"error"`
}

type Notification struct {
	Message string `json:"message"`
}
