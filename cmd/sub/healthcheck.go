package sub

import (
	"net/http"
	"time"
)

func RunHealthcheck() int {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}
	resp, err := client.Get("http://localhost:8080/readyz")
	if err != nil {
		return 1
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
