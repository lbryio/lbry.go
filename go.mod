module github.com/lbryio/lbry.go/v2

replace github.com/btcsuite/btcd => github.com/lbryio/lbrycrd.go v0.0.0-20200203050410-e1076f12bf19

require (
	github.com/asaskevich/govalidator v0.0.0-20190424111038-f61b66f89f4a // indirect
	github.com/btcsuite/btcd v0.0.0-20190213025234-306aecffea32
	github.com/btcsuite/btcutil v0.0.0-20190425235716-9e5f4b9a998d
	github.com/davecgh/go-spew v1.1.1
	github.com/fatih/structs v1.1.0
	github.com/go-errors/errors v1.1.1
	github.com/go-ini/ini v1.48.0
	github.com/go-ozzo/ozzo-validation v3.6.0+incompatible // indirect
	github.com/golang/protobuf v1.3.2
	github.com/google/go-cmp v0.3.1 // indirect
	github.com/gopherjs/gopherjs v0.0.0-20190915194858-d3ddacdb130f // indirect
	github.com/gorilla/mux v1.7.3
	github.com/gorilla/rpc v1.2.0
	github.com/gorilla/websocket v1.4.1 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/kr/pretty v0.1.0 // indirect
	github.com/lbryio/ozzo-validation v0.0.0-20170323141101-d1008ad1fd04
	github.com/lbryio/types v0.0.0-20201019032447-f0b4476ef386
	github.com/lyoshenka/bencode v0.0.0-20180323155644-b7abd7672df5
	github.com/mitchellh/mapstructure v1.1.2
	github.com/nlopes/slack v0.6.0
	github.com/onsi/ginkgo v1.10.2 // indirect
	github.com/onsi/gomega v1.7.0 // indirect
	github.com/pkg/errors v0.8.1 // indirect
	github.com/sebdah/goldie v0.0.0-20190531093107-d313ffb52c77
	github.com/sergi/go-diff v1.0.0
	github.com/shopspring/decimal v0.0.0-20191009025716-f1972eb1d1f5
	github.com/sirupsen/logrus v1.4.2
	github.com/smartystreets/assertions v1.0.1 // indirect
	github.com/smartystreets/goconvey v0.0.0-20190731233626-505e41936337 // indirect
	github.com/spf13/cast v1.3.0
	github.com/stretchr/testify v1.7.0
	github.com/ybbus/jsonrpc v0.0.0-20180411222309-2a548b7d822d
	go.uber.org/atomic v1.4.0
	golang.org/x/crypto v0.0.0-20191002192127-34f69633bfdc
	golang.org/x/net v0.0.0-20191009170851-d66e71096ffb
	golang.org/x/oauth2 v0.0.0-20180821212333-d2e6202438be
	golang.org/x/sys v0.0.0-20191009170203-06d7bd2c5f4f // indirect
	golang.org/x/text v0.3.2
	golang.org/x/time v0.0.0-20190921001708-c4c64cad1fd0
	google.golang.org/genproto v0.0.0-20191009194640-548a555dbc03 // indirect
	google.golang.org/grpc v1.24.0
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/ini.v1 v1.48.0 // indirect
	gopkg.in/nullbio/null.v6 v6.0.0-20161116030900-40264a2e6b79
	gopkg.in/yaml.v2 v2.2.4 // indirect
	gotest.tools v2.2.0+incompatible
)

go 1.15
