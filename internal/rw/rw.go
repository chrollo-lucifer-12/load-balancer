package rw

import "net/http"

type ResponseWrapper struct {
	w         http.ResponseWriter
	HeaderMap http.Header
	buf       []byte
	Status    int
}

func NewResponseWrapper(w http.ResponseWriter) *ResponseWrapper {
	return &ResponseWrapper{
		w:         w,
		HeaderMap: make(http.Header),
		Status:    http.StatusOK,
		buf:       make([]byte, 0, 4096),
	}
}

func (r *ResponseWrapper) Header() http.Header {
	return r.HeaderMap
}

func (r *ResponseWrapper) Write(p []byte) (int, error) {
	r.buf = append(r.buf, p...)
	return len(p), nil
}

func (r *ResponseWrapper) WriteHeader(status int) {
	r.Status = status
}

func (r *ResponseWrapper) Reset() {
	clear(r.HeaderMap)

	r.buf = r.buf[:0]
	r.Status = http.StatusOK
}

func (r *ResponseWrapper) Flush() error {

	for k, values := range r.HeaderMap {
		r.w.Header()[k] = values
	}

	r.w.WriteHeader(r.Status)

	_, err := r.w.Write(r.buf)
	if err != nil {
		return err
	}

	r.Reset()
	return nil
}
