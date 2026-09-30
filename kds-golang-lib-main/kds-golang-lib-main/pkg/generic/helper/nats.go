package helper

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	INTERVAL_PING    = 30 * time.Second
	INTERVAL_TIMEOUT = 5
)

func KDSNatsLocalConnect(name string) (*nats.Conn, error) {
	nc, err := nats.Connect(nats.DefaultURL, nats.Timeout(30*time.Second), nats.Name(name), nats.PingInterval(INTERVAL_PING), nats.MaxPingsOutstanding(INTERVAL_TIMEOUT))
	return nc, err
}

func KDSNatsLocalConnectOverride(name string, ip string) (*nats.Conn, error) {
	nc, err := nats.Connect("nats://"+ip+":4222", nats.Timeout(30*time.Second), nats.Name(name), nats.PingInterval(INTERVAL_PING), nats.MaxPingsOutstanding(INTERVAL_TIMEOUT))
	return nc, err
}

// KDSNatsLocalConnectPort is KDSNatsLocalConnect with a configurable port -
// production always runs exactly one kdswebserver + one kdsnats per
// terminal on the default port, so this exists purely for local testing
// (running a second terminal's full stack on the same machine without a
// port clash), matching the same reasoning as kds-webserver-main's own
// -port flag.
func KDSNatsLocalConnectPort(name string, port int) (*nats.Conn, error) {
	nc, err := nats.Connect(fmt.Sprintf("nats://127.0.0.1:%d", port), nats.Timeout(30*time.Second), nats.Name(name), nats.PingInterval(INTERVAL_PING), nats.MaxPingsOutstanding(INTERVAL_TIMEOUT))
	return nc, err
}

func KDSNatsRemoteConnect(name string) (*nats.Conn, error) {
	nc, err := nats.Connect("nats://nats.tabletpos.co.za:4222", nats.Name(name), nats.Token("KDS2021@Tabletpos"), nats.Timeout(INTERVAL_PING), nats.PingInterval(30*time.Second), nats.MaxPingsOutstanding(INTERVAL_TIMEOUT))
	return nc, err
}

func KDSNatsRemoteConnectOld(name string) (*nats.Conn, error) {
	nc, err := nats.Connect("nats://admin.tabletpos.co.za:4222", nats.Name(name), nats.Token("KDS2021@Tabletpos"), nats.Timeout(INTERVAL_PING), nats.PingInterval(30*time.Second), nats.MaxPingsOutstanding(INTERVAL_TIMEOUT))
	return nc, err
}

func KDSNATSPublish(nc *nats.Conn, subject string, data []byte) {
	//fmt.Println("Publishing Local On : " + subject)
	if nc != nil && nc.IsConnected() {
		nc.Publish(subject, data)
		fmt.Println("PUB : " + subject)
	}
}

func KDSNATSPublishGlobal(nc *nats.Conn, subject string, data []byte) {
	//fmt.Println("Publishing Global On : " + subject)
	if nc != nil && nc.IsConnected() {
		nc.Publish(subject, data)
	}
}

func TestNats() {
	nc, _ := KDSNatsRemoteConnect("test")
	nc.Publish("kds.0856124963c78c5bee7cb9c4189fa973e34ff15eadac942b2196132a37d2ae05eebb8427424ae2861d2ffc63e57a6e4517a792f9229151561d74ca7a746e575f", []byte("HALLO"))
}
