module github.com/reverent911/kds-webserver

go 1.21

toolchain go1.23.3

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/getsentry/sentry-go v0.29.1
	github.com/glebarez/sqlite v1.11.0
	github.com/go-sql-driver/mysql v1.8.1
	github.com/gorilla/mux v1.8.1
	github.com/jinzhu/gorm v1.9.16
	github.com/nats-io/nats.go v1.37.0
	github.com/newrelic/go-agent/v3 v3.35.1
	github.com/reverent911/kds-golang-lib v0.0.0-20241204191511-52ca97e8b382
	github.com/stretchr/testify v1.8.4
	gorm.io/gorm v1.25.7
)

// Build against the local checkout with the "in production" feature until
// it's published upstream.
replace github.com/reverent911/kds-golang-lib => ../../kds-golang-lib-main/kds-golang-lib-main

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/aws/aws-lambda-go v1.28.0 // indirect
	github.com/aws/aws-sdk-go v1.43.19 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/glebarez/go-sqlite v1.21.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/jung-kurt/gofpdf v1.16.2 // indirect
	github.com/klauspost/compress v1.17.7 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/nats-io/nkeys v0.4.7 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/richardlehane/mscfb v1.0.3 // indirect
	github.com/richardlehane/msoleps v1.0.1 // indirect
	github.com/xuri/efp v0.0.0-20210322160811-ab561f5b45e3 // indirect
	github.com/xuri/excelize/v2 v2.5.0 // indirect
	golang.org/x/crypto v0.23.0 // indirect
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
	golang.org/x/text v0.15.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240528184218-531527333157 // indirect
	google.golang.org/grpc v1.65.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gorm.io/driver/mysql v1.3.2 // indirect
	modernc.org/libc v1.22.5 // indirect
	modernc.org/mathutil v1.5.0 // indirect
	modernc.org/memory v1.5.0 // indirect
	modernc.org/sqlite v1.23.1 // indirect
)
