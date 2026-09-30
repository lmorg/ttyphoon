package userquestion

import (
	"context"
	"reflect"
	"testing"
)

func TestStructuredAskUserInput(t *testing.T) {
	tool := &AskUser{}
	inputType := tool.InputType()
	if inputType.Kind() != reflect.Struct {
		t.Fatalf("InputType() = %v, want object", inputType)
	}
	if _, ok := inputType.FieldByName("Question"); !ok {
		t.Fatal("input type is missing question")
	}
	if _, ok := inputType.FieldByName("Choices"); !ok {
		t.Fatal("input type is missing choices")
	}

	response, err := tool.CallStructured(context.Background(), &askUserInput{})
	if err != nil {
		t.Fatalf("CallStructured() error = %v", err)
	}
	if response != "ERROR: 'question' is required" {
		t.Fatalf("CallStructured() = %q, want required-question response", response)
	}
}
