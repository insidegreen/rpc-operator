/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package controller

import "fmt"

const clusterConfigFile = "connect.yaml"

// clusterConfigYAML returns the Redpanda Connect main config loaded by every
// streams-mode instance in a PipelineCluster. It enables the HTTP server (for
// the streams API + health probes on httpPort) and sets the logger format.
func clusterConfigYAML(jsonLogging bool) string {
	format := "logfmt"
	if jsonLogging {
		format = "json"
	}
	return fmt.Sprintf(`http:
  address: 0.0.0.0:%d
  enabled: true
logger:
  level: INFO
  format: %s
  add_timestamp: true
`, httpPort, format)
}
