package mcp_config

import (
	"fmt"

	"github.com/lmorg/mxtty/types"
)

const _INPUT_SCHEMA_TYPE_PROMPT_STRING = "promptString"

/*
   {
     "type": "promptString",
     "id": "aws_access_key",
     "description": "AWS Access Key ID",
     "password": true
   },
*/

type InputT struct {
	Type        string `json:"type"`
	Id          string `json:"id"`
	Description string `json:"description"`
	Password    bool   `json:"password"`
}

type InputsT map[string]InputT

func (inputs InputsT) Get(renderer types.Renderer, id string) (string, error) {
	input, ok := inputs[id]
	if !ok {
		return "", fmt.Errorf("missing input schema for `%s`", id)
	}

	if input.Type != _INPUT_SCHEMA_TYPE_PROMPT_STRING {
		return "", fmt.Errorf("input schema for `%s` is '%s', expecting '%s'", id, input.Type, _INPUT_SCHEMA_TYPE_PROMPT_STRING)
	}

	ch := make(chan string)

	renderer.DisplayInputBox(input.Description, "", func(s string) {
		ch <- s
	}, nil)

	val := <-ch
	return val, nil
}
