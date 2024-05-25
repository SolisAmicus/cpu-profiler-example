package collector

import (
	"context"
	"runtime/pprof"
)

func CtxWithLabel(ctx context.Context, label string) context.Context {
	return pprof.WithLabels(ctx, pprof.Labels(labelKey, label))
}

func CtxWithLabels(ctx context.Context, labels []string) context.Context {
	labelPairs := []string{}
	for _, label := range labels {
		labelPairs = append(labelPairs, labelKey, label)
	}
	return pprof.WithLabels(ctx, pprof.Labels(labelPairs...))
}
