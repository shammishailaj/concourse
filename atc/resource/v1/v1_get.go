package v1

import (
	"context"

	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/atc/resource/source"
	"github.com/concourse/concourse/atc/worker"
)

type getRequest struct {
	Source  atc.Source  `json:"source"`
	Params  atc.Params  `json:"params,omitempty"`
	Version atc.Version `json:"version,omitempty"`
}

func (r *Resource) Get(
	ctx context.Context,
	volume worker.Volume,
	ioConfig source.IOConfig,
	src atc.Source,
	params atc.Params,
	version atc.Version,
) (source.VersionedSource, error) {
	var vr source.VersionResult

	err := r.runScript(
		ctx,
		"/opt/resource/in",
		[]string{source.ResourcesDir("get")},
		getRequest{src, params, version},
		&vr,
		ioConfig.Stderr,
		true,
	)
	if err != nil {
		return nil, err
	}

	return source.NewGetVersionedSource(volume, vr.Version, vr.Metadata), nil
}
