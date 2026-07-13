#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$repo_root"

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

bash -n scripts/k3s-router-go-no-go.sh scripts/k3s-router-rollback-check.sh
shellcheck -x scripts/k3s-router-go-no-go.sh scripts/k3s-router-rollback-check.sh

scripts/k3s-router-rollback-check.sh --self-test
scripts/k3s-router-go-no-go.sh --self-test

grep -Fq 'bin/clianything status --backend podman' scripts/k3s-router-rollback-check.sh ||
  fail 'rollback check does not select the Podman CLI backend explicitly'
grep -Fq 'apache_config=/etc/apache2/sites-enabled/router.atius.com.br-le-ssl.conf' scripts/k3s-router-rollback-check.sh ||
  fail 'rollback check does not pin the canonical enabled Apache vhost'
grep -Fq 'apache2ctl configtest' scripts/k3s-router-rollback-check.sh ||
  fail 'rollback check does not run Apache configtest'
grep -Fq '/unexpected router route fixture was accepted' scripts/k3s-router-rollback-check.sh ||
  fail 'rollback check does not reject an extra router path'
grep -Fq 'live EndpointSlices differ from shadow apply/smoke snapshots' scripts/k3s-router-go-no-go.sh ||
  fail 'go/no-go does not compare complete live EndpointSlices with apply/smoke'
grep -Fq 'replacement EndpointSlice with the same endpoint was accepted' scripts/k3s-router-go-no-go.sh ||
  fail 'go/no-go does not reject a replacement EndpointSlice with the same endpoint'
if rg -n 'PHASE29_APACHE_CONFIG|--apache-config' scripts/k3s-router-rollback-check.sh >/dev/null; then
  fail 'rollback check still allows an Apache path override'
fi
grep -Fq "rollback-\$run_id.json" scripts/k3s-router-go-no-go.sh ||
  fail 'go/no-go does not generate a fresh rollback artifact for its run_id'
grep -Fq "live-identity-\$run_id.json" scripts/k3s-router-go-no-go.sh ||
  fail 'go/no-go does not persist an exact run-bound live identity map'
if scripts/k3s-router-go-no-go.sh --verify-existing /tmp/fabricated-decision.json >/dev/null 2>&1; then
  fail '--verify-existing accepted an externally supplied decision'
fi

if rg -n '\b(systemctl[[:space:]]+(--user[[:space:]]+)?(start|stop|restart|reload)|apache2ctl[[:space:]]+(graceful|restart)|apachectl[[:space:]]+(graceful|restart)|a2en(site|mod)|a2dis(site|mod)|kubectl[[:space:]]+(apply|patch|delete|edit|label|annotate|replace|scale|rollout[[:space:]]+restart)|podman[[:space:]]+(start|stop|restart|rm|kill|run|exec|compose))\b' \
  scripts/k3s-router-go-no-go.sh scripts/k3s-router-rollback-check.sh >/dev/null; then
  fail 'Phase 29 final gates contain a runtime mutation command'
fi

if rg -n '(Authorization:|Bearer[[:space:]]+\$|POSTGRES_PASSWORD=|REDIS_PASSWORD=|SESSION_SECRET=|postgres(ql)?://)' \
  scripts/k3s-router-go-no-go.sh scripts/k3s-router-rollback-check.sh >/dev/null; then
  fail 'Phase 29 final gates contain a secret-bearing pattern'
fi

echo 'phase29 GO/NO-GO and rollback self-tests: PASS'
