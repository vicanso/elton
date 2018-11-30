package cod

import "net/http"

var (
	methods = []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodHead,
		http.MethodConnect,
		http.MethodOptions,
		http.MethodTrace,
	}
)
