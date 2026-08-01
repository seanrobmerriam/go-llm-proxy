package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

func main() {
	upstreamURL := getenv("LLM_UPSTREAM_URL", "https://api.openai.com")
	upstreamAPIKey := os.Getenv("LLM_UPSTREAM_API_KEY")

	if upstreamAPIKey == "" {
		log.Fatal("LLM_UPSTREAM_API_KEY is required")
	}

	target, err := url.Parse(upstreamURL)
	if err != nil {
		log.Fatalf("invalid LLM_UPSTREAM_URL: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	originalDirector := proxy.Director

	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		// The client never needs access to the provider API key.
		req.Header.Set("Authorization", "Bearer "+upstreamAPIKey)

		// Preserve the original Host value for auditing if useful.
		req.Header.Set("X-Forwarded-Host", req.Host)

		// The upstream provider expects its own hostname.
		req.Host = target.Host
	}

	proxy.ModifyResponse = func(res *http.Response) error {
		// Do not expose unnecessary provider-specific headers.
		res.Header.Del("Set-Cookie")
		res.Header.Del("Server")

		// Prevent buffering by intermediaries when streaming SSE.
		if strings.Contains(res.Header.Get("Content-Type"), "text/event-stream") {
			res.Header.Set("Cache-Control", "no-cache")
			res.Header.Set("X-Accel-Buffering", "no")
		}

		return nil
	}

	proxy.ErrorHandler = func(
		writer http.ResponseWriter,
		request *http.Request,
		err error,
	) {
		log.Printf("upstream request failed: %v", err)

		http.Error(
			writer,
			`{"error":{"message":"upstream LLM request failed"}}`,
			http.StatusBadGateway,
		)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"status":"ok"}`))
	})

	// Forwards:
	//
	// /v1/chat/completions
	// /v1/models
	// /v1/embeddings
	// etc.
	mux.Handle("/v1/", authenticateClient(proxy))

	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 10 * 1e9,
	}

	log.Printf("LLM proxy listening on %s", server.Addr)
	log.Printf("forwarding requests to %s", target)

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func authenticateClient(next http.Handler) http.Handler {
	proxyAPIKey := os.Getenv("PROXY_API_KEY")

	return http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		if proxyAPIKey != "" {
			provided := strings.TrimPrefix(
				request.Header.Get("Authorization"),
				"Bearer ",
			)

			if provided != proxyAPIKey {
				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(http.StatusUnauthorized)
				_, _ = writer.Write(
					[]byte(`{"error":{"message":"unauthorized"}}`),
				)
				return
			}
		}

		next.ServeHTTP(writer, request)
	})
}

func getenv(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}

	return value
}
