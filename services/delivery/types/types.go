package types

type Schedule struct {
	Mask int64 `json:"mask"`
}

type Order struct {
	ID        int64  `json:"id"`
	Status    string `json:"status,omitempty"`
	StartTime string `json:"start_time,omitempty"`
	EndTime   string `json:"end_time,omitempty"`
}

type HTTPError struct {
	Error string `json:"error"`
}
