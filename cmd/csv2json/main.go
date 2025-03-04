package csv2json

import (
	"context"
	"log"
	"os"

	c2j "github.com/trichner/tb/pkg/csv2json"
)

func Exec(ctx context.Context, args []string) {
	err := c2j.Convert(os.Stdin, os.Stdout)
	if err != nil {
		log.Fatalf("cannot convert csv to json: %v", err)
	}
}
