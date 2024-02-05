package integration

import (
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"testing"
)

// getMetricValue returns the sum of the Counter metrics associated with the Collector
// e.g. the metric for a non-vector, or the sum of the metrics for vector labels.
// If the metric is a Histogram then number of samples is used.
func getMetricValue(col prometheus.Collector) float64 {
	var total float64
	collect(col, func(m *dto.Metric) {
		if h := m.GetHistogram(); h != nil {
			total += float64(h.GetSampleCount())
		} else {
			total += m.GetCounter().GetValue()
		}
	})
	return total
}

// collect calls the function for each metric associated with the Collector
func collect(col prometheus.Collector, do func(*dto.Metric)) {
	c := make(chan prometheus.Metric)
	go func(c chan prometheus.Metric) {
		col.Collect(c)
		close(c)
	}(c)
	for x := range c { // eg range across distinct label vector values
		m := dto.Metric{}
		_ = x.Write(&m)
		do(&m)
	}
}

func Test_updateIntegrationPhase(t *testing.T) {
	type args struct {
		iId      string
		p        string
		expected float64
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test should fail with empty id", args: args{
				iId:      "",
				p:        "running",
				expected: 0,
			},
		},
		{
			name: "test should fail with empty phase", args: args{
				iId:      "int-1",
				p:        "",
				expected: 0,
			},
		},
		{
			name: "test should pass and increase the counter", args: args{
				iId:      "int-1",
				p:        "running",
				expected: 1,
			},
		},
	}
	for _, tt := range tests {
		//		integration.Reset()
		t.Run(tt.name, func(t *testing.T) {
			updateIntegrationPhase(tt.args.iId, tt.args.p)
			labels := map[string]string{"phase": tt.args.p, "id": tt.args.iId}
			if i, err := integration.GetMetricWith(labels); err == nil {
				val := getMetricValue(i)
				assert.Equal(t, val, tt.args.expected)
			}
		})
	}
}
