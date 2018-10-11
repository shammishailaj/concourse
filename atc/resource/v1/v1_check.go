package v1

import (
	"context"

	"github.com/concourse/concourse/atc"
)

type checkRequest struct {
	Source  atc.Source  `json:"source"`
	Version atc.Version `json:"version"`
}

func (r *Resource) Check(ctx context.Context, source atc.Source, fromVersion atc.Version) ([]atc.Version, error) {
	var versions []atc.Version

	err := r.runScript(
		ctx,
		"/opt/resource/check",
		nil,
		checkRequest{source, fromVersion},
		&versions,
		nil,
		false,
	)
	if err != nil {
		return nil, err
	}

	return versions, nil
}
