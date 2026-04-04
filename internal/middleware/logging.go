package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
)

type contextKey string

const QueriesContextKey contextKey = "sql_queries"

type responseWriter struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		queries := make([]string, 0)
		ctx := context.WithValue(r.Context(), QueriesContextKey, &queries)
		r = r.WithContext(ctx)

		rw := &responseWriter{
			ResponseWriter: w,
			body:           &bytes.Buffer{},
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		routeCtx := chi.RouteContext(r.Context())
		pathParams := make(map[string]string)
		if routeCtx != nil && routeCtx.URLParams.Keys != nil {
			for i, key := range routeCtx.URLParams.Keys {
				if i < len(routeCtx.URLParams.Values) {
					pathParams[key] = routeCtx.URLParams.Values[i]
				}
			}
		}

		queryParams := r.URL.Query()

		var respBody any
		if rw.body.Len() > 0 {
			if err := json.Unmarshal(rw.body.Bytes(), &respBody); err != nil {
				// Fallback to string if not JSON
				respBody = rw.body.String()
			}
		}

		logger.Info("HTTP Request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rw.statusCode),
			slog.String("duration", duration.String()),
			slog.Any("query_params", queryParams),
			slog.Any("path_params", pathParams),
			slog.Any("sql_queries", queries),
			slog.Any("response", respBody),
		)
	})
}
