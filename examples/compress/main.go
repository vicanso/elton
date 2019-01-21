package main

import (
	"math/rand"

	"github.com/vicanso/cod"
	"github.com/vicanso/cod/middleware"
)

// curl -H "Accept-Encoding:gzip" -v 'http://127.0.0.1:8001/ids'

func main() {
	d := cod.New()

	d.Use(middleware.NewRecover())

	d.Use(middleware.NewFresh(middleware.FreshConfig{}))
	d.Use(middleware.NewETag(middleware.ETagConfig{}))

	d.Use(middleware.NewCompress(middleware.CompressConfig{}))

	d.Use(middleware.NewResponder(middleware.ResponderConfig{}))

	d.GET("/users/me", func(c *cod.Context) (err error) {
		c.Body = &struct {
			Account string `json:"account"`
		}{
			"tree.xie",
		}
		return
	})

	d.GET("/ids", func(c *cod.Context) (err error) {
		max := 1000
		arr := make([]int, max)
		for index := 0; index < max; index++ {
			arr[index] = rand.Int()
		}
		c.Body = map[string]interface{}{
			"ids": arr,
		}
		return
	})

	d.ListenAndServe(":8001")
}
