---
description: 各类常用的中间件
---

# Middleware

- [basic auth](https://github.com/vicanso/elton-basic-auth) HTTP Basic Auth，建议只用于内部管理系统使用
- [body parser](https://github.com/vicanso/elton-body-parser) 请求数据的解析中间件，支持`application/json`以及`application/x-www-form-urlencoded`两种数据类型
- [compress](https://github.com/vicanso/elton-compress) 数据压缩中间件，默认支持gzip、brotli、snappy、s2、zstd以及lz4，也可根据需要增加相应的压缩处理
- [concurrent limiter](https://github.com/vicanso/elton-concurrent-limiter) 根据指定参数限制并发请求，可用于订单提交等防止重复提交或限制提交频率的场景
- [error handler](https://github.com/vicanso/elton-error-handler) 用于将处理函数的Error转换为对应的响应数据，如HTTP响应中的状态码(40x, 50x)，对应的出错类别等，建议在实际使用中根据项目自定义的Error对象生成相应的响应数据
- [etag](https://github.com/vicanso/elton-etag) 用于生成HTTP响应数据的ETag
- [fresh](https://github.com/vicanso/elton-fresh) 判断HTTP请求是否未修改(Not Modified)
- [json picker](https://github.com/vicanso/elton-json-picker) 用于从响应的JSON中筛选指定字段
- [logger](https://github.com/vicanso/elton-logger) 生成HTTP请求日志，支持从请求头、响应头中获取相应信息
- [proxy](https://github.com/vicanso/elton-proxy) Proxy中间件，可定义请求转发至其它的服务
- [recover](https://github.com/vicanso/elton-recover) 捕获程序的panic异常，避免程序崩溃
- [responder](https://github.com/vicanso/elton-responder) 响应处理中间件，用于将`Context.Body`(interface{})转换为对应的JSON数据并输出。如果系统使用xml等输出响应数据，可参考此中间件实现interface{}至xml的转换
- [router-concurrent-limiter](https://github.com/vicanso/elton-router-concurrent-limiter) 路由并发限制中间件，可以针对路由限制并发请求量。
- [session](https://github.com/vicanso/elton-session) Session中间件，默认支持保存至redis或内存中，也可自定义相应的存储
- [stats](https://github.com/vicanso/elton-stats) 请求处理的统计中间件，包括处理时长、状态码、响应数据长度、连接数等信息
- [static serve](https://github.com/vicanso/elton-static-serve) 静态文件处理中间件，默认支持从目录中读取静态文件或实现StaticFile的相关接口，从[packr](github.com/gobuffalo/packr/v2)或者数据库(mongodb)等读取文件
- [tracker](https://github.com/vicanso/elton-tracker) 可以用于在POST、PUT等提交类的接口中增加跟踪日志，此中间件将输出QueryString，Params以及RequestBody部分，并能将指定的字段做"***"的处理，避免输出敏感信息
