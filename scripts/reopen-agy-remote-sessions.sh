#!/usr/bin/env zsh
# ==============================================================================
# reopen-agy-remote-sessions.sh
# 
# Encerra sessões agy ativas no tmux que NÃO possuem a flag --remote-control
# e as reabre nos mesmos painéis tmux com a configuração canônica completa:
#   agy --remote-control --dangerously-skip-permissions --mode accept-edits --conversation=<UUID>
#
# Preserva integralmente sessões que já possuem --remote-control (ex: router-ai-atius).
# Limpa processos órfãos desconectados que retêm travas de presença em ~/.gemini/.../presence/.
# ==============================================================================

set -e

DRY_RUN=0
VERBOSE=0

for arg in "$@"; do
    case "$arg" in
        --dry-run)
            DRY_RUN=1
            ;;
        --verbose|-v)
            VERBOSE=1
            ;;
        --help|-h)
            echo "Uso: $0 [--dry-run] [--verbose]"
            echo ""
            echo "Opções:"
            echo "  --dry-run    Apenas exibe as sessões e o que seria feito, sem alterar nada"
            echo "  --verbose    Exibe detalhes adicionais de processos e inspeção"
            echo "  --help       Exibe esta mensagem de ajuda"
            exit 0
            ;;
    esac
done

echo "======================================================================"
echo "   REABERTURA DE SESSÕES AGY COM REMOTE CONTROL (CANÔNICO ATIUS)     "
echo "======================================================================"
echo "Horário: $(date '+%Y-%m-%d %H:%M:%S %Z')"
echo "Host:    $(hostname)"
echo "Shell:   ~/.zshrc (Canônico)"
echo ""

# 1. Verificar status do daemon PM2 atius-srv-1-agy
echo "▸ [1/5] Verificando daemon PM2 atius-srv-1-agy..."
if command -v pm2 >/dev/null 2>&1; then
    PM2_STATUS=$(pm2 jlist 2>/dev/null | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    p = next((x for x in data if x.get('name') == 'atius-srv-1-agy'), None)
    if p:
        print(p.get('pm2_env', {}).get('status', 'unknown'))
    else:
        print('not_found')
except Exception as e:
    print('error')
")
    if [ "$PM2_STATUS" = "online" ]; then
        echo "   ✓ Daemon PM2 'atius-srv-1-agy' está ONLINE."
    else
        echo "   ⚠️ Aviso: Daemon PM2 'atius-srv-1-agy' status: $PM2_STATUS"
    fi
else
    echo "   ⚠️ Aviso: pm2 não encontrado no PATH."
fi

# 2. Executar script de análise em Python
echo ""
echo "▸ [2/5] Inspecionando painéis tmux e travas de presença..."

PYTHON_ANALYSIS=$(python3 - <<'EOF'
import subprocess, os, psutil, glob, json

# Mapeamento de travas de presença
presence_dir = os.path.expanduser('~/.gemini/antigravity-cli/presence')
locks = {}
if os.path.exists(presence_dir):
    for lf in glob.glob(os.path.join(presence_dir, '*.lock')):
        uuid = os.path.basename(lf).replace('.lock', '')
        try:
            pids = subprocess.check_output(['fuser', lf], stderr=subprocess.DEVNULL).decode().strip().split()
            for p in pids:
                locks[int(p)] = uuid
        except:
            pass

# Listar painéis tmux
panes = []
try:
    raw = subprocess.check_output(['tmux', 'list-panes', '-a', '-F', '#{session_name}:#{window_index}.#{pane_index} #{pane_pid} #{pane_current_path}']).decode()
    for line in raw.splitlines():
        if line.strip():
            parts = line.strip().split()
            panes.append({'target': parts[0], 'pane_pid': int(parts[1]), 'cwd': parts[2]})
except Exception as e:
    pass

active_agy_pids = set()
pane_results = []

for p in panes:
    try:
        pane_proc = psutil.Process(p['pane_pid'])
        children = pane_proc.children(recursive=True)
        agy_proc = None
        for c in children:
            if 'agy' in c.name().lower() or (c.cmdline() and 'agy' in c.cmdline()[0]):
                agy_proc = c
                break
        
        has_remote = False
        conv_id = None
        agy_pid = None
        cmdline = []
        
        if agy_proc:
            agy_pid = agy_proc.pid
            active_agy_pids.add(agy_pid)
            cmdline = agy_proc.cmdline()
            has_remote = '--remote-control' in cmdline
            
            for arg in cmdline:
                if arg.startswith('--conversation='):
                    conv_id = arg.split('=', 1)[1]
                elif arg == '--conversation' and len(cmdline) > cmdline.index(arg) + 1:
                    conv_id = cmdline[cmdline.index(arg) + 1]
            
            if not conv_id and agy_pid in locks:
                conv_id = locks[agy_pid]

        pane_results.append({
            'target': p['target'],
            'cwd': p['cwd'],
            'agy_pid': agy_pid,
            'has_remote': has_remote,
            'conv_id': conv_id,
            'cmd': ' '.join(cmdline)
        })
    except Exception:
        pass

# Detectar processos agy CLI órfãos que não pertencem a nenhum painel tmux
orphans = []
for proc in psutil.process_iter(['pid', 'name', 'cmdline', 'ppid']):
    try:
        if proc.info['pid'] in active_agy_pids:
            continue
        c = proc.info['cmdline']
        if not c:
            continue
        # Apenas binário agy CLI exato, nunca servidores IDE ou VSCode
        base_cmd = os.path.basename(c[0]).lower()
        if base_cmd != 'agy':
            continue
        cmd_str = ' '.join(c)
        if any(ign in cmd_str for ign in ['remote-control serve', 'start-agy.sh', 'mcp', 'ide-server', 'language_server', ' -p ', ' --print']):
            continue
        orphans.append({
            'pid': proc.info['pid'],
            'cmd': cmd_str,
            'lock': locks.get(proc.info['pid'], None)
        })
    except:
        pass

print(json.dumps({'panes': pane_results, 'orphans': orphans}))
EOF
)

export PYTHON_ANALYSIS
export DRY_RUN

# 3. Processar resultados da análise
python3 - <<'EOF'
import json, os, sys

data = json.loads(os.environ['PYTHON_ANALYSIS'])

print('Sessões detectadas:')
for p in data['panes']:
    if not p['agy_pid']:
        status = '\033[0;37mSEM AGY ATIVO\033[0m'
    elif p['has_remote']:
        status = '\033[1;32mCORRETO (com --remote-control)\033[0m'
    else:
        status = '\033[1;31mINCORRETO (falta --remote-control)\033[0m'
    print(f" • Painel {p['target']} [{p['cwd']}]: {status} (PID {p['agy_pid'] or '-'}, Conv: {p['conv_id'] or 'nova'})")

if data['orphans']:
    print('\nProcessos órfãos detectados (fora de painéis tmux):')
    for o in data['orphans']:
        print(f" • PID {o['pid']}: {o['cmd'][:70]}... (Trava: {o['lock'] or 'nenhuma'})")
EOF

# 4. Limpeza de órfãos
echo ""
echo "▸ [3/5] Limpando processos órfãos e liberando travas presas..."
if [ "$DRY_RUN" -eq 1 ]; then
    echo "   [DRY-RUN] Processos órfãos e travas seriam finalizados aqui."
else
    python3 - <<'EOF'
import json, os, signal, time, psutil, subprocess

data = json.loads(os.environ['PYTHON_ANALYSIS'])
for o in data['orphans']:
    pid = o['pid']
    try:
        p = psutil.Process(pid)
        print(f"   Finalizando processo órfão PID {pid}...")
        p.send_signal(signal.SIGTERM)
        time.sleep(0.5)
        if p.is_running():
            p.send_signal(signal.SIGKILL)
    except Exception:
        pass

# Limpar locks não referenciados por processos vivos
presence_dir = os.path.expanduser('~/.gemini/antigravity-cli/presence')
if os.path.exists(presence_dir):
    for lf in os.listdir(presence_dir):
        if lf.endswith('.lock'):
            p = os.path.join(presence_dir, lf)
            try:
                out = subprocess.check_output(['fuser', p], stderr=subprocess.DEVNULL).decode().strip()
                if not out:
                    os.remove(p)
                    print(f"   Removida trava órfã vazia: {lf}")
            except Exception:
                pass
EOF
    echo "   ✓ Processos órfãos e travas limpos."
fi

# 5. Executar reabertura nos painéis incorretos
echo ""
echo "▸ [4/5] Executando encerramento e reabertura com --remote-control..."

python3 - <<'EOF'
import json, subprocess, time, os, psutil, signal

data = json.loads(os.environ['PYTHON_ANALYSIS'])
dry_run = (os.environ.get('DRY_RUN') == '1')

target_panes = [p for p in data['panes'] if p['agy_pid'] and not p['has_remote']]

if not target_panes:
    print("   Nenhum painel precisa de reabertura! Todas as sessões agy já possuem --remote-control.")
else:
    for p in target_panes:
        target = p['target']
        pid = p['agy_pid']
        conv_id = p['conv_id']
        cwd = p['cwd']
        
        print(f"\n   ------------------------------------------------------------------")
        print(f"   Processando painel {target} ({cwd})")
        print(f"   PID Atual: {pid} | Conversa: {conv_id}")
        
        if dry_run:
            print(f"   [DRY-RUN] Enviaria C-c para {target}, aguardaria término do PID {pid}")
            if conv_id:
                print(f"   [DRY-RUN] Reabriria com: agy --remote-control --dangerously-skip-permissions --mode accept-edits --conversation={conv_id}")
            else:
                print(f"   [DRY-RUN] Reabriria com: agy --remote-control --dangerously-skip-permissions --mode accept-edits -c")
            continue
            
        # 1. Enviar C-c para o painel tmux
        print(f"   Enviando sinal de saída (C-c) para painel {target}...")
        subprocess.run(['tmux', 'send-keys', '-t', target, 'C-c'])
        
        # 2. Aguardar processo sair
        terminated = False
        for _ in range(6):
            time.sleep(0.5)
            if not psutil.pid_exists(pid):
                terminated = True
                break
                
        if not terminated:
            print(f"   Processo {pid} ainda ativo. Enviando SIGTERM...")
            try:
                os.kill(pid, signal.SIGTERM)
                time.sleep(1)
                if psutil.pid_exists(pid):
                    print(f"   Enviando SIGKILL para {pid}...")
                    os.kill(pid, signal.SIGKILL)
                    time.sleep(0.5)
            except ProcessLookupError:
                pass
                
        # 3. Liberar trava específica se ainda existir
        if conv_id:
            lock_path = os.path.expanduser(f'~/.gemini/antigravity-cli/presence/{conv_id}.lock')
            if os.path.exists(lock_path):
                try:
                    subprocess.run(['fuser', '-k', lock_path], stderr=subprocess.DEVNULL)
                    time.sleep(0.3)
                    if os.path.exists(lock_path):
                        os.remove(lock_path)
                except Exception:
                    pass

        # 4. Reabrir com o comando canônico completo
        cmd_args = "agy --remote-control --dangerously-skip-permissions --mode accept-edits"
        if conv_id:
            cmd_args += f" --conversation={conv_id}"
        else:
            cmd_args += " -c"
            
        print(f"   Reabrindo sessão com Remote Control...")
        print(f"   Comando enviado: {cmd_args}")
        subprocess.run(['tmux', 'send-keys', '-t', target, cmd_args, 'Enter'])
        time.sleep(2)

EOF

# 6. Validação final
echo ""
echo "▸ [5/5] Validação do estado pós-execução..."

if [ "$DRY_RUN" -eq 1 ]; then
    echo "   [DRY-RUN] Validação concluída (nenhuma alteração foi feita)."
else
    sleep 2
    python3 - <<'EOF'
import subprocess, psutil

raw = subprocess.check_output(['tmux', 'list-panes', '-a', '-F', '#{session_name}:#{window_index}.#{pane_index} #{pane_pid} #{pane_current_path}']).decode()
print("Estado atualizado dos painéis:")
for line in raw.splitlines():
    if not line.strip(): continue
    parts = line.strip().split()
    target, ppid, cwd = parts[0], int(parts[1]), parts[2]
    
    try:
        proc = psutil.Process(ppid)
        children = proc.children(recursive=True)
        agy = None
        for c in children:
            if 'agy' in c.name().lower() or (c.cmdline() and 'agy' in c.cmdline()[0]):
                agy = c
                break
        if agy:
            cmd = ' '.join(agy.cmdline())
            remote = '--remote-control' in cmd
            st = '\033[1;32m✓ ONLINE COM REMOTE CONTROL\033[0m' if remote else '\033[1;31m✗ SEM REMOTE CONTROL\033[0m'
            print(f" • Painel {target} [{cwd}]: PID {agy.pid} -> {st}")
        else:
            print(f" • Painel {target} [{cwd}]: Sem agy ativo")
    except Exception as e:
        print(f" • Painel {target}: erro de leitura ({e})")
EOF
fi

echo ""
echo "======================================================================"
echo "Sessões registradas no Antigravity Remote Control (host: atius-srv-1-agy)."
echo "Acesse https://antigravity.google.com para interagir com as sessões ativas."
echo "======================================================================"
