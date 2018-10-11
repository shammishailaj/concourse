package v1

import (
	"github.com/concourse/concourse/atc/worker"
)

type Resource struct {
	container worker.Container

	ScriptFailure bool
}

func NewResource(container worker.Container) *Resource {
	return &Resource{
		container: container,
	}
}

func (r *Resource) Container() worker.Container {
	return r.container
}
