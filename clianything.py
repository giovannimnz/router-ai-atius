#!/usr/bin/env python3
"""
Atius Browser Control CLI - clianything
Control the Atius AI Router frontend via CLI/HTTP.
Can be used standalone or as an MCP server.

Usage:
  # CLI mode
  python clianything.py --cmd "goto" --url "https://router.atius.com.br"
  python clianything.py --cmd "screenshot" --path "/tmp/screen.png"
  
  # Server mode (for AI/MCP control)
  python clianything.py --server --port 8899
  
  # MCP mode (stdio)
  python clianything.py --mcp
"""

import argparse
import asyncio
import json
import sys
import os
import base64
import time
import threading
from typing import Optional, Dict, Any, List
from multiprocessing.connection import Client, Listener
from pathlib import Path

try:
    from playwright.async_api import async_playwright
except ImportError:
    print("Installing playwright...")
    os.system("pip install playwright -q")
    os.system("playwright install chromium")
    from playwright.async_api import async_playwright


SOCKET_PATH = "/tmp/clianything.sock"


class BrowserController:
    def __init__(self, headless: bool = True):
        self.browser = None
        self.context = None
        self.page = None
        self.headless = headless
        self.playwright = None
        self._browser_launched = False
        
    async def launch(self):
        if self._browser_launched:
            return
        self.playwright = await async_playwright().start()
        self.browser = await self.playwright.chromium.launch(headless=self.headless)
        self.context = await self.browser.new_context(viewport={"width": 1280, "height": 800})
        self.page = await self.context.new_page()
        self._browser_launched = True
    
    async def close(self):
        if self.page:
            await self.page.close()
        if self.context:
            await self.context.close()
        if self.browser:
            await self.browser.close()
        if self.playwright:
            await self.playwright.stop()
        self._browser_launched = False
        self.browser = None
        self.page = None
        self.context = None
    
    async def __aenter__(self):
        await self.launch()
        return self
    
    async def __aexit__(self, *args):
        await self.close()
    
    async def execute_command(self, cmd: str, args: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        args = args or {}
        
        if not self._browser_launched:
            if cmd == "launch":
                await self.launch()
                return {"ok": True, "action": "launch"}
            return {"error": "Browser not initialized. Use 'launch' command first."}
        
        cmd = cmd.strip().lower()
        
        try:
            if cmd == "goto":
                url = args.get("url")
                if not url:
                    return {"error": "url required for 'goto' command"}
                response = await self.page.goto(url, timeout=30000)
                return {"ok": True, "status": response.status if response else None, "url": url}
            
            elif cmd == "click":
                selector = args.get("selector")
                if not selector:
                    return {"error": "selector required for 'click' command"}
                await self.page.click(selector, timeout=10000)
                return {"ok": True, "action": "click", "selector": selector}
            
            elif cmd == "fill":
                selector = args.get("selector")
                value = str(args.get("value", ""))
                if not selector:
                    return {"error": "selector required for 'fill' command"}
                await self.page.fill(selector, value, timeout=10000)
                return {"ok": True, "action": "fill", "selector": selector, "value_length": len(value)}
            
            elif cmd == "type":
                selector = args.get("selector")
                value = str(args.get("value", ""))
                if not selector:
                    return {"error": "selector required for 'type' command"}
                await self.page.fill(selector, "")
                await self.page.type(selector, value, delay=50)
                return {"ok": True, "action": "type", "selector": selector, "value_length": len(value)}
            
            elif cmd == "hover":
                selector = args.get("selector")
                if not selector:
                    return {"error": "selector required for 'hover' command"}
                await self.page.hover(selector, timeout=10000)
                return {"ok": True, "action": "hover", "selector": selector}
            
            elif cmd == "wait":
                seconds = float(args.get("seconds", 1))
                await asyncio.sleep(seconds)
                return {"ok": True, "action": "wait", "seconds": seconds}
            
            elif cmd == "wait_selector":
                selector = args.get("selector")
                timeout = int(args.get("timeout", 10000))
                state = args.get("state", "visible")
                if not selector:
                    return {"error": "selector required for 'wait_selector' command"}
                await self.page.wait_for_selector(selector, state=state, timeout=timeout)
                return {"ok": True, "action": "wait_selector", "selector": selector, "state": state}
            
            elif cmd == "screenshot":
                path = args.get("path", "/tmp/screenshot.png")
                full_page = bool(args.get("full_page", False))
                await self.page.screenshot(path=path, full_page=full_page)
                size = os.path.getsize(path)
                with open(path, "rb") as f:
                    img_data = base64.b64encode(f.read()).decode()
                return {"ok": True, "action": "screenshot", "path": path, "size": size, "data": img_data[:200] + "..."}
            
            elif cmd == "html":
                content = await self.page.content()
                return {"ok": True, "action": "html", "length": len(content), "content": content[:5000]}
            
            elif cmd == "title":
                return {"ok": True, "action": "title", "title": await self.page.title()}
            
            elif cmd == "url":
                return {"ok": True, "action": "url", "url": self.page.url}
            
            elif cmd == "evaluate":
                code = args.get("code")
                if not code:
                    return {"error": "code required for 'evaluate' command"}
                result = await self.page.evaluate(code)
                result_str = str(result)
                if len(result_str) > 2000:
                    result_str = result_str[:2000] + "..."
                return {"ok": True, "action": "evaluate", "result": result_str}
            
            elif cmd == "query":
                selector = args.get("selector")
                if not selector:
                    return {"error": "selector required for 'query' command"}
                count = await self.page.locator(selector).count()
                first_text = ""
                if count > 0:
                    try:
                        first_text = (await self.page.locator(selector).first.text_content())[:200] or ""
                    except Exception:
                        pass
                return {"ok": True, "action": "query", "selector": selector, "count": count, "first_text": first_text}
            
            elif cmd == "press":
                selector = args.get("selector")
                key = args.get("key")
                if not selector or not key:
                    return {"error": "selector and key required for 'press' command"}
                await self.page.locator(selector).press(key, timeout=5000)
                return {"ok": True, "action": "press", "selector": selector, "key": key}
            
            elif cmd == "scroll":
                x = int(args.get("x", 0))
                y = int(args.get("y", 0))
                await self.page.evaluate(f"window.scrollTo({x}, {y})")
                return {"ok": True, "action": "scroll", "x": x, "y": y}
            
            elif cmd == "scroll_down":
                pixels = int(args.get("pixels", 300))
                await self.page.evaluate(f"window.scrollBy(0, {pixels})")
                return {"ok": True, "action": "scroll_down", "pixels": pixels}
            
            elif cmd == "scroll_up":
                pixels = int(args.get("pixels", 300))
                await self.page.evaluate(f"window.scrollBy(0, -{pixels})")
                return {"ok": True, "action": "scroll_up", "pixels": pixels}
            
            elif cmd == "refresh":
                await self.page.reload()
                return {"ok": True, "action": "refresh"}
            
            elif cmd == "launch":
                await self.launch()
                return {"ok": True, "action": "launch"}
            
            elif cmd == "close":
                await self.close()
                return {"ok": True, "action": "close"}
            
            elif cmd == "cookies":
                cookies = await self.context.cookies()
                return {"ok": True, "action": "cookies", "count": len(cookies), "cookies": cookies}
            
            else:
                return {"error": f"Unknown command: {cmd}"}
        
        except Exception as e:
            return {"error": str(e), "command": cmd}


# ─────────────────────────────────────────────────────────────────
# Background thread server (uses Unix socket for browser state)
# ─────────────────────────────────────────────────────────────────

def bg_server():
    """Background server that maintains browser state."""
    loop = asyncio.new_event_loop()
    asyncio.set_event_loop(loop)
    
    bc = BrowserController(headless=True)
    loop.run_until_complete(bc.launch())
    
    def handle_request(data):
        cmd = data.get("command", "")
        args = data.get("args", {})
        return loop.run_until_complete(bc.execute_command(cmd, args))
    
    if os.path.exists(SOCKET_PATH):
        os.unlink(SOCKET_PATH)
    
    listener = Listener(SOCKET_PATH, authkey=b"clianything")
    
    while True:
        try:
            conn = listener.accept()
            try:
                data = conn.recv()
                result = handle_request(data)
                conn.send(result)
            finally:
                conn.close()
        except Exception as e:
            pass


async def run_cli_command(args):
    cmd = args.cmd.lower()
    cmd_args: Dict[str, Any] = {}
    
    if args.selector:
        cmd_args["selector"] = args.selector
    if args.value is not None:
        cmd_args["value"] = args.value
    if args.url:
        cmd_args["url"] = args.url
    if args.path:
        cmd_args["path"] = args.path
    if args.full_page:
        cmd_args["full_page"] = True
    if args.seconds is not None:
        cmd_args["seconds"] = args.seconds
    if args.timeout:
        cmd_args["timeout"] = args.timeout
    if args.code:
        cmd_args["code"] = args.code
    if args.json:
        try:
            cmd_args.update(json.loads(args.json))
        except json.JSONDecodeError as e:
            print(f"JSON parse error: {e}")
            sys.exit(1)
    
    async with BrowserController(headless=args.headless) as bc:
        result = await bc.execute_command(cmd, cmd_args)
        print(json.dumps(result, indent=2, ensure_ascii=False))


async def run_mcp():
    """MCP mode - read JSON-RPC from stdin, write to stdout."""
    from concurrent.futures import ThreadPoolExecutor
    
    bc = BrowserController(headless=True)
    await bc.launch()
    
    executor = ThreadPoolExecutor(max_workers=1)
    loop = asyncio.get_event_loop()
    
    def do_command(cmd, args):
        return asyncio.run_coroutine_threadsafe(bc.execute_command(cmd, args), loop).result(60)
    
    while True:
        try:
            line = sys.stdin.readline()
            if not line:
                break
            
            req = json.loads(line.strip())
            method = req.get("method", "")
            params = req.get("params", {})
            req_id = req.get("id")
            
            if method == "initialize":
                result = {
                    "protocolVersion": "2024-11-05",
                    "capabilities": {"roots": {"listChanged": True}, "sampling": {}},
                    "clientInfo": {"name": "clianything", "version": "1.0.0"}
                }
                response = {"jsonrpc": "2.0", "id": req_id, "result": result}
            
            elif method == "tools/list":
                result = {
                    "tools": [
                        {"name": "goto", "description": "Navigate to URL", "inputSchema": {"type": "object", "properties": {"url": {"type": "string"}}, "required": ["url"]}},
                        {"name": "click", "description": "Click element", "inputSchema": {"type": "object", "properties": {"selector": {"type": "string"}}, "required": ["selector"]}},
                        {"name": "fill", "description": "Fill input field", "inputSchema": {"type": "object", "properties": {"selector": {"type": "string"}, "value": {"type": "string"}}, "required": ["selector", "value"]}},
                        {"name": "wait", "description": "Wait seconds", "inputSchema": {"type": "object", "properties": {"seconds": {"type": "number"}}, "required": ["seconds"]}},
                        {"name": "screenshot", "description": "Take screenshot", "inputSchema": {"type": "object", "properties": {"path": {"type": "string"}, "full_page": {"type": "boolean"}}, "required": []}},
                        {"name": "html", "description": "Get page HTML", "inputSchema": {"type": "object", "properties": {}}},
                        {"name": "title", "description": "Get page title", "inputSchema": {"type": "object", "properties": {}}},
                        {"name": "url", "description": "Get current URL", "inputSchema": {"type": "object", "properties": {}}},
                        {"name": "evaluate", "description": "Evaluate JavaScript", "inputSchema": {"type": "object", "properties": {"code": {"type": "string"}}, "required": ["code"]}},
                        {"name": "query", "description": "Query elements", "inputSchema": {"type": "object", "properties": {"selector": {"type": "string"}}, "required": ["selector"]}},
                        {"name": "scroll_down", "description": "Scroll down", "inputSchema": {"type": "object", "properties": {"pixels": {"type": "number"}}, "required": []}},
                        {"name": "batch", "description": "Run multiple commands", "inputSchema": {"type": "object", "properties": {"commands": {"type": "array"}}, "required": ["commands"]}},
                    ]
                }
                response = {"jsonrpc": "2.0", "id": req_id, "result": result}
            
            elif method == "tools/call":
                tool_name = params.get("name", "")
                tool_args = params.get("arguments", {})
                
                if tool_name == "batch":
                    commands = tool_args.get("commands", [])
                    results = []
                    for c in commands:
                        cmd = c.get("command", "")
                        cmd_args = c.get("args", {})
                        r = await bc.execute_command(cmd, cmd_args)
                        results.append({"command": cmd, "result": r})
                    content = [{"type": "text", "text": json.dumps(results, indent=2)}]
                else:
                    cmd = tool_name
                    cmd_args = tool_args
                    r = await bc.execute_command(cmd, cmd_args)
                    content = [{"type": "text", "text": json.dumps(r, indent=2)}]
                
                result = {"content": content}
                response = {"jsonrpc": "2.0", "id": req_id, "result": result}
            
            elif method == "ping":
                result = {"pong": True}
                response = {"jsonrpc": "2.0", "id": req_id, "result": result}
            
            else:
                response = {"jsonrpc": "2.0", "id": req_id, "error": {"code": -32601, "message": f"Unknown method: {method}"}}
            
            print(json.dumps(response), flush=True)
        
        except Exception as e:
            error_resp = {"jsonrpc": "2.0", "id": None, "error": {"code": -32603, "message": str(e)}}
            print(json.dumps(error_resp), flush=True)


async def main():
    parser = argparse.ArgumentParser(description="Atius Browser Control CLI", prog="clianything")
    parser.add_argument("--cmd", "-c", help="Command to execute")
    parser.add_argument("--selector", "-s", help="CSS selector")
    parser.add_argument("--value", "-v", help="Value to fill/type")
    parser.add_argument("--url", "-u", help="URL for goto")
    parser.add_argument("--path", "-p", help="Screenshot path")
    parser.add_argument("--full-page", action="store_true")
    parser.add_argument("--seconds", type=float, default=None)
    parser.add_argument("--timeout", type=int, default=None)
    parser.add_argument("--code", help="JavaScript to evaluate")
    parser.add_argument("--json", "-j", help="JSON args")
    parser.add_argument("--server", action="store_true", help="Start server")
    parser.add_argument("--mcp", action="store_true", help="MCP mode")
    parser.add_argument("--port", type=int, default=8899)
    parser.add_argument("--headless", action="store_true")
    
    args = parser.parse_args()
    
    if args.mcp:
        await run_mcp()
    elif args.server:
        from aiohttp import web
        
        app = web.Application()
        
        bc = BrowserController(headless=True)
        await bc.launch()
        
        async def handle_cmd(request):
            data = await request.json()
            cmd = str(data.get("command", ""))
            cmd_args = data.get("args", {})
            result = await bc.execute_command(cmd, cmd_args)
            return web.json_response(result)
        
        async def handle_launch(request):
            await bc.launch()
            return web.json_response({"ok": True})
        
        async def handle_close(request):
            await bc.close()
            return web.json_response({"ok": True})
        
        async def handle_batch(request):
            data = await request.json()
            commands = data.get("commands", [])
            results = []
            for c in commands:
                cmd = str(c.get("command", ""))
                cmd_args = c.get("args", {})
                result = await bc.execute_command(cmd, cmd_args)
                results.append({"command": cmd, "result": result})
            return web.json_response({"ok": True, "results": results})
        
        async def handle_health(request):
            return web.json_response({
                "ok": True,
                "browser_launched": bc._browser_launched,
                "current_url": bc.page.url if bc.page else None,
            })
        
        app.router.add_post("/cmd", handle_cmd)
        app.router.add_post("/launch", handle_launch)
        app.router.add_post("/close", handle_close)
        app.router.add_post("/batch", handle_batch)
        app.router.add_get("/health", handle_health)
        
        runner = web.AppRunner(app)
        await runner.setup()
        site = web.TCPSite(runner, "0.0.0.0", args.port)
        await site.start()
        
        print(f"Atius Browser Control CLI running on http://0.0.0.0:{args.port}")
        print(f"  POST /cmd     - Execute single command")
        print(f"  POST /batch   - Execute multiple commands")
        print(f"  GET  /health  - Health check")
        print(f"\nPress Ctrl+C to stop")
        
        # Keep running
        await asyncio.Event().wait()
    elif args.cmd:
        await run_cli_command(args)
    else:
        parser.print_help()
        print("\nCommands: goto, click, fill, type, wait, screenshot, html, title, url, evaluate, query, scroll, scroll_down, scroll_up, launch, close")


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        print("\nInterrupted")
        sys.exit(0)