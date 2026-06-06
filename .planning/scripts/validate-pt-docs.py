#!/usr/bin/env -S uv run --quiet --with websocket-client python
"""Phase 04 visual validation — chromium headless + CDP raw WS bypass.
Validates /pt/docs/ loads Next.js with CSS, logo, sidebar, nav."""
import json, sys, urllib.request, urllib.parse, websocket, time, base64, os, signal, subprocess, argparse

PORT = 9333
CHROME_BIN = "/snap/bin/chromium"
PROFILE = "/tmp/chrome-phase04-validate"
CDP = f"http://127.0.0.1:{PORT}"

p = argparse.ArgumentParser()
p.add_argument("--url", required=True)
p.add_argument("--out", default=None, help="Optional screenshot path")
p.add_argument("--label", default="page")
args = p.parse_args()

# Kill old chromium on this port
subprocess.run(["pkill", "-9", "-f", f"remote-debugging-port={PORT}"], stderr=subprocess.DEVNULL)
os.makedirs(PROFILE, exist_ok=True)

# Spawn
proc = subprocess.Popen(
    [CHROME_BIN, "--headless=new", "--no-sandbox", "--disable-gpu",
     "--disable-dev-shm-usage",
     f"--remote-debugging-port={PORT}",
     "--remote-allow-origins=*",
     f"--user-data-dir={PROFILE}",
     "--window-size=1920,1200",
     "about:blank"],
    stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)

# Wait CDP
for _ in range(30):
    try: urllib.request.urlopen(f"{CDP}/json/version", timeout=1); break
    except: time.sleep(0.5)
else:
    print("CDP TIMEOUT"); sys.exit(1)

# Open target
req = urllib.request.Request(f"{CDP}/json/new?{urllib.parse.quote(args.url, safe='')}", method="PUT")
target = json.loads(urllib.request.urlopen(req).read())
print(f"[{args.label}] target: {target['id']} | {target['url'][:80]}")

ws = websocket.create_connection(target["webSocketDebuggerUrl"], timeout=60, suppress_origin=True)
def send(method, params=None, rid=1, wait_s=20):
    ws.send(json.dumps({"id": rid, "method": method, "params": params or {}}))
    ws.settimeout(wait_s)
    while True:
        j = json.loads(ws.recv())
        if j.get("id") == rid: return j

send("Page.enable", rid=1)
send("Runtime.enable", rid=2)
send("Page.navigate", {"url": args.url}, rid=10, wait_s=20)
time.sleep(8)  # SPA hydration

# Eval state
expr = """JSON.stringify({
  url: window.location.href,
  title: document.title,
  lang: document.documentElement.lang,
  body200: (document.body?.innerText || '').substring(0, 250),
  cssVars: {
    navHeight: getComputedStyle(document.documentElement).getPropertyValue('--fd-nav-height').trim(),
    sidebarWidth: getComputedStyle(document.documentElement).getPropertyValue('--fd-layout-offset').trim(),
  },
  cssRuleCount: (function(){ try{ let n=0; for(const s of document.styleSheets){try{n+=s.cssRules.length}catch(e){}} return n }catch(e){ return -1 }})(),
  sidebarPresent: !!document.querySelector('aside, [class*="sidebar" i]'),
  navLinkSample: Array.from(document.querySelectorAll('nav a, header a')).slice(0, 6).map(a => a.innerText.trim()).filter(Boolean),
  imgCount: document.querySelectorAll('img').length,
  logoImgs: Array.from(document.querySelectorAll('img')).slice(0, 4).map(i => ({src: i.src.replace(location.origin,''), complete: i.complete, w: i.naturalWidth, h: i.naturalHeight})),
  theme: document.documentElement.className,
})"""
r = send("Runtime.evaluate", {"expression": expr, "returnByValue": True}, rid=20, wait_s=15)
state = json.loads(r["result"]["result"]["value"])
print(f"[{args.label}] STATE:")
print(json.dumps(state, indent=2, ensure_ascii=False))

# Screenshot
if args.out:
    r = send("Page.captureScreenshot", {"format": "png", "captureBeyondViewport": False}, rid=30, wait_s=15)
    img = base64.b64decode(r["result"]["data"])
    with open(args.out, "wb") as f: f.write(img)
    print(f"[{args.label}] screenshot: {args.out} ({len(img)} bytes)")

ws.close()
proc.terminate()
proc.wait(timeout=5)
