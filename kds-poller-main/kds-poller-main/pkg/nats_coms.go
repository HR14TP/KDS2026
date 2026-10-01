package main

import (
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/reverent911/kds-golang-lib/pkg/generic/helper"
	"github.com/reverent911/kds-golang-lib/pkg/kds/kdsmodels"
)

var NC *nats.Conn
var NCglobal *nats.Conn
var conerror error
var natsLocalPort = 4222

func checkConnections(kdsconfig kdsmodels.KDSConfig, dbLocation string) {
	//fmt.Println("Checking Connections")

	if NC == nil || !NC.IsConnected() {
		fmt.Println("POLL: Local NATS : Connecting")
		initLocalConnection(kdsconfig, dbLocation)
	}

	if NCglobal == nil || !NCglobal.IsConnected() {
		//initGlobalConnection()
	}
}

func initLocalConnection(kdsconfig kdsmodels.KDSConfig, dbLocation string) {
	NC, conerror = helper.KDSNatsLocalConnectPort("POLL", natsLocalPort)
	//myerror.HandelGlobalError(conerror, myerror.GetSentryServices().KDSPOLLER)
	if NC != nil && NC.IsConnected() {
		fmt.Println("Local Client Connected")
		NC.Subscribe("incoming", func(m *nats.Msg) {
			fmt.Println("INCOMING")
			obj := kdsmodels.KDSPollObject{}
			obj.Config = kdsconfig
			ProcessMessage(obj, m.Data, FileLocation, DBLocation)
		})
	}
}

func initGlobalConnection() {
	fmt.Println("POLL: Global NATS : Connecting")
	NCglobal, conerror = helper.KDSNatsRemoteConnect("POLL")
	if NCglobal != nil && NCglobal.IsConnected() {
		fmt.Println("Global Client Connected")
	}
}
