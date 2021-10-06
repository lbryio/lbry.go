module github.com/lbryio/lbry.go/v3

replace github.com/btcsuite/btcd => github.com/lbryio/lbcd v0.0.0-20200203050410-e1076f12bf19

require (
	github.com/cockroachdb/errors v1.8.6
	github.com/davecgh/go-spew v1.1.1
	github.com/go-errors/errors v1.1.1 // indirect
	github.com/go-ini/ini v1.48.0
	github.com/golang/protobuf v1.5.2
	github.com/gorilla/mux v1.7.3
	github.com/gorilla/rpc v1.2.0
	github.com/lbryio/lbcd v0.22.101-beta
	github.com/lbryio/lbcutil v1.0.201
	github.com/lbryio/lbry.go/v2 v2.7.1
	github.com/lbryio/types v0.0.0-20201019032447-f0b4476ef386
	github.com/lyoshenka/bencode v0.0.0-20180323155644-b7abd7672df5
	github.com/sebdah/goldie v0.0.0-20190531093107-d313ffb52c77
	github.com/sergi/go-diff v1.1.0
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/cast v1.3.0
	github.com/stretchr/testify v1.7.0
	go.uber.org/atomic v1.4.0
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5
	golang.org/x/time v0.0.0-20190921001708-c4c64cad1fd0
	gotest.tools v2.2.0+incompatible
)

go 1.16
