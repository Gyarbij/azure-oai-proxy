# Changelog

All notable changes to the Azure OpenAI Proxy project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Native Anthropic Messages API Integration** (`pkg/anthropic/` module)
  - Complete Anthropic Messages API type definitions ([types.go](pkg/anthropic/types.go))
  - Bidirectional OpenAI ↔ Anthropic format conversion ([converter.go](pkg/anthropic/converter.go))
  - Native Anthropic SSE (Server-Sent Events) streaming support
  - Automatic system message extraction and handling
  - Proper header management (`x-api-key`, `anthropic-version`)
- ARCHITECTURE.md documentation with complete technical overview
- Support for configurable Anthropic API version via `ANTHROPIC_APIVERSION` environment variable (default: `2023-06-01`)

### Changed
- **Breaking**: Claude models now route to `/anthropic/v1/messages` instead of `/openai/deployments/...`
- Claude model handling refactored to use dedicated Anthropic module
- Updated [README.md](README.md) with Claude-specific documentation and examples
- Improved streaming reliability for Claude models in OpenWebUI
- Azure proxy now uses `anthropic.StreamConverter` for Claude SSE stream processing

### Fixed
- **Critical**: Resolved streaming cutoff issue with Claude models in OpenWebUI
  - Previous implementation incorrectly treated Anthropic API as OpenAI format
  - Streaming now properly handles Anthropic SSE events (`message_start`, `content_block_delta`, `message_stop`)
- Fixed Content-Type header detection to handle charset parameters
- Added `FlushInterval: -1` to reverse proxy for immediate SSE flushing

### Technical Details
The Anthropic integration follows the official [Anthropic Messages API specification](https://docs.anthropic.com/en/api/messages):

**Request Flow:**
1. Client sends OpenAI chat completion request
2. Proxy detects Claude model via `anthropic.IsClaudeModel()`
3. `anthropic.ConvertOpenAIToAnthropic()` converts request format
4. Routes to `https://[endpoint]/anthropic/v1/messages`
5. Response converted via `anthropic.StreamConverter` (streaming) or `anthropic.ReadNonStreamingResponse()` (non-streaming)
6. Client receives OpenAI-formatted response

**SSE Event Mapping:**
- Anthropic `message_start` → OpenAI initial chunk with role
- Anthropic `content_block_delta` → OpenAI delta chunks with content
- Anthropic `message_stop` → OpenAI final chunk with `[DONE]`

## [1.0.8] - 2025-08-03

### Added
- Comprehensive support for Azure OpenAI Responses API
- Automatic reasoning model detection (O1, O3, O4 series)
- Streaming conversion for Responses API

## [1.0.7] - 2025-01-24

### Added
- Support for Azure OpenAI API version 2024-12-01-preview
- New model fetching mechanism

## [1.0.6] - 2024-12-14

### Added
- Comprehensive Azure OpenAI in Microsoft Foundry support
  - GPT-5.2 series (preview)
  - GPT-5.1 series
  - GPT-5 series
  - GPT-4.1 series
  - Complete O-series reasoning models
  - Codex models
  - Audio/realtime models
  - Image generation (gpt-image-1)
  - Video generation (sora, sora-2)
  - Open-weight models
- Updated API versions to 2024-08-01-preview

## [1.0.5] - 2024-07-25

### Added
- Azure AI Studio deployments support
- Support for Meta Llama 3.1, Mistral-2407, and Cohere AI models

## [1.0.4] - 2024-07-18

### Added
- Support for gpt-4o-mini

## [1.0.3] - 2024-06-23

### Added
- Dynamic model fetching for `/v1/models` endpoint
- Unified token handling mechanism
- Audio-related endpoints support
- Model capabilities endpoint
- Improved CORS handling

### Changed
- Flexible environment variable handling

## [1.0.2] - 2024-06-22

### Added
- Image generation endpoint support
- Fine-tuning operations support
- File management support
- Better error handling and logging

### Changed
- Improved rate limiting handling
- Updated model mappings for latest models

[Unreleased]: https://github.com/gyarbij/azure-oai-proxy/compare/v1.0.8...HEAD
[1.0.8]: https://github.com/gyarbij/azure-oai-proxy/compare/v1.0.7...v1.0.8
[1.0.7]: https://github.com/gyarbij/azure-oai-proxy/compare/v1.0.6...v1.0.7
[1.0.6]: https://github.com/gyarbij/azure-oai-proxy/compare/v1.0.5...v1.0.6
[1.0.5]: https://github.com/gyarbij/azure-oai-proxy/compare/v1.0.4...v1.0.5
[1.0.4]: https://github.com/gyarbij/azure-oai-proxy/compare/v1.0.3...v1.0.4
[1.0.3]: https://github.com/gyarbij/azure-oai-proxy/compare/v1.0.2...v1.0.3
[1.0.2]: https://github.com/gyarbij/azure-oai-proxy/releases/tag/v1.0.2
