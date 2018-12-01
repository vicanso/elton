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

const (
	// HeaderXForwardedFor x-forwarded-for
	HeaderXForwardedFor = "X-Forwarded-For"
	// HeaderXRealIp x-real-ip
	HeaderXRealIp = "X-Real-Ip"
	// HeaderSetCookie Set-Cookie
	HeaderSetCookie = "Set-Cookie"
)
