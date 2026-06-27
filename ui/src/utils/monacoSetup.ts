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
import { discriminateComponentTypes, stripSkeletonSuggestions } from './monacoSchemaTransforms'

declare global {
  interface Window {
    MonacoEnvironment?: monaco.Environment
  }
}

/** Quelle des Schemas; erscheint auch als Quelle in Hover-Tooltips. */
const SCHEMA_URI = 'https://redpanda.com/rpk-connect.schema.json'

/**
 * Entfernt die „object"-Skelett-Vorschläge aus der YAML-Completion.
 *
 * `discriminateComponentTypes` setzt `required` an jedem Komponenten-Zweig, was
 * den yaml-language-server pro Zweig einen „parent skeleton"-Vorschlag (Label
 * „object") erzeugen lässt — 73 Stück, die die Typ-Liste zumüllen (siehe
 * monacoSchemaTransforms). monaco-yaml verwirft deren SortText, sie sortieren in
 * der UI also nicht ans Ende, sondern mitten hinein. Hier wird der von
 * monaco-yaml registrierte „yaml"-Completion-Provider umschlossen und die
 * Skelett-Vorschläge werden herausgefiltert. Muss VOR `configureMonacoYaml`
 * laufen, damit der Provider beim Registrieren erfasst wird.
 */
function filterSkeletonCompletions(monacoInstance: typeof monaco): void {
  const { languages } = monacoInstance
  const register = languages.registerCompletionItemProvider.bind(languages)
  languages.registerCompletionItemProvider = (selector, provider) => {
    const provideCompletionItems = provider.provideCompletionItems
    if (selector === 'yaml' && provideCompletionItems) {
      provider.provideCompletionItems = async (...args) => {
        const list = await provideCompletionItems.apply(provider, args)
        if (list?.suggestions) {
          list.suggestions = stripSkeletonSuggestions(list.suggestions)
        }
        return list
      }
    }
    return register(selector, provider)
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

    // Provider-Wrapper installieren, bevor monaco-yaml seinen Provider registriert.
    filterSkeletonCompletions(monaco)

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
