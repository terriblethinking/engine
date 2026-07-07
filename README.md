# agent

---

## intro

this is the starting point for the agentic harness terrible thinking is going to create.

it is a very minimal implmentation of an agentic loop, focused on making adding and plugging in other stuff as easy and quick as possible.

this is basically a template **anyone** can use; whether you are going to work with the terriblethinking harness or not.

that is why there is very little files and actual logic besides the agentic loop; this repo gives you a ready engine you can plugin into virtually anything you want to create.

---

## restrictions

of course, there are somethings that the project depends on and you can't change; for example, the whole thing works exclusively on [bifrost](https://github.com/maximhq/bifrost), an AI gateway which has 23 built in providers and allows you to add any custom ones that support the OpenAI structure.

another minor restriction lies in what you can name the tools you give to your agent; any tool which includes "-" in the name will be treated as a MCP tool, to be run with bifrost.ExecuteChatMCPTool, not a simple function.

otherwise, everything is in your hands.

---

## contributing

open an issue if you find one, push requests will almost 100% be closed, we are currently not accepting any.
