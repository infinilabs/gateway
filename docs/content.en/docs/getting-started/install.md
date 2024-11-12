---
weight: 20
title: Installing the Gateway
asciinema: true
---

# Installing the Gateway

INFINI Gateway supports mainstream operating systems and platforms. The program package is small, with no extra external dependency. So, the gateway can be installed very rapidly.

## Installation Demo

{{< asciinema key="/install"  autoplay="1" speed="2" rows="30" preload="1" >}}

## Downloading

**Automatic install**

```bash
curl -sSL http://get.infini.cloud | bash -s -- -p gateway
```

> The above script can automatically download the latest version of the corresponding platform's gateway and extract it to /opt/gateway

> The optional parameters for the script are as follows:

> &nbsp;&nbsp;&nbsp;&nbsp;_-v [version number]（Default to use the latest version number）_

> &nbsp;&nbsp;&nbsp;&nbsp;_-d [installation directory] (default installation to /opt/gateway)_

**Manual install**

Select a package for downloading in the following URL based on your operating system and platform:

[https://release.infinilabs.com/](https://release.infinilabs.com/gateway/)

## Container Deployment

INFINI Gateway also supports Docker container deployment.

{{< button relref="./docker" >}}Learn More{{< /button >}}

## Verifying the Installation

After downloading and decompressing INFINI Gateway installation package, run the following command to check whether the installation package is effective:

```
✗ ./bin/gateway -v
gateway 1.0.0_SNAPSHOT 2021-01-03 22:45:28 6a54bb2
```

If the above version information is displayed, the gateway program is in good condition.

## Starting the Gateway

Run the gateway program as an administrator to start INFINI Gateway, as follows:

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

If the above startup information is displayed, the gateway is running successfully and listening on specified port.

## Accessing the Gateway

The back-end Elasticsearch service can be accessed using a browser or other clients through the gateway that serves as a proxy:

{{% load-img "/img/access-gateway.jpg" "Service Gateway" %}}

## Shutting Down the Gateway

To shut down INFINI Gateway, hold down `Ctrl+C`. The following information will be displayed:

```
^C
[GATEWAY] got signal: interrupt, start shutting down
[01-11 16:44:41] [INF] [app.go:303] gateway now terminated.
[GATEWAY] 1.0.0_SNAPSHOT, uptime: 1m10.550336s

Thanks for using GATEWAY, have a good day!
```

## System Service

To run the data platform of INFINI Gateway as a background task, run the following commands:

```
➜ ./gateway -service install
Success
➜ ./gateway -service start
Success
```

Unloading the service is simple. To unload the service, run the following commands:

```
➜ ./gateway -service stop
Success
➜ ./gateway -service uninstall
Success
```

Customize service name:

```
sudo SERVICE_NAME=mygw ./bin/gateway -service install
sudo SERVICE_NAME=mygw ./bin/gateway -service start
sudo SERVICE_NAME=mygw ./bin/gateway -service stop
sudo SERVICE_NAME=mygw ./bin/gateway -service uninstall
```

INFINI Gateway has been completely installed. Next, configure the gateway.

{{< button relref="./configuration" >}}Configuring INFINI Gateway{{< /button >}}
