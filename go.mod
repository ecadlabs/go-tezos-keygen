module github.com/ecadlabs/go-tezos-keygen

go 1.21

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/ecadlabs/gotez/v2 v2.0.0
	github.com/ecadlabs/hdw v0.0.0-20221019154344-0b9e0a5909f0
	github.com/gorilla/mux v1.8.0
	github.com/sirupsen/logrus v1.9.3
	github.com/stretchr/testify v1.8.4
	go.etcd.io/bbolt v1.3.7
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.2.0 // indirect
	github.com/ecadlabs/goblst v1.0.0 // indirect
	github.com/ecadlabs/pretty v0.0.0-20230412124801-f948fc689a04 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	golang.org/x/crypto v0.18.0 // indirect
	golang.org/x/sys v0.16.0 // indirect
)

// replace github.com/ecadlabs/gotez/v2 => ../gotez
