package middleware

import (
	"log"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
)

type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

var reqID atomic.Uint64

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := reqID.Add(1)
		requestID := strconv.FormatUint(id, 10)
		w.Header().Set("X-Request-Id", requestID)

		start := time.Now()
		rec := &responseRecorder{ResponseWriter: w}

		next.ServeHTTP(rec, r)

		log.Printf("request_id=%s method=%s path=%s status=%d bytes=%d duration=%s",
			requestID,
			r.Method,
			r.URL.Path,
			rec.status,
			rec.bytes,
			time.Since(start),
		)
	})
}
