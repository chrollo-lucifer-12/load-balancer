package rw

import (
	"net/http"
	"sync"
)

type ResponseWrapper struct {
	w      http.ResponseWriter
	buf    []byte
	Status int
}

var responseWrapperPool = sync.Pool{
	New: func() any {
		return &ResponseWrapper{
			Status: http.StatusOK,
			buf:    make([]byte, 0, 4096),
		}
	},
}

func NewResponseWrapper(w http.ResponseWriter) *ResponseWrapper {
	r := responseWrapperPool.Get().(*ResponseWrapper)
	r.w = w

	return r
}

func (r *ResponseWrapper) Header() http.Header {
	return r.w.Header()
}

func (r *ResponseWrapper) Write(p []byte) (int, error) {
	r.buf = append(r.buf, p...)
	return len(p), nil
}

func (r *ResponseWrapper) WriteHeader(status int) {
	r.Status = status
}

func (r *ResponseWrapper) Reset() {
	r.buf = r.buf[:0]
	r.Status = http.StatusOK
}

func (r *ResponseWrapper) Flush() error {

	r.w.WriteHeader(r.Status)

	_, err := r.w.Write(r.buf)
	return err
}

func PutResponseWrapper(r *ResponseWrapper) {
	r.Reset()
	r.w = nil
	responseWrapperPool.Put(r)
}
