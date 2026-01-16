go 1.25

module github.com/lbryio/lbry.go/v2

replace github.com/btcsuite/btcd => github.com/lbryio/lbrycrd.go v0.0.0-20200203050410-e1076f12bf19

require (
	github.com/btcsuite/btcd v0.0.0-20190213025234-306aecffea32
	github.com/btcsuite/btcutil v0.0.0-20190425235716-9e5f4b9a998d
	github.com/davecgh/go-spew v1.1.1
	github.com/fatih/structs v1.1.0
	github.com/go-errors/errors v1.5.1
	github.com/golang/protobuf v1.5.4
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/rpc v1.2.1
	github.com/lbryio/ozzo-validation v3.0.3-0.20170512160344-202201e212ec+incompatible
	github.com/lbryio/types v0.0.0-20220224142228-73610f6654a6
	github.com/lyoshenka/bencode v0.0.0-20180323155644-b7abd7672df5
	github.com/mitchellh/mapstructure v1.5.0
	github.com/sebdah/goldie v1.0.0
	github.com/sergi/go-diff v1.4.0
	github.com/shopspring/decimal v1.4.0
	github.com/sirupsen/logrus v1.9.4
	github.com/slack-go/slack v0.17.3
	github.com/spf13/cast v1.10.0
	github.com/stretchr/testify v1.11.1
	github.com/ybbus/jsonrpc/v2 v2.1.7
	go.uber.org/atomic v1.11.0
	golang.org/x/crypto v0.47.0
	golang.org/x/net v0.49.0
	golang.org/x/oauth2 v0.34.0
	golang.org/x/text v0.33.0
	golang.org/x/time v0.14.0
	google.golang.org/grpc v1.78.0
	gopkg.in/ini.v1 v1.67.1
	gopkg.in/nullbio/null.v6 v6.0.0-20161116030900-40264a2e6b79
	gotest.tools v2.2.0+incompatible
)

require (
	github.com/asaskevich/govalidator v0.0.0-20190424111038-f61b66f89f4a // indirect
	github.com/btcsuite/btclog v0.0.0-20170628155309-84c8d2346e9f // indirect
	github.com/btcsuite/go-socks v0.0.0-20170105172521-4720035b7bfd // indirect
	github.com/btcsuite/websocket v0.0.0-20150119174127-31079b680792 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/onsi/gomega v1.7.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	google.golang.org/genproto v0.0.0-20230110181048-76db0878b65f // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
