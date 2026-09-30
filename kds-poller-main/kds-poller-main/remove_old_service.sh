#!/bin/bash
sudo systemctl stop kdspoller
sudo systemctl disable kdspoller
sudo rm -rf /usr/bin/kdspoller

sudo systemctl stop kdstcpserver
sudo systemctl disable kdstcpserver
sudo rm -rf /usr/bin/kdstcpserver

sudo systemctl stop kdsnats
sudo systemctl disable kdsnats
sudo rm -rf /usr/bin/nats-server

sudo systemctl stop kdswebserver
sudo systemctl disable kdswebserver
sudo rm -rf /usr/bin/kdswebserver