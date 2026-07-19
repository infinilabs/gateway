module infini.sh/gateway

go 1.25.0

replace infini.sh/framework => ../framework

replace github.com/cihub/seelog => ../framework/lib/seelog

replace github.com/dop251/goja => github.com/infinilabs/framework-vendor/src/github.com/dop251/goja v0.0.0-20230228080227-6d95946e4353

replace github.com/dop251/goja_nodejs => github.com/infinilabs/framework-vendor/src/github.com/dop251/goja_nodejs v0.0.0-20230228080227-6d95946e4353

require (
	github.com/OneOfOne/xxhash v1.2.8
	github.com/bsm/extsort v0.6.1
	github.com/buger/jsonparser v1.2.0
	github.com/cespare/xxhash v1.1.0
	github.com/dgraph-io/ristretto v0.2.0
	github.com/dop251/goja v0.0.0-20220408131256-ffe77e20c6f1
	github.com/dop251/goja_nodejs v0.0.0-20211022123610-8dd9abb0616d
	github.com/drewlanenga/govector v0.0.0-20220726163947-b958ac08bc93
	github.com/emirpasic/gods v1.18.1
	github.com/fsnotify/fsnotify v1.10.1
	github.com/go-redis/redis/v8 v8.11.5
	github.com/j-keck/arping v1.0.3
	github.com/magiconair/properties v1.8.10
	github.com/mailru/easyjson v0.9.2
	github.com/minio/minio-go/v7 v7.1.0
	github.com/pierrec/xxHash v0.1.5
	github.com/pkg/errors v0.9.1
	github.com/savsgio/gotils v0.0.0-20250924091648-bce9a52d7761
	github.com/segmentio/fasthash v1.0.3
	github.com/segmentio/kafka-go v0.4.51
	github.com/shaj13/libcache v1.2.1
	github.com/stretchr/testify v1.11.1
	github.com/twmb/franz-go v1.21.2
	golang.org/x/net v0.55.0
)

require (
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/dlclark/regexp2 v1.11.4 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/go-sourcemap/sourcemap v2.1.4+incompatible // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/klauspost/crc32 v1.3.0 // indirect
	github.com/minio/crc64nvme v1.1.1 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.26 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/tinylib/msgp v1.6.1 // indirect
	github.com/twmb/franz-go/pkg/kmsg v1.13.1 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/zeebo/xxh3 v1.1.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.52.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
