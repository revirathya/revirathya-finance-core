package middleware

import "net/http"

// StatusRecorder captures the status code a handler sends
type StatusRecorder struct {
	http.ResponseWriter
	Status int
}

func NewStatusRecorder(w http.ResponseWriter) *StatusRecorder {
	return &StatusRecorder{
		ResponseWriter: w,
	}
}

// WriteHeader keeps the first status, like net/http does
func (r *StatusRecorder) WriteHeader(code int) {
	if r.Status == 0 {
		r.Status = code
	}
	r.ResponseWriter.WriteHeader(code)
}

// Write without WriteHeader means 200, as in net/http
func (r *StatusRecorder) Write(b []byte) (int, error) {
	if r.Status == 0 {
		r.Status = http.StatusOK
	}
	return r.ResponseWriter.Write(b)
}

func (r *StatusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

// Code returns the status sent, or 200 if the handler wrote nothing
func (r *StatusRecorder) Code() int {
	if r.Status == 0 {
		return http.StatusOK
	}
	return r.Status
}
