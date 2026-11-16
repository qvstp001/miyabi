package metadata

import (
	"context"

	"github.com/ppxb/miyabi/internal/domain"
)

// MediaSource only reads images referenced by existing metadata.
type MediaSource interface {
	ID() string
	Media(context.Context, string) (domain.Media, error)
}

// ImageReader has no metadata lookup, settings, database, or background work.
type ImageReader struct {
	sources  map[string]MediaSource
	capacity chan struct{}
}

func NewImageReader(sources ...MediaSource) *ImageReader {
	reader := &ImageReader{sources: make(map[string]MediaSource), capacity: make(chan struct{}, 8)}
	for _, source := range sources {
		reader.sources[source.ID()] = source
	}
	return reader
}

func (reader *ImageReader) Image(ctx context.Context, candidate domain.ImageCandidate) (domain.Media, error) {
	source := reader.sources[candidate.Provider]
	if source == nil {
		return domain.Media{}, domain.E(domain.KindInvalid, "未知的图片来源", nil)
	}
	select {
	case reader.capacity <- struct{}{}:
	case <-ctx.Done():
		return domain.Media{}, ctx.Err()
	}
	defer func() { <-reader.capacity }()
	return source.Media(ctx, candidate.URL)
}

func (reader *ImageReader) Close() {
	for _, source := range reader.sources {
		if closer, ok := source.(interface{ Close() }); ok {
			closer.Close()
		}
	}
}
