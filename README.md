```
make update-plugins
make build
(cd ~/go/src/infini.sh/framework/ && make build-cmd)
(~/go/src/infini.sh/framework/bin/plugin-discovery -dir plugins -pkg config -import_prefix infini.sh/gateway -out config/plugins.go)

```