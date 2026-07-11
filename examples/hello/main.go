// Hello example: Recommended middleware stack + simple JSON routes.
//
//	go run ./examples/hello
//	curl -i http://127.0.0.1:3000/
package main

import (
	"log"

	"github.com/vicanso/elton/v2"
	"github.com/vicanso/elton/v2/middleware"
)

func main() {
	e := elton.New()
	e.Use(middleware.Recommended()...)

	e.GET("/", func(c *elton.Context) error {
		c.Body = map[string]string{
			"message":   "Hello, World!",
			"requestId": middleware.GetRequestID(c),
		}
		return nil
	})

	e.GET("/books/{id}", func(c *elton.Context) error {
		c.Body = map[string]string{"id": c.Param("id")}
		return nil
	})

	log.Println("listening on :3000")
	if err := e.ListenAndServe(":3000"); err != nil {
		panic(err)
	}
}
