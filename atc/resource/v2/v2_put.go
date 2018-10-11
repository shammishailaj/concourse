package v2

import (
	"context"

	"github.com/concourse/concourse/atc"
	res "github.com/concourse/concourse/atc/resource"
)

type putRequest struct {
	Source atc.Source `json:"source"`
	Params atc.Params `json:"params,omitempty"`
}

func (r *resource) Put(
	ctx context.Context,
	ioConfig res.IOConfig,
	source atc.Source,
	params atc.Params,
) (res.VersionedSource, error) {
	resourceDir := res.ResourcesDir("put")

	var versionResult res.VersionResult
	err := res.RunScript(
		ctx,
		"/opt/resource/out",
		[]string{resourceDir},
		putRequest{
			Params: params,
			Source: source,
		},
		&versionResult,
		ioConfig.Stderr,
		true,
		r.container,
	)
	if err != nil {
		return nil, err
	}

	return res.NewPutVersionedSource(versionResult, r.container, resourceDir), nil
}
