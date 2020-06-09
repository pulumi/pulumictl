package github

import (
	"context"
	"net/http"

	"github.com/google/go-github/v32/github"
	"golang.org/x/oauth2"
)

var (
	tokenClient *http.Client
)

func CreateGithubClient(token string) (context.Context, *github.Client) {
	ctx := context.Background()

	tokenClient = nil
	if token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		tokenClient = oauth2.NewClient(ctx, ts)
	}
	return ctx, github.NewClient(tokenClient)
}
