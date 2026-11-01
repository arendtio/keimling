#!/usr/bin/env python3
"""Replay agent-to-agent turns against a local llama.cpp Responses server.

The cases are the Instructor/Entertainer failures: answer a joke, look the
other agent up, send the joke back to the thread, and answer "Did you tell me?".

Worked examples in the system prompt were a regression on Ling-3.0-tiny.
What passed is a short rule plus a developer line on that turn naming
instance_id and the text. That line is peerReplyHint in src/model_input.go.

    python3 docs/lab/agent-reply/run.py --variant rules

Default server: http://127.0.0.1:8080/v1/responses
"""

import argparse
import json
import sys
import urllib.error
import urllib.request

URL = "http://127.0.0.1:8080/v1/responses"
MODEL = "inclusionAI/Ling-3.0-tiny-GGUF:Q4_K_M"

THREAD = "20261031-225911-490222439"
INSTRUCTOR = "agent-20261101-005219-472607549"
ENTERTAINER = "agent-20261101-005219-472142000"
JOKE = "Why don't scientists trust atoms? Because they make up everything!"

LIST_OUTPUT = (
    "You are Instructor. Your instance id is " + INSTRUCTOR + ". "
    "Do not pass your own id to message_agent.\n"
    "Pass one of these instance_id values:\n"
    "- instance_id=" + ENTERTAINER + ' name="Entertainer" peer_allowed=true shared_workspace=true\n'
)


def tool(name, desc, props, required):
    return {
        "type": "function",
        "name": name,
        "description": desc,
        "parameters": {
            "type": "object",
            "properties": props,
            "required": required,
            "additionalProperties": False,
        },
        "strict": True,
    }


TOOLS = [
    tool(
        "list_agents",
        "List other agent instances this agent may contact. Pass one of those instance_id values to message_agent. Never pass your own id.",
        {"unused": {"type": "string", "description": "Pass empty string"}},
        ["unused"],
    ),
    tool(
        "message_agent",
        "Send text to one recipient. instance_id is copied from the message you are answering. text is your answer, not a copy of their words. Writing in chat does not deliver it.",
        {
            "instance_id": {"type": "string", "description": "Recipient id copied from the message you are answering."},
            "text": {"type": "string", "description": "The answer for that recipient."},
        },
        ["instance_id", "text"],
    ),
]

CURRENT = """This chat is your channel with the user. Text you write here is not sent to other agents or to workspace threads.
A received line that names a thread_id or an instance_id and says via message_agent was delivered by message_agent. You never write that line. It is not the user.
You do not have to answer. If you answer that line, the answer must be a message_agent call to that id copied exactly. An assistant message in this chat does not deliver it.
A request from another agent or a thread is for you. Send the answer back to that sender. Do not rewrite it as a delivery to the user.
A message that arrived via message_agent is not the human in this chat. If you answer, call message_agent. text is your answer, not a copy of their words, and not a line in this chat. Do not type curly braces or a square bracket.
If they asked you to get something and bring it back, "me" is that sender. Send the result with message_agent to the same id. Writing it here does not deliver it.
One example. Sample ids only. Copy the id from the real line. Do not reuse a sample id.
A thread says: Ask Ada for a joke and tell it to me. The thread id is 20261031-192630-608087535.
Call list_agents. Ada's instance id in this sample is agent-20261031-120000-000000001.
Call message_agent. instance_id is agent-20261031-120000-000000001. text is: Tell me a joke.
Ada's joke arrives. You do not write the joke here.
Call message_agent. instance_id is 20261031-192630-608087535. text is the joke.
When Ada is asked "Tell me a joke.", Ada's text is the joke, not Tell me a joke.
"""

IF_THEN = """You are Instructor. Your instance id is """ + INSTRUCTOR + """.
When a message contains via message_agent, answer with the message_agent tool only. Do not write a chat reply.
instance_id is the id in that message. If the message says thread_id, that value is instance_id.
text is the answer, not a copy of the request.
If the message says to ask someone and tell the sender, first call list_agents, then message_agent to that agent. When the answer comes back, call message_agent to the original id with that answer. Do not write the answer in chat.
If the message is "Tell me a joke.", text is a joke.
"""

MAIL = """You are Instructor. Your instance id is """ + INSTRUCTOR + """.
A developer message is mail, not the human.
Answer mail only by calling message_agent.
instance_id is the Reply-to value. text is the answer.
If the body says to ask another agent and bring the result back, call list_agents, then message_agent to that agent. When their answer arrives, call message_agent to the original Reply-to with that answer.
If the body is "Tell me a joke.", text is a joke.
Do not write chat text about the mail.
"""


def user_bracket(who, body):
    return {"role": "user", "content": who + body}


def developer_mail(reply_to, body):
    return {
        "role": "developer",
        "content": "Mail via message_agent.\nReply-to: " + reply_to + "\nBody: " + body,
    }


def cases(presentation):
    if presentation == "bracket":
        ask = user_bracket(
            "[From thread_id=" + THREAD + " via message_agent] ",
            "Ask the Entertainer for a joke and tell it to me.",
        )
        joke_in = user_bracket(
            "[From Entertainer 15 instance_id=" + ENTERTAINER + " via message_agent] ",
            JOKE,
        )
        joke_req = user_bracket(
            "[From Instructor 15 instance_id=" + INSTRUCTOR + " via message_agent] ",
            "Tell me a joke.",
        )
        did = user_bracket(
            "[From thread_id=" + THREAD + " via message_agent] ",
            "Did you tell me the joke?",
        )
    else:
        ask = developer_mail(THREAD, "Ask the Entertainer for a joke and tell it to me.")
        joke_in = developer_mail(ENTERTAINER, JOKE)
        joke_req = developer_mail(INSTRUCTOR, "Tell me a joke.")
        did = developer_mail(THREAD, "Did you tell me the joke?")

    relay_input = [
        ask,
        {"type": "function_call", "name": "list_agents", "call_id": "call_list", "arguments": "{\"unused\":\"\"}"},
        {"type": "function_call_output", "call_id": "call_list", "output": LIST_OUTPUT},
        {
            "type": "function_call",
            "name": "message_agent",
            "call_id": "call_ask",
            "arguments": json.dumps({"instance_id": ENTERTAINER, "text": "Tell me a joke."}),
        },
        {"type": "function_call_output", "call_id": "call_ask", "output": "Sent to Entertainer: Tell me a joke."},
        joke_in,
    ]
    return {
        "answer": {
            "input": [joke_req],
            "expect": {"tool": "message_agent", "instance_id": INSTRUCTOR, "text_is": "joke", "no_chat": True},
        },
        "lookup": {
            "input": [ask],
            "expect": {"tool": "list_agents", "no_chat": True},
        },
        "relay": {
            "input": relay_input,
            "expect": {"tool": "message_agent", "instance_id": THREAD, "text_has": "atoms", "no_chat": True},
        },
        "did": {
            "input": relay_input + [
                {
                    "type": "function_call",
                    "name": "message_agent",
                    "call_id": "call_back",
                    "arguments": json.dumps({"instance_id": THREAD, "text": JOKE}),
                },
                {"type": "function_call_output", "call_id": "call_back", "output": "Sent to thread: " + JOKE},
                did,
            ],
            "expect": {"tool": "message_agent", "instance_id": THREAD, "no_chat": True},
        },
    }


RULES = """The last message arrived via message_agent. It is not the human. Do not write a chat reply.
Call message_agent.
If the last message asks you for a joke, instance_id is the id on that message and text is a joke you invent. text is not the request sentence.
If the last message is the answer you asked another agent for, instance_id is the thread_id from the first message, not the id on the last message. text is that answer.
If the first message says to ask someone, and you have not asked yet, call list_agents.
"""

SHORT = """The last message arrived via message_agent. It is not the human.
Call the message_agent tool. Copy the id from that message into instance_id.
text is your answer. If they asked for a joke, text is the joke itself.
If they asked you to get something and bring it back, call list_agents first, then message_agent to the other agent. When the result arrives, call message_agent to the original id with that result.
Do not write the result in chat.
"""

VARIANTS = {
    "current": CURRENT,
    "if-then": IF_THEN,
    "mail": MAIL,
    "short": SHORT,
    "rules": RULES,
}


REQUIRE_TOOL = False
NUDGE = False
NUDGE_TEXT = (
    "The message above arrived via message_agent. It is not the human. "
    "Respond with a function call only. Do not write chat text. "
    "If that message is an answer you were waiting for, message_agent instance_id is the thread_id from the first message and text is that answer."
)


def post(instructions, items):
    body = {
        "model": MODEL,
        "store": False,
        "stream": False,
        "temperature": 0,
        "max_output_tokens": 700,
        "instructions": instructions,
        "input": items,
        "tools": TOOLS,
        "parallel_tool_calls": False,
    }
    if REQUIRE_TOOL:
        body["tool_choice"] = "required"
    data = json.dumps(body).encode()
    req = urllib.request.Request(URL, data=data, headers={"Content-Type": "application/json"})
    try:
        with urllib.request.urlopen(req, timeout=180) as res:
            return json.loads(res.read().decode())
    except urllib.error.HTTPError as e:
        detail = e.read().decode(errors="replace")
        if "temperature" in detail and "temperature" in body:
            del body["temperature"]
            data = json.dumps(body).encode()
            req = urllib.request.Request(URL, data=data, headers={"Content-Type": "application/json"})
            with urllib.request.urlopen(req, timeout=180) as res:
                return json.loads(res.read().decode())
        raise SystemExit("HTTP %s\n%s" % (e.code, detail[:1200]))


def split(resp):
    calls = []
    chat = []
    for item in resp.get("output") or []:
        if item.get("type") == "function_call":
            try:
                args = json.loads(item.get("arguments") or "{}")
            except json.JSONDecodeError:
                args = {"_raw": item.get("arguments")}
            calls.append({"name": item.get("name"), "args": args})
        elif item.get("type") == "message":
            for part in item.get("content") or []:
                text = part.get("text") or ""
                if text.strip():
                    chat.append(text.strip())
    return calls, chat


def grade(expect, calls, chat):
    reasons = []
    if not calls:
        reasons.append("no tool call")
    else:
        call = calls[0]
        if call["name"] != expect["tool"]:
            reasons.append("tool %s" % call["name"])
        if "instance_id" in expect and call["args"].get("instance_id") != expect["instance_id"]:
            reasons.append("instance_id %r" % call["args"].get("instance_id"))
        text = str(call["args"].get("text") or "")
        if expect.get("text_is") == "joke":
            if text.strip() in ("", "Tell me a joke.", "Tell me a joke"):
                reasons.append("text is not a joke: %r" % text)
            if text.strip().startswith("[") or text.strip().startswith("{"):
                reasons.append("text copies a template")
        if "text_has" in expect and expect["text_has"].lower() not in text.lower():
            reasons.append("text missing %r: %r" % (expect["text_has"], text[:120]))
    if expect.get("no_chat") and chat:
        reasons.append("chat: %s" % chat[0][:160])
    return reasons


def run(variant, presentation, only):
    instructions = VARIANTS[variant]
    if presentation == "mail" and variant != "mail":
        instructions = MAIL
    results = []
    for name, case in cases(presentation).items():
        if only and name != only:
            continue
        items = list(case["input"])
        if NUDGE:
            items.append({"role": "developer", "content": NUDGE_TEXT})
        resp = post(instructions, items)
        calls, chat = split(resp)
        reasons = grade(case["expect"], calls, chat)
        results.append((name, not reasons, reasons, calls, chat))
        mark = "PASS" if not reasons else "FAIL"
        print("%s %s/%s %s" % (mark, variant, presentation, name))
        if reasons:
            print("  " + "; ".join(reasons))
        if calls:
            print("  call", json.dumps(calls[0], ensure_ascii=False)[:300])
        sys.stdout.flush()
    return results


def main():
    p = argparse.ArgumentParser()
    p.add_argument("--variant", choices=sorted(VARIANTS), action="append")
    p.add_argument("--presentation", choices=("bracket", "mail"), default="bracket")
    p.add_argument("--case", choices=("answer", "lookup", "relay", "did"))
    p.add_argument("--both", action="store_true", help="run bracket and mail presentations")
    p.add_argument("--require-tool", action="store_true")
    p.add_argument("--nudge", action="store_true", help="append a developer reminder after the last message")
    args = p.parse_args()
    global REQUIRE_TOOL, NUDGE
    REQUIRE_TOOL = args.require_tool
    NUDGE = args.nudge
    variants = args.variant or ["current"]
    presentations = ["bracket", "mail"] if args.both else [args.presentation]
    failed = 0
    for variant in variants:
        for presentation in presentations:
            # mail instructions only make sense with the mail presentation
            if variant == "mail" and presentation == "bracket":
                continue
            if variant != "mail" and presentation == "mail":
                continue
            for _, ok, _, _, _ in run(variant, presentation, args.case):
                if not ok:
                    failed += 1
    print("failed", failed)
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())
