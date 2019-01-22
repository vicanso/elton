# static file

静态文件处理，用于读取静态文件并输出。下面先来看看此中间件的参数配置：

- `Path` 静态文件目录，此参数用于指定静态文件的根目录，只能处理此目录下的文件，文件真实路径为`Path + url`
- `Mount` 静态文件请求mount的路径，如果配置此参数，会判断请求的静态文件url是否以此mount为前缀，如果不是，则执行next。如果是，则文件名为文件url去除mount部分。如请求的静态文件是`/assets/admin/index.html`，如果Mount为`/assets`，则文件真实路径为`Path` + `"/admin/index.html"`。
- `MaxAge` 设置HTTP Cache-Control中的max-age的值。
- `SMaxAge` 设置HTTP Cache-Control中的s-maxage的值（如果有使用缓存中间件）
- `Header` 对静态文件添加的自定义响应头，可根据实际需要添加所需响应头。
- `DenyQueryString` 禁止querystring，如果静态文件放在CDN回源，可以禁止querystring，避免误用导致生成过多的无用文件，建议启用。
- `DenyDot` 禁止dot，主要用于避免返回静态文件目录中有一些以.开头的目录或文件（隐藏文件），建议启用。
- `DisableETag` 禁止生成ETag，如果认为不需要生成ETag则可禁用，不建议启用。
- `DisableLastModified` 禁止生成Last-Modified，如果已生成了ETag，可不再生成Last-Modifed。
- `NotFoundNext` 如果404未发现文件时，是否转至下一个中间件，如果不设置，直接返回出错，不建议启用。

```go
fs := &middleware.FS{}
static := middleware.NewStaticServe(fs, middleware.StaticServeConfig{
  Path:            "/Users/xieshuzhou/tmp",
  Mount:           "/assets",
  DenyQueryString: true,
  DenyDot:         true,
})
d.GET("/assets/*file", static, func(c *cod.Context) (err error) {
  return
})
```