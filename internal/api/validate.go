package api

import (
	"fmt"
	"regexp"

	"sigs.k8s.io/yaml"

	rpcv1alpha1 "github.com/insidegreen/rpc-operator-claude/api/v1alpha1"
	"github.com/insidegreen/rpc-operator-claude/internal/projectroute"
	"github.com/insidegreen/rpc-operator-claude/internal/render"
)

var envVarNameRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ValidationError describes a single schema or render validation failure.
type ValidationError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// ValidatePipeline checks a Pipeline: rawYAML must be present and render-valid;
// projectRef and clusterRef are mutually exclusive; secretRefs must be well-formed.
func ValidatePipeline(p *rpcv1alpha1.Pipeline) []ValidationError {
	if p.Spec.ProjectRef != nil && p.Spec.ClusterRef != "" {
		return []ValidationError{{Path: "spec.projectRef", Message: "use projectRef or clusterRef, not both"}}
	}
	var errs []ValidationError
	if p.Spec.RawYAML == "" {
		errs = append(errs, ValidationError{Path: "spec.rawYAML", Message: "rawYAML is required"})
	} else if _, err := render.RenderPipelineYAML(&p.Spec); err != nil {
		errs = append(errs, ValidationError{Path: "spec.rawYAML", Message: err.Error()})
	}
	errs = append(errs, validateSecretRefs(p.Spec.SecretRefs)...)
	return errs
}

// validateSecretRefs checks that every SecretRef has valid, non-duplicate fields.
func validateSecretRefs(refs []rpcv1alpha1.SecretRef) []ValidationError {
	var errs []ValidationError
	seen := map[string]bool{}
	for i, r := range refs {
		path := fmt.Sprintf("spec.secretRefs[%d]", i)
		if r.EnvVar == "" {
			errs = append(errs, ValidationError{Path: path + ".envVar", Message: "envVar is required"})
		} else if !envVarNameRe.MatchString(r.EnvVar) {
			errs = append(errs, ValidationError{Path: path + ".envVar", Message: "envVar must match [A-Za-z_][A-Za-z0-9_]*"})
		} else if seen[r.EnvVar] {
			errs = append(errs, ValidationError{Path: path + ".envVar", Message: fmt.Sprintf("duplicate envVar %q", r.EnvVar)})
		}
		seen[r.EnvVar] = true
		if r.SecretName == "" {
			errs = append(errs, ValidationError{Path: path + ".secretName", Message: "secretName is required"})
		}
		if r.Key == "" {
			errs = append(errs, ValidationError{Path: path + ".key", Message: "key is required"})
		}
	}
	return errs
}

// ValidateProject validates a project's route graph against the given pipelines
// (all pipelines in the project's namespace). Returns nil when valid.
func ValidateProject(p *rpcv1alpha1.PipelineProject, pipelines []rpcv1alpha1.Pipeline) []ValidationError {
	views := make(map[string]projectroute.PipelineView, len(pipelines))
	for i := range pipelines {
		pp := &pipelines[i]
		proj := ""
		if pp.Spec.ProjectRef != nil {
			proj = pp.Spec.ProjectRef.Name
		}
		views[pp.Name] = projectroute.PipelineView{
			Name:        pp.Name,
			ProjectName: proj,
			HasInput:    rawTopKey(pp.Spec.RawYAML, "input"),
			HasOutput:   rawTopKey(pp.Spec.RawYAML, "output"),
		}
	}
	verrs := projectroute.ValidateProject(p, views)
	out := make([]ValidationError, 0, len(verrs))
	for _, e := range verrs {
		out = append(out, ValidationError{Path: "spec.routes", Message: e.Message})
	}
	return out
}

// rawTopKey reports whether rawYAML parses to a mapping containing key.
func rawTopKey(rawYAML, key string) bool {
	var m map[string]any
	if err := yaml.Unmarshal([]byte(rawYAML), &m); err != nil {
		return false
	}
	_, ok := m[key]
	return ok
}
