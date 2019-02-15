# compress

响应数据压缩中间件，此中间件可根据配置的最小压缩长度、客户端接受压缩类型选择对应的压缩算法，以达到更好的效果。

```go

type BrComprssor struct {}

func (br *BrComprssor) Accept(c *cod.Context) (acceptable bool, encoding string) {
	return middleware.AcceptEncoding(c, "br")
}

func (br *BrComprssor) Compress(buf []byte, level int) ([]byte, error) {
  return cbrotli.Encode(buf, cbrotli.WriterOptions{
      Quality: level,
      LGWin:   0,
    })
}

d := cod.New()

compressorList := make([]middleware.Compressor, 1)
// 增加新的压缩方式 brotli
compressorList[0] = &BrComprssor{} 
d.Use(middleware.NewCompresss(middleware.CompressConfig{
  // 最小压缩尺寸，如果不设置为1KB
  MinLength:       1,
  // 压缩级别
  Level:           9,
  // 用于对响应数据类型判断是否使用压缩
  Checker:         regexp.MustCompile("text|javascript|json"),
  CompressorList: compressorList,
}))

d.Use(middleware.NewResponder(middleware.ResponderConfig{}))

d.GET("/ping", func(c *cod.Context) (err error) {
  c.Body = "pong"
  return
})
```

如果所有属性都不配置，则默认为针对长度大于1024的`text|javascript|json`类数据使用gzip压缩。

由于`brotli`压缩需要编译，因此默认未支持此方法，如果使用的场景为移动客户端，现在大部分都已支持`brotli`，建议自已添加支持节约带宽。如果是内部系统之间的调用，还可根据自己的需要增加其它类型的压缩方式。
