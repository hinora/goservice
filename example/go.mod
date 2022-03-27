module example

go 1.18

replace github.com/hinora/goservice => ../

require github.com/hinora/goservice v1.0.0

require (
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/go-redis/redis/v8 v8.11.5 // indirect
)
