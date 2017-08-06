package modemClient

import (
	"net"
	"fmt"
	"time"
	"regexp"
	//"errors"
	"./fdReader"
	"os"
	"context"
)

type ModemClient struct {
	Hostname string
	Username string
	Password string
	connection net.Conn
	fdReader *fdReader.FdReader
}


func (self *ModemClient) Run() {
	//for {
		if conn, err := net.Dial("tcp", fmt.Sprintf("%s:23", self.Hostname)); err==nil {
			conn.SetReadDeadline(time.Now().Add(time.Second * 10))
			self.connection = conn
			fdr := fdReader.New(conn)
			self.fdReader = &fdr
			if self.login() {
				fmt.Println(self.chat("ifconfig\n", `(?m)WAP>`, time.Second*4))
				// ppp\d+\s+.*\n\s+inet addr:(\S+).*\n.*\n\s+RX packets:(\d+).*\n\s+TX packets:(\d+).*\n.*\n\s+RX bytes:(\d+).*TX bytes:(\d+).*
				// ppp\d+\s+.*\n(.+\n)*
			} else {
				fmt.Fprintln(os.Stderr, "Login Failed!")
			}
			if err := conn.Close(); err!=nil {
				fmt.Fprintln(os.Stderr, err.Error())
			}
		} else {
			fmt.Println(err)
		}
		time.Sleep(time.Second)
	//}
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


func New(hostname string, username string, password string) ModemClient {
	mc := ModemClient{
		Hostname: hostname,
		Username: username,
		Password: password,
	}
	mc.Run()
	return mc
}