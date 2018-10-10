package ocgin

import "go.opencensus.io/tag"

type addedTagsKey struct{}

type addedTags struct {
	t []tag.Mutator
}
