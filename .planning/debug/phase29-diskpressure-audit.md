---
status: diagnosed
trigger: "Read-only live infrastructure audit for planning Phase 29. On atius-srv-1, determine why kubelet DiskPressure remains True with 28GB free, exact eviction thresholds, safe reclaim candidates, current k3s/containerd storage usage, and what minimum condition makes scheduler remove taint. Also verify whether Apache on host can safely target ClusterIP, NodePort, or hostPort for a pod pinned to srv1. Do not delete, label, apply, restart or edit anything. Return commands/evidence and safest recommendation."
created: 2026-07-12T23:15:00-03:00
updated: 2026-07-12T23:45:00-03:00
---

## Current Focus

hypothesis: confirmed: DiskPressure persists because evictionMinimumReclaim adds 10 percentage points to the 10% hard threshold, requiring 20% free on the shared nodefs/imagefs before pressure clears
test: compare effective configz thresholds and transition period with df bytes and eviction/image-GC logs; probe ClusterIP from host and inspect kube-proxy/Apache state
expecting: current free space below 20% matches kubelet reclaim target, and host reaches ClusterIP through kube-proxy
next_action: return read-only diagnosis and safest Phase 29 recommendation

## Symptoms

expected: kubelet reports no DiskPressure when storage has adequate headroom; scheduler can place a pod pinned to srv1; host Apache has a stable safe backend path
actual: kubelet DiskPressure reportedly remains True despite approximately 28GB free, retaining a scheduler taint
errors: node.kubernetes.io/disk-pressure:NoSchedule
reproduction: inspect atius-srv-1 node conditions, taints, kubelet eviction state, filesystem usage and candidate Apache-to-pod network paths
started: present during Phase 29 planning audit; exact onset unknown

## Eliminated

## Evidence

- timestamp: 2026-07-12T23:15:00-03:00
  checked: Graphify status and query
  found: graph is current at commit 20ae5c7; task-specific query returned no nodes
  implication: focused planning/runtime reads and live SSH evidence are required

- timestamp: 2026-07-12T23:15:00-03:00
  checked: Codex memory, GBrain and Obsidian context
  found: historical context confirms DiskPressure on atius-srv-1 as a migration blocker and warns that pinned admin/observability workloads amplify pressure
  implication: treat historical explanation only as a hypothesis and verify current live state

- timestamp: 2026-07-12T23:39:05-03:00
  checked: live node condition and filesystem
  found: DiskPressure=True and node.kubernetes.io/disk-pressure:NoSchedule since 2026-07-12T05:30:04Z; /dev/sda1 is 207907635200 bytes with 28275568640 bytes free (87% used); root inodes are only 13% used
  implication: byte availability, not inode exhaustion, drives pressure

- timestamp: 2026-07-12T23:39:05-03:00
  checked: effective kubelet configz
  found: evictionHard nodefs.available<10% and imagefs.available<10%; evictionMinimumReclaim is 10% for both; evictionPressureTransitionPeriod is 5m; image GC high/low thresholds are 85%/80%
  implication: recovery requires at least 20% free for the transition period, not merely crossing back above 10%

- timestamp: 2026-07-12T23:40:00-03:00
  checked: k3s journal
  found: eviction manager retries ephemeral-storage reclaim every 10s; image GC reports 87% used, target 80%, amountToFree about 13.3GB, freed 0 bytes, and no active pods left to evict
  implication: the computed shortfall to 20% is confirmed directly by kubelet behavior

- timestamp: 2026-07-12T23:41:00-03:00
  checked: k3s/containerd and host storage
  found: /var/lib/rancher/k3s totals 4.7G, including 3.5G local-path storage and about 1.0G server data; containerd has no listed images or containers after eviction; notable external consumers include /var/tmp 8.5G, /srv/Shared 9.1G, /var/backups 1.4G, /var/log 2.1G, /home/ubuntu/.cache 4.4G, /root/.cache 1.6G, /root/.npm 1.2G, /home/ubuntu/.npm 893M, /home/ubuntu/.bun 618M, and 35 etcd snapshots totaling 580M
  implication: reclaim must focus primarily outside k3s; candidates require owner/retention validation before deletion

- timestamp: 2026-07-12T23:43:00-03:00
  checked: host-to-cluster networking
  found: kube-proxy mode is iptables with localhostNodePorts=true and unrestricted nodePortAddresses; host TCP probes to 10.43.0.1:443 and 10.43.28.32:80 succeeded, and HTTP to the latter returned 200 in 2ms; no NodePort services currently exist
  implication: ClusterIP is directly usable by host Apache and avoids adding externally exposed node ports

- timestamp: 2026-07-12T23:44:00-03:00
  checked: Apache and hostPort state
  found: Apache currently proxies mainly to loopback or node addresses; no ClusterIP target exists; one config uses loopback port 31810; hostNetwork/hostPort is used by node exporter on srv1 port 9100
  implication: hostPort works in principle but creates host port collision/lifecycle coupling and is less safe than ClusterIP for Apache

## Resolution

root_cause: kubelet uses a shared root filesystem for nodefs/imagefs and is configured with a 10% hard threshold plus 10% minimum reclaim; with only 13.6% free it correctly remains under pressure until at least 20% is free and stable for 5 minutes
fix: read-only audit; no fix authorized
verification: effective configz, df byte math, kubelet journal, storage inventory, kube-proxy config and live ClusterIP probes agree
files_changed: []
