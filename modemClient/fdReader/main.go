package fdReader

import (
	"net"
	"fmt"
	"io"
	"sync"
	"context"
	"time"
	"regexp"
	"bytes"
)

type ReadData struct {
	Timestamp time.Time
	Data []byte
	Error error
}

type FdReader struct {
	fd io.ReadWriter
	readQueue chan ReadData
	readMu sync.Mutex
	readCancelFunc context.CancelFunc
}

func (self *FdReader) readerRoutine(ctx context.Context) {
	self.readMu.Lock()
	read := true
	for read {
		select {
			case <-ctx.Done():
				read = false
			default:
				readBytes := make([]byte, 1024)
				readBytesCount, err := self.fd.Read(readBytes)
				returnData := make([]byte, readBytesCount)
				copy(returnData, readBytes)
				rd := ReadData{
					Timestamp: time.Now(),
					Data: returnData,
					Error: err,
				}
				self.readQueue <- rd
		}
	}
	self.readMu.Unlock()
}

func (self *FdReader) ReadUntilExpect(expect *regexp.Regexp, notEarlier time.Time, ctx context.Context) ([]byte, bool) {
	match := false
	read := true
	returnData := bytes.NewBuffer(make([]byte, 0))
	for read {
		select {
			case <- ctx.Done():
				read = false
			case readData := <- self.readQueue:
				if readData.Timestamp.Sub(notEarlier) > 0 {
					returnData.Write(readData.Data)
					match = expect.Match(returnData.Bytes())
					if match {
						read = false
					}
				} else {
					fmt.Printf("Got message from the past: %v\n%s\n%v\n\n", readData.Timestamp,
						string(readData.Data), readData.Error)
				}
		}
	}
	return returnData.Bytes(), match
}


func New(readWriter io.ReadWriter) FdReader {
	ctx, cancel := context.WithCancel(context.Background())
	fdReader := FdReader{
		fd: readWriter,
		readQueue: make(chan ReadData, 1024),
		readCancelFunc: cancel,
	}
	go fdReader.readerRoutine(ctx)
	return fdReader
}

func main() {
	if conn, err := net.Dial("tcp", "localhost:8082"); err==nil {
		//response := make([]byte, 0)
		//rr := bytes.NewBuffer(response)
		//if _, err := conn.Read(response); err!=nil {
		//	fmt.Println(err)
		//}
		//fmt.Println(io.Copy(rr, conn))
		//fmt.Println(string(response))
		r := New(conn)
		ctx, _ := context.WithTimeout(context.Background(), time.Second*15)
		fmt.Println(r.ReadUntilExpect(regexp.MustCompile(`(?m)^Login:`), time.Now(), ctx))
	}
}
