package cod

import (
	"testing"
)

func TestError(t *testing.T) {
	err := &HTTPError{
		StatusCode: 400,
		Category:   "custom",
		Message:    "error",
	}
	if err.Error() != "category=custom, status=400, message=error" {
		t.Fatalf("get error fail")
	}
}
