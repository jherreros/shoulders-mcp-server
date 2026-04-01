// Package manifests embeds the platform manifests so the CLI works standalone.
//
// The embedded files are copied from canonical sources in the repo root.
// Run "go generate ./..." from shoulders-cli/ to refresh them before building.
package manifests

import _ "embed"

//go:generate mkdir -p flux
//go:generate cp ../../../1-cluster/vind-config.yaml vind-config.yaml
//go:generate cp ../../../1-cluster/authentication-config.yaml authentication-config.yaml
//go:generate cp ../../../2-addons/flux/git-repository.yaml flux/git-repository.yaml
//go:generate cp ../../../2-addons/flux/kustomizations.yaml flux/kustomizations.yaml
//go:generate cp ../../../2-addons/manifests/crds/gateway-api.yaml gateway-api-crds.yaml

//go:embed vind-config.yaml
var VindConfig []byte

//go:embed authentication-config.yaml
var AuthenticationConfig []byte

//go:embed flux/git-repository.yaml
var FluxGitRepository []byte

//go:embed flux/kustomizations.yaml
var FluxKustomizations []byte

//go:embed gateway-api-crds.yaml
var GatewayAPICRDs []byte
