package main

import (
	p "github.com/pulumi/pulumi-go-provider"

	"github.com/iwahbe/pulumi-deployment/provider"
	"github.com/iwahbe/pulumi-deployment/version"
)

func main() {
	p.RunProvider("deployment", version.Version, provider.Provider())
}
