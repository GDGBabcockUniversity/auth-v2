package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUnmatchedRouteReturnsNotFound(t *testing.T) {
	app := (API{}).App()
	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/", nil))
	if err != nil {
		t.Fatalf("send request: %v", err)
	}

	defer func(){
		if err = response.Body.Close(); err != nil{
			return
		}
	}()

	if response.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", response.StatusCode, http.StatusNotFound)
	}
}
