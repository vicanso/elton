---
description: Group的相关方法说明
---

# Group

## NewGroup

创建一个组，它包括Path的前缀以及组内公共中间件（非全局），适用于创建有相同前置校验条件的路由处理，如用户相关的操作。返回的Group对象包括`GET`，`POST`，`PUT`等方法，与Elton类似，之后可以通过`AddGroup`将所有路由处理添加至Elton实例。

**Example**
```go
```