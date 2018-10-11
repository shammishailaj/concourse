package v2

import (
	res "github.com/concourse/concourse/atc/resource"
	"github.com/concourse/concourse/atc/worker"
)

type resource struct {
	container worker.Container
	info      res.ResourceInfo

	ScriptFailure bool
}

func NewResource(container worker.Container, info res.ResourceInfo) res.Resource {
	return &resource{
		container: container,
		info:      info,
	}
}

func (r *resource) Container() worker.Container {
	return r.container
}
