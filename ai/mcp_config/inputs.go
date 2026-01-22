package mcp_config

import (
	"fmt"

	"github.com/lmorg/ttyphoon/types"
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

type InputsT []InputT

func (input *InputT) Get(renderer types.Renderer) (string, error) {
	if input.Type != _INPUT_SCHEMA_TYPE_PROMPT_STRING {
		return "", fmt.Errorf("input schema for `%s` is '%s', expecting '%s'", input.Id, input.Type, _INPUT_SCHEMA_TYPE_PROMPT_STRING)
	}

	ch := make(chan string)
	var err error

	renderer.DisplayInputBox(input.Description, "",
		func(s string) {
			ch <- s
		},
		func(_ string) {
			err = fmt.Errorf("input required for '%s'", input.Id)
			ch <- ""
		})

	val := <-ch
	return val, err
}
