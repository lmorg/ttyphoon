package dispatcher

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
)

type RespFunc func(*IpcMessageT)

type IpcMessageT struct {
	EventName  string            `json:"eventName"`
	Parameters map[string]string `json:"parameters"`
	Error      error             `json:"error"`
}

type IpcT struct {
	r        io.Reader
	w        io.Writer
	respFunc RespFunc
}

func (ipc *IpcT) listen() {
	reader := bufio.NewReader(ipc.r)
	for {
		line, err := reader.ReadString('\n')
		if err != nil { //&& err != io.EOF {
			//ipc.respFunc(&IpcMessageT{Error: err})
			os.Stderr.WriteString(err.Error())
			return
		}

		msg := new(IpcMessageT)
		err = json.Unmarshal([]byte(line), msg)
		if err != nil {
			//ipc.respFunc(&IpcMessageT{Error: err})
			os.Stderr.WriteString(err.Error())
			continue
		}

		os.Stderr.WriteString("received: " + line + "\n")
		ipc.respFunc(msg)
	}
}

func (ipc *IpcT) Send(msg *IpcMessageT) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = ipc.w.Write(append(b, '\n'))
	os.Stderr.WriteString("sent: " + string(b) + "\n")
	return err
}
