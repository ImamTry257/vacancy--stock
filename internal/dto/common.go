package dto

type ErrorResponse struct {
	Message string `json:"message"`
}

type HealthResponse struct {
	Status string `json:"status"`
}
