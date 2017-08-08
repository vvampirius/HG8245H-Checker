package modemClient

import (
	"net"
	"fmt"
	"time"
	"regexp"
	"./fdReader"
	"os"
	"context"
	"strconv"
	"errors"
	"sync"
)

type Iface struct {
	Timestamp time.Time
	Name string
	Ip string
	RXPackets int
	TXPackets int
	RXBytes int
	TXBytes int
}

type ModemClient struct {
	Hostname string
	Username string
	Password string
	connection net.Conn
	fdReader *fdReader.FdReader
	IfaceChan chan Iface
	runMutex sync.Mutex
	Interval time.Duration
	Cancel context.CancelFunc
}


func (self *ModemClient) Run(ctx context.Context) {
	self.runMutex.Lock()
	repeatChan := make(chan time.Time, 1)
	repeatChan <- time.Now()
	active := true
	for active {
		select {
		case <-repeatChan:
			if conn, err := net.Dial("tcp", fmt.Sprintf("%s:23", self.Hostname)); err == nil {
				conn.SetReadDeadline(time.Now().Add(time.Second * 10))
				self.connection = conn
				fdr := fdReader.New(conn)
				self.fdReader = &fdr
				if self.login() {
					if ifconfig, ok := self.chat("ifconfig\n", `(?m)WAP>`, time.Second*4); ok {
						if iface, err := self.getInterfaceValues(ifconfig, `ppp\d+\s+.*\n(?:.+\r\n)*`);
							err == nil {
							self.IfaceChan <- iface
						}
					}
					fmt.Fprintln(conn, `quit`)
				} else {
					fmt.Fprintln(os.Stderr, "Login Failed!")
				}
				if err := conn.Close(); err != nil {
					fmt.Fprintln(os.Stderr, err.Error())
				}
			} else {
				fmt.Println(err)
			}
		case <-time.After(self.Interval):
			repeatChan <- time.Now()
		case <-ctx.Done():
			active = false
		}
	}
	self.runMutex.Unlock()
}

func (self *ModemClient) login() bool {
	if _, ok := self.chat(``, `(?m)^Login:`, time.Second*2); ok {
		if _, ok := self.chat(self.Username+"\n", `(?m)^Password:`, time.Second*2); ok {
			if _, ok := self.chat(self.Password+"\n", `(?m)WAP>`, time.Second*2); ok {
				return true
			}
		}
	}
	return false
}

func (self *ModemClient) chat(send string, expectRegexp string, timeout time.Duration) (string, bool) {
	timestamp := time.Now()
	if len(send)>0 {
		if _, err := fmt.Fprint(self.connection, send); err!=nil {
			fmt.Fprintf(os.Stderr, "An arror while sending '%s':\n%s\n", send, err.Error())
			return ``, false
		}
	}
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	response, match := self.fdReader.ReadUntilExpect(regexp.MustCompile(expectRegexp), timestamp, ctx)
	if !match {
		fmt.Fprintf(os.Stderr, "Can't match '%s' in response '%s'\n", expectRegexp, string(response))
	}
	return string(response), match
}

func (self *ModemClient) getInterfaceValues(rawIfconfigOutput string, interfaceRegexpMatch string) (Iface, error) {
	_iface := Iface{
		Timestamp: time.Now(),
	}
	iface := regexp.MustCompile(interfaceRegexpMatch).FindString(rawIfconfigOutput)
	if len(iface) > 0 {
		ifaceNameRxp := regexp.MustCompile(`^(\S+)\s+`)
		ifaceNameRxpSubmatch := ifaceNameRxp.FindStringSubmatch(iface)
		if len(ifaceNameRxpSubmatch) == 2 {
			_iface.Name = ifaceNameRxpSubmatch[1]
		}
		ifaceIpRxp := regexp.MustCompile(`inet addr:(\S+)`)
		ifaceIpRxpSubmatch := ifaceIpRxp.FindStringSubmatch(iface)
		if len(ifaceIpRxpSubmatch) == 2 {
			_iface.Ip = ifaceIpRxpSubmatch[1]
		}
		ifaceRxPacketsRxp := regexp.MustCompile(`RX packets:(\d+)`)
		ifaceRxPacketsRxpSubmatch := ifaceRxPacketsRxp.FindStringSubmatch(iface)
		if len(ifaceRxPacketsRxpSubmatch) == 2 {
			if i, err := strconv.Atoi(ifaceRxPacketsRxpSubmatch[1]); err==nil {
				_iface.RXPackets = i
			}
		}
		ifaceTxPacketsRxp := regexp.MustCompile(`TX packets:(\d+)`)
		ifaceTxPacketsRxpSubmatch := ifaceTxPacketsRxp.FindStringSubmatch(iface)
		if len(ifaceTxPacketsRxpSubmatch) == 2 {
			if i, err := strconv.Atoi(ifaceTxPacketsRxpSubmatch[1]); err==nil {
				_iface.TXPackets = i
			}
		}
		ifaceBytesRxp := regexp.MustCompile(`RX bytes:(\d+).*TX bytes:(\d+)`)
		ifaceBytesRxpSubmatch := ifaceBytesRxp.FindStringSubmatch(iface)
		if len(ifaceBytesRxpSubmatch) == 3 {
			if i, err := strconv.Atoi(ifaceBytesRxpSubmatch[1]); err==nil {
				_iface.RXBytes = i * 1000
			}
			if i, err := strconv.Atoi(ifaceBytesRxpSubmatch[2]); err==nil {
				_iface.TXBytes = i * 1000
			}
		}
	} else {
		return _iface, errors.New(`Cant find interface!`)
	}
	return _iface, nil
}

func New(hostname string, username string, password string) ModemClient {
	ctx, cancelFunc := context.WithCancel(context.Background())
	mc := ModemClient{
		Hostname: hostname,
		Username: username,
		Password: password,
		IfaceChan: make(chan Iface, 100),
		Interval: time.Second*30,
		Cancel: cancelFunc,
	}
	go mc.Run(ctx)
	return mc
}