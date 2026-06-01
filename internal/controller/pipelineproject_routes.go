/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package controller

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	rpcv1alpha1 "github.com/insidegreen/rpc-operator-claude/api/v1alpha1"
	"github.com/insidegreen/rpc-operator-claude/internal/nats"
	"github.com/insidegreen/rpc-operator-claude/internal/projectroute"
)

// validateProjectRoutes lists the namespace's pipelines and validates the
// project's route graph. Returns the first error message (verbatim from the
// spec) or "" when valid.
func (r *PipelineProjectReconciler) validateProjectRoutes(ctx context.Context, project *rpcv1alpha1.PipelineProject) (string, error) {
	var pipes rpcv1alpha1.PipelineList
	if err := r.List(ctx, &pipes, client.InNamespace(project.Namespace)); err != nil {
		return "", err
	}
	views := make(map[string]projectroute.PipelineView, len(pipes.Items))
	for i := range pipes.Items {
		p := &pipes.Items[i]
		proj := ""
		if p.Spec.ProjectRef != nil {
			proj = p.Spec.ProjectRef.Name
		}
		views[p.Name] = projectroute.PipelineView{
			Name:        p.Name,
			ProjectName: proj,
			HasInput:    pipelineHasInput(p),
			HasOutput:   pipelineHasOutput(p),
		}
	}
	if errs := projectroute.ValidateProject(project, views); len(errs) > 0 {
		return errs[0].Message, nil
	}
	return "", nil
}

// reconcileRouteStreams ensures one JetStream stream exists per route and
// deletes streams for routes that no longer exist (compared to last status).
func (r *PipelineProjectReconciler) reconcileRouteStreams(ctx context.Context, project *rpcv1alpha1.PipelineProject) ([]rpcv1alpha1.ProjectRouteStatus, error) {
	natsURL := projectroute.NATSURL(project.Name, project.Namespace)
	defaultRet := projectDefaultRetention(project)

	desired := map[string]bool{}
	statuses := make([]rpcv1alpha1.ProjectRouteStatus, 0, len(project.Spec.Routes))
	for i := range project.Spec.Routes {
		route := &project.Spec.Routes[i]
		stream := projectroute.StreamName(project.Name, route.Name)
		subject := projectroute.Subject(project.Name, route.Name)
		desired[stream] = true

		ret := defaultRet
		if route.Retention != nil {
			ret = mergeRetention(defaultRet, route.Retention)
		}
		st := rpcv1alpha1.ProjectRouteStatus{Name: route.Name, Subject: subject, Stream: stream, Phase: "Ready"}
		if err := r.Streams.EnsureStream(ctx, natsURL, stream, subject, ret); err != nil {
			st.Phase = "Failed"
			st.Conditions = []metav1.Condition{{
				Type: "Ready", Status: metav1.ConditionFalse, Reason: "StreamError",
				Message: err.Error(), LastTransitionTime: metav1.Now(),
			}}
		}
		statuses = append(statuses, st)
	}

	for _, old := range project.Status.Routes {
		if !desired[old.Stream] && old.Stream != "" {
			if err := r.Streams.DeleteStream(ctx, natsURL, old.Stream); err != nil {
				return statuses, fmt.Errorf("delete stale stream %s: %w", old.Stream, err)
			}
		}
	}
	return statuses, nil
}

// projectDefaultRetention reads spec.nats.retention into a nats.Retention,
// applying spec defaults (maxAge 24h, maxBytes 1Gi) when unset.
func projectDefaultRetention(project *rpcv1alpha1.PipelineProject) nats.Retention {
	if project.Spec.NATS == nil {
		return nats.Retention{MaxAge: 24 * time.Hour, MaxBytes: 1 << 30}
	}
	ret := retentionFrom(&project.Spec.NATS.Retention)
	if ret.MaxAge == 0 {
		ret.MaxAge = 24 * time.Hour
	}
	if ret.MaxBytes == 0 {
		ret.MaxBytes = 1 << 30
	}
	return ret
}

// mergeRetention overlays a per-route override on the project default; unset
// override fields keep the default.
func mergeRetention(base nats.Retention, override *rpcv1alpha1.ProjectNATSRetention) nats.Retention {
	o := retentionFrom(override)
	if o.MaxAge != 0 {
		base.MaxAge = o.MaxAge
	}
	if o.MaxBytes != 0 {
		base.MaxBytes = o.MaxBytes
	}
	if o.MaxMsgs != 0 {
		base.MaxMsgs = o.MaxMsgs
	}
	return base
}

func retentionFrom(r *rpcv1alpha1.ProjectNATSRetention) nats.Retention {
	var out nats.Retention
	if r == nil {
		return out
	}
	if r.MaxAge != nil {
		out.MaxAge = r.MaxAge.Duration
	}
	if r.MaxBytes != nil {
		out.MaxBytes = r.MaxBytes.Value()
	}
	if r.MaxMsgs != nil {
		out.MaxMsgs = *r.MaxMsgs
	}
	return out
}

// pipelineHasOutput reports whether a pipeline defines a user output. Structured
// pipelines: Output.Type set. Raw pipelines: a top-level `output:` key present.
func pipelineHasOutput(p *rpcv1alpha1.Pipeline) bool {
	if p.Spec.RawYAML != "" {
		return rawHasTopKey(p.Spec.RawYAML, "output")
	}
	return p.Spec.Output.Type != ""
}

func pipelineHasInput(p *rpcv1alpha1.Pipeline) bool {
	if p.Spec.RawYAML != "" {
		return rawHasTopKey(p.Spec.RawYAML, "input")
	}
	return p.Spec.Input.Type != ""
}

// rawHasTopKey reports whether rawYAML parses to a mapping containing key.
func rawHasTopKey(rawYAML, key string) bool {
	var m map[string]any
	if err := yaml.Unmarshal([]byte(rawYAML), &m); err != nil {
		return false
	}
	_, ok := m[key]
	return ok
}
