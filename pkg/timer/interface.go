package timer

import "context"

type Timer interface {
	Validate(ctx context.Context) error
	Start(ctx context.Context) error
}
