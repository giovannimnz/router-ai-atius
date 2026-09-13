---
status: diagnosed
trigger: "Audite de forma read-only a causa de terminais/conhost/PowerShell visiveis ou orfaos no GIOVANNI-W11-PC. Acesso WSL: ssh -p 8222 muniz@ssh-giovanni-wsl-pc.atius.com.br; Windows: ssh -p 8122 muniz@ssh-giovanni-w11-pc.atius.com.br. Arquitetura: PM2 no WSL chama /home/muniz/.local/share/mcp-gateway-local/run-windows-bridge.sh, que usa pwsh -NoLogo -NoProfile -NonInteractive -ExecutionPolicy Bypass -WindowStyle Hidden para scripts em C:\\Users\\muniz\\.local\\share\\mcp-gateway-windows; tarefas agendadas e CUA daemon. Antes havia 510 conhost, 466 orfaos foram encerrados, e um WindowsTerminal visivel foi fechado. Nao altere arquivos/processos e nao abra UI. Foque em loops/restarts, launchers visiveis e risco de recorrencia. Retorne hipoteses com evidencias, comandos zero-UI e recomendacoes precisas. Voce nao esta sozinho; nao reverta mudancas."
created: 2026-08-22T05:25:16-03:00
updated: 2026-08-22T05:48:00-03:00
---

## Current Focus

hypothesis: confirmed: PM2 owns only the WSL/pwsh wrapper, while detached Windows Start-Process descendants survive abrupt wrapper stop/replacement; aggressive restart policy multiplies the leaked trees and conhosts
test: completed read-only comparison of PM2 lifecycle, Windows parentage, logs, task settings, launchers, registry, and current readiness
expecting: diagnosis complete
next_action: report ranked findings, zero-UI audit commands, and precise remediation recommendations without applying changes

## Symptoms

expected: Background bridge, scheduled tasks, and CUA daemon run without visible terminal windows and without accumulating orphaned conhost or PowerShell processes.
actual: Previously 510 conhost processes existed, 466 orphans were terminated, and one visible WindowsTerminal was closed; recurrence risk is unknown.
errors: No explicit error message supplied; symptom is visible Windows Terminal/conhost/PowerShell processes and prior orphan accumulation.
reproduction: Observe the Windows desktop/process inventory while WSL PM2 launches the Windows bridge, scheduled tasks run, and the CUA daemon operates.
started: Exact start time unknown; large orphan population and one visible WindowsTerminal were observed before this audit.

## Eliminated

- hypothesis: A persistent multi-entry PSModulePath in powershell.config.json or the PM2 daemon environment causes the exact PowerShell startIndex startup exception.
  evidence: PowerShell 7.6.3 has no user powershell.config.json, the all-users config contains only execution policy and compatibility deny-list, and /proc/366/environ contains no PSModulePath or WSLENV. The machine PSModulePath is the normal Windows two-entry value.
  timestamp: 2026-08-22T05:36:00-03:00

- hypothesis: An app-specific PM2 PSModulePath/WSLENV override causes the transient PowerShell startIndex exception.
  evidence: Live PM2 metadata for the CUA apps contains only a Linux PATH among selected module/path/terminal variables; PSModulePath and WSLENV are absent.
  timestamp: 2026-08-22T05:37:00-03:00

- hypothesis: A standard HKCU/HKLM Run or Startup-folder entry launched the 04:34 WSL/Hermes process and visible terminal.
  evidence: Run/RunOnce keys contain no WSL/Hermes/Terminal launcher; user Startup contains only desktop.ini and common Startup only AnyDesk plus desktop.ini.
  timestamp: 2026-08-22T05:39:00-03:00

- hypothesis: CUA/Hermes launchers directly invoke WindowsTerminal or intentionally create visible console windows.
  evidence: CUA and Hermes VBS use WScript.Shell.Run window style 0; pwsh uses WindowStyle Hidden; bridge proxy/tunnel children use WindowStyle Hidden; no task/Run/Startup action calls wt.exe or WindowsTerminal.
  timestamp: 2026-08-22T05:47:00-03:00

- hypothesis: node_repl.exe alone accounts for the prior 466 orphan conhosts.
  evidence: Only six node_repl/conhost pairs currently exist; the active population is distributed across ssh, mcp-proxy, node, wslhost, cmd, winpty, CUA, and other parents.
  timestamp: 2026-08-22T05:47:00-03:00

## Evidence

- timestamp: 2026-08-22T05:25:16-03:00
  checked: user-provided architecture and prior cleanup evidence
  found: WSL PM2 invokes run-windows-bridge.sh, which invokes pwsh with NoLogo, NoProfile, NonInteractive, ExecutionPolicy Bypass, and WindowStyle Hidden; 510 conhost existed previously, 466 classified as orphans were ended, and one visible WindowsTerminal was closed.
  implication: The bridge's documented PowerShell flags should hide its own shell, so recurrence may originate in a child launcher, scheduled-task action, daemon restart loop, or detachment behavior rather than the top-level pwsh invocation alone.

- timestamp: 2026-08-22T05:27:00-03:00
  checked: mandatory debugger patterns and relevant Windows/WSL and PM2 operational guidance
  found: Resource growth maps to async/timing cleanup leaks; PM2 normally autorestarts exited processes and requires explicit delay/backoff/max-restart controls; Windows CUA commonly starts from an interactive scheduled task at logon.
  implication: Restart counts/timing, child-process ownership, and scheduled-task logon/window settings are discriminating evidence; merely seeing WindowStyle Hidden on the top-level pwsh command is not sufficient.

- timestamp: 2026-08-22T05:28:00-03:00
  checked: live WSL process inventory and complete run-windows-bridge.sh
  found: PM2 daemon PID 366 and one direct Node process were alive for about 3h42m; no persistent pwsh/powershell bridge process appeared in the filtered WSL inventory. The wrapper sources resolve-wslinterop.sh and then uses exec to replace itself with PowerShell 7 using NoLogo, NoProfile, NonInteractive, ExecutionPolicy Bypass, WindowStyle Hidden, and a fixed -File path.
  implication: The wrapper does not retain an intermediate Bash parent and explicitly requests a hidden PowerShell window. A visible terminal therefore requires either a downstream child/task that ignores those flags, an independently launched terminal, or a platform-specific console allocation despite hidden startup; PM2 restart counters are needed to test recurrence pressure.

- timestamp: 2026-08-22T05:29:00-03:00
  checked: PM2 daemon ancestry and persisted dump.pm2 lifecycle fields
  found: Live PM2 daemon PID 366 is owned by the WSL user systemd instance and currently has no application descendants in pstree. The 05:12 PM2 dump lists six autorestarting apps with 3-second restart_delay and max_restarts 50. Both Chrome bridges and fronts record zero restarts; bridge-cua-driver-w11 records 10 and mcp-cua-driver-w11_http records 5, with unstable_restarts 0. PM2 itself has been alive since 01:44.
  implication: Restart activity is specific to the CUA chain rather than global PM2 instability. The absence of current PM2 descendants conflicts with the persisted online status and makes live counters/log timing necessary; the dump alone cannot establish an ongoing loop.

- timestamp: 2026-08-22T05:30:00-03:00
  checked: live PM2 daemon lifecycle state using the confirmed existing daemon and absolute CLI path
  found: All six PM2 apps are currently stopped with pid 0. bridge-cua-driver-w11 increased from 10 restarts in the 05:12 dump to 60 live restarts, while its front increased from 5 to 6. Both CUA apps have 3-second restart_delay; bridge max_restarts is 50. Chrome apps remain at zero restarts.
  implication: A real CUA-specific restart loop occurred after the persisted dump and exhausted the restart policy. Each bridge iteration invokes Windows PowerShell, creating a credible recurrence mechanism for console/conhost accumulation. The failing child or script still needs identification; PM2 is an amplifier, not yet the root trigger.

- timestamp: 2026-08-22T05:31:00-03:00
  checked: existing PM2 CUA logs and app arguments
  found: bridge-cua-driver-w11 runs run-cua.ps1. Its error log records repeated run-cua.ps1 line 11 failures, "CUA daemon did not become ready", roughly every 29-33 seconds through 05:10. From 05:13:56 through 05:19:29, fresh pwsh launches fail during shell initialization with "Index was out of range... startIndex" roughly every 6-8 seconds. The CUA front repeatedly failed to reach 127.0.0.1:3193, later connected successfully at 05:12:16, and received shutdown signals at 05:14:04 and 05:26:06.
  implication: The bridge exits are directly observed and temporally match the PM2 restart growth. There are at least two failure modes: daemon readiness failure and PowerShell initialization failure. PM2 repeatedly launches Windows shells in response, which is a confirmed recurrence mechanism even before Windows parentage is inspected.

- timestamp: 2026-08-22T05:32:00-03:00
  checked: complete Windows run-cua.ps1 via noninteractive OpenSSH PowerShell
  found: Test-CuaReady requires exactly one cua-driver.exe serve process and exactly one named pipe. If false, run-cua.ps1 invokes Start-ScheduledTask cua-driver-serve, polls for only 20 seconds, throws on failure, then would start run-bridge-supervisor.ps1. The Windows host has been up about 5.8 days, while WSL/PM2 restarted about 3.8 hours ago.
  implication: PM2 and Task Scheduler are coupled restart layers. Any condition with zero or more than one serve process, a missing pipe, or readiness over 20 seconds causes PM2 to launch another pwsh and call the scheduled task again. Task instance policy and action visibility determine whether this produces visible or duplicate consoles.

- timestamp: 2026-08-22T05:33:00-03:00
  checked: current Windows process census with creation-time-aware parent validation and window handles
  found: 35 conhost.exe remain: 9 in session 0 and 26 in interactive session 2; none has a missing/reused parent and none exposes a main window. No WindowsTerminal.exe exists. One cua-driver.exe serve process has been stable since 04:42:31 in session 2, parented by pwsh.exe started at 04:42:29 through wscript.exe; pwsh uses NoLogo, NoProfile, NonInteractive, ExecutionPolicy Bypass, WindowStyle Hidden. Its conhost child is valid and windowless. One unrelated wsl.exe has an invalid/missing original parent. The audit's own transient SSH pwsh/conhost appeared in session 0 and was windowless.
  implication: There is no current conhost/PowerShell/Terminal orphan or visible-window incident. The risk is recurrence on restart, not an active accumulation. Because the daemon predates the 05:05-05:19 bridge failures, readiness was false despite a long-lived process, pointing to named-pipe availability or exact-count semantics rather than simple daemon absence.

- timestamp: 2026-08-22T05:34:00-03:00
  checked: cua-driver-serve scheduled task metadata and exported XML
  found: The task runs in the interactive muniz token at limited privilege, is Running, has Hidden=false, MultipleInstances=IgnoreNew, RestartOnFailure count 3 at 1-minute intervals, and no execution limit. Its action is wscript.exe //B //NoLogo invoking cua-driver-serve-unrestricted-hidden.vbs. LastRunTime is 05:10:20 and LastTaskResult is 2147946720 (0x800710E0), while the live task process actually began at 04:42:29.
  implication: Repeated Start-ScheduledTask calls do not create parallel task instances; they are rejected/ignored while the existing instance runs. The task may independently relaunch up to three times only if its process exits. The task XML is not marked hidden, but action-level VBS/pwsh flags likely control window visibility and must be verified.

- timestamp: 2026-08-22T05:35:00-03:00
  checked: complete VBS, daemon PowerShell launcher, bridge supervisor, and upstream PowerShell issue for the exact initialization exception
  found: VBS uses WScript.Shell.Run with window style 0 and waits synchronously. The daemon launcher uses a global mutex, exits if a bypass-enabled serve process exists, and otherwise runs cua-driver in the foreground. The bridge supervisor starts both mcp-proxy and ssh with WindowStyle Hidden, redirects stdio, and force-cleans both children in finally blocks. PowerShell issue #20706 reproduces the exact startIndex initialization exception when PowerShell 7.4+ processes a multi-path PSModulePath setting in powershell.config.json, failing inside ModuleIntrinsics.UpdatePath.
  implication: No CUA launcher calls Windows Terminal or intentionally opens a window; the visible WindowsTerminal likely came from another startup path. The exact startup error has a specific falsifiable PSModulePath/config cause that should be tested before broader speculation.

- timestamp: 2026-08-22T05:36:00-03:00
  checked: PowerShell 7 version/config scopes, registry module path, and selected PM2 daemon environment
  found: PowerShell is 7.6.3. The all-users powershell.config.json defines only RemoteSigned and a compatibility module deny-list; user config files are absent. User PSModulePath is unset, machine/process values are ordinary Windows module paths. PM2 daemon environment contains only its Linux PATH among selected relevant variables and no PSModulePath, WSLENV, WT_SESSION, TERM, or COMSPEC.
  implication: The known exact exception signature is relevant to PowerShell's ModuleIntrinsics startup path, but its documented persistent multi-path config trigger is not present. App-specific PM2 environment is the remaining cheap falsification check.

- timestamp: 2026-08-22T05:37:00-03:00
  checked: selected app-specific PM2 environment metadata
  found: CUA PM2 entries expose only PATH=/home/muniz/.local/bin:/usr/local/bin:/usr/bin:/bin among the selected relevant variables; no PSModulePath, WSLENV, terminal, or COMSPEC override is persisted.
  implication: The PowerShell startup failure remains a transient secondary failure with no persistent configuration trigger found. The visible WindowsTerminal should be traced independently, especially the live orphaned-parent wsl.exe command that starts hermes-serve.service.

- timestamp: 2026-08-22T05:38:00-03:00
  checked: matching Windows scheduled tasks and last-run metadata
  found: No task calls WindowsTerminal or wt.exe. CUA and Chrome CDP use wscript //B launchers. Active Hermes_WSL_All uses a VBS launcher and last ran at Windows boot on 2026-08-16. ADB tasks explicitly use WindowStyle Hidden. Codex_Startup explicitly uses WindowStyle Hidden. Atius-WSL-SSH-PortProxy is Interactive, runs powershell.exe without WindowStyle Hidden, and last ran on 2026-08-19, not during the current incident. Several older Hermes tasks are disabled.
  implication: No scheduled task timestamp matches the 04:34 wsl.exe bootstrap. The port-proxy task is a precise recurrence risk at future logon but not the source of today's visible terminal. Startup/Run remains the leading source candidate.

- timestamp: 2026-08-22T05:39:00-03:00
  checked: HKCU/HKLM Run and RunOnce plus user/common Startup directories
  found: No Run value invokes WSL, Hermes, PowerShell without hidden flags, or Windows Terminal. The only PowerShell Run value explicitly uses WindowStyle Hidden for an Obsidian tunnel. User Startup contains no launcher; common Startup contains only AnyDesk.
  implication: The visible-terminal path is not a conventional Startup/Run entry. Provenance must come from a helper script, manual/agent launch, or event history.

- timestamp: 2026-08-22T05:40:00-03:00
  checked: initial broad exact-string search across Hermes launcher areas
  found: Search output was dominated and truncated by Hermes cache/result artifacts despite intended exclusions, so it cannot prove script absence. Existing canonical documentation embedded in that output identifies start-hermes-desktop-wsl.ps1 as a wrapper that starts hermes-serve.service through wsl.exe.
  implication: Repeat with strict extension and cache scope; treat this as a candidate, not confirmed provenance.

- timestamp: 2026-08-22T05:40:39-03:00
  checked: extension-limited launcher search and relevant start-hermes-desktop-wsl.ps1 lines
  found: Two VBS helpers use WScript.Shell.Run with window style 0. The desktop wrapper invokes wsl.exe with Start-Process -WindowStyle Hidden to run exactly systemctl --user start hermes-serve.service, does not retain the Process object, and does not wait for that WSL child.
  implication: The 04:34 wsl.exe command and orphaned parent are explained by intentional detached startup from the desktop wrapper. It is a process-lifetime leak risk, but explicit hidden launch makes it unlikely to be the observed WindowsTerminal window.

- timestamp: 2026-08-22T05:43:00-03:00
  checked: Security 4688 and Sysmon process-create history around 04:34:57
  found: Security contains no matching 4688 events in the interval, and the Sysmon Operational log does not exist.
  implication: The historical parent executable for PID 32400 cannot be recovered from current event telemetry. The visible WindowsTerminal's exact launcher remains unproven after closure; future recurrence needs process-create observability captured before cleanup.

- timestamp: 2026-08-22T05:43:00-03:00
  checked: full existing bridge-cua-driver-w11 error log pattern counts and temporal bounds
  found: There are 10 readiness failures from 05:05:41 through 05:10:15, each represented twice in the formatted stack (20 matching lines), followed by exactly 50 pwsh initialization/startIndex failures from 05:13:56 through 05:19:29. All are on 2026-08-22.
  implication: The PM2 counter of 60 is fully explained by one bounded current-day episode. It cannot by itself account for 466 removed orphan conhosts, so at least one additional historical launcher/leak source existed.

- timestamp: 2026-08-22T05:44:00-03:00
  checked: current conhost parent distribution and node_repl ancestry
  found: Conhost count increased from 35 at 05:32:46 to 51 at 05:44:03, with zero invalid parents. Largest groups are ssh.exe 9, node_repl.exe 6, mcp-proxy.exe 5, wslhost.exe 5, node.exe 5, cmd.exe 4, and winpty-agent.exe 3; only one belongs to pwsh.exe/CUA and one directly to cua-driver.exe. Six node_repl processes are Codex cua_node runtime children of wsl.exe, each with one valid conhost.
  implication: node_repl is not large enough to explain 466 historical orphans by itself. Active concurrent bridge/tool launches are creating valid console hosts across several executables. If cleanup fails when these parents exit, accumulation is cross-launcher; CUA's 60-restart burst is one amplifier among several.

- timestamp: 2026-08-22T05:45:00-03:00
  checked: second live PM2 state snapshot after concurrent process growth
  found: The prior six-app PM2 list was replaced around 05:32 with ten newly named apps covering four Chrome profiles plus CUA and five HTTP fronts. Every app is now stopped with pid 0 and restart_time 0. Nevertheless Windows still contains mcp-proxy/node/ssh processes and conhosts created at 05:32-05:35.
  implication: Another actor reconfigured/started/stopped the bridge set during this audit. Persisting Windows descendants after all PM2 owners are stopped is direct evidence of a cross-boundary lifecycle mismatch and a stronger recurrence mechanism than CUA retries alone.

- timestamp: 2026-08-22T05:46:00-03:00
  checked: detailed ancestry for Windows bridge descendants created during the 05:32 PM2 lifecycle
  found: Five mcp-proxy.exe roots (ports 13091-13095) and five ssh.exe reverse tunnels (WSL ports 3191-3195) remain alive and windowless. Every one has a missing parent PID corresponding to an exited supervisor pwsh process. Their conhost children have valid parents only because the orphaned mcp-proxy/ssh roots remain alive. Downstream node.exe and cua-driver.exe mcp children remain under Python proxy processes. All ten PM2 apps are stopped.
  implication: The supervisor finally blocks did not execute or could not clean children when PM2/WSL interop ownership ended. Repeating start/stop or crash cycles necessarily accumulates orphaned Windows proxy/tunnel trees and their conhosts. This directly explains recurrence risk and plausibly the prior hundreds of orphan consoles.

- timestamp: 2026-08-22T05:48:00-03:00
  checked: replacement PM2 retry policy, current CUA readiness, and Windows default terminal delegation
  found: All ten replacement PM2 apps retain autorestart=true, fixed restart_delay=3000, max_restarts=100, min_uptime=10000, no exponential backoff, and no stop_exit_codes. CUA is currently healthy with one serve process and one named pipe. Registry DelegationConsole/DelegationTerminal GUIDs select Windows Terminal as the default terminal application.
  implication: Recurrence risk is high when any bridge set is restarted: each app can retry up to 100 times and abrupt stop can leak detached Windows trees. Any interactive console launcher that lacks explicit hiding can surface as Windows Terminal; Atius-WSL-SSH-PortProxy is the identified scheduled-task risk, although it did not run during today's incident.

## Resolution

root_cause: PM2 supervises the WSL-interoperability pwsh wrapper, but run-bridge-supervisor.ps1 creates mcp-proxy.exe and ssh.exe as detached Windows Start-Process children without Job Object kill-on-close ownership. When PM2 apps are stopped/replaced or wrappers fail, supervisor finally cleanup is bypassed/ineffective; Windows proxy/tunnel/downstream trees and conhosts survive. CUA added a confirmed 60-launch retry burst due 10 readiness failures plus 50 transient pwsh initialization failures. The observed WindowsTerminal window was a separate unhidden-console exposure; exact provenance was lost after closure, but Windows Terminal is default and Atius-WSL-SSH-PortProxy lacks WindowStyle Hidden.
fix: No fix applied by read-only instruction. Recommended direction is Windows Job Object ownership with JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE or moving bridge ownership fully to Windows; graceful stop IPC plus stale-instance fail-closed checks; decouple CUA task start from bridge retries; reduce PM2 max_restarts and add backoff/stop exit codes; hide the port-proxy task; make Hermes WSL bootstrap lifecycle synchronous or explicitly supervised; enable retained process-create telemetry.
verification: Read-only evidence shows all ten PM2 apps stopped while five mcp-proxy.exe and five ssh.exe roots created by that lifecycle remain alive with missing supervisor parents, each retaining conhost/downstream children. CUA log counts exactly match 60 PM2 restarts. Current snapshot has no visible WindowsTerminal and current conhosts expose no windows, but retry policy remains unsafe.
files_changed: []
