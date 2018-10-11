package resource

import (
	"context"

	"github.com/concourse/concourse/atc/resource/v1"
	"github.com/concourse/concourse/atc/worker"
)

type ResourceInfo struct {
	Artifacts Artifacts
}

type Artifacts struct {
	APIVersion string `json:"api_version"`
	Check      string `json:"check"`
	Get        string `json:"get"`
	Put        string `json:"put"`
}

// XXX: better name?
type UnversionedResource interface {
	Info(context.Context) (ResourceInfo, error)
}

type unversionedResource struct {
	container worker.Container
}

func NewUnversionedResource(container worker.Container) UnversionedResource {
	return &unversionedResource{
		container: container,
	}
}

func (resource *unversionedResource) Info(ctx context.Context) (ResourceInfo, error) {
	var info ResourceInfo
	err := v1.RunScript(
		ctx,
		"/info",
		nil,
		nil,
		&info,
		nil,
		false,
		resource.container,
	)
	if err != nil {
		return ResourceInfo{}, err
	}

	return info, nil
}
