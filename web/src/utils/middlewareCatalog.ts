// Curated Goma Gateway middleware kinds with descriptions and example rule
// templates. Free-form types are still allowed by typing into the field.
export interface MwType {
  type: string
  label: string
  description: string
  template: Record<string, unknown>
}

export const middlewareCatalog: MwType[] = [
  { type: 'basicAuth', label: 'Basic Auth', description: 'HTTP Basic authentication with a list of users (bcrypt/SHA-1/plaintext passwords).', template: { realm: 'Restricted', forwardUsername: true, users: [{ username: 'admin', password: '$2y$05$...' }] } },
  { type: 'jwt', label: 'JWT', description: 'Validate a JSON Web Token from the Authorization header.', template: { secret: 'your-signing-secret', algo: 'HS256' } },
  { type: 'rateLimit', label: 'Rate Limit', description: 'Throttle requests per client over a time window.', template: { requestsPerUnit: 100, unit: 'minute' } },
  { type: 'accessPolicy', label: 'Access Policy', description: 'Allow or deny requests by source IP / CIDR.', template: { action: 'DENY', sourceRanges: ['10.0.0.0/8'] } },
  { type: 'addPrefix', label: 'Add Prefix', description: 'Prepend a path prefix before forwarding upstream.', template: { prefix: '/api' } },
  { type: 'rewrite', label: 'Rewrite', description: 'Rewrite the request path with a pattern/replacement.', template: { pattern: '^/old/(.*)', replacement: '/new/$1' } },
  { type: 'cors', label: 'CORS', description: 'Cross-Origin Resource Sharing headers.', template: { origins: ['*'], allowMethods: ['GET', 'POST'], allowHeaders: ['Content-Type'] } },
  { type: 'redirectScheme', label: 'Redirect Scheme', description: 'Force HTTP→HTTPS (or another scheme).', template: { scheme: 'https', permanent: true } },
  { type: 'forwardAuth', label: 'Forward Auth', description: 'Delegate auth to an external authorization service.', template: { authUrl: 'http://auth.internal/verify', authSignin: '' } },
]

export const middlewareTypes = middlewareCatalog.map((c) => c.type)

export function middlewareTypeInfo(t: string): MwType | undefined {
  return middlewareCatalog.find((c) => c.type === t)
}
