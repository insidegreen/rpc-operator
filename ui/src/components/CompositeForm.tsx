import Form from '@rjsf/core'
import validator from '@rjsf/validator-ajv8'
import { NestedComponentEditor } from './NestedComponentEditor'
import type { SchemaFormComponent } from './NestedComponentEditor'
import type { CatalogComponent, CompositeField, ComponentSpec } from '../types'

interface Props {
  component: CatalogComponent
  value: unknown
  catalogCache: Map<string, CatalogComponent>
  depth: number
  SchemaFormComp: SchemaFormComponent
  onChange: (value: unknown) => void
}

export function CompositeForm({
  component,
  value,
  catalogCache,
  depth,
  SchemaFormComp,
  onChange,
}: Props) {
  const compositeFields = component.compositeFields ?? []

  // Pattern B: field="" → config itself is the array
  const isDirectArray =
    compositeFields.length === 1 && compositeFields[0].field === ''
  if (isDirectArray) {
    const cf = compositeFields[0]
    const items = Array.isArray(value) ? (value as ComponentSpec[]) : []
    return (
      <NestedComponentEditor
        kind={cf.kind}
        items={items}
        catalogCache={catalogCache}
        depth={depth}
        SchemaFormComp={SchemaFormComp}
        onChange={onChange}
      />
    )
  }

  // Pattern A: config is an object; composite fields are rendered separately.
  const configObj =
    value && typeof value === 'object' && !Array.isArray(value)
      ? (value as Record<string, unknown>)
      : {}

  function updateCompositeField(cf: CompositeField, items: ComponentSpec[]) {
    onChange({ ...configObj, [cf.field]: items })
  }

  function updateScalarFields(formData: unknown) {
    // Merge RJSF scalar-field changes with existing composite field values
    const compositeKeys = new Set(compositeFields.map(cf => cf.field))
    const compositeValues = Object.fromEntries(
      Object.entries(configObj).filter(([k]) => compositeKeys.has(k)),
    )
    onChange({ ...(formData as object), ...compositeValues })
  }

  const schema = component.configSchema as {
    properties?: Record<string, unknown>
  }
  const hasScalarFields =
    schema.properties != null && Object.keys(schema.properties).length > 0

  return (
    <div>
      {compositeFields.map(cf => (
        <div key={cf.field} style={{ marginBottom: 12 }}>
          <div
            style={{
              fontSize: 12,
              fontWeight: 600,
              color: '#555',
              marginBottom: 4,
              textTransform: 'uppercase',
              letterSpacing: '0.05em',
            }}
          >
            {cf.field || cf.kind}
          </div>
          <NestedComponentEditor
            kind={cf.kind}
            items={
              Array.isArray(configObj[cf.field])
                ? (configObj[cf.field] as ComponentSpec[])
                : []
            }
            catalogCache={catalogCache}
            depth={depth}
            SchemaFormComp={SchemaFormComp}
            onChange={items => updateCompositeField(cf, items)}
          />
        </div>
      ))}

      {hasScalarFields && (
        <div style={{ marginTop: 8 }}>
          <Form
            schema={component.configSchema as object}
            validator={validator}
            formData={Object.fromEntries(
              Object.entries(configObj).filter(
                ([k]) => !compositeFields.some(cf => cf.field === k),
              ),
            )}
            onChange={({ formData }) => updateScalarFields(formData)}
            uiSchema={{ 'ui:submitButtonOptions': { norender: true } }}
          />
        </div>
      )}
    </div>
  )
}
