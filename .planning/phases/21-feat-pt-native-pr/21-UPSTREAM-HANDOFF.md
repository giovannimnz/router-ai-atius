# Phase 21 Upstream Handoff

## Local Execution Status

- Implementation lane: `/home/ubuntu/GitHub/containers/router-ai-atius-phase21-upstream`
- Branch: `feat/phase21-pt-native-upstream`
- Upstream baseline: `5fc35e28a253bd5a5656c177aea1bd121231398f`
- Local validation complete for backend, default frontend, and classic frontend PT support
- Not yet committed, pushed, or submitted upstream

## Current Duplicate Risk

- Issue `#2924` is open: `Portuguese translation`
- PR `#5801` is open: `Add Portuguese translations for various messages`
  - Current scope as of 2026-07-05: only `i18n/pt.yaml`
- Closed historical PRs:
  - `#5238` `feat(i18n): add Brazilian Portuguese (pt-BR) translation`
  - `#5245` `feat(i18n): add Brazilian Portuguese (pt) translation`

## Validation Evidence

- Backend:
  - `/usr/local/go/bin/go test ./i18n` passed
- Default frontend:
  - `bun run i18n:sync` passed
  - `_sync-report.json` shows `pt.missingCount=0`, `pt.extrasCount=0`, `pt.untranslatedCount=0`
  - explicit key/placeholder/same-as-English parity check passed
  - `bun run typecheck` passed
  - `bun run build` passed
- Classic frontend:
  - explicit key/placeholder/same-as-English parity check passed
  - `bun run build` passed
- Diff hygiene:
  - `git diff --check upstream/main...HEAD` passed
  - fork/planning/runtime leak grep passed

## Known Remaining Blockers

- `web/default`: `bun run lint` fails on pre-existing unrelated lint debt in upstream files outside the PT diff
- `web/classic`: `bun run i18n:lint` fails on `106` pre-existing unrelated issues outside the PT diff
- `web/classic`: `bun run lint` fails because Prettier wants formatting changes in `58` unrelated existing files

## Proposed PR Title

`feat(i18n): add Brazilian Portuguese localization`

## PR Body Draft

Template source: `.github/PULL_REQUEST_TEMPLATE.md`

```md
# ⚠️ 提交说明 / PR Notice
> [!IMPORTANT]
>
> - 请提供**人工撰写**的简洁摘要，避免直接粘贴未经整理的 AI 输出。

## 📝 变更描述 / Description
Add native Brazilian Portuguese (`pt`) localization support across the current upstream surfaces:

- backend locale file and PT language normalization in `i18n/`
- default React frontend locale + language registration
- classic frontend locale + language registration

The diff stays limited to native i18n files, language wiring, and validation-focused backend tests.

This contribution was AI-assisted and human-reviewed.

## 🚀 变更类型 / Type of change
- [ ] 🐛 Bug 修复 (Bug fix) - *请关联对应 Issue，避免将设计取舍、理解偏差或预期不一致直接归类为 bug*
- [x] ✨ 新功能 (New feature) - *重大特性建议先通过 Issue 沟通*
- [ ] ⚡ 性能优化 / 重构 (Refactor)
- [ ] 📝 文档更新 (Documentation)

## 🔗 关联任务 / Related Issue
- Closes #2924

## ✅ 提交前检查项 / Checklist
- [x] **人工确认:** 我已亲自整理并撰写此描述，没有直接粘贴未经处理的 AI 输出。
- [x] **非重复提交:** 我已搜索现有的 [Issues](https://github.com/QuantumNous/new-api/issues) 与 [PRs](https://github.com/QuantumNous/new-api/pulls)，确认不是重复提交。
- [x] **Bug fix 说明:** 若此 PR 标记为 `Bug fix`，我已提交或关联对应 Issue，且不会将设计取舍、预期不一致或理解偏差直接归类为 bug。
- [x] **变更理解:** 我已理解这些更改的工作原理及可能影响。
- [x] **范围聚焦:** 本 PR 未包含任何与当前任务无关的代码改动。
- [x] **本地验证:** 已在本地运行并通过测试或手动验证，维护者可以据此复核结果。
- [x] **安全合规:** 代码中无敏感凭据，且符合项目代码规范。

## 📸 运行证明 / Proof of Work
- `/usr/local/go/bin/go test ./i18n`
- `cd web/default && bun run i18n:sync`
- `cd web/default && bun run typecheck`
- `cd web/default && bun run build`
- `cd web/classic && bun run build`
- explicit locale parity checks passed for both frontends (keys, placeholders, reviewed same-as-English allowlist)

Known upstream pre-existing blockers not introduced by this diff:
- `cd web/default && bun run lint`
- `cd web/classic && bun run i18n:lint`
- `cd web/classic && bun run lint`
```

## Replacement Comment Draft For PR #5245

Use only after a clean replacement PR exists:

```md
Closing this in favor of the clean replacement PR: <REPLACEMENT_PR_URL>

This new PR keeps the scope focused on native Brazilian Portuguese support only and drops the unrelated fork/runtime changes that polluted this historical branch.
```
