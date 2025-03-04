package kraki

import (
	"context"

	"github.com/trichner/oauthflows"
	"github.com/trichner/tb/pkg/directory"
)

func deleteUser(email string) error {
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

	svc, err := directory.NewService(ctx, tokenSource)
	if err != nil {
		return err
	}

	user, err := svc.DeleteUserByPrimaryEmail(ctx, email)
	if err != nil {
		return err
	}

	return printJson(user)
}
