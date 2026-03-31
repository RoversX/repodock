package sources

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/roversx/repodock/internal/domain"
)

type Provider interface {
	Name() string
	Load(ctx context.Context) ([]domain.Project, error)
}

type Registry struct {
	providers []Provider
}

func NewRegistry(providers ...Provider) Registry {
	return Registry{providers: providers}
}

func (r Registry) Providers() []Provider {
	return append([]Provider(nil), r.providers...)
}

// LoadAll loads projects from all providers in best-effort mode: a failing
// provider is skipped and its error is collected rather than aborting the
// whole load. Callers receive the partial merged project list alongside any
// per-provider errors.
func LoadAll(ctx context.Context, providers ...Provider) ([]domain.Project, []error) {
	merged := make([]domain.Project, 0)
	indexByPath := make(map[string]int)
	var errs []error

	for _, provider := range providers {
		projects, err := provider.Load(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", provider.Name(), err))
			continue
		}

		for _, project := range projects {
			key := filepath.Clean(project.Path)
			if index, ok := indexByPath[key]; ok {
				merged[index].Sources = mergeSources(merged[index].Sources, project.Sources)
				continue
			}

			indexByPath[key] = len(merged)
			project.Path = key
			merged = append(merged, project)
		}
	}

	return merged, errs
}

func mergeSources(current, incoming []domain.Source) []domain.Source {
	out := append([]domain.Source(nil), current...)
	for _, source := range incoming {
		if slices.Contains(out, source) {
			continue
		}
		out = append(out, source)
	}
	return out
}
