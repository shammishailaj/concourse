package v1

import (
	"context"

	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/atc/resource/source"
)

type putRequest struct {
	Source atc.Source `json:"source"`
	Params atc.Params `json:"params,omitempty"`
}

func (r *Resource) Put(
	ctx context.Context,
	ioConfig source.IOConfig,
	src atc.Source,
	params atc.Params,
) (source.VersionedSource, error) {
	resourceDir := source.ResourcesDir("put")

	var versionResult source.VersionResult
	err := r.runScript(
		ctx,
		"/opt/resource/out",
		[]string{resourceDir},
		putRequest{
			Params: params,
			Source: src,
		},
		&versionResult,
		ioConfig.Stderr,
		true,
	)
	if err != nil {
		return nil, err
	}

	return source.NewPutVersionedSource(versionResult, r.container, resourceDir), nil
}
