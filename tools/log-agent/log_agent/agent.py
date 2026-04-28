import json
import logging
from dataclasses import dataclass, field
from typing import Any

from log_agent.tools import TOOL_SCHEMAS, ToolContext, dispatch

log = logging.getLogger(__name__)


@dataclass
class AgentResult:
    text: str
    iterations: int
    aborted: bool = False
    abort_reason: str | None = None


@dataclass
class Agent:
    ctx: ToolContext
    client: Any                  # anthropic.Anthropic
    system_prompt: str
    history: list[dict[str, Any]] = field(default_factory=list)

    def run(self, user_message: str) -> AgentResult:
        checkpoint = len(self.history)
        self.history.append({"role": "user", "content": user_message})
        iterations = 0
        cumulative_input_tokens = 0
        max_iter = self.ctx.config.periodic.max_iterations
        max_tokens_in = self.ctx.config.limits.max_session_input_tokens

        try:
            while True:
                try:
                    resp = self.client.messages.create(
                        model=self.ctx.config.model,
                        max_tokens=4096,
                        system=[
                            {
                                "type": "text",
                                "text": self.system_prompt,
                                "cache_control": {"type": "ephemeral"},
                            }
                        ],
                        tools=TOOL_SCHEMAS,
                        messages=list(self.history),
                    )
                except Exception:
                    # Restore so the next run() doesn't see a dangling user message.
                    del self.history[checkpoint:]
                    raise

                # Anthropic's input_tokens counts the entire context window each turn,
                # so the cumulative sum overcounts vs. real new-token cost. The configured
                # limit is therefore conservative — the loop aborts well before the model's
                # actual token budget would.
                cumulative_input_tokens += resp.usage.input_tokens

                if resp.stop_reason == "end_turn":
                    # SUCCESS: keep history (assistant text appended below)
                    text = "".join(
                        block.text for block in resp.content if block.type == "text"
                    )
                    self.history.append({"role": "assistant", "content": text})
                    return AgentResult(text=text, iterations=iterations)

                if resp.stop_reason == "tool_use":
                    # Append assistant message with full content (text + tool_use blocks)
                    assistant_content = []
                    for block in resp.content:
                        if block.type == "text":
                            assistant_content.append({"type": "text", "text": block.text})
                        elif block.type == "tool_use":
                            assistant_content.append({
                                "type": "tool_use",
                                "id": block.id,
                                "name": block.name,
                                "input": block.input,
                            })
                    self.history.append({"role": "assistant", "content": assistant_content})

                    # Dispatch each tool_use, append a single user message with all tool_results
                    tool_results = []
                    for block in resp.content:
                        if block.type != "tool_use":
                            continue
                        result = dispatch(self.ctx, block.name, block.input)
                        is_error = "error" in result
                        tool_results.append({
                            "type": "tool_result",
                            "tool_use_id": block.id,
                            "content": json.dumps(result, default=str),
                            "is_error": is_error,
                        })
                    self.history.append({"role": "user", "content": tool_results})

                    iterations += 1
                    if iterations >= max_iter:
                        log.warning("agent aborted: max_iterations=%d", max_iter)
                        del self.history[checkpoint:]
                        return AgentResult(
                            text="(aborted: max iterations reached)",
                            iterations=iterations,
                            aborted=True,
                            abort_reason="max_iterations",
                        )
                    if cumulative_input_tokens >= max_tokens_in:
                        log.warning(
                            "agent aborted: max_session_input_tokens=%d (used %d)",
                            max_tokens_in, cumulative_input_tokens,
                        )
                        del self.history[checkpoint:]
                        return AgentResult(
                            text="(aborted: token budget exceeded)",
                            iterations=iterations,
                            aborted=True,
                            abort_reason="max_session_input_tokens",
                        )
                    continue

                # Any other stop reason — bail out
                log.warning("agent: unexpected stop_reason=%s", resp.stop_reason)
                del self.history[checkpoint:]
                return AgentResult(
                    text=f"(unexpected stop_reason: {resp.stop_reason})",
                    iterations=iterations,
                    aborted=True,
                    abort_reason=f"stop_reason:{resp.stop_reason}",
                )
        except Exception:
            # Defensive: any unhandled exception (besides the create() one above) — clean up.
            if len(self.history) > checkpoint:
                del self.history[checkpoint:]
            raise
