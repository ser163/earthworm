# earthworm

#### 介绍
将数据库内容,同步到飞书多维文档.

#### 软件架构
1. 拉取远程服务器id做对比
2. 如果有差异则进行更新


#### 安装教程
改名配置文件

`
mv config.yaml.ex config.yaml
`


 编译运行
## 编译
Linux编译
```shell
go build -ldflags "-s -w" -o earth main.go
```

windows下交叉编译
```shell
set GOOS=linux
set GOARCH=amd64
go build -ldflags "-s -w" -o earth main.go
```

#### 使用说明

1. 将程序放进定时任务,即可实时同步.


