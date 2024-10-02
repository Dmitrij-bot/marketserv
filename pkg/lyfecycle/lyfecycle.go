package lyfecycle

import "context"

type Lyfecycle interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}
