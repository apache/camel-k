package cmd

import (
	"testing"

	"github.com/apache/camel-k/pkg/util/test"
)

func TestLogsAlias(t *testing.T) {
	options, rootCommand := kamelTestPreAddCommandInit()
	logCommand, _ := newCmdLog(options)
	rootCommand.AddCommand(logCommand)

	kamelTestPostAddCommandInit(t, rootCommand)

	_, err := test.ExecuteCommand(rootCommand, "logs")

	//in case of error we expect this to be the log default message
	if err != nil && err.Error() != "log expects an integration name argument" {
		t.Fatalf("Expected error result for invalid alias `logs`")
	}
}
