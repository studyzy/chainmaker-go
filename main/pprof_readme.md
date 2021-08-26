# pprof使用

若要开启pprof，请在启动项中加入`--pprof`

## Go tool pprof常用基本调试基本命令(默认30s采集时间，可通过--seconds)

- Heap profile:

```
go tool pprof --text http://localhost:6060/debug/pprof/heap
```


- CPU profile:
文本显示：

```
go tool pprof --text http://localhost:6060/debug/pprof/profile
```

图片显示：

```
go tool pprof http://localhost:6060/debug/pprof/profile
#结束后直接进入交互：
(pprof)
web
(pprof)
```

- Goroutine blocking profile:

```
go tool pprof --text http://localhost:6060/debug/pprof/block
```


