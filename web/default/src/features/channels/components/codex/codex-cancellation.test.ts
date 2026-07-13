/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'

import {
  CodexDeviceAuthorizationDetails,
  type CodexDevicePollingScheduler,
  startCodexDevicePolling,
} from './codex-regenerate-dialog'

import {
  coalesceCodexCancellation,
  confirmCodexDeviceAuthorizationCancellation,
  restoreCodexCancellationUiAfterFailure,
  runAfterCodexRegenerationCancellation,
} from '../drawers/channel-mutate-drawer'

describe('Codex device authorization cancellation confirmation', () => {
  test('coalesces concurrent cancellation attempts into one server request', async () => {
    let resolveCancellation: ((value: boolean) => void) | undefined
    let serverCalls = 0
    const startCancellation = () => {
      serverCalls += 1
      return new Promise<boolean>((resolve) => {
        resolveCancellation = resolve
      })
    }

    const first = coalesceCodexCancellation(null, startCancellation)
    const second = coalesceCodexCancellation(first, startCancellation)

    assert.equal(first, second)
    assert.equal(serverCalls, 1)
    resolveCancellation?.(false)
    assert.equal(await second, false)
  })

  test('applies a user close intent after an expiry cancellation is coalesced', async () => {
    let resolveCancellation: ((value: boolean) => void) | undefined
    let serverCalls = 0
    let dialogOpen = true
    const startCancellation = () => {
      serverCalls += 1
      return new Promise<boolean>((resolve) => {
        resolveCancellation = resolve
      })
    }

    const expiryCancellation = coalesceCodexCancellation(
      null,
      startCancellation
    )
    const userCancellation = runAfterCodexRegenerationCancellation(
      () => coalesceCodexCancellation(expiryCancellation, startCancellation),
      () => {
        dialogOpen = false
      }
    )

    assert.equal(serverCalls, 1)
    assert.equal(dialogOpen, true)
    resolveCancellation?.(true)
    assert.equal(await userCancellation, true)
    assert.equal(dialogOpen, false)
  })

  test('does not resolve until the server confirms cancellation', async () => {
    let confirmCancel: (() => void) | undefined
    let markCancelStarted: (() => void) | undefined
    const cancelStarted = new Promise<void>((resolve) => {
      markCancelStarted = resolve
    })
    let settled = false
    const cancellation = confirmCodexDeviceAuthorizationCancellation({
      previousCancelRequest: null,
      startRequest: Promise.resolve(),
      activeChannelId: 5,
      cancel: async () => {
        await new Promise<void>((resolve) => {
          confirmCancel = resolve
          markCancelStarted?.()
        })
        return { success: true }
      },
      fallbackMessage: 'cancel failed',
    }).then(() => {
      settled = true
    })

    await cancelStarted
    assert.equal(settled, false)
    confirmCancel?.()
    await cancellation
    assert.equal(settled, true)
  })

  test('rejects a failed server cancellation', async () => {
    await assert.rejects(
      confirmCodexDeviceAuthorizationCancellation({
        previousCancelRequest: null,
        startRequest: Promise.resolve(),
        activeChannelId: 5,
        cancel: async () => ({ success: false, message: 'store unavailable' }),
        fallbackMessage: 'cancel failed',
      }),
      /store unavailable/
    )
  })

  test('keeps close and reset blocked when cancellation fails', async () => {
    let closed = false
    let reset = false

    const confirmed = await runAfterCodexRegenerationCancellation(
      async () => false,
      () => {
        closed = true
        reset = true
      }
    )

    assert.equal(confirmed, false)
    assert.equal(closed, false)
    assert.equal(reset, false)
  })

  test('restores the real device view and polling after cancellation fails', async () => {
    const authorization = {
      flow: 'device_code' as const,
      user_code: 'ABCD-EFGH',
      verification_url: 'https://auth.openai.com/codex/device',
      interval_seconds: 1,
      expires_at: new Date(10_000).toISOString(),
    }
    let cancellationError: unknown
    try {
      await confirmCodexDeviceAuthorizationCancellation({
        previousCancelRequest: null,
        startRequest: Promise.resolve(),
        activeChannelId: 5,
        cancel: async () => ({ success: false, message: 'store unavailable' }),
        fallbackMessage: 'cancel failed',
      })
    } catch (error) {
      cancellationError = error
    }
    assert.match(String(cancellationError), /store unavailable/)

    const restored = restoreCodexCancellationUiAfterFailure({
      blocked: true,
      regenerating: true,
      dialogOpen: true,
      deviceAuthorization: authorization,
    })

    let polling = false
    let pollCount = 0
    let timerId = 0
    const timers = new Map<
      number,
      { delay: number; callback: () => void | Promise<void> }
    >()
    const scheduler: CodexDevicePollingScheduler = {
      now: () => 0,
      setTimeout: (callback, delay) => {
        timerId += 1
        timers.set(timerId, { delay, callback })
        return timerId
      },
      clearTimeout: (id) => {
        timers.delete(id)
      },
    }
    const stopPolling = startCodexDevicePolling({
      expiresAt: restored.deviceAuthorization.expires_at,
      intervalMs: restored.deviceAuthorization.interval_seconds * 1000,
      poll: async () => {
        pollCount += 1
        return { status: 'pending' }
      },
      onCompleted: () => assert.fail('recovered polling must remain pending'),
      onExpired: () => assert.fail('recovered polling must not expire'),
      onError: (error) => assert.fail(String(error)),
      onPollingChange: (value) => {
        polling = value
      },
      scheduler,
    })
    const nextPoll = [...timers.values()].sort(
      (left, right) => left.delay - right.delay
    )[0]
    await nextPoll.callback()

    const markup = renderToStaticMarkup(
      createElement(CodexDeviceAuthorizationDetails, {
        deviceAuthorization: restored.deviceAuthorization,
        deviceFlowError: '',
        isPolling: polling,
        onRestartDevice: () => undefined,
        translate: (key) => key,
      })
    )

    assert.equal(restored.blocked, false)
    assert.equal(restored.regenerating, false)
    assert.equal(restored.dialogOpen, true)
    assert.equal(pollCount, 1)
    assert.match(markup, /ABCD-EFGH/)
    assert.match(markup, /https:\/\/auth\.openai\.com\/codex\/device/)
    assert.match(markup, /Waiting for OpenAI authorization/)
    stopPolling()
  })

  test('awaits cancellation confirmation before close and reset', async () => {
    let confirmCancellation: ((confirmed: boolean) => void) | undefined
    let closed = false
    let reset = false
    const cancellation = new Promise<boolean>((resolve) => {
      confirmCancellation = resolve
    })

    const completion = runAfterCodexRegenerationCancellation(
      () => cancellation,
      () => {
        closed = true
        reset = true
      }
    )

    await Promise.resolve()
    assert.equal(closed, false)
    assert.equal(reset, false)
    confirmCancellation?.(true)
    assert.equal(await completion, true)
    assert.equal(closed, true)
    assert.equal(reset, true)
  })
})
