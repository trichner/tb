package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"

	"github.com/trichner/tb/cmd/csv2json"
	"github.com/trichner/tb/cmd/sheet2json"
	"github.com/trichner/tb/pkg/cmdreg"

	"github.com/trichner/tb/cmd/sql2json"

	"github.com/trichner/tb/cmd/jiracli"
	"github.com/trichner/tb/cmd/json2sheet"
	"github.com/trichner/tb/cmd/kraki"
)

func main() {
	tintOpts := &tint.Options{
		Level:      new(slog.LevelVar),
		TimeFormat: time.TimeOnly,
	}
	slog.SetDefault(slog.New(tint.NewHandler(os.Stderr, tintOpts)))

	r := cmdreg.New(cmdreg.WithProgramName("tb"))

	r.RegisterFunc("csv2json", csv2json.Exec)
	r.RegisterFunc("jiracli", jiracli.Exec)
	r.RegisterFunc("json2sheet", json2sheet.Exec)
	r.RegisterFunc("kraki", kraki.Exec)
	r.RegisterFunc("sheet2json", sheet2json.Exec, cmdreg.WithCompletion(sheet2json.Completions()))
	r.RegisterFunc("sql2json", sql2json.Exec)

	r.RegisterFunc("help", help(r))

	ctx := context.Background()
	r.Exec(ctx, os.Args)
}

func help(r *cmdreg.CommandRegistry) cmdreg.CommandFunc {
	return func(_ context.Context, args []string) {
		r.PrintHelp(os.Stdout)
	}
}
