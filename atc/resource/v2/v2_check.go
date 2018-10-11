package v2

import (
	"context"

	"github.com/concourse/concourse/atc"
	res "github.com/concourse/concourse/atc/resource"
)

// CheckResponse contains a default space and a list of resource versions. This response is returned from the check of the resource v2 interface. The default space can be empty for resources that do not have a default space (ex. PR resource).
type CheckResponse struct {
	DefaultSpace string `json:"default_space"`
	Versions     []ResourceVersion
}

type ResourceVersion struct {
	Space    string              `json:"space"`
	Version  atc.Version         `json:"version"`
	Metadata []atc.MetadataField `json:"metadata"`
}

type checkRequest struct {
	Source  atc.Source  `json:"source"`
	Version atc.Version `json:"version"`
}

func (r *resource) Check(ctx context.Context, source atc.Source, fromVersion atc.Version) ([]atc.Version, error) {
	var versions []atc.Version

	err := res.RunScript(
		ctx,
		r.info.Artifacts.Check,
		nil,
		checkRequest{source, fromVersion},
		&versions,
		nil,
		false,
		r.container,
	)
	if err != nil {
		return nil, err
	}

	return versions, nil
}
