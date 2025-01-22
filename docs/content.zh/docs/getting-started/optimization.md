---
weight: 40
title: 系统调优
---

# 系统调优

要保证极限网关运行在最佳状态，其所在服务器的操作系统也需要进行相应的调优，以 Linux 为例。

## 系统参数

```
sudo tee /etc/security/limits.d/21-infini.conf <<-'EOF'
*                soft    nofile         1048576
*                hard    nofile         1048576
*                soft    memlock        unlimited
*                hard    memlock        unlimited
root             soft    nofile         1048576
root             hard    nofile         1048576
root             soft    memlock        unlimited
root             hard    memlock        unlimited
EOF
```

## 内核调优

```
cat << SETTINGS | sudo tee /etc/sysctl.d/70-infini.conf
fs.file-max=10485760
fs.nr_open=10485760
vm.max_map_count=262144

net.core.somaxconn=65535
net.core.netdev_max_backlog=65535
net.core.rmem_default = 262144
net.core.wmem_default = 262144
net.core.rmem_max=4194304
net.core.wmem_max=4194304

net.ipv4.ip_forward = 1
net.ipv4.ip_nonlocal_bind=1
net.ipv4.ip_local_port_range = 1024 65535
net.ipv4.conf.default.accept_redirects = 0
net.ipv4.conf.default.rp_filter = 1
net.ipv4.conf.all.accept_redirects = 0
net.ipv4.conf.all.send_redirects = 0
net.ipv4.tcp_tw_reuse=1
net.ipv4.tcp_tw_recycle = 1
net.ipv4.tcp_max_tw_buckets = 300000
net.ipv4.tcp_timestamps=1
net.ipv4.tcp_syncookies=1
net.ipv4.tcp_max_syn_backlog=65535
net.ipv4.tcp_synack_retries=0
net.ipv4.tcp_keepalive_intvl = 30
net.ipv4.tcp_keepalive_time = 900
net.ipv4.tcp_keepalive_probes = 3
net.ipv4.tcp_fin_timeout = 10
net.ipv4.tcp_max_orphans = 131072
net.ipv4.tcp_rmem = 4096 4096 16777216
net.ipv4.tcp_wmem = 4096 4096 16777216
net.ipv4.tcp_mem = 786432 3145728  4194304
SETTINGS
```

执行下面的命令验证配置参数是否合法。

```
sysctl -p
```

最后重启操作系统让配置生效。
