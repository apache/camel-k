package kubernetes

import (
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"time"
)

type ResourceRetrieveFunction func() (interface{}, error)

type ResourceCheckFunction func(interface{}) (bool, error)

const (
	sleepTime = 400 * time.Millisecond
)

func WaitCondition(obj runtime.Object, condition ResourceCheckFunction, maxDuration time.Duration) error {
	start := time.Now()

	for start.Add(maxDuration).After(time.Now()) {
		err := sdk.Get(obj)
		if err != nil {
			time.Sleep(sleepTime)
			continue
		}

		satisfied, err := condition(obj)
		if err != nil {
			return errors.Wrap(err, "error while evaluating condition")
		} else if !satisfied {
			time.Sleep(sleepTime)
			continue
		}

		return nil
	}
	return errors.New("timeout while waiting condition")
}
