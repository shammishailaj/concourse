package v2

import (
	"context"

	"github.com/concourse/concourse/atc"
	res "github.com/concourse/concourse/atc/resource"
	"github.com/concourse/concourse/atc/worker"
)

type getRequest struct {
	Source  atc.Source  `json:"source"`
	Params  atc.Params  `json:"params,omitempty"`
	Version atc.Version `json:"version,omitempty"`
}

func (r *resource) Get(
	ctx context.Context,
	volume worker.Volume,
	ioConfig res.IOConfig,
	source atc.Source,
	params atc.Params,
	version atc.Version,
) (res.VersionedSource, error) {
	var vr res.VersionResult

	err := res.RunScript(
		ctx,
		"/opt/resource/in",
		[]string{res.ResourcesDir("get")},
		getRequest{source, params, version},
		&vr,
		ioConfig.Stderr,
		true,
		r.container,
	)
	if err != nil {
		return nil, err
	}

	return res.NewGetVersionedSource(volume, vr.Version, vr.Metadata), nil
}
