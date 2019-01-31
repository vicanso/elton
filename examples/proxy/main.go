package main

import (
	"net/url"

	"github.com/vicanso/cod"
	"github.com/vicanso/cod/middleware"
)

// curl 'http://127.0.0.1:8001/baidu/'
// baidu html

func main() {
	d := cod.New()

	d.Use(middleware.NewRecover())

	d.Use(middleware.NewDefaultFresh())
	d.Use(middleware.NewDefaultETag())

	d.Use(middleware.NewDefaultResponder())

	d.GET("/users/me", func(c *cod.Context) (err error) {
		c.Body = &struct {
			Account string `json:"account"`
		}{
			"tree.xie",
		}
		return
	})

	target, _ := url.Parse("https://www.baidu.com")
	baiduPorxy := middleware.NewProxy(middleware.ProxyConfig{
		// 转发的target
		Target: target,
		Host:   "www.baidu.com",
		// 转发时重写url
		Rewrites: []string{
			"/baidu/*:/$1",
		},
	})
	// 以/baidu前缀的GET请求的处理
	d.GET("/baidu/*path", baiduPorxy)

	d.ListenAndServe(":8001")
}
