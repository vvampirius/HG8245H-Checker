package main

import (
	"./modemClient"
	"fmt"
	"time"
)

const MODEM_HOSTNAME  = "192.168.100.1"
const MODEM_USERNAME  = "root"
const MODEM_PASSWORD  = "admin"


type Collector struct {
	Iface modemClient.Iface
}

func (self *Collector) CollectorRoutine(ifaceChan <-chan modemClient.Iface) {
	for true {
		self.Iface = <- ifaceChan
	}
}

func (self *Collector) ZabbixNotifier() {
	var lastTimestamp time.Time
	for true {
		if self.Iface.Timestamp != lastTimestamp {
			lastTimestamp = self.Iface.Timestamp
			fmt.Println(self.Iface)
		}
		time.Sleep(time.Second*1)
	}
}


func main()  {
	mc := modemClient.New(MODEM_HOSTNAME, MODEM_USERNAME, MODEM_PASSWORD)
	collector := Collector{}
	go collector.CollectorRoutine(mc.IfaceChan)
	collector.ZabbixNotifier()

}