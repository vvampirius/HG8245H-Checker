package main

import (
	"./modemClient"
	"fmt"
)

const MODEM_HOSTNAME  = "192.168.100.1"
const MODEM_USERNAME  = "root"
const MODEM_PASSWORD  = "admin"

func main()  {
	mc := modemClient.New(MODEM_HOSTNAME, MODEM_USERNAME, MODEM_PASSWORD)
	fmt.Println(mc)
	//for true {
	//}
}