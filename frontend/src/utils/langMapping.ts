import { javascript } from '@codemirror/lang-javascript'
import { json } from '@codemirror/lang-json'
import { python } from '@codemirror/lang-python'
import { html } from '@codemirror/lang-html'
import { css } from '@codemirror/lang-css'
import { markdown } from '@codemirror/lang-markdown'
import { xml } from '@codemirror/lang-xml'
import { cpp } from '@codemirror/lang-cpp'
import { java } from '@codemirror/lang-java'
import { go } from '@codemirror/lang-go'
import { rust } from '@codemirror/lang-rust'
import { php } from '@codemirror/lang-php'
import { sql, PostgreSQL } from '@codemirror/lang-sql'
import { StreamLanguage } from '@codemirror/language'
import { yaml } from '@codemirror/legacy-modes/mode/yaml'
import { toml } from '@codemirror/legacy-modes/mode/toml'
import { properties } from '@codemirror/legacy-modes/mode/properties'
import { ruby } from '@codemirror/legacy-modes/mode/ruby'
import { swift } from '@codemirror/legacy-modes/mode/swift'
import { csharp, objectiveC, kotlin, scala, dart } from '@codemirror/legacy-modes/mode/clike'
import { groovy } from '@codemirror/legacy-modes/mode/groovy'
import { lua } from '@codemirror/legacy-modes/mode/lua'
import { r } from '@codemirror/legacy-modes/mode/r'
import { shell } from '@codemirror/legacy-modes/mode/shell'
import { powerShell } from '@codemirror/legacy-modes/mode/powershell'
import { mscgen } from '@codemirror/legacy-modes/mode/mscgen'
import { dockerFile } from '@codemirror/legacy-modes/mode/dockerfile'
import { cmake } from '@codemirror/legacy-modes/mode/cmake'
import { protobuf } from '@codemirror/legacy-modes/mode/protobuf'
import { stex } from '@codemirror/legacy-modes/mode/stex'
import { elm } from '@codemirror/legacy-modes/mode/elm'
import { erlang } from '@codemirror/legacy-modes/mode/erlang'
import { haskell } from '@codemirror/legacy-modes/mode/haskell'
import { clojure } from '@codemirror/legacy-modes/mode/clojure'
import { fortran } from '@codemirror/legacy-modes/mode/fortran'
import { perl } from '@codemirror/legacy-modes/mode/perl'
import { scheme } from '@codemirror/legacy-modes/mode/scheme'
import type { Extension } from '@codemirror/state'

// Official CM6 language packages (tree-sitter grammars)
const officialMap: Record<string, () => Extension> = {
  javascript,
  typescript: javascript,
  json,
  python,
  html,
  css,
  scss: css,
  less: css,
  markdown,
  xml,
  c: cpp,
  cpp,
  java,
  go,
  rust,
  php,
  sql,
  pgsql: () => sql({ dialect: PostgreSQL }),
}

// Legacy mode mapping (StreamLanguage wrappers)
const legacyMap: Record<string, any> = {
  yaml,
  toml,
  ini: properties,
  ruby,
  swift,
  objectivec: objectiveC,
  kotlin,
  scala,
  groovy,
  csharp,
  lua,
  r,
  dart,
  bash: shell,
  fish: shell,
  powershell: powerShell,
  dos: mscgen,
  dockerfile: dockerFile,
  makefile: cmake,
  cmake,
  protobuf,
  latex: stex,
  elm,
  erlang,
  haskell,
  clojure,
  fortran,
  perl,
  scheme,
}

const legacyCache: Record<string, Extension> = {}

function loadLegacy(lang: string): Extension | null {
  if (legacyCache[lang]) return legacyCache[lang]
  const mode = legacyMap[lang]
  if (!mode) return null
  const ext = StreamLanguage.define(mode)
  legacyCache[lang] = ext
  return ext
}

export function getLanguageExtension(lang: string): Extension | null {
  if (!lang || lang === 'plaintext') return null
  const fn = officialMap[lang]
  if (fn) return fn()
  return loadLegacy(lang)
}
