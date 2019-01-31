package main

import (
	"encoding/json"
	"errors"
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
		OnStats: func(stats *middleware.StatsInfo, _ *cod.Context) {
			fmt.Println(stats)
		},
	}))

	d.Use(middleware.NewDefaultErrorHandler())

	d.Use(middleware.NewDefaultFresh())
	d.Use(middleware.NewDefaultETag())

	d.Use(middleware.NewDefaultResponder())

	d.Use(middleware.NewDefaultBodyParser())

	d.GET("/users/me", func(c *cod.Context) (err error) {
		c.Body = &struct {
			Account string `json:"account"`
		}{
			"tree.xie",
		}
		return
	})

	d.POST("/users/login", func(c *cod.Context) (err error) {
		m := make(map[string]interface{})
		err = json.Unmarshal(c.RequestBody, &m)
		if err != nil {
			return
		}
		c.Body = m
		return
	})

	d.GET("/error", func(_ *cod.Context) error {
		return errors.New("abcd")
	})

	d.ListenAndServe(":8001")
}
