---
weight: 20
title: 安装网关
asciinema: true
---

# 安装网关

极限网关支持主流的操作系统和平台，程序包很小，没有任何额外的外部依赖，安装起来应该是很快的 ：）

## 安装演示

{{< asciinema key="/install"  autoplay="1" speed="2" rows="30" preload="1" >}}

## 下载安装

**自动安装**

```bash
curl -sSL http://get.infini.cloud | bash -s -- -p gateway
```

> 通过以上脚本可自动下载相应平台的 gateway 最新版本并解压到/opt/gateway

> 脚本的可选参数如下：

> &nbsp;&nbsp;&nbsp;&nbsp;_-v [版本号]（默认采用最新版本号）_

> &nbsp;&nbsp;&nbsp;&nbsp;_-d [安装目录]（默认安装到/opt/gateway）_

**手动安装**

根据您所在的操作系统和平台选择下面相应的下载地址：

[https://release.infinilabs.com/](https://release.infinilabs.com/gateway/)

## 容器部署

极限网关也支持 Docker 容器方式部署。

{{< button relref="./docker" >}}了解更多{{< /button >}}

## 验证安装

极限网关下载解压之后，我们可以执行这个命令来验证安装包是否有效，如下：

```
✗ ./bin/gateway -v
gateway 1.0.0_SNAPSHOT 2021-01-03 22:45:28 6a54bb2
```

如果能够正常看到上面的版本信息，说明网关程序本身一切正常。

## 启动网关

以管理员身份直接运行网关程序即可启动极限网关了，如下：

```
➜ sudo ./bin/gateway
   ___   _   _____  __  __    __  _
  / _ \ /_\ /__   \/__\/ / /\ \ \/_\ /\_/\
 / /_\///_\\  / /\/_\  \ \/  \/ //_\\\_ _/
/ /_\\/  _  \/ / //__   \  /\  /  _  \/ \
\____/\_/ \_/\/  \__/    \/  \/\_/ \_/\_/

[GATEWAY] A light-weight, powerful and high-performance elasticsearch gateway.
[GATEWAY] 1.0.0_SNAPSHOT, 4daf6e9, Mon Jan 11 11:40:44 2021 +0800, medcl, add response_header_filter
[01-11 16:43:31] [INF] [instance.go:24] workspace: data/gateway/nodes/0
[01-11 16:43:31] [INF] [api.go:255] api server listen at: http://0.0.0.0:2900
[01-11 16:43:31] [INF] [runner.go:59] pipeline: primary started with 1 instances
[01-11 16:43:31] [INF] [runner.go:59] pipeline: nodes_index started with 1 instances
[01-11 16:43:31] [INF] [entry.go:262] entry [es_gateway] listen at: https://0.0.0.0:8000
[01-11 16:43:32] [INF] [floating_ip.go:170] floating_ip listen at: 192.168.3.234, echo port: 61111
[01-11 16:43:32] [INF] [app.go:254] gateway now started.
```

看到上面的启动信息，说明网关已经成功运行了，并且监听了相应的端口。

## 访问网关

使用浏览器或者其它客户端即可正常访问由网关代理的后端 Elasticsearch 服务了，如下：

{{% load-img "/img/access-gateway.jpg" "服务网关" %}}

## 停止网关

如果需要停止网关，按 `Ctrl+C` 即可停止极限网关，如下：

```
^C
[GATEWAY] got signal: interrupt, start shutting down
[01-11 16:44:41] [INF] [app.go:303] gateway now terminated.
[GATEWAY] 1.0.0_SNAPSHOT, uptime: 1m10.550336s

Thanks for using GATEWAY, have a good day!
```

## 系统服务

如果希望将极限网关以后台任务的方式运行，如下：

```
➜ ./gateway -service install
Success
➜ ./gateway -service start
Success
```

卸载服务也很简单，如下：

```
➜ ./gateway -service stop
Success
➜ ./gateway -service uninstall
Success
```

也支持自定义服务名称（如果有多个实例安装在一台机器上面）:

```
sudo SERVICE_NAME=mygw ./bin/gateway -service install
sudo SERVICE_NAME=mygw ./bin/gateway -service start
sudo SERVICE_NAME=mygw ./bin/gateway -service stop
sudo SERVICE_NAME=mygw ./bin/gateway -service uninstall
```

到这里极限网关就已经安装好了，下一步我们来看如何配置极限网关。

{{< button relref="./configuration" >}}配置网关{{< /button >}}
