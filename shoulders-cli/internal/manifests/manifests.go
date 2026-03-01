// Package manifests embeds the platform manifests so the CLI works standalone.
package manifests

import _ "embed"

//go:embed kind-config.yaml
var KindConfig []byte

//go:embed authentication-config.yaml
var AuthenticationConfig []byte

//go:embed flux/git-repository.yaml
var FluxGitRepository []byte

//go:embed flux/kustomizations.yaml
var FluxKustomizations []byte
