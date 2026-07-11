// Logo helpers for apps and databases. Curated resources carry a brand logo URL
// (simpleicons CDN); everything else falls back to an MDI glyph or a generated
// initials avatar rendered by <ResourceIcon>.
import type { DBEngine } from '@/api/types'

const ENGINE_SLUG: Record<string, string> = {
  postgres: 'postgresql',
  mysql: 'mysql',
  mariadb: 'mariadb',
  redis: 'redis',
  mongodb: 'mongodb',
  libsql: 'sqlite',
}

// engineLogo returns the brand-logo URL for a database engine, or '' if unknown.
export function engineLogo(engine: DBEngine | string): string {
  const slug = ENGINE_SLUG[engine] || ''
  return slug ? `https://cdn.simpleicons.org/${slug}` : ''
}

// engineMdi is the MDI fallback used when the logo URL fails to load (offline).
export function engineMdi(engine: DBEngine | string): string {
  switch (engine) {
    case 'postgres':
      return 'mdi-elephant'
    case 'redis':
      return 'mdi-database-outline'
    case 'mongodb':
      return 'mdi-leaf'
    case 'libsql':
      return 'mdi-database-search'
    default:
      return 'mdi-database'
  }
}
