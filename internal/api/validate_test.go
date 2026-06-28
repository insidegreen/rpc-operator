package api_test

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rpcv1alpha1 "github.com/insidegreen/rpc-operator-claude/api/v1alpha1"
	"github.com/insidegreen/rpc-operator-claude/internal/api"
)

func TestValidate_RawYAML_Valid(t *testing.T) {
	p := &rpcv1alpha1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{Name: "raw-test"},
		Spec: rpcv1alpha1.PipelineSpec{
			RawYAML: "input:\n  generate:\n    mapping: 'root = \"hi\"'\n    interval: 1s\noutput:\n  stdout: {}\n",
		},
	}
	errs := api.ValidatePipeline(p)
	if len(errs) != 0 {
		t.Errorf("expected no errors for valid rawYAML, got %v", errs)
	}
}

func TestValidate_RawYAML_InvalidYAML(t *testing.T) {
	p := &rpcv1alpha1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{Name: "raw-test"},
		Spec: rpcv1alpha1.PipelineSpec{
			RawYAML: "{invalid: yaml: [",
		},
	}
	errs := api.ValidatePipeline(p)
	if len(errs) == 0 {
		t.Fatal("expected ValidationError for invalid YAML")
	}
	if errs[0].Path != "spec.rawYAML" {
		t.Errorf("expected path spec.rawYAML, got %q", errs[0].Path)
	}
}

func TestValidate_RawYAML_NotAMapping(t *testing.T) {
	p := &rpcv1alpha1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{Name: "raw-test"},
		Spec: rpcv1alpha1.PipelineSpec{
			RawYAML: "- item1\n- item2\n",
		},
	}
	errs := api.ValidatePipeline(p)
	if len(errs) == 0 {
		t.Fatal("expected ValidationError for non-mapping YAML")
	}
	if errs[0].Path != "spec.rawYAML" {
		t.Errorf("expected path spec.rawYAML, got %q", errs[0].Path)
	}
}

func TestValidate_RawYAML_Required(t *testing.T) {
	p := &rpcv1alpha1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{Name: "raw-test"},
		Spec:       rpcv1alpha1.PipelineSpec{},
	}
	errs := api.ValidatePipeline(p)
	if len(errs) == 0 {
		t.Fatal("expected ValidationError for missing rawYAML")
	}
	if errs[0].Path != "spec.rawYAML" {
		t.Errorf("expected path spec.rawYAML, got %q", errs[0].Path)
	}
}

func TestValidateSecretRefs_Valid(t *testing.T) {
	p := &rpcv1alpha1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec: rpcv1alpha1.PipelineSpec{
			RawYAML: "input:\n  stdin: {}\noutput:\n  stdout: {}",
			SecretRefs: []rpcv1alpha1.SecretRef{
				{EnvVar: "MY_PASSWORD", SecretName: "my-secret", Key: "password"},
			},
		},
	}
	errs := api.ValidatePipeline(p)
	if len(errs) != 0 {
		t.Errorf("expected no errors for valid secretRef, got %v", errs)
	}
}

func TestValidateSecretRefs_EmptyFields(t *testing.T) {
	p := &rpcv1alpha1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec: rpcv1alpha1.PipelineSpec{
			RawYAML: "input:\n  stdin: {}\noutput:\n  stdout: {}",
			SecretRefs: []rpcv1alpha1.SecretRef{
				{EnvVar: "", SecretName: "", Key: ""},
			},
		},
	}
	errs := api.ValidatePipeline(p)
	paths := make(map[string]bool)
	for _, e := range errs {
		paths[e.Path] = true
	}
	for _, want := range []string{
		"spec.secretRefs[0].envVar",
		"spec.secretRefs[0].secretName",
		"spec.secretRefs[0].key",
	} {
		if !paths[want] {
			t.Errorf("expected error at path %q, got errors: %v", want, errs)
		}
	}
}

func TestValidateSecretRefs_InvalidEnvVar(t *testing.T) {
	p := &rpcv1alpha1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec: rpcv1alpha1.PipelineSpec{
			RawYAML: "input:\n  stdin: {}\noutput:\n  stdout: {}",
			SecretRefs: []rpcv1alpha1.SecretRef{
				{EnvVar: "123_INVALID", SecretName: "s", Key: "k"},
			},
		},
	}
	errs := api.ValidatePipeline(p)
	if len(errs) == 0 {
		t.Error("expected error for invalid envVar name, got none")
	}
	if errs[0].Path != "spec.secretRefs[0].envVar" {
		t.Errorf("expected path spec.secretRefs[0].envVar, got %q", errs[0].Path)
	}
}

func TestValidateSecretRefs_Duplicate(t *testing.T) {
	p := &rpcv1alpha1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec: rpcv1alpha1.PipelineSpec{
			RawYAML: "input:\n  stdin: {}\noutput:\n  stdout: {}",
			SecretRefs: []rpcv1alpha1.SecretRef{
				{EnvVar: "MY_VAR", SecretName: "s1", Key: "k1"},
				{EnvVar: "MY_VAR", SecretName: "s2", Key: "k2"},
			},
		},
	}
	errs := api.ValidatePipeline(p)
	found := false
	for _, e := range errs {
		if e.Path == "spec.secretRefs[1].envVar" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected duplicate envVar error at spec.secretRefs[1].envVar, got: %v", errs)
	}
}
