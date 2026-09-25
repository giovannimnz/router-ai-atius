#!/usr/bin/env bash
# ==============================================================================
# scripts/sync-vendor-icons.sh — Sincronizador de Ícones SVG de Fornecedores e Modelos
#
# Mantém a separação estrita da arquitetura Router AI Atius:
# - Versão Monocromática (b&w): Fornecedores (Vendor column, Pricing, badges neutros)
# - Versão Colorida (color): Modelos de IA e Channel logos dedicados
# ==============================================================================
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEFAULT_SRC_DIR="/home/ubuntu/Imagens"
SRC_DIR="${VENDOR_ICONS_SRC_DIR:-$DEFAULT_SRC_DIR}"
PUBLIC_IMAGES_DIR="$ROOT_DIR/web/default/public/images"
PUBLIC_LOGOS_DIR="$PUBLIC_IMAGES_DIR/logos"
DIST_IMAGES_DIR="$ROOT_DIR/web/default/dist/images"
DIST_LOGOS_DIR="$DIST_IMAGES_DIR/logos"
COMPONENTS_DIR="$ROOT_DIR/web/default/src/components"

usage() {
  cat <<EOF
Uso: $0 [COMANDO|OPÇÃO]

Comandos:
  auto                  Coleta e sincroniza automaticamente todos os ícones da pasta de origem ($SRC_DIR)
  status                Exibe status de sincronização e hashes SHA256 entre origem e destino
  select [fornecedor]   Escolhe fornecedor específico para atualizar (antigravity, atius, etc.)
  build                 Executa typecheck e build do frontend sob o cgroup CPU guardrail (20% CPU)

Opções:
  --src <dir>           Define diretório de origem (padrão: $DEFAULT_SRC_DIR)
  --with-build          Executa typecheck e build após sincronização
  -h, --help            Exibe esta ajuda

Exemplos:
  $0 auto
  $0 select antigravity
  $0 select atius
  $0 status
  $0 auto --with-build
EOF
}

log_info() {
  echo -e "\033[1;34m[INFO]\033[0m $*"
}

log_success() {
  echo -e "\033[1;32m[OK]\033[0m $*"
}

log_warn() {
  echo -e "\033[1;33m[AVISO]\033[0m $*"
}

log_error() {
  echo -e "\033[1;31m[ERRO]\033[0m $*" >&2
}

sync_antigravity() {
  log_info "Sincronizando ícones do Antigravity..."
  local bw_src="$SRC_DIR/antigravity.svg"
  local col_src="$SRC_DIR/antigravity-color.svg"

  if [[ ! -f "$bw_src" ]]; then
    log_warn "Arquivo monocromático não encontrado: $bw_src"
  else
    cp "$bw_src" "$PUBLIC_IMAGES_DIR/antigravity.svg"
    [[ -d "$DIST_IMAGES_DIR" ]] && cp "$bw_src" "$DIST_IMAGES_DIR/antigravity.svg"
    log_success "Copiado $bw_src -> $PUBLIC_IMAGES_DIR/antigravity.svg"
  fi

  if [[ ! -f "$col_src" ]]; then
    log_warn "Arquivo colorido não encontrado: $col_src"
  else
    cp "$col_src" "$PUBLIC_IMAGES_DIR/antigravity-color.svg"
    [[ -d "$DIST_IMAGES_DIR" ]] && cp "$col_src" "$DIST_IMAGES_DIR/antigravity-color.svg"
    log_success "Copiado $col_src -> $PUBLIC_IMAGES_DIR/antigravity-color.svg"
  fi
}

sync_atius() {
  log_info "Sincronizando ícones do Atius..."
  local bw_src="$SRC_DIR/atius.svg"
  local col_src="$SRC_DIR/atius-color.svg"

  if [[ ! -f "$bw_src" ]]; then
    log_warn "Arquivo monocromático não encontrado: $bw_src"
  else
    cp "$bw_src" "$PUBLIC_IMAGES_DIR/atius.svg"
    cp "$bw_src" "$PUBLIC_LOGOS_DIR/logo-atius.svg"
    if [[ -d "$DIST_IMAGES_DIR" ]]; then
      mkdir -p "$DIST_LOGOS_DIR"
      cp "$bw_src" "$DIST_IMAGES_DIR/atius.svg"
      cp "$bw_src" "$DIST_LOGOS_DIR/logo-atius.svg"
    fi
    log_success "Copiado $bw_src -> $PUBLIC_IMAGES_DIR/atius.svg e $PUBLIC_LOGOS_DIR/logo-atius.svg"
  fi

  if [[ ! -f "$col_src" ]]; then
    log_warn "Arquivo colorido não encontrado: $col_src"
  else
    cp "$col_src" "$PUBLIC_IMAGES_DIR/atius-color.svg"
    cp "$col_src" "$PUBLIC_LOGOS_DIR/logo-atius-color.svg"
    if [[ -d "$DIST_IMAGES_DIR" ]]; then
      mkdir -p "$DIST_LOGOS_DIR"
      cp "$col_src" "$DIST_IMAGES_DIR/atius-color.svg"
      cp "$col_src" "$DIST_LOGOS_DIR/logo-atius-color.svg"
    fi
    log_success "Copiado $col_src -> $PUBLIC_IMAGES_DIR/atius-color.svg e $PUBLIC_LOGOS_DIR/logo-atius-color.svg"
  fi
}

sync_typesafe() {
  log_info "Sincronizando ícones do TypeSafe..."
  local ts_src="$SRC_DIR/typesafe.svg"
  if [[ -f "$ts_src" ]]; then
    cp "$ts_src" "$PUBLIC_IMAGES_DIR/typesafe.svg"
    cp "$ts_src" "$PUBLIC_LOGOS_DIR/logo-typesafe.svg"
    cp "$ts_src" "$PUBLIC_LOGOS_DIR/typesafe.svg"
    if [[ -d "$DIST_IMAGES_DIR" ]]; then
      mkdir -p "$DIST_LOGOS_DIR"
      cp "$ts_src" "$DIST_IMAGES_DIR/typesafe.svg"
      cp "$ts_src" "$DIST_LOGOS_DIR/logo-typesafe.svg"
      cp "$ts_src" "$DIST_LOGOS_DIR/typesafe.svg"
    fi
    log_success "Copiado $ts_src para diretórios de destino"
  else
    log_warn "Arquivo TypeSafe não encontrado em $ts_src"
  fi
}

sync_all() {
  log_info "Iniciando sincronização completa a partir de: $SRC_DIR"
  [[ -d "$SRC_DIR" ]] || { log_error "Diretório de origem não existe: $SRC_DIR"; exit 1; }
  mkdir -p "$PUBLIC_IMAGES_DIR" "$PUBLIC_LOGOS_DIR"

  sync_antigravity
  sync_atius
  sync_typesafe
  log_success "Sincronização de arquivos SVG concluída!"
}

show_status() {
  echo "=== Status de Ícones SVG de Fornecedores ==="
  echo "Origem:  $SRC_DIR"
  echo "Destino: $PUBLIC_IMAGES_DIR"
  echo ""

  local files=("antigravity.svg" "antigravity-color.svg" "atius.svg" "atius-color.svg" "typesafe.svg")
  for f in "${files[@]}"; do
    local src_file="$SRC_DIR/$f"
    local dst_file="$PUBLIC_IMAGES_DIR/$f"

    echo "--- $f ---"
    if [[ -f "$src_file" ]]; then
      local src_sha
      src_sha="$(sha256sum "$src_file" | awk '{print $1}')"
      echo "  Origem:  $src_sha ($(date -r "$src_file" '+%Y-%m-%d %H:%M:%S'))"
    else
      echo "  Origem:  [NÃO ENCONTRADO]"
    fi

    if [[ -f "$dst_file" ]]; then
      local dst_sha
      dst_sha="$(sha256sum "$dst_file" | awk '{print $1}')"
      echo "  Destino: $dst_sha ($(date -r "$dst_file" '+%Y-%m-%d %H:%M:%S'))"
      if [[ -f "$src_file" ]]; then
        if [[ "$src_sha" == "$dst_sha" ]]; then
          echo "  Status:  SINCRONIZADO (OK)"
        else
          echo "  Status:  DESATUALIZADO (Divergência de hash)"
        fi
      fi
    else
      echo "  Destino: [NÃO ENCONTRADO]"
    fi
  done
}

run_build() {
  log_info "Executando typecheck e build sob CPU Guardrail (20% CPU)..."
  "$ROOT_DIR/scripts/podman-admin.sh" profile-run -- bash -lc "cd '$ROOT_DIR/web/default' && bun run typecheck"
  log_success "Typecheck concluído com sucesso!"
}

# Processamento de argumentos
DO_BUILD=0
CMD=""
TARGET=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --src)
      SRC_DIR="$2"
      shift 2
      ;;
    --with-build)
      DO_BUILD=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    auto|status|build)
      CMD="$1"
      shift
      ;;
    select)
      CMD="select"
      TARGET="${2:-}"
      if [[ -n "$TARGET" ]]; then
        shift 2
      else
        shift
      fi
      ;;
    *)
      log_error "Argumento desconhecido: $1"
      usage
      exit 1
      ;;
  esac
done

if [[ -z "$CMD" ]]; then
  CMD="auto"
fi

case "$CMD" in
  auto)
    sync_all
    ;;
  status)
    show_status
    ;;
  select)
    if [[ -z "$TARGET" ]]; then
      echo "Escolha o fornecedor para sincronizar:"
      echo "  1) Antigravity (antigravity.svg + antigravity-color.svg)"
      echo "  2) Atius (atius.svg + atius-color.svg)"
      echo "  3) TypeSafe (typesafe.svg)"
      echo "  4) Todos"
      read -r -p "Opção [1-4]: " opt
      case "$opt" in
        1) sync_antigravity ;;
        2) sync_atius ;;
        3) sync_typesafe ;;
        4) sync_all ;;
        *) log_error "Opção inválida"; exit 1 ;;
      esac
    else
      case "${TARGET,,}" in
        antigravity|agy) sync_antigravity ;;
        atius|local)     sync_atius ;;
        typesafe)        sync_typesafe ;;
        all|todos)       sync_all ;;
        *)
          log_error "Fornecedor desconhecido: $TARGET. Opções: antigravity, atius, typesafe, all"
          exit 1
          ;;
      esac
    fi
    ;;
  build)
    run_build
    exit 0
    ;;
esac

if [[ "$DO_BUILD" -eq 1 ]]; then
  run_build
fi
