# CPU Profiler 示例

项目结构：

```
.
├── collector
│   ├── collector.go
│   └── labels.go
├── cpuprofile
│   ├── cpuprofile.go
│   └── cpuprofile_test.go
├── testutil
│   └── util.go
├── util
│   └── util.go
├── main.go
├── go.mod
└── go.sum
```

## 前提

- $\text {Go} \ 1.21.10$ 或更高版本

## 安装

1. 克隆代码库

```shell
$ git clone https://github.com/SolisAmicus/cpu-profiler-example.git

$ cd cpu-profiler-example
```

2. 安装依赖项

```shell
$ go mod tidy
```

## 运行示例

```shell
$ go run main.go
```
