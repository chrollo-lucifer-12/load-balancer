package static

import "net/http"

func NewStaticServer(dir, path string) http.Handler {
	return http.StripPrefix(
		path,
		http.FileServer(http.Dir(dir)),
	)
}
