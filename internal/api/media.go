package api

import (
	"context"

	"github.com/ppxb/miyabi/internal/domain"
)

// ImageReader serves preview URLs already stored in imported metadata.
type ImageReader interface {
	Image(context.Context, domain.ImageCandidate) (domain.Media, error)
}
