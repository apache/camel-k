package event

import "k8s.io/client-go/tools/record"

type Injectable interface {
	InjectRecorder(recorder record.EventRecorder)
}
