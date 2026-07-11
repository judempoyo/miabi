import { parse, stringify } from 'yaml'

/** Parse a YAML string into an object (throws on invalid YAML). */
export function parseYaml(text: string): Record<string, unknown> {
  const out = parse(text)
  return (out && typeof out === 'object' ? out : {}) as Record<string, unknown>
}

/** Serialize an object to YAML. */
export function toYaml(value: unknown): string {
  return stringify(value ?? {})
}
