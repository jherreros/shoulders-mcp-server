package bootstrap

import (
	"strings"
	"testing"

	"sigs.k8s.io/yaml"
)

func TestCiliumHelmReleaseDeletePatchIsValidYAML(t *testing.T) {
	patch := ciliumHelmReleaseDeletePatch()
	if strings.Contains(patch, "\t") {
		t.Fatalf("delete patch should not contain tabs: %q", patch)
	}

	var parsed map[string]interface{}
	if err := yaml.Unmarshal([]byte(patch), &parsed); err != nil {
		t.Fatalf("delete patch should be valid YAML: %v\n%s", err, patch)
	}
}
