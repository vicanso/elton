// API example: Recommended + CORS + Timeout + nested groups.
//
//	go run ./examples/api
//	curl -i -H 'Origin: https://app.example' http://127.0.0.1:3000/api/v1/ping
//	curl -i -X OPTIONS -H 'Origin: https://app.example' \
//	  -H 'Access-Control-Request-Method: POST' http://127.0.0.1:3000/api/v1/ping
package main

import (
	"log"
	"time"

	"github.com/vicanso/elton/v2"
	"github.com/vicanso/elton/v2/middleware"
)

func main() {
	e := elton.New()

	// CORS 尽量靠前，预检可短路
	e.Use(middleware.NewCORS(middleware.CORSConfig{
		AllowOrigins:     []string{"https://app.example", "http://localhost:5173"},
		AllowCredentials: true,
		MaxAge:           time.Hour,
		ExposeHeaders:    []string{middleware.HeaderXRequestID},
	}))
	// 全局限时：业务应监听 c.Context()
	e.Use(middleware.NewDefaultTimeout(5 * time.Second))
	e.Use(middleware.Recommended()...)

	api := elton.NewGroup("/api")
	v1 := api.NewGroup("/v1")
	v1.GET("/ping", func(c *elton.Context) error {
		c.Body = map[string]any{
			"pong":      true,
			"requestId": middleware.GetRequestID(c),
		}
		return nil
	})
	e.AddGroup(api)

	log.Println("listening on :3000")
	if err := e.ListenAndServe(":3000"); err != nil {
		panic(err)
	}
}
