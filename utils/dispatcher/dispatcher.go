package dispatcher

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
)

const DISPATCHER_PORT = 59486

const ENV_WINDOW = "MXTTY_WINDOW"

func Start() {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", DISPATCHER_PORT))
	if err != nil {
		panic(err)
	}
	defer ln.Close()

	go startSdl()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}

		go handleConn(conn)
	}
}

type ActionT string

const (
	ActionStart   ActionT = "start"
	ActionStop    ActionT = "stop"
	ActionMessage ActionT = "message"
)

type dispatchEventT struct {
	Source      WindowNameT
	Destination WindowNameT
	Payload     string
	Action      ActionT
}

func handleConn(c net.Conn) {
	defer c.Close()
	r := bufio.NewReader(c)

	for {
		msg, err := r.ReadBytes('\n')
		if err != nil {
			if err.Error() != "EOF" {
				log.Printf("read error: %v", err)
			}
			return
		}
		// strip newline
		go handleMessage(msg)
	}
}

func handleMessage(msg []byte) {
	var evt dispatchEventT
	err := json.Unmarshal(msg, &evt)
	if err != nil {
		log.Println(err)
		return
	}

	switch evt.Action {

	}
}

func SendMessage(dstWindow WindowNameT, payload any) error {
	params, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	conn, err := net.Dial("TCP", fmt.Sprintf("127.0.0.1:%d", DISPATCHER_PORT))
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write(params)
	return err
}
