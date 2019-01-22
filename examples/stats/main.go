package main

import (
	"fmt"

	"github.com/vicanso/cod"
	"github.com/vicanso/cod/middleware"
)

// curl 'http://127.0.0.1:8001/users/me'
// {"account":"tree.xie"}

func main() {
	d := cod.New()

	d.Use(middleware.NewRecover())

	d.Use(middleware.NewStats(middleware.StatsConfig{
		OnStats: func(info *middleware.StatsInfo, c *cod.Context) {
			// 输出请求的相关信息，状态码，处理时长等
			// &{ 127.0.0.1 GET /users/me /users/me 200 737.705µs 2 22 1}
			fmt.Println(info)
		},
	}))

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
