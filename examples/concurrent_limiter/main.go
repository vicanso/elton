package main

import (
	"sync"
	"time"

	"github.com/vicanso/cod"
	"github.com/vicanso/cod/middleware"
)

// 开两个命令行，连续执行两次，后一次则报错
// curl -XPOST 'http://127.0.0.1:8001/orders/123'

func main() {
	d := cod.New()

	d.Use(middleware.NewRecover())

	d.Use(middleware.NewDefaultErrorHandler())

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

	// 订单处理

	orderLockMap := new(sync.Map)
	orderLimit := middleware.NewConcurrentLimiter(middleware.ConcurrentLimiterConfig{
		// 限制同一个id只能有一个在处理
		Keys: []string{
			"p:id",
		},
		Lock: func(key string, c *cod.Context) (bool, func(), error) {
			locked := false
			// 演示方便直接用sync.Map，实际使用建议基于redis等控制并发
			_, loaded := orderLockMap.LoadOrStore(key, true)
			if loaded {
				// 已有其它的locked成功（因为数据已存在）
				locked = false
			} else {
				locked = true
			}
			// 在处理完成时，回调删除
			unlock := func() {
				orderLockMap.Delete(key)
			}
			return locked, unlock, nil
		},
	})
	d.POST("/orders/:id", orderLimit, func(c *cod.Context) (err error) {
		// 为了方便模拟并发
		time.Sleep(10 * time.Second)
		c.Body = &struct {
			ID string `json:"id"`
		}{
			c.Param("id"),
		}
		return
	})

	d.ListenAndServe(":8001")
}
