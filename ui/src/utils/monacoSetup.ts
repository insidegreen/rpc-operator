/**
 * Monaco-Setup für den RAW-YAML-Pipeline-Editor.
 *
 * Bündelt Monaco lokal (statt CDN-Loader) und verknüpft es per `monaco-yaml`
 * mit dem nativen Redpanda-Connect-Config-Schema (`/schemas/rpk.json`). Das
 * liefert echte Schema-Validierung, Hover und kontextsensitive Code-Completion
 * im tatsächlich bearbeiteten YAML-Format (input/pipeline.processors/output …).
 *
 * Wird lazy aus dem Editor-Chunk aufgerufen, damit Monaco nicht im Haupt-Bundle
 * landet. `setupMonaco()` ist idempotent (einmal pro Application).
 */
import * as monaco from 'monaco-editor'
import { loader } from '@monaco-editor/react'
import { configureMonacoYaml, type JSONSchema } from 'monaco-yaml'
import EditorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker'
import YamlWorker from 'monaco-yaml/yaml.worker?worker'

declare global {
  interface Window {
    MonacoEnvironment?: monaco.Environment
  }
}

/** Quelle des Schemas; erscheint auch als Quelle in Hover-Tooltips. */
const SCHEMA_URI = 'https://redpanda.com/rpk-connect.schema.json'

/**
 * Macht die Komponenten-Typ-Completion „eine Ebene tiefer" funktionsfähig.
 *
 * Das RPC-Config-Schema beschreibt jede Komponente (input/output/processor …)
 * als `allOf: [{ anyOf: [ { properties: { <typ>: <config> } }, … ] }, …]`. Ohne
 * Diskriminator matchen bei `input: { generate: {} }` ALLE 73 anyOf-Zweige
 * gleichermaßen — der yaml-language-server kann den gewählten Typ nicht
 * auflösen und bietet keine Attribut-Completion (mapping/interval/…).
 *
 * Fix: jedem Ein-Schlüssel-Zweig `required: [<typ>]` geben. Dann matcht ein
 * gesetzter Typ genau seinen Zweig → Attribute werden vorgeschlagen. Die
 * Typ-Liste auf der Elternebene bleibt erhalten (Verifikation: yaml-language-
 * server gegen das echte Schema). Mutiert `schema` in-place.
 */
function discriminateComponentTypes(schema: JSONSchema): void {
  for (const def of Object.values(schema.definitions ?? {})) {
    if (typeof def !== 'object' || !Array.isArray(def.allOf)) continue
    for (const element of def.allOf) {
      if (typeof element !== 'object' || !Array.isArray(element.anyOf)) continue
      for (const branch of element.anyOf) {
        if (typeof branch !== 'object' || branch.required || !branch.properties) continue
        const keys = Object.keys(branch.properties)
        if (keys.length === 1) branch.required = keys
      }
    }
  }
}

let setupPromise: Promise<void> | null = null

/**
 * Konfiguriert Monaco + monaco-yaml einmalig. Muss laufen, bevor
 * `@monaco-editor/react` den Editor mountet (daher `loader.config`).
 */
export function setupMonaco(): Promise<void> {
  if (setupPromise) return setupPromise

  setupPromise = (async () => {
    // Worker-Verdrahtung für gebündeltes Monaco + monaco-yaml.
    window.MonacoEnvironment = {
      getWorker(_workerId, label) {
        if (label === 'yaml') return new YamlWorker()
        return new EditorWorker()
      },
    }

    // @monaco-editor/react auf das gebündelte Monaco zeigen lassen (kein CDN).
    loader.config({ monaco })

    let schema: JSONSchema | undefined
    try {
      const res = await fetch('/schemas/rpk.json')
      if (!res.ok) {
        throw new Error(`${res.status} ${res.statusText}`)
      }
      schema = (await res.json()) as JSONSchema
      discriminateComponentTypes(schema)
    } catch (err) {
      // Ohne Schema bleibt der Editor voll funktionsfähig, nur ohne Completion.
      console.error('RPK-Schema konnte nicht geladen werden:', err)
    }

    configureMonacoYaml(monaco, {
      enableSchemaRequest: false,
      validate: true,
      hover: true,
      completion: true,
      schemas: schema
        ? [{ uri: SCHEMA_URI, fileMatch: ['*.yaml', '*.yml'], schema }]
        : [],
    })
  })()

  return setupPromise
}
