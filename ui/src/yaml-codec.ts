import yaml from 'js-yaml'
import type { PipelineSpec } from './types'

export function specToYaml(spec: PipelineSpec): string {
  return yaml.dump(spec, { lineWidth: 120, quotingType: '"' })
}

export function yamlToSpec(text: string): PipelineSpec {
  const parsed = yaml.load(text)
  if (typeof parsed !== 'object' || parsed === null) {
    throw new Error('YAML muss ein Object sein')
  }
  const obj = parsed as Record<string, unknown>
  if (!obj.input || !obj.output) {
    throw new Error('YAML benötigt input und output Felder')
  }
  return obj as unknown as PipelineSpec
}
