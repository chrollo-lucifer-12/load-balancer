package middleware

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/lb/internal/config"
	"github.com/lb/internal/rw"
)

type Logger struct {
	logger *log.Logger
	level  string
	format string
}

func NewLogger(cfg config.LoggerConfig) *Logger {
	var output *os.File = os.Stdout

	if cfg.Output == "stderr" {
		output = os.Stderr
	}

	return &Logger{
		logger: log.New(output, "", 0),
		level:  cfg.Level,
		format: cfg.Format,
	}
}

func (l *Logger) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		rec, ok := w.(*rw.ResponseWrapper)
		if !ok {
			return
		}

		switch l.format {
		case "json":
			l.logger.Printf(
				`{"level":%q,"method":%q,"path":%q,"status":%d,"duration":%q}`,
				l.level,
				r.Method,
				r.URL.Path,
				rec.Status,
				time.Since(start).String(),
			)

		default:
			l.logger.Printf(
				"level=%s method=%s path=%s status=%d duration=%s",
				l.level,
				r.Method,
				r.URL.Path,
				rec.Status,
				time.Since(start),
			)
		}
	})
}
