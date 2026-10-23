package metrics_test

import (
	"runtime"
	"testing"

	"github.com/revirathya/revirathya-finance-core/internal/platform/metrics"
	"github.com/revirathya/revirathya-finance-core/testutil"
)

func TestBuildInfo(t *testing.T) {
	m := metrics.NewMetrics(nil)
	m.SetBuildInfo("0.2.0")

	labels := map[string]string{"version": "0.2.0", "go_version": runtime.Version()}
	if v, ok := testutil.MetricValue(t, m.Registry, "financecore_build_info", labels); !ok || v != 1 {
		t.Fatalf("financecore_build_info%v = %v (exported: %v), want 1", labels, v, ok)
	}
}
