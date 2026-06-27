/**
 * Reine (seiteneffektfreie) Transformationen rund um das RPK-Config-Schema und
 * die daraus erzeugte Monaco-Completion. Bewusst OHNE `monaco`/`monaco-yaml`-
 * Laufzeit-Import (nur Typen), damit dieses Modul in Node/Vitest ohne Web-Worker
 * und ohne Editor-Bundle getestet werden kann.
 */
import type { JSONSchema } from 'monaco-yaml'

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
export function discriminateComponentTypes(schema: JSONSchema): void {
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

/**
 * Label, das der yaml-language-server den „parent skeleton"-Vorschlägen gibt.
 *
 * Sobald ein anyOf-Zweig `required` besitzt (siehe discriminateComponentTypes),
 * erzeugt der LS pro Zweig zusätzlich einen Vorschlag, der das ganze Komponenten-
 * Objekt als Snippet einfügt. Dessen Label ist `getSchemaTypeName(branch)` —
 * mangels `title`/`$ref` am Zweig also schlicht der Typname „object". Bei 73
 * Komponententypen sind das 73 mit „object" beschriftete Einträge, die die
 * eigentliche Typ-Liste (generate/kafka_franz/…) zumüllen; einen Mehrwert haben
 * sie nicht (der Typname-Vorschlag fügt dasselbe Gerüst ein).
 */
export const SKELETON_COMPLETION_LABEL = 'object'

/**
 * Erkennt die oben beschriebenen Skelett-Vorschläge anhand ihres Labels.
 *
 * Das Label wird von monaco-yaml unverändert vom LSP übernommen (Kind/SortText
 * dagegen werden umgeschrieben bzw. verworfen), daher ist der Label-Vergleich das
 * stabile Kriterium über die LSP→Monaco-Konvertierung hinweg. Kein echter RPC-
 * Komponententyp und kein Attribut heißt „object" (gegen das Schema verifiziert),
 * also gibt es keine Falsch-Positiven. Akzeptiert sowohl das String-Label als
 * auch Monacos `CompletionItemLabel`-Objektform.
 */
export function isSkeletonCompletionLabel(label: string | { label: string }): boolean {
  const text = typeof label === 'string' ? label : label.label
  return text === SKELETON_COMPLETION_LABEL
}

/** Filtert die Skelett-Vorschläge aus einer Monaco-Vorschlagsliste. */
export function stripSkeletonSuggestions<T extends { label: string | { label: string } }>(
  suggestions: T[],
): T[] {
  return suggestions.filter((s) => !isSkeletonCompletionLabel(s.label))
}
