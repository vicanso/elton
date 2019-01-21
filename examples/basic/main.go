package main

import (
	"github.com/vicanso/cod"
	"github.com/vicanso/cod/middleware"
)

// curl 'http://127.0.0.1:8001/users/me'
// {"account":"tree.xie"}

func main() {
	d := cod.New()

	d.Use(middleware.NewRecover())

	d.Use(middleware.NewFresh(middleware.FreshConfig{}))
	d.Use(middleware.NewETag(middleware.ETagConfig{}))

	d.Use(middleware.NewResponder(middleware.ResponderConfig{}))

	d.GET("/users/me", func(c *cod.Context) (err error) {
		c.Body = &struct {
			Account string `json:"account"`
		}{
			"tree.xie",
		}
		return
	})

	d.ListenAndServe(":8001")
}
