Goal
----
Replace the monolithic, regex-driven `Chat()` flow with an LLM-first orchestrator that: 
- Uses the existing tool registry (ai_companion_tools_*.go) as callable functions.
- Injects contextual memory & entity context into prompts.
- Streams LLM tokens to the frontend via existing SSE endpoint and emits structured `tool_call` events when tools are invoked.

Phases
------
1) Design & safety checks (this doc)
2) Add `llmOrchestrator` small surface: call LLM, parse JSON tool_call responses, call tools, stream back chunks
3) Replace `Chat()` to delegate to `llmOrchestrator` while keeping existing rule fallbacks
4) Add unit tests for orchestration and tool invocation
5) Tune latency and retries for streaming

API changes
-----------
- Keep the SSE endpoint `/api/v1/ai/chat/stream` compatible.
- Add lightweight metadata in SSE events: `event: tool_call`, `event: tool_result`, `event: llm_chunk`.
- Add optional query param `prefer_llm=true` to force LLM-first mode for testing.

Implementation notes
--------------------
- Use the existing `registerTool()` helpers to obtain callable tool funcs; implement a small adapter that converts `tool_call` (name + args) into a function call.
- When LLM returns a `tool_call`, orchestrator should: 1) stream a `tool_call` SSE event; 2) execute the tool (with timeout + context); 3) stream `tool_result` SSE event; 4) include tool output back into the LLM conversation (function-calling loop) until LLM finishes.
- Persisted memory retrieval: call existing memory APIs (or helper) to assemble a short context window (max tokens budget).

Backwards compatibility
-----------------------
- If a request body contains `force_rules=true`, fall back to the old `Chat()` rules engine.
- Keep event formats backward-compatible: produce `event: text` chunks when LLM final text is available.

Next immediate action
---------------------
Create a new file `api/internal/services/ai_companion_orchestrator.go` with a minimal LLM-orchestrator skeleton and unit tests.
