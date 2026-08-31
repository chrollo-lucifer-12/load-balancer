package rw

import (
	"net/http"
)

type ResponseWrapper struct {
	http.ResponseWriter

	Status      int
	WroteHeader bool
	Bytes       int64
}

func NewResponseWrapper(w http.ResponseWriter) *ResponseWrapper {
	return &ResponseWrapper{
		ResponseWriter: w,
		Status:         http.StatusOK,
	}
}

func (r *ResponseWrapper) WriteHeader(status int) {

	if r.WroteHeader {
		return
	}

	r.Status = status
	r.WroteHeader = true

	r.ResponseWriter.WriteHeader(status)
}

func (r *ResponseWrapper) Write(p []byte) (int, error) {

	if !r.WroteHeader {
		r.WriteHeader(http.StatusOK)
	}

	n, err := r.ResponseWriter.Write(p)
	r.Bytes += int64(n)

	return n, err
}

func (r *ResponseWrapper) Flush() {
	if !r.WroteHeader {
		r.WriteHeader(http.StatusOK)
	}

	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (r *ResponseWrapper) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}
