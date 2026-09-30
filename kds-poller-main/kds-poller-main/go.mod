module github.com/reverent911/kds-poller

go 1.16

require (
	github.com/davecgh/go-spew v1.1.1
	github.com/getsentry/sentry-go v0.13.0
	github.com/glebarez/sqlite v1.11.0
	github.com/nats-io/nats.go v1.13.1-0.20220308171302-2f2f6968e98d
	github.com/newrelic/go-agent/v3 v3.15.2
	github.com/reverent911/kds-golang-lib v0.0.0-20250117123613-ff648ff5a855
	github.com/stretchr/testify v1.7.0 // indirect
	gorm.io/gorm v1.25.7
)

// Build against the local checkout with the "in production" feature until
// it's published upstream.
replace github.com/reverent911/kds-golang-lib => ../../kds-golang-lib-main/kds-golang-lib-main
