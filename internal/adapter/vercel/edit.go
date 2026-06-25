package vercel

import (
	"context"

	"github.com/ncxton/potaco/internal/adapter"
)

// Edit returns ErrEditNotSupported because Vercel AI Gateway does not support image editing.
func (a *Adapter) Edit(context.Context, adapter.EditRequest) (*adapter.GenerateResponse, error) {
	return nil, adapter.ErrEditNotSupported
}
