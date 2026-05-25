// Mirrors api/v1alpha1/pipeline_types.go and internal/api/catalog/catalog.go

export interface ComponentSpec {
  type: string
  label?: string
  // config is unknown: string (scalar), object (object/composite Pattern A),
  // or ComponentSpec[] directly (composite Pattern B: for_each, fallback)
  config?: unknown
}

export interface SecretRef {
  envVar: string
  secretName: string
  key: string
}

export interface PipelineSpec {
  input?: ComponentSpec
  processors?: ComponentSpec[]
  output?: ComponentSpec
  rawYAML?: string
  replicas?: number
  image?: string
  secretRefs?: SecretRef[]
  /** F45: when true, the operator removes the pipeline pod and keeps it stopped. */
  stopped?: boolean
  /** F47 Phase 2: run this pipeline as a stream on the named PipelineCluster instead of its own pod. */
  clusterRef?: string
}

export interface Pipeline {
  apiVersion: string
  kind: string
  metadata: {
    name: string
    namespace: string
    resourceVersion?: string
    creationTimestamp?: string
  }
  spec: PipelineSpec
  status?: {
    phase?: 'Pending' | 'Running' | 'Failed' | 'Stopped'
    podName?: string
    /** F47 Phase 2/3a: set when the pipeline runs as a stream on a cluster. */
    assignedCluster?: string
    assignedInstance?: string
    streamID?: string
    observedGeneration?: number
    conditions?: Array<{
      type: string
      status: string
      message?: string
      reason?: string
      lastTransitionTime?: string
    }>
  }
}

// Mirrors catalog.CompositeField
export interface CompositeField {
  field: string        // field name in config; "" = config itself is the array (Pattern B)
  kind: 'inputs' | 'processors' | 'outputs'
  multi: boolean
}

export interface CatalogComponent {
  name: string
  category: 'inputs' | 'processors' | 'outputs'
  status: string
  summary: string
  bodyKind: 'object' | 'scalar' | 'composite'
  replicaSafety: string
  configSchema: object            // JSON Schema Draft-07 (non-composite fields only)
  compositeFields?: CompositeField[]
}

export interface ValidationError {
  path: string
  message: string
}

export interface ValidateResponse {
  valid: boolean
  errors?: ValidationError[]
}

// The metric series the backend can compute, shared by pipeline and cluster
// metrics endpoints. Mirrors the server's knownQueries map.
export type MetricQuery = 'throughput' | 'error_rate' | 'input_rate' | 'processor_error_rate'

export interface MetricsDatapoint {
  t: number  // Unix timestamp (seconds)
  v: number  // value (msg/s)
}

export interface MetricsResponse {
  query: string
  unit: string
  datapoints: MetricsDatapoint[]
}

// Mirrors api/v1alpha1/pipelinecluster_types.go
export interface PipelineClusterSpec {
  replicas?: number
  image?: string
  jsonLogging?: boolean
  resources?: object
}

export interface PipelineCluster {
  apiVersion?: string
  kind?: string
  metadata: {
    name: string
    namespace: string
    resourceVersion?: string
    creationTimestamp?: string
  }
  spec: PipelineClusterSpec
  status?: {
    phase?: 'Pending' | 'Ready' | 'Degraded'
    readyReplicas?: number
    observedGeneration?: number
    conditions?: Array<{
      type: string
      status: string
      message?: string
      reason?: string
      lastTransitionTime?: string
    }>
  }
}

// Mirrors the internal/api cluster-distribution response (Phase 3b /instances).
export interface ClusterInstance {
  name: string
  ordinal: number
  ready: boolean
  assignedPipelines: string[]
}

export interface StalePlacement {
  pipeline: string
  assignedInstance: string
}

export interface ClusterDistribution {
  cluster: string
  namespace: string
  phase: string
  desiredReplicas: number
  readyReplicas: number
  instances: ClusterInstance[]
  stalePlacements: StalePlacement[]
}
