package kraki

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/trichner/oauthflows"
	"github.com/trichner/tb/pkg/directory"
)

func batchDelete(filename string) error {
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
		log.Printf("deleting %q", email)

		user, err := directoryService.DeleteUserByPrimaryEmail(ctx, email)
		if err != nil {
			return err
		}
		log.Printf("deleted %q", user.PrimaryEmail)
	}

	return nil
}
