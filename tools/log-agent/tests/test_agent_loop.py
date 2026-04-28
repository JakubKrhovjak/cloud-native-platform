from types import SimpleNamespace
from unittest.mock import MagicMock

import pytest

from log_agent.agent import Agent, AgentResult


@pytest.fixture
def ctx(mocker, tmp_path):
    from log_agent.config import Config, PeriodicConfig, AlertsConfig, LimitsConfig
    from log_agent.state import AlertState
    from log_agent.tools import ToolContext

    cfg = Config(
        namespaces=["apps"],
        model="claude-sonnet-4-6",
        periodic=PeriodicConfig(lookback_seconds=900, max_iterations=3),
        alerts=AlertsConfig(dedup_ttl_seconds=3600, prefilter_pattern=r"ERROR"),
        limits=LimitsConfig(max_log_lines=2000, max_session_input_tokens=100),
    )
    return ToolContext(
        config=cfg,
        state=AlertState(tmp_path / "alerts.json", ttl_seconds=3600),
        core_v1=MagicMock(),
        notify=MagicMock(return_value=True),
    )


def _text_response(text: str, input_tokens: int = 10):
    return SimpleNamespace(
        stop_reason="end_turn",
        content=[SimpleNamespace(type="text", text=text)],
        usage=SimpleNamespace(input_tokens=input_tokens, output_tokens=5),
    )


def _tool_use_response(tool_name: str, tool_input: dict, tool_id: str = "id1", input_tokens: int = 10):
    return SimpleNamespace(
        stop_reason="tool_use",
        content=[
            SimpleNamespace(type="text", text="thinking..."),
            SimpleNamespace(type="tool_use", id=tool_id, name=tool_name, input=tool_input),
        ],
        usage=SimpleNamespace(input_tokens=input_tokens, output_tokens=5),
    )


def test_agent_returns_text_when_end_turn(ctx, mocker):
    client = MagicMock()
    client.messages.create.return_value = _text_response("all clear")
    agent = Agent(ctx=ctx, client=client, system_prompt="sys")
    result = agent.run("any problems?")
    assert isinstance(result, AgentResult)
    assert result.text == "all clear"
    assert result.iterations == 0
    assert result.aborted is False


def test_agent_dispatches_tool_then_ends(ctx, mocker):
    client = MagicMock()
    ctx.core_v1.list_namespaced_pod.return_value = SimpleNamespace(items=[])
    client.messages.create.side_effect = [
        _tool_use_response("list_pods", {"namespace": "apps"}),
        _text_response("done"),
    ]
    agent = Agent(ctx=ctx, client=client, system_prompt="sys")
    result = agent.run("check")
    assert result.text == "done"
    assert result.iterations == 1
    assert client.messages.create.call_count == 2


def test_agent_aborts_on_max_iterations(ctx, mocker):
    client = MagicMock()
    ctx.core_v1.list_namespaced_pod.return_value = SimpleNamespace(items=[])
    # always return tool_use -> never ends
    client.messages.create.return_value = _tool_use_response(
        "list_pods", {"namespace": "apps"}
    )
    agent = Agent(ctx=ctx, client=client, system_prompt="sys")
    result = agent.run("loop forever")
    assert result.aborted is True
    assert result.abort_reason == "max_iterations"
    assert result.iterations == ctx.config.periodic.max_iterations


def test_agent_aborts_on_max_session_tokens(ctx, mocker):
    client = MagicMock()
    ctx.core_v1.list_namespaced_pod.return_value = SimpleNamespace(items=[])
    client.messages.create.return_value = _tool_use_response(
        "list_pods", {"namespace": "apps"}, input_tokens=60  # 60 + 60 > 100 -> aborts after 2nd
    )
    agent = Agent(ctx=ctx, client=client, system_prompt="sys")
    result = agent.run("expensive")
    assert result.aborted is True
    assert result.abort_reason == "max_session_input_tokens"


def test_agent_unknown_tool_returns_error_to_claude(ctx, mocker):
    client = MagicMock()
    client.messages.create.side_effect = [
        _tool_use_response("nonexistent_tool", {}),
        _text_response("noted"),
    ]
    agent = Agent(ctx=ctx, client=client, system_prompt="sys")
    result = agent.run("test")
    assert result.text == "noted"
    second_call_messages = client.messages.create.call_args_list[1].kwargs["messages"]
    last = second_call_messages[-1]
    tool_result_block = last["content"][0]
    assert tool_result_block["is_error"] is True


def test_agent_chat_preserves_history_across_calls(ctx, mocker):
    client = MagicMock()
    client.messages.create.side_effect = [
        _text_response("first answer"),
        _text_response("second answer"),
    ]
    agent = Agent(ctx=ctx, client=client, system_prompt="sys")
    r1 = agent.run("q1")
    r2 = agent.run("q2")
    assert r1.text == "first answer"
    assert r2.text == "second answer"
    second_messages = client.messages.create.call_args_list[1].kwargs["messages"]
    # history must include q1, assistant1, q2 at minimum
    user_texts = [m for m in second_messages if m["role"] == "user"]
    assert len(user_texts) >= 2


def test_agent_dispatches_tool_with_correct_history_shape(ctx, mocker):
    """Verify the assistant + user(tool_results) message shape is correct."""
    client = MagicMock()
    ctx.core_v1.list_namespaced_pod.return_value = SimpleNamespace(items=[])
    client.messages.create.side_effect = [
        _tool_use_response("list_pods", {"namespace": "apps"}, tool_id="tu_abc"),
        _text_response("done"),
    ]
    agent = Agent(ctx=ctx, client=client, system_prompt="sys")
    agent.run("check")

    second_call_messages = client.messages.create.call_args_list[1].kwargs["messages"]
    # history snapshot at second call: [user(check), assistant(text+tool_use), user(tool_result)]
    assert second_call_messages[0] == {"role": "user", "content": "check"}
    assistant = second_call_messages[1]
    assert assistant["role"] == "assistant"
    assert any(b.get("type") == "tool_use" and b.get("id") == "tu_abc" for b in assistant["content"])
    user_tr = second_call_messages[2]
    assert user_tr["role"] == "user"
    tr_block = user_tr["content"][0]
    assert tr_block["type"] == "tool_result"
    assert tr_block["tool_use_id"] == "tu_abc"
    assert tr_block["is_error"] is False


def test_agent_unexpected_stop_reason_returns_abort(ctx, mocker):
    """Cover the unexpected stop_reason fallback branch."""
    client = MagicMock()
    client.messages.create.return_value = SimpleNamespace(
        stop_reason="weird_unknown_reason",
        content=[],
        usage=SimpleNamespace(input_tokens=5, output_tokens=0),
    )
    agent = Agent(ctx=ctx, client=client, system_prompt="sys")
    result = agent.run("test")
    assert result.aborted is True
    assert "weird_unknown_reason" in result.abort_reason


def test_agent_history_restored_after_abort(ctx, mocker):
    """After max_iterations abort, next run() must start from a clean checkpoint
    so we don't get consecutive user messages."""
    client = MagicMock()
    ctx.core_v1.list_namespaced_pod.return_value = SimpleNamespace(items=[])
    # First run: tool_use looped -> max_iterations abort
    client.messages.create.side_effect = (
        [_tool_use_response("list_pods", {"namespace": "apps"})] * ctx.config.periodic.max_iterations
        + [_text_response("recovered")]  # second run() succeeds
    )
    agent = Agent(ctx=ctx, client=client, system_prompt="sys")
    r1 = agent.run("loop")
    assert r1.aborted is True

    # Second call should start clean — no leftover user message
    r2 = agent.run("hello again")
    assert r2.text == "recovered"
    second_call_messages = client.messages.create.call_args_list[-1].kwargs["messages"]
    user_msgs = [m for m in second_call_messages if m["role"] == "user"]
    # Only the new user message should be present (no leftover from aborted run)
    assert len(user_msgs) == 1
    assert user_msgs[0]["content"] == "hello again"


def test_agent_history_restored_after_sdk_exception(ctx, mocker):
    """If client.messages.create raises, the user message must be cleaned up
    so the next run() starts clean."""
    client = MagicMock()
    client.messages.create.side_effect = [
        RuntimeError("simulated API error"),
        _text_response("recovered"),
    ]
    agent = Agent(ctx=ctx, client=client, system_prompt="sys")
    with pytest.raises(RuntimeError):
        agent.run("first")
    r2 = agent.run("second")
    assert r2.text == "recovered"
    second_call_messages = client.messages.create.call_args_list[-1].kwargs["messages"]
    user_msgs = [m for m in second_call_messages if m["role"] == "user"]
    assert len(user_msgs) == 1
    assert user_msgs[0]["content"] == "second"
