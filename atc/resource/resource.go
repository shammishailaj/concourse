package resource

import (
	"context"

	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/atc/db"
	"github.com/concourse/concourse/atc/resource/source"
	"github.com/concourse/concourse/atc/worker"
)

//go:generate counterfeiter . Resource

type Resource interface {
	Get(context.Context, worker.Volume, source.IOConfig, atc.Source, atc.Params, atc.Version) (source.VersionedSource, error)
	Put(context.Context, source.IOConfig, atc.Source, atc.Params) (source.VersionedSource, error)
	Check(context.Context, atc.Source, atc.Version) ([]atc.Version, error)
	Container() worker.Container
}

type ResourceType string

type Session struct {
	Metadata db.ContainerMetadata
}

type Metadata interface {
	Env() []string
}
