import Form from '@rjsf/core'
import validator from '@rjsf/validator-ajv8'
import { CompositeForm } from './CompositeForm'
import type { CatalogComponent } from '../types'

interface Props {
  component: CatalogComponent
  value: unknown
  catalogCache: Map<string, CatalogComponent>
  depth?: number
  onChange: (value: unknown) => void
}

// SchemaForm is the central renderer for all bodyKind variants.
// It passes itself as SchemaFormComp to CompositeForm → NestedComponentEditor,
// enabling full recursive rendering of nested composite components.
export function SchemaForm({ component, value, catalogCache, depth = 0, onChange }: Props) {
  if (component.bodyKind === 'scalar') {
    return (
      <textarea
        value={typeof value === 'string' ? value : ''}
        onChange={e => onChange(e.target.value)}
        rows={3}
        style={{
          width: '100%',
          fontFamily: 'monospace',
          fontSize: 12,
          boxSizing: 'border-box',
        }}
        placeholder={`${component.name} expression…`}
      />
    )
  }

  if (component.bodyKind === 'composite') {
    return (
      <CompositeForm
        component={component}
        value={value}
        catalogCache={catalogCache}
        depth={depth}
        SchemaFormComp={SchemaForm}
        onChange={onChange}
      />
    )
  }

  // bodyKind === 'object' (default)
  return (
    <Form
      schema={component.configSchema as object}
      validator={validator}
      formData={value ?? {}}
      onChange={({ formData }) => onChange(formData)}
      uiSchema={{ 'ui:submitButtonOptions': { norender: true } }}
    />
  )
}
