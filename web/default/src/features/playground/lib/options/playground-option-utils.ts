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
import type { GroupOption, ModelOption } from '../../types'

function supportsParameter(model: ModelOption | null, parameter: string): boolean {
  if (!model?.supportedParameters || model.supportedParameters.length === 0) {
    return true
  }

  return model.supportedParameters.some(
    (value) => value.toLowerCase() === parameter.toLowerCase()
  )
}

export function getModelOption(
  models: ModelOption[],
  currentModel: string
): ModelOption | null {
  return models.find((model) => model.value === currentModel) ?? null
}

export function getEffectiveParameterEnabled(
  parameterEnabled: Record<
    | 'temperature'
    | 'top_p'
    | 'max_tokens'
    | 'frequency_penalty'
    | 'presence_penalty'
    | 'seed',
    boolean
  >,
  model: ModelOption | null
) {
  return {
    temperature:
      parameterEnabled.temperature && supportsParameter(model, 'temperature'),
    top_p: parameterEnabled.top_p && supportsParameter(model, 'top_p'),
    max_tokens:
      parameterEnabled.max_tokens &&
      (supportsParameter(model, 'max_tokens') ||
        supportsParameter(model, 'max_completion_tokens')),
    frequency_penalty:
      parameterEnabled.frequency_penalty &&
      supportsParameter(model, 'frequency_penalty'),
    presence_penalty:
      parameterEnabled.presence_penalty &&
      supportsParameter(model, 'presence_penalty'),
    seed: parameterEnabled.seed && supportsParameter(model, 'seed'),
  }
}

export function getModelFallback(
  models: ModelOption[],
  currentModel: string
): string | null {
  const hasCurrentModel = models.some((model) => model.value === currentModel)

  if (hasCurrentModel || models.length === 0) {
    return null
  }

  return models[0].value
}

export function shouldClearModelForGroup(
  models: ModelOption[],
  currentModel: string
): boolean {
  if (currentModel === '') {
    return false
  }

  return !models.some((model) => model.value === currentModel)
}

export function getGroupFallback(
  groups: GroupOption[],
  currentGroup: string
): string | null {
  const hasCurrentGroup = groups.some((group) => group.value === currentGroup)

  if (hasCurrentGroup || groups.length === 0) {
    return null
  }

  return (
    groups.find((group) => group.value === 'default')?.value ?? groups[0].value
  )
}

export function getOptionLoadErrorMessage(
  error: unknown,
  fallbackMessage: string
): string {
  return error instanceof Error ? error.message : fallbackMessage
}
