package backend

import "net/http"

//Middleware to check for
func MaxBytesMiddleware(maxBytes int64) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			//Just in case if someone provides different content length
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			if r.ContentLength > maxBytes {
				http.Error(w, "File too large", http.StatusBadRequest)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
