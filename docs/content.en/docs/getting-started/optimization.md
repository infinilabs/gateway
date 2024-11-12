---
weight: 40
title: System Optimization
---

# System Optimization

The operating system of the server where INFINI Gateway is installed needs to be optimized to ensure that INFINI Gateway runs in the best possible state. The following uses Linux as an example.

## System Parameters

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

## Kernel Optimization

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

Run the following command to check whether configuration parameters are valid.

```
sysctl -p
```

Restart the operating system to make the configuration take effect.
