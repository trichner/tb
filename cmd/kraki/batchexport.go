package kraki

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/trichner/oauthflows"
	"github.com/trichner/tb/pkg/directory"
	vault2 "github.com/trichner/tb/pkg/vault"
)

func batchExport(filename string, resources []ExportResource) error {
	ctx := context.Background()

	config, err := getOAuth2Config()
	if err != nil {
		return err
	}
	config.Scopes = scopes

	tokenSource, err := oauthflows.NewBrowserFlowTokenSource(ctx, config)
	if err != nil {
		return err
	}

	directoryService, err := directory.NewService(ctx, tokenSource)
	if err != nil {
		return err
	}

	vaultService, err := vault2.NewService(ctx, tokenSource)
	if err != nil {
		return err
	}

	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("cannot read batch file %q: %w", filename, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// optionally, resize scanner's capacity for lines over 64K, see next example
	for scanner.Scan() {
		email := scanner.Text()
		if email == "" || strings.HasPrefix(email, "#") {
			log.Printf("skipping %s", email)
			continue
		}
		log.Printf("exporting %q", email)
		matter, _, err := doUserExport(ctx, email, resources, directoryService, vaultService)
		if err != nil {
			return fmt.Errorf("failed to export %q: %w", email, err)
		}
		err = writeState(matter)
		if err != nil {
			return fmt.Errorf("failed to write state for %q: %w", email, err)
		}
	}

	return nil
}

func writeState(matter *vault2.Matter) error {
	data, err := json.MarshalIndent(matter, "", " ")
	if err != nil {
		return err
	}

	filename := "matter_" + matter.Name + ".json"
	return os.WriteFile(filename, data, 0o644)
}
