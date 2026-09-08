package tmux

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/lmorg/ttyphoon/debug"
)

const (
	CMD_SELECT_WINDOW = `select-window`
	CMD_SEND_PREFIX   = `send-prefix`
)

const _SEPARATOR = `|||`

func (tmux *Tmux) SendCommand(b []byte) (*tmuxResponseT, error) {
	if tmux.readerDead.Load() {
		return nil, errors.New("tmux control channel is closed")
	}

	tmux.limiter.Lock()
	defer tmux.limiter.Unlock()

	// Discard any response orphaned by a previously timed out command so it
	// cannot be mis-attributed to this one.
	select {
	case <-tmux.resp:
		debug.Log("discarded stale tmux response")
	default:
	}

	debug.Log(b)
	_, err := tmux.tty.Write(append(b, '\n'))
	//_, err := tmux.writePipe.Write(append(b, '\n'))
	if err != nil {
		debug.Log(fmt.Sprintf("error (%s): %v", string(b), err))
		return nil, err
	}

	select {
	case resp := <-tmux.resp:
		if resp.IsErr {
			return nil, fmt.Errorf("tmux command failed: %s", string(bytes.Join(resp.Message, []byte(": "))))
		}
		return resp, nil

	case <-time.After(tmuxCommandTimeout):
		return nil, fmt.Errorf("timed out after %s waiting for tmux response to: %s", tmuxCommandTimeout, string(b))
	}
}

func (tmux *Tmux) SendCommandWithReflection(command string, t reflect.Type, parameters ...string) (any, error) {
	resp, err := tmux.SendCommand(mkCmdLine(command, t, parameters...))
	if err != nil {
		return nil, err
	}

	var slice []any

	for i := range resp.Message {
		v := reflect.New(t)
		err = parseMxttyLine(resp.Message[i], v)
		if err != nil {
			if debug.Enabled {
				panic(err)
			}
			return nil, err
		}
		slice = append(slice, v.Interface())
	}

	return slice, nil
}

func mkCmdLine(command string, t reflect.Type, parameters ...string) []byte {
	var fields []string

	structTags := getStructTags(t)

	for i := range structTags {
		fields = append(fields, fmt.Sprintf(`%s:#{%s}`, structTags[i][0], structTags[i][1]))
	}

	cmdLine := fmt.Sprintf("%s %s -F '%s'", command, strings.Join(parameters, " "), strings.Join(fields, _SEPARATOR))
	return []byte(cmdLine)
}

func getStructTags(t reflect.Type) [][2]string {
	if t.Kind() != reflect.Struct {
		panic("provided value is not a struct")
	}

	var structTags [][2]string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag, ok := field.Tag.Lookup("tmux")
		if !ok {
			continue
		}
		structTags = append(structTags, [2]string{field.Name, tag})
	}

	return structTags
}

func parseMxttyLine(b []byte, v reflect.Value) error {
	fields := strings.Split(string(b), _SEPARATOR)

	for i := range fields {
		values := strings.SplitN(fields[i], ":", 2)

		if len(values) < 2 {
			return fmt.Errorf("too few values: %d", len(values))
		}

		err := setFieldValue(v, values[0], values[1])
		if err != nil {
			return err
		}
	}

	return nil
}

func setFieldValue(v reflect.Value, name string, value string) error {
	// Ensure that we have a pointer to a struct
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("expected a pointer to a struct")
	}

	// Get the actual struct value
	v = v.Elem()

	// Get the field by name
	field := v.FieldByName(name)
	if !field.IsValid() {
		return fmt.Errorf("no such field: %s in struct", name)
	}

	// Ensure the field is settable
	if !field.CanSet() {
		return fmt.Errorf("cannot set field: %s", name)
	}

	// Set the value, ensuring the types match
	switch field.Type().String() {
	case reflect.String.String():
		field.Set(reflect.ValueOf(value))

	case reflect.Int.String():
		value = strings.TrimSpace(value)
		i, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf(`cannot convert field to int: "%s": "%s" (%v)`, name, value, []byte(value))
		}
		field.Set(reflect.ValueOf(i))

	case reflect.Bool.String():
		if value == "true" {
			field.Set(reflect.ValueOf(true))
		} else {
			field.Set(reflect.ValueOf(false))
		}
	}

	return nil
}
