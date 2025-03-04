package kraki

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/trichner/oauthflows"
	"github.com/trichner/tb/pkg/directory"
	vault2 "github.com/trichner/tb/pkg/vault"
)

func exportUser(email string, resources []ExportResource) error {
	ctx := context.Background()

	config, err := getOAuth2Config()
	if err != nil {
		return err
	}
	config.Scopes = scopes

	src, err := oauthflows.NewBrowserFlowTokenSource(ctx, config)
	if err != nil {
		return err
	}

	directoryService, err := directory.NewService(ctx, src)
	if err != nil {
		return err
	}

	vaultService, err := vault2.NewService(ctx, src)
	if err != nil {
		return err
	}

	_, exports, err := doUserExport(ctx, email, resources, directoryService, vaultService)
	if err != nil {
		return fmt.Errorf("failed to export user %q: %w", email, err)
	}

	bytes, err := json.MarshalIndent(exports, "", "  ")
	fmt.Printf("%s", bytes)

	return nil
}

func doUserExport(ctx context.Context, email string, resources []ExportResource, directoryService *directory.Service, vaultService *vault2.VaultService) (*vault2.Matter, []*vault2.Export, error) {
	user, err := directoryService.FindUserByPrimaryEmail(ctx, email)
	if err != nil {
		return nil, nil, err
	}

	matter, err := vaultService.CreateMatter(ctx, user.PrimaryEmail)
	if err != nil {
		return nil, nil, err
	}

	matterId := matter.Id

	exports := []*vault2.Export{}

	if contains(resources, ResourceDrive) {
		driveExport, err := vaultService.CreateDriveExportForMatter(ctx, matterId, email)
		exports = append(exports, driveExport)
		if err != nil {
			return nil, nil, err
		}
	}

	if contains(resources, ResourceEmail) {
		emailExport, err := vaultService.CreateEmailExportForMatter(ctx, matterId, email)
		exports = append(exports, emailExport)
		if err != nil {
			return nil, nil, err
		}
	}

	return matter, exports, nil
}

func contains(r []ExportResource, e ExportResource) bool {
	for _, n := range r {
		if n == e {
			return true
		}
	}
	return false
}
