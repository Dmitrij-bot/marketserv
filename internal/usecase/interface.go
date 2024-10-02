package usecase

import "context"

type Interface interface {
	FindClientByUsername(ctx context.Context, req FindClientByUsernameRequest) (resp FindClientByUsernameResponse, err error)
}
