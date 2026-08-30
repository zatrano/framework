package rag

import (
	"context"

	"github.com/zatrano/framework/packages/ai"
)

// FromAI returns an Embedder that calls ai.Manager.Embed (optional model override).
func FromAI(mgr *ai.Manager, model string, provider ...string) Embedder {
	return FuncEmbedder(func(ctx context.Context, texts []string) ([][]float64, error) {
		req := ai.EmbedRequest{Model: model, Input: texts}
		var resp *ai.EmbedResponse
		var err error
		if len(provider) > 0 && provider[0] != "" {
			resp, err = mgr.Using(provider[0]).Embed(ctx, req)
		} else {
			resp, err = mgr.Embed(ctx, req)
		}
		if err != nil {
			return nil, err
		}
		return resp.Embeddings, nil
	})
}
