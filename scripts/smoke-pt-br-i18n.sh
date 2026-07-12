#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

required_files=(
  i18n/locales/pt.yaml
  web/default/src/i18n/locales/pt.json
  web/classic/src/i18n/locales/pt.json
)
for path in "${required_files[@]}"; do
  test -s "$path" || { echo "PT-BR ausente: $path" >&2; exit 1; }
done

rg -q "LangPt[[:space:]]+= \"pt\"" i18n/i18n.go
rg -q "locales/pt.yaml" i18n/i18n.go
rg -q "import pt from './locales/pt.json'" web/default/src/i18n/config.ts
rg -q "supportedLngs:.*'pt'" web/default/src/i18n/config.ts
rg -q "code: 'pt', label: 'Português'" web/default/src/i18n/languages.ts
rg -q "normalized.startsWith\('pt'\)" web/default/src/i18n/languages.ts
rg -q "ptTranslation" web/classic/src/i18n/i18n.js
rg -q "lower.startsWith\('pt'\)" web/classic/src/i18n/language.js

node <<'NODE'
const fs = require('node:fs')

function translation(path) {
  return JSON.parse(fs.readFileSync(path, 'utf8')).translation
}

function assertSameKeys(basePath, ptPath) {
  const base = translation(basePath)
  const pt = translation(ptPath)
  const baseKeys = Object.keys(base).sort()
  const ptKeys = Object.keys(pt).sort()
  if (JSON.stringify(baseKeys) !== JSON.stringify(ptKeys)) {
    throw new Error(`PT-BR fora de sincronia: ${ptPath}`)
  }

  const placeholder = /\{\{[^}]+\}\}/g
  for (const key of baseKeys) {
    const expected = String(base[key]).match(placeholder) ?? []
    const actual = String(pt[key]).match(placeholder) ?? []
    if (JSON.stringify(expected.sort()) !== JSON.stringify(actual.sort())) {
      throw new Error(`Placeholders divergentes em ${ptPath}: ${key}`)
    }
  }
}

assertSameKeys(
  'web/default/src/i18n/locales/en.json',
  'web/default/src/i18n/locales/pt.json'
)
assertSameKeys(
  'web/classic/src/i18n/locales/en.json',
  'web/classic/src/i18n/locales/pt.json'
)
NODE

echo "PT-BR i18n smoke: OK"
