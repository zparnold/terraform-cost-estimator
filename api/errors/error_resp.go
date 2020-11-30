package errors

type APIErrorResp struct {
	StatusCode int    `json:"status_code"`
	Error      string `json:"error"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion"`
	TraceId    string `json:"trace_id"`
}
