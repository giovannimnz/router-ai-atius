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

import {
  type CodexDevicePollingScheduler,
  getTrustedCodexAuthorizationUrl,
  startCodexDevicePolling,
} from './codex-regenerate-dialog'

class FakePollingScheduler implements CodexDevicePollingScheduler {
  private nowMs = 0
  private nextTimerId = 1
  private readonly timers = new Map<
    number,
    { at: number; callback: () => void | Promise<void> }
  >()

  now = () => this.nowMs

  setTimeout = (callback: () => void | Promise<void>, delayMs: number) => {
    const timerId = this.nextTimerId++
    this.timers.set(timerId, {
      at: this.nowMs + delayMs,
      callback,
    })
    return timerId
  }

  clearTimeout = (timerId: number) => {
    this.timers.delete(timerId)
  }

  async advanceBy(delayMs: number) {
    const targetTime = this.nowMs + delayMs
    while (true) {
      const nextTimer = [...this.timers.entries()]
        .filter(([, timer]) => timer.at <= targetTime)
        .sort(
          ([leftId, left], [rightId, right]) =>
            left.at - right.at || leftId - rightId
        )[0]
      if (!nextTimer) break

      const [timerId, timer] = nextTimer
      this.nowMs = timer.at
      this.timers.delete(timerId)
      await timer.callback()
    }
    this.nowMs = targetTime
  }
}

describe('Codex device authorization polling lifecycle', () => {
  test('stops exactly at expires_at and does not schedule more polls', async () => {
    const scheduler = new FakePollingScheduler()
    const pollingStates: boolean[] = []
    let pollCount = 0
    let expiredCount = 0

    startCodexDevicePolling({
      expiresAt: new Date(5_000).toISOString(),
      intervalMs: 2_000,
      poll: async () => {
        pollCount += 1
        return { status: 'pending' }
      },
      onCompleted: () => assert.fail('polling must not complete'),
      onExpired: () => {
        expiredCount += 1
      },
      onError: (error) => assert.fail(String(error)),
      onPollingChange: (polling) => pollingStates.push(polling),
      scheduler,
    })

    await scheduler.advanceBy(4_999)
    assert.equal(pollCount, 2)
    assert.equal(expiredCount, 0)

    await scheduler.advanceBy(1)
    await scheduler.advanceBy(20_000)
    assert.equal(pollCount, 2)
    assert.equal(expiredCount, 1)
    assert.deepEqual(pollingStates, [true, false])
  })

  test('stops only when the poll result is explicitly terminal', async () => {
    const scheduler = new FakePollingScheduler()
    const terminalError = new Error('device authorization expired')
    const errors: unknown[] = []
    let pollCount = 0

    startCodexDevicePolling({
      expiresAt: new Date(10_000).toISOString(),
      intervalMs: 1_000,
      poll: async () => {
        pollCount += 1
        return { status: 'terminal', error: terminalError }
      },
      onCompleted: () => assert.fail('polling must not complete'),
      onExpired: () => assert.fail('terminal error must clear expiry timer'),
      onError: (error) => errors.push(error),
      onPollingChange: () => undefined,
      scheduler,
    })

    await scheduler.advanceBy(1_000)
    await scheduler.advanceBy(20_000)
    assert.equal(pollCount, 1)
    assert.deepEqual(errors, [terminalError])
  })

  test('cancels pending timers during dialog or drawer cleanup', async () => {
    const scheduler = new FakePollingScheduler()
    let pollCount = 0
    let expiredCount = 0

    const cancel = startCodexDevicePolling({
      expiresAt: new Date(10_000).toISOString(),
      intervalMs: 1_000,
      poll: async () => {
        pollCount += 1
        return { status: 'pending' }
      },
      onCompleted: () => assert.fail('cancelled polling must not complete'),
      onExpired: () => {
        expiredCount += 1
      },
      onError: (error) => assert.fail(String(error)),
      onPollingChange: () => undefined,
      scheduler,
    })

    cancel()
    await scheduler.advanceBy(20_000)
    assert.equal(pollCount, 0)
    assert.equal(expiredCount, 0)
  })

  test('aborts an in-flight poll and ignores its late response', async () => {
    const scheduler = new FakePollingScheduler()
    let pollSignal: AbortSignal | undefined
    let resolvePoll: ((status: { status: 'completed' }) => void) | undefined
    let completedCount = 0
    const pollResult = new Promise<{ status: 'completed' }>((resolve) => {
      resolvePoll = resolve
    })

    const cancel = startCodexDevicePolling({
      expiresAt: new Date(10_000).toISOString(),
      intervalMs: 1_000,
      poll: (signal) => {
        pollSignal = signal
        return pollResult
      },
      onCompleted: () => {
        completedCount += 1
      },
      onExpired: () => assert.fail('cancelled polling must not expire'),
      onError: (error) => assert.fail(String(error)),
      onPollingChange: () => undefined,
      scheduler,
    })

    const advancePromise = scheduler.advanceBy(1_000)
    await Promise.resolve()
    assert.equal(pollSignal?.aborted, false)

    cancel()
    assert.equal(pollSignal?.aborted, true)
    resolvePoll?.({ status: 'completed' })
    await advancePromise
    await scheduler.advanceBy(20_000)
    assert.equal(completedCount, 0)
  })

  test('retries transient polling failures without cancelling the flow', async () => {
    const scheduler = new FakePollingScheduler()
    let pollCount = 0
    let completedCount = 0
    let errorCount = 0

    startCodexDevicePolling({
      expiresAt: new Date(10_000).toISOString(),
      intervalMs: 1_000,
      poll: async () => {
        pollCount += 1
        if (pollCount === 1) throw new Error('temporary store failure')
        if (pollCount === 2) {
          return { status: 'pending', retryAfterMs: 2_000 }
        }
        return { status: 'completed' }
      },
      onCompleted: () => {
        completedCount += 1
      },
      onExpired: () => assert.fail('retryable polling must not expire'),
      onError: () => {
        errorCount += 1
      },
      onPollingChange: () => undefined,
      scheduler,
    })

    await scheduler.advanceBy(3_999)
    assert.equal(pollCount, 2)
    assert.equal(completedCount, 0)
    await scheduler.advanceBy(1)
    assert.equal(pollCount, 3)
    assert.equal(completedCount, 1)
    assert.equal(errorCount, 0)
  })
})

describe('Codex authorization URL validation', () => {
  test('allows only the exact OpenAI origin and expected paths', () => {
    assert.equal(
      getTrustedCodexAuthorizationUrl(
        'https://auth.openai.com/codex/device?user_code=ABCD'
      ),
      'https://auth.openai.com/codex/device?user_code=ABCD'
    )
  })

  test('rejects hostile HTTPS origins and non-allowlisted paths', () => {
    const hostileCandidates = [
      'https://auth.openai.com.evil.example/codex/device',
      'https://auth.openai.com@evil.example/codex/device',
      'https://evil.example/oauth/authorize',
      'https://auth.openai.com/api/accounts/deviceauth/usercode',
      'javascript:alert(1)',
      'not a URL',
    ]

    for (const candidate of hostileCandidates) {
      assert.equal(getTrustedCodexAuthorizationUrl(candidate), null)
    }
  })
})
