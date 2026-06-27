import { describe, it, expect } from 'vitest'
import type { JSONSchema } from 'monaco-yaml'
import {
  discriminateComponentTypes,
  isSkeletonCompletionLabel,
  stripSkeletonSuggestions,
} from './monacoSchemaTransforms'

// Nachbau der Komponenten-Definition aus dem RPK-Schema: allOf[anyOf[…typ-zweige…]].
function componentSchema(): JSONSchema {
  return {
    definitions: {
      input: {
        allOf: [
          {
            anyOf: [
              { properties: { generate: { type: 'object' } } },
              { properties: { kafka_franz: { type: 'object' } } },
            ],
          },
          // Gemeinsame Properties (label/meta/…) — kein Ein-Schlüssel-Typ-Zweig.
          { properties: { label: { type: 'string' }, processors: { type: 'array' } } },
        ],
      },
    },
  }
}

describe('discriminateComponentTypes', () => {
  it('setzt required:[<typ>] an jedem Ein-Schlüssel-anyOf-Zweig', () => {
    const schema = componentSchema()
    discriminateComponentTypes(schema)

    const branches = (schema.definitions!.input as JSONSchema).allOf![0] as JSONSchema
    expect((branches.anyOf![0] as JSONSchema).required).toEqual(['generate'])
    expect((branches.anyOf![1] as JSONSchema).required).toEqual(['kafka_franz'])
  })

  it('lässt Mehr-Schlüssel-Elemente (gemeinsame Properties) unangetastet', () => {
    const schema = componentSchema()
    discriminateComponentTypes(schema)

    const common = (schema.definitions!.input as JSONSchema).allOf![1] as JSONSchema
    expect(common.required).toBeUndefined()
  })

  it('überschreibt vorhandenes required nicht', () => {
    const schema = componentSchema()
    const firstBranch = ((schema.definitions!.input as JSONSchema).allOf![0] as JSONSchema)
      .anyOf![0] as JSONSchema
    firstBranch.required = ['already']
    discriminateComponentTypes(schema)
    expect(firstBranch.required).toEqual(['already'])
  })

  it('ist robust gegen ein Schema ohne definitions', () => {
    expect(() => discriminateComponentTypes({})).not.toThrow()
  })
})

describe('isSkeletonCompletionLabel', () => {
  it('erkennt das String-Label "object"', () => {
    expect(isSkeletonCompletionLabel('object')).toBe(true)
  })

  it('erkennt die CompletionItemLabel-Objektform', () => {
    expect(isSkeletonCompletionLabel({ label: 'object' })).toBe(true)
  })

  it('lässt echte Typ-/Attribut-Labels durch', () => {
    expect(isSkeletonCompletionLabel('generate')).toBe(false)
    expect(isSkeletonCompletionLabel({ label: 'mapping' })).toBe(false)
  })
})

describe('stripSkeletonSuggestions', () => {
  it('entfernt nur die object-Skelett-Vorschläge', () => {
    const suggestions = [
      { label: 'generate' },
      { label: 'object' },
      { label: { label: 'kafka_franz' } },
      { label: 'object' },
    ]
    const result = stripSkeletonSuggestions(suggestions)
    expect(result.map((s) => (typeof s.label === 'string' ? s.label : s.label.label))).toEqual([
      'generate',
      'kafka_franz',
    ])
  })
})
