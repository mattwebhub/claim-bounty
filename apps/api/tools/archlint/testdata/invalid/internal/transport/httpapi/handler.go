package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
)

func invalidHandler(r *http.Request) {
	_ = context.Background()
	_ = json.NewDecoder(r.Body)
}
