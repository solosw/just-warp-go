const iconMap: Record<string, string> = {
  // TypeScript / JavaScript
  ts: '\u{1F537}', tsx: '\u{1F537}', mts: '\u{1F537}', cts: '\u{1F537}',
  js: '\u{1F7E8}', jsx: '\u{1F7E8}', mjs: '\u{1F7E8}', cjs: '\u{1F7E8}',
  // Web
  vue: '\u{1F49A}', svelte: '\u{1F9E1}',
  html: '\u{1F310}', htm: '\u{1F310}',
  css: '\u{1F3A8}', scss: '\u{1F3A8}', sass: '\u{1F3A8}', less: '\u{1F3A8}',
  // Data
  json: '\u{1F4CB}', jsonc: '\u{1F4CB}', json5: '\u{1F4CB}',
  xml: '\u{1F4C4}', svg: '\u{1F5BC}',
  yaml: '\u{2699}', yml: '\u{2699}',
  toml: '\u{2699}',
  // Markup
  md: '\u{1F4DD}', mdx: '\u{1F4DD}', markdown: '\u{1F4DD}',
  // Go
  go: '\u{1F535}',
  // Python
  py: '\u{1F40D}', pyw: '\u{1F40D}', ipynb: '\u{1F4D3}',
  // Rust
  rs: '\u{2699}',
  // C / C++
  c: '\u{1F52E}', h: '\u{1F52E}', cpp: '\u{1F52E}', cxx: '\u{1F52E}', cc: '\u{1F52E}', hpp: '\u{1F52E}',
  // JVM
  java: '\u{2615}', kt: '\u{1F7E3}', kts: '\u{1F7E3}', scala: '\u{1F534}',
  groovy: '\u{1F48E}',
  // PHP
  php: '\u{1F418}',
  // C#
  cs: '\u{1F7E3}',
  // Ruby
  rb: '\u{1F48E}', rake: '\u{1F48E}',
  // Swift / ObjC
  swift: '\u{1F7E0}', m: '\u{1F535}', mm: '\u{1F535}',
  // Shell
  sh: '\u{1F427}', bash: '\u{1F427}', zsh: '\u{1F427}', fish: '\u{1F41F}',
  bat: '\u{1F4BB}', cmd: '\u{1F4BB}', ps1: '\u{1F4BB}',
  // Docker
  dockerfile: '\u{1F433}',
  // Config
  env: '\u{1F512}', ini: '\u{1F527}', cfg: '\u{1F527}', conf: '\u{1F527}',
  properties: '\u{1F527}',
  // GraphQL
  gql: '\u{1F310}', graphql: '\u{1F310}',
  // SQL
  sql: '\u{1F5C4}', pgsql: '\u{1F5C4}', mysql: '\u{1F5C4}', sqlite: '\u{1F5C4}',
  // Lua
  lua: '\u{1F319}',
  // R
  r: '\u{1F4CA}',
  // Dart
  dart: '\u{1F3AF}',
  // Elixir
  ex: '\u{1F48E}', exs: '\u{1F48E}',
  // Haskell
  hs: '\u{1F52C}', lhs: '\u{1F52C}',
  // Erlang
  erl: '\u{1F4E6}',
  // Clojure
  clj: '\u{1F33F}', cljs: '\u{1F33F}', edn: '\u{1F33F}',
  // Makefile
  makefile: '\u{1F527}',
  // CMake
  cmake: '\u{1F527}',
  // License / readme special
  license: '\u{1F4DC}',
  // Git
  gitignore: '\u{1F512}',
  // Logs
  log: '\u{1F4C4}',
  // Diff / patch
  diff: '\u{1F4C4}', patch: '\u{1F4C4}',
  // Images (rarely shown, but just in case)
  png: '\u{1F5BC}', jpg: '\u{1F5BC}', jpeg: '\u{1F5BC}', gif: '\u{1F5BC}',
  ico: '\u{1F5BC}', webp: '\u{1F5BC}',
  // Fonts
  ttf: '\u{1F520}', otf: '\u{1F520}', woff: '\u{1F520}', woff2: '\u{1F520}',
  // Lock files
  lock: '\u{1F512}',
}

export function getFileIcon(fileName: string): string {
  const name = fileName.toLowerCase()
  const ext = name.split('.').pop() || ''

  // Special filenames
  const base = name.split('/').pop()?.split('\\').pop() || name
  if (base === 'dockerfile') return iconMap.dockerfile
  if (base === 'makefile') return iconMap.makefile
  if (base === 'license') return iconMap.license
  if (base === '.gitignore') return iconMap.gitignore
  if (base.endsWith('.d.ts')) return '\u{1F537}' // declaration files

  return iconMap[ext] || '\u{1F4C4}' // default: page
}

export function isDirectory(name: string): boolean {
  // No extension suggests directory (used in file tree context)
  return !name.includes('.')
}
