package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	p "github.com/pulumi/pulumi-go-provider"
	infer "github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi-go-provider/middleware/schema"
	service "github.com/pulumi/pulumi-pulumiservice/sdk/go/pulumiservice"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func Provider() p.Provider {
	return infer.Provider(infer.Options{
		Metadata: schema.Metadata{
			DisplayName: "Pulumi Service Deployment",
			Publisher:   "iwahbe",
		},
		Components: []infer.InferredComponent{
			infer.Component[*GitHub, GitHubArgs, GitHub](),
		},
		ModuleMap: map[tokens.ModuleName]tokens.ModuleName{
			"provider": "index",
		},
	})
}

type GitHub struct {
	pulumi.ComponentResource
	DeploymentID pulumi.StringOutput `pulumi:"DeploymentID"`
}

func (d *GitHub) Annotate(a infer.Annotator) {
	a.Describe(d, `A [deployment](https://www.pulumi.com/docs/pulumi-cloud/deployments/) in the Pulumi service with all the defaults.

The deployment is set to trigger on the main branch of the project this resource is in.

This resource must be used from a GitHub Repo.`)

	a.Describe(&d.DeploymentID, "The ID of the resulting deployment")
}

type GitHubArgs struct{}

func (d *GitHub) Construct(ctx *pulumi.Context, name, typ string, inputs GitHubArgs, opts pulumi.ResourceOption) (GitHub, error) {
	ctx.RegisterComponentResource(typ, name, d, opts)

	gh, err := getCurrentGHRepo(ctx.Context())
	if err != nil {
		return *d, err
	}

	defaultBranch, err := getDefaultBranch(ctx.Context(), gh)
	if err != nil {
		return *d, err
	}

	repoDir, err := getRepoDir(ctx.Context())
	if err != nil {
		return *d, err
	}

	s, err := service.NewDeploymentSettings(ctx, "deployment", &service.DeploymentSettingsArgs{
		Organization: pulumi.String(ctx.Organization()),
		Project:      pulumi.String(ctx.Project()),
		Stack:        pulumi.String(ctx.Stack()),
		Github: &service.DeploymentSettingsGithubArgs{
			Repository: pulumi.String(gh),
		},
		SourceContext: &service.DeploymentSettingsSourceContextArgs{
			Git: &service.DeploymentSettingsGitSourceArgs{
				Branch:  pulumi.String(defaultBranch),
				RepoDir: pulumi.String(repoDir),
			},
		},
	}, pulumi.Parent(d))
	if err != nil {
		return *d, err
	}

	d.DeploymentID = s.ID().ApplyT(identity[string]).(pulumi.StringOutput)

	return *d, nil
}

func identity[T any](t T) T { return t }

func getCurrentGHRepo(ctx context.Context) (string, error) {
	var stdout bytes.Buffer
	c := exec.CommandContext(ctx, "git", "--get", "remote.origin.url")
	c.Stdout = &stdout
	if err := c.Run(); err != nil {
		return "", err
	}

	const (
		prefix = "https://github.com/"
		suffix = ".git"
	)

	rest, ok := strings.CutPrefix(stdout.String(), prefix)
	if !ok {
		return "", fmt.Errorf(
			"failed to find the current GH repo: missing prefix %q found %q",
			prefix, stdout.String())
	}

	gh, ok := strings.CutSuffix(rest, suffix)
	if !ok {
		return "", fmt.Errorf("failed to find the current GH repo: missing suffix %q found %q",
			suffix, stdout.String())
	}
	return gh, nil
}

func getDefaultBranch(ctx context.Context, repo string) (string, error) {
	var stdout bytes.Buffer
	c := exec.CommandContext(ctx, "gh", "repo", repo, "view", "--json", "defaultBranchRef")
	c.Stdout = &stdout
	if err := c.Run(); err != nil {
		return "", err
	}

	var result struct {
		DefaultBranchRef struct {
			Name string `json:"name"`
		} `json:"defaultBranchRef"`
	}

	err := json.Unmarshal(stdout.Bytes(), &result)
	return result.DefaultBranchRef.Name, err
}

func getRepoDir(ctx context.Context) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	var path string

	for {
		dir, err := os.ReadDir(cwd)
		if err != nil {
			return "", err
		}

		for _, d := range dir {
			if d.Name() == ".git" && d.IsDir() {
				return path, nil
			}
		}
		var component string
		cwd, component = filepath.Split(cwd)
		path = filepath.Join(component, path)

	}
}
