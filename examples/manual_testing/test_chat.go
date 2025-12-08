package main

import (
	"context"
	"fmt"
	"github.com/guild-framework/guild-core/pkg/config"
	"github.com/guild-framework/guild-core/pkg/gerror"
)

func main() {
	fmt.Println("Testing error detection...")

	_, err := config.LoadGuildConfig(context.Background(), ".")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Printf("Error code: %v\n", gerror.GetCode(err))
		fmt.Printf("Is NotFound: %v\n", gerror.GetCode(err) == gerror.ErrCodeNotFound)

		// Check wrapped error
		if gerror.GetCode(err) == gerror.ErrCodeNotFound {
			fmt.Println("✅ Correctly detected NotFound error")
		}
	}
}
