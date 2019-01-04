package controller

import (
	"github.com/apache/camel-k/pkg/controller/integration"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, integration.Add)
}
