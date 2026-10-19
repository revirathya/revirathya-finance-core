package sub

import (
	"fmt"
	"net/http"
	"time"
)

func RunHealthcheck(port string) int {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}
	resp, err := client.Get(fmt.Sprintf("http://localhost%s/readyz", port))
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
