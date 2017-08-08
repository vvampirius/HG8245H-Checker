package main

import (
	"./modemClient"
	"fmt"
	"time"
	"bytes"
	"os"
	"os/exec"
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
			postData := make([]byte, 0)
			postDataBuffer := bytes.NewBuffer(postData)
			if len(self.Iface.Ip)>0 {
				fmt.Fprintf(postDataBuffer, "modem internet.ip %s\n", self.Iface.Ip)
			}
			fmt.Fprintf(postDataBuffer, "modem internet.rx.pkts %d\n", self.Iface.RXPackets)
			fmt.Fprintf(postDataBuffer, "modem internet.tx.pkts %d\n", self.Iface.TXPackets)
			fmt.Fprintf(postDataBuffer, "modem internet.rx.bytes %d\n", self.Iface.RXBytes)
			fmt.Fprintf(postDataBuffer, "modem internet.tx.bytes %d\n", self.Iface.TXBytes)
			//os.Stdout.Write(postDataBuffer.Bytes())
			cmd := exec.Command("zabbix_sender", "-z", "127.0.0.1", "-p", "10051", "-i", "-")
			//cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if stdin, err := cmd.StdinPipe(); err==nil {
				stdin.Write(postDataBuffer.Bytes())
				stdin.Close()
			} else {
				fmt.Fprintln(os.Stderr, err.Error())
			}
			if err := cmd.Start(); err!=nil {
				fmt.Fprintln(os.Stderr, err.Error())
			}

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