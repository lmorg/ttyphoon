package dispatcher

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

const address = "localhost:59687"

var connections = make(chan net.Conn)

type RespFunc func(*IpcMessageT)

type IpcMessageT struct {
	EventName  string
	Parameters map[string]string
	Error      error
}

type IpcT struct {
	mutex sync.Mutex
	conn  net.Conn
	resp  RespFunc
}

func (ipc *IpcT) hostListener() {
	ipc.mutex.Lock()
	ipc.conn = <-connections
	ipc.mutex.Unlock()
	ipc.clientListener()
}

func (ipc *IpcT) clientListener() {
	for {
		reader := bufio.NewReader(ipc.conn)
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				ipc.resp(&IpcMessageT{Error: err})
			}
			return
		}

		msg := new(IpcMessageT)
		err = json.Unmarshal([]byte(line), &msg)
		if err != nil {
			ipc.resp(&IpcMessageT{Error: err})
			continue
		}

		ipc.resp(msg)
	}
}

func (ipc *IpcT) Send(msg *IpcMessageT) error {
	// this is some ugly code :'(
isNil:
	ipc.mutex.Lock()
	if ipc.conn == nil {
		ipc.mutex.Unlock()
		time.Sleep(10 * time.Millisecond)
		goto isNil
	}
	ipc.mutex.Unlock()

	return ipc.send(msg)
}

func (ipc *IpcT) send(msg *IpcMessageT) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = ipc.conn.Write(append(b, '\n'))
	return err
}

func StartIpcServer() error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Println(err)
				continue
			}
			connections <- conn
		}
		//defer listener.Close()
	}()

	return nil
}

func hostListen(resp RespFunc) *IpcT {
	ipc := &IpcT{
		resp: resp,
	}
	go ipc.hostListener()
	return ipc
}

func ClientConnect(resp RespFunc) (*IpcT, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}

	ipc := &IpcT{
		conn: conn,
		resp: resp,
	}

	go ipc.clientListener()

	return ipc, nil
}
