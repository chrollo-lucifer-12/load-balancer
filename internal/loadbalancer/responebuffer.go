package loadbalancer

import "net/http"

type ResponseBuffer struct {
	w         http.ResponseWriter
	HeaderMap http.Header
	buf       []byte
	status    int
}

func NewResponseBuffer(w http.ResponseWriter) *ResponseBuffer {
	return &ResponseBuffer{
		w:         w,
		HeaderMap: make(http.Header),
		status:    http.StatusOK,
		buf:       make([]byte, 0, 4096),
	}
}

func (r *ResponseBuffer) Header() http.Header {
	return r.HeaderMap
}

func (r *ResponseBuffer) Write(p []byte) (int, error) {
	r.buf = append(r.buf, p...)
	return len(p), nil
}

func (r *ResponseBuffer) WriteHeader(status int) {
	r.status = status
}

func (r *ResponseBuffer) Reset() {
	clear(r.HeaderMap)

	r.buf = r.buf[:0]
	r.status = http.StatusOK
}

func (r *ResponseBuffer) Flush() error {

	for k, values := range r.HeaderMap {
		r.w.Header()[k] = values
	}

	r.w.WriteHeader(r.status)

	_, err := r.w.Write(r.buf)
	if err != nil {
		return err
	}

	r.Reset()
	return nil
}
