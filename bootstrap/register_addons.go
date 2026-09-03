package bootstrap

// Side-effect imports so remaining in-tree service packages register themselves.
// Addon packages live in github.com/zatrano/packages and must be blank-imported
// by the consumer application.
import (
	_ "github.com/zatrano/framework/packages/ai"
)
