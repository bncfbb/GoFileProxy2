##What's GoProxy2
GoProxy2 is a reverse proxy for speeding up file download program.

##Get Start
Help param

* Usage of FileProxy2:

  -debug
        是否启用log调试模式(true或false), 默认为false
  -listen string
        设置监听地址 (default "[::]:8099")
  -timeout int
        下载链接超时时间(秒) (default 3600)"""

Examples

* `./goproxy2`
* `./goproxy2 -timeout=7200 -listen=0.0.0.0:8080 -denug=true`
