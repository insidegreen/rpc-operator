/**
 * Monaco Schema Loader für RPK (Redpanda Connect) Schema
 * 
 * Lädt das RPK JSON Schema und registriert es bei Monaco Editor
 * für intelligente Code-Completion in YAML-Dateien.
 */

// Custom Interface für Monaco Instance
// eslint-disable-next-line @typescript-eslint/no-explicit-any
interface MonacoInstance extends Record<string, any> {
  languages: {
    json: {
      jsonDefaults: {
        setDiagnosticsOptions: (options: unknown) => void;
      };
    };
    yaml?: {
      yamlDefaults?: {
        setDiagnosticsOptions?: (options: unknown) => void;
      };
    };
    registerCompletionItemProvider?: (language: string, provider: unknown) => void;
  };
}

let schemaInitializationPromise: Promise<void> | null = null;

/**
 * Extrahiert alle Komponenten-Namen aus dem RPK Schema nach Kategorie.
 */
function extractComponentNames(rpkSchema: any): {
  inputs: string[];
  processors: string[];
  outputs: string[];
} {
  const result = {
    inputs: [] as string[],
    processors: [] as string[],
    outputs: [] as string[],
  };

  if (!rpkSchema?.definitions) return result;

  for (const name of Object.keys(rpkSchema.definitions)) {
    if (name.startsWith('input_')) result.inputs.push(name);
    else if (name.startsWith('processor_')) result.processors.push(name);
    else if (name.startsWith('output_')) result.outputs.push(name);
  }

  result.inputs.sort();
  result.processors.sort();
  result.outputs.sort();
  return result;
}

/**
 * Extrahiert Properties aus RPK Schema Definitions (handelt allOf/anyOf).
 */
function extractPropertiesFromDef(def: any): Record<string, any> {
  const props: Record<string, any> = {};

  if (def.allOf) {
    for (const item of def.allOf) {
      if (item.anyOf) {
        for (const anyOf of item.anyOf) {
          if (anyOf.properties) Object.assign(props, anyOf.properties);
        }
      } else if (item.properties) {
        Object.assign(props, item.properties);
      }
    }
  } else if (def.properties) {
    Object.assign(props, def.properties);
  }

  return props;
}

/**
 * Erstellt Definitions für Monaco aus dem RPK Schema.
 */
function createDefinitions(rpkSchema: any): Record<string, object> {
  const definitions: Record<string, object> = {};

  if (!rpkSchema?.definitions) return definitions;

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  for (const [name, def] of Object.entries(rpkSchema.definitions) as [string, any][]) {
    if (typeof def !== 'object' || def === null) continue;

    const properties = extractPropertiesFromDef(def);
    const required: string[] = [];
    let additionalProperties = true;

    // Sammle required Felder
    if (def.allOf) {
      for (const item of def.allOf) {
        if (item.anyOf) {
          for (const anyOf of item.anyOf) {
            if (anyOf.required) {
              required.push(...anyOf.required.filter((r: string) => !required.includes(r)));
            }
          }
        } else if (item.required) {
          required.push(...item.required.filter((r: string) => !required.includes(r)));
        }
        if (item.additionalProperties === false) additionalProperties = false;
      }
    } else if (def.required) {
      required.push(...def.required);
    }
    if (def.additionalProperties === false) additionalProperties = false;

    definitions[name] = {
      type: 'object',
      properties,
      ...(required.length > 0 && { required }),
      additionalProperties,
    };
  }

  return definitions;
}

/**
 * Erstellt das Root-Schema mit oneOf für kontextsensitive Completion.
 * oneOf ermöglicht Monaco, basierend auf dem type-Feld die richtige Definition zu wählen.
 */
function createRootSchema(
  components: { inputs: string[]; processors: string[]; outputs: string[] },
  definitions: Record<string, object>
): object {
  // Input/Output Schema (single object)
  const ioSchema = (compList: string[]) => ({
    oneOf: compList.map(compName => ({
      type: 'object',
      properties: {
        type: { const: compName },
        label: { type: 'string' },
        config: { $ref: `#/definitions/${compName}` },
      },
      required: ['type'],
      additionalProperties: false,
    })),
  });

  // Processors Schema (array of objects)
  const processorsSchema = {
    type: 'array',
    items: {
      oneOf: components.processors.map(compName => ({
        type: 'object',
        properties: {
          type: { const: compName },
          label: { type: 'string' },
          config: { $ref: `#/definitions/${compName}` },
        },
        required: ['type'],
        additionalProperties: false,
      })),
    },
  };

  return {
    $schema: 'http://json-schema.org/draft-07/schema#',
    type: 'object',
    definitions,
    properties: {
      input: ioSchema(components.inputs),
      processors: processorsSchema,
      output: ioSchema(components.outputs),
    },
    additionalProperties: false,
  };
}

/**
 * Erstellt ein minimales Fallback-Schema.
 */
function createFallbackSchema(): object {
  return {
    $schema: 'http://json-schema.org/draft-07/schema#',
    type: 'object',
    properties: {
      input: { type: 'object' },
      processors: { type: 'array', items: { type: 'object' } },
      output: { type: 'object' },
    },
    additionalProperties: false,
  };
}

/**
 * Lädt und adaptiert das RPK Schema für Monaco.
 */
async function loadAndAdaptSchema(): Promise<object> {
  const response = await fetch('/schemas/rpk.json');
  if (!response.ok) {
    throw new Error(`Failed to load RPK schema: ${response.status} ${response.statusText}`);
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const rpkSchema = await response.json() as any;
  const components = extractComponentNames(rpkSchema);
  const definitions = createDefinitions(rpkSchema);
  return createRootSchema(components, definitions);
}

/**
 * Initialisiert Monaco Editor mit dem RPK Schema.
 * Diese Funktion sollte einmalig beim Start der Anwendung aufgerufen werden.
 */
export async function initializeMonacoSchema(monaco: MonacoInstance): Promise<void> {
  if (schemaInitializationPromise) return schemaInitializationPromise;

  schemaInitializationPromise = (async () => {
    try {
      const adaptedSchema = await loadAndAdaptSchema();

      // Registriere das Schema für JSON
      monaco.languages.json.jsonDefaults.setDiagnosticsOptions({
        validate: true,
        schemas: [
          {
            uri: 'redpanda-connect://rpk-pipeline',
            fileMatch: ['*.json'],
            schema: adaptedSchema,
          },
        ],
        enableSchemaRequest: true,
        comments: 'ignore',
        compact: false,
      });

      // Registriere das Schema auch für YAML (falls yamlDefaults existiert)
      if (monaco.languages.yaml?.yamlDefaults?.setDiagnosticsOptions) {
        monaco.languages.yaml.yamlDefaults.setDiagnosticsOptions({
          validate: true,
          schemas: [
            {
              uri: 'redpanda-connect://rpk-pipeline',
              fileMatch: ['*.yaml', '*.yml'],
              schema: adaptedSchema,
            },
          ],
          enableSchemaRequest: true,
          comments: 'ignore',
          compact: false,
        });
      }

      // Versuche, YAML Language Support zu aktivieren
      // Monaco verwendet oft die JSON Schema Validation auch für YAML
      monaco.languages.json.jsonDefaults.setDiagnosticsOptions({
        validate: true,
        schemas: [
          {
            uri: 'redpanda-connect://rpk-pipeline',
            fileMatch: ['*.yaml', '*.yml', '*'],
            schema: adaptedSchema,
          },
        ],
        enableSchemaRequest: true,
        comments: 'ignore',
        compact: false,
      });

      // Registriere Custom Completion Provider für YAML
      // Dies ist notwendig, da Monaco YAML nicht nativ mit JSON Schema verknüpft
      registerYAMLCompletionProvider(monaco, adaptedSchema);

      console.log('✅ Monaco Editor initialized with RPK schema');
      console.log(`   Schema URI: redpanda-connect://rpk-pipeline`);
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      console.log(`   Components: ${Object.keys((adaptedSchema as any).definitions || {}).length} definitions loaded`);
    } catch (error) {
      console.error('❌ Failed to initialize Monaco with RPK schema:', error);
      // Fallback Schema registrieren
      monaco.languages.json.jsonDefaults.setDiagnosticsOptions({
        validate: true,
        schemas: [
          {
            uri: 'redpanda-connect://fallback-pipeline',
            fileMatch: ['*.yaml', '*.yml', '*'],
            schema: createFallbackSchema(),
          },
        ],
      });
    }
  })();

  return schemaInitializationPromise;
}

/**
 * Setzt die Schema-Initialisierung zurück (für Tests).
 */
export function resetSchemaInitialization(): void {
  schemaInitializationPromise = null;
}

/**
 * Prüft, ob Monaco bereits initialisiert wurde.
 */
export function isMonacoInitialized(): boolean {
  return schemaInitializationPromise !== null;
}

/**
 * Registriert einen Custom Completion Provider für YAML.
 * Da Monaco YAML nicht nativ mit JSON Schema verknüpft,
 * müssen wir manuell Completion für die wichtigsten Fälle hinzufügen.
 */
function registerYAMLCompletionProvider(monaco: MonacoInstance, schema: object): void {
  // Es gibt ein Problem: Monaco's eingebaute YAML-Unterstützung funktioniert
  // nur mit JSON Schema, wenn die YAML Language registriert ist.
  // Da @monaco-editor/react YAML support hat, sollten wir das Schema
  // für die YAML-Sprache registrieren.
  
  // Registriere einen einfachen Completion Provider für YAML
  // der die Top-Level Felder und Komponenten-Typen vorschlägt
  if (monaco.languages.registerCompletionItemProvider) {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const m = monaco as any;
    
    // Extrahiere alle Komponenten-Namen
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const allComponents = Object.keys((schema as any).definitions || {});

    m.languages.registerCompletionItemProvider('yaml', {
      provideCompletionItems: (model: any, position: any) => {
        const line = model.getLineContent(position.lineNumber);
        const textUntilPosition = line.substring(0, position.column - 1);
        const lineTrimmed = line.trim();

        // 1. Leere Zeile oder nur Whitespace: Vorschläge für Top-Level Felder
        if (lineTrimmed === '' || /^\s*$/.test(lineTrimmed)) {
          return {
            suggestions: [
              { label: 'input:', kind: m.languages.CompletionItemKind.Field, insertText: 'input:\n  type: ' },
              { label: 'processors:', kind: m.languages.CompletionItemKind.Field, insertText: 'processors:\n  - type: ' },
              { label: 'output:', kind: m.languages.CompletionItemKind.Field, insertText: 'output:\n  type: ' },
            ],
          };
        }

        // 2. Nach "input:" oder "output:" - schlage type-Feld vor
        if (textUntilPosition.match(/(input|output):\s*$/)) {
          return {
            suggestions: [
              { label: 'type:', kind: m.languages.CompletionItemKind.Field, insertText: 'type: ' },
            ],
          };
        }

        // 3. Nach "processors:" - schlage Array-Item vor
        if (textUntilPosition.match(/processors:\s*$/)) {
          return {
            suggestions: [
              { label: '- type:', kind: m.languages.CompletionItemKind.Field, insertText: '\n  - type: ' },
            ],
          };
        }

        // 4. Nach "- type:" in processors-Array
        if (textUntilPosition.match(/- type:\s*$/)) {
          const suggestions = allComponents.map((compName: string) => ({
            label: compName,
            kind: m.languages.CompletionItemKind.Enum,
            insertText: compName,
          }));
          return { suggestions };
        }

        // 5. Nach "type:" (für input und output)
        if (textUntilPosition.match(/type:\s*$/)) {
          const suggestions = allComponents.map((compName: string) => ({
            label: compName,
            kind: m.languages.CompletionItemKind.Enum,
            insertText: compName,
          }));
          return { suggestions };
        }

        // 6. Nach type-Wert - schlage Standard-Felder vor
        if (textUntilPosition.match(/type:\s*\w+/)) {
          return {
            suggestions: [
              { label: 'label:', kind: m.languages.CompletionItemKind.Field, insertText: 'label: ' },
              { label: 'config:', kind: m.languages.CompletionItemKind.Field, insertText: 'config: ' },
            ],
          };
        }

        return { suggestions: [] };
      },
    });
  }
}
