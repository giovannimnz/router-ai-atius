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

function flatYaml(path) {
  const entries = new Map()
  let currentKey = null

  for (const [index, line] of fs.readFileSync(path, 'utf8').split(/\r?\n/).entries()) {
    if (/^\s*(?:#|$)/.test(line)) continue

    const entry = line.match(/^([^\s][^:]*):(?:\s*(.*))?$/)
    if (entry) {
      currentKey = entry[1]
      if (entries.has(currentKey)) {
        throw new Error(`Chave YAML duplicada em ${path}:${index + 1}: ${currentKey}`)
      }
      entries.set(currentKey, entry[2] ?? '')
      continue
    }

    if (/^\s+/.test(line) && currentKey) {
      entries.set(currentKey, `${entries.get(currentKey)} ${line.trim()}`)
      continue
    }

    throw new Error(`YAML backend inesperado em ${path}:${index + 1}`)
  }

  return entries
}

function placeholders(value) {
  return String(value).match(/\{\{[^}]+\}\}/g)?.sort() ?? []
}

function expectedJsonPlaceholders(key, baseValue) {
  return String(baseValue).trim() === '' ? placeholders(key) : placeholders(baseValue)
}

function assertSameYamlKeysAndPlaceholders(basePath, ptPath) {
  const base = flatYaml(basePath)
  const pt = flatYaml(ptPath)
  const baseKeys = [...base.keys()].sort()
  const ptKeys = [...pt.keys()].sort()
  if (JSON.stringify(baseKeys) !== JSON.stringify(ptKeys)) {
    const missing = baseKeys.filter((key) => !pt.has(key))
    const extra = ptKeys.filter((key) => !base.has(key))
    throw new Error(
      `PT-BR backend fora de sincronia: ausentes=${missing.join(',') || '-'}; extras=${extra.join(',') || '-'}`,
    )
  }

  for (const key of baseKeys) {
    if (JSON.stringify(placeholders(base.get(key))) !== JSON.stringify(placeholders(pt.get(key)))) {
      throw new Error(`Placeholders backend divergentes em ${ptPath}: ${key}`)
    }
  }
}

function assertNoEmptyValues(path) {
  const emptyKeys = Object.entries(translation(path))
    .filter(([, value]) => typeof value !== 'string' || value.trim() === '')
    .map(([key]) => key)
  if (emptyKeys.length > 0) {
    throw new Error(`Valores PT-BR vazios em ${path}: ${emptyKeys.join(', ')}`)
  }
}

function assertSameKeys(basePath, ptPath) {
  const base = translation(basePath)
  const pt = translation(ptPath)
  const baseKeys = Object.keys(base).sort()
  const ptKeys = Object.keys(pt).sort()
  if (JSON.stringify(baseKeys) !== JSON.stringify(ptKeys)) {
    throw new Error(`PT-BR fora de sincronia: ${ptPath}`)
  }

  for (const key of baseKeys) {
    if (JSON.stringify(expectedJsonPlaceholders(key, base[key])) !== JSON.stringify(placeholders(pt[key]))) {
      throw new Error(`Placeholders divergentes em ${ptPath}: ${key}`)
    }
  }
}

assertSameYamlKeysAndPlaceholders(
  'i18n/locales/en.yaml',
  'i18n/locales/pt.yaml'
)

assertSameKeys(
  'web/default/src/i18n/locales/en.json',
  'web/default/src/i18n/locales/pt.json'
)
assertSameKeys(
  'web/classic/src/i18n/locales/en.json',
  'web/classic/src/i18n/locales/pt.json'
)
assertNoEmptyValues('web/default/src/i18n/locales/pt.json')
assertNoEmptyValues('web/classic/src/i18n/locales/pt.json')
NODE

echo "PT-BR i18n smoke: OK"
