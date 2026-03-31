package store

import (
	"context"

	"github.com/roversx/repodock/internal/domain"
)

type ProjectStore interface {
	ListProjects(ctx context.Context) ([]domain.Project, error)
}

type MemoryProjectStore struct {
	Projects []domain.Project
}

func (s MemoryProjectStore) ListProjects(context.Context) ([]domain.Project, error) {
	return append([]domain.Project(nil), s.Projects...), nil
}
