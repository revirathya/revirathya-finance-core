package version

import (
	"encoding/json"
	"net/http"
)

type response struct {
	Version string `json:"version"`
}

// Handler serves the binary's build version as {"version": "..."}
func Handler(version string) http.HandlerFunc {
	body, _ := json.Marshal(response{Version: version})
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}
}
