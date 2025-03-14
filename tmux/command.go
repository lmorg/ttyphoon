package tmux

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/lmorg/mxtty/debug"
)

const (
	CMD_SELECT_WINDOW = `select-window`
	CMD_SEND_PREFIX   = `send-prefix`
)

const _SEPARATOR = `|||`

func (tmux *Tmux) SendCommand(b []byte) (*tmuxResponseT, error) {
	//tmux.limiter.Lock()

	_, err := tmux.tty.Write(append(b, '\n'))
	//_, err := tmux.writePipe.Write(append(b, '\n'))
	if err != nil {
		debug.Log(fmt.Sprintf("error (%s): %v", string(b), err))
		return nil, err
	}

	resp := <-tmux.resp

	//tmux.limiter.Unlock()

	if resp.IsErr {
		return nil, fmt.Errorf("tmux command failed: %s", string(bytes.Join(resp.Message, []byte(": "))))
	}

	return resp, nil
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
		i, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("cannot convert field to int: %s", name)
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
