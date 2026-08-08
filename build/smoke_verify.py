#!/usr/bin/env python3
"""Verify the current vocat deployment on 192.168.2.222:
1. notification channels trimmed to telegram/email/webhook/bark/pushplus
2. removed channels (feishu/qq/weixin) rejected
3. the new unsaved-config front-proxy probe endpoint (UDP Associate for VoWiFi)
4. the served frontend bundle reflects the changes (probe button present, API
   docs button and Feishu/QQ tabs gone)
"""
import http.client
import http.cookiejar
import json
import re
import time
import urllib.error
import urllib.request

BASE = "http://192.168.2.222:7575"
USERNAME = "admin"
PASSWORD = "vocat520"

jar = http.cookiejar.CookieJar()
opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar))
results = []


def _once(method, path, body, csrf, timeout, raw):
    req = urllib.request.Request(BASE + path, method=method)
    data = None
    if body is not None:
        req.add_header("Content-Type", "application/json")
        data = json.dumps(body).encode()
    if csrf:
        req.add_header("X-CSRF-Token", csrf)
    req.add_header("Connection", "close")
    try:
        with opener.open(req, data=data, timeout=timeout) as resp:
            payload = resp.read().decode("utf-8", "replace")
            return (resp.status, payload) if raw else (resp.status, json.loads(payload) if payload else {})
    except urllib.error.HTTPError as exc:
        text = exc.read().decode("utf-8", "replace")
        if raw:
            return exc.code, text
        try:
            return exc.code, json.loads(text)
        except json.JSONDecodeError:
            return exc.code, {"raw": text[:200]}


def call(method, path, body=None, csrf=None, timeout=40, raw=False):
    # The Windows TCP stack sometimes reports a RST right after the server has
    # already sent a complete response; retry those client-side resets.
    last = None
    for attempt in range(4):
        try:
            return _once(method, path, body, csrf, timeout, raw)
        except (ConnectionResetError, urllib.error.URLError, http.client.RemoteDisconnected) as exc:
            last = exc
            time.sleep(0.4 * (attempt + 1))
    raise last


def check(name, cond, detail=""):
    results.append((name, bool(cond)))
    print(("PASS " if cond else "FAIL ") + name + (f"  | {detail}" if detail and not cond else ""))


def bundle_has(bundle, s):
    # bundle may keep Chinese literally or \uXXXX-escaped depending on charset
    if s in bundle:
        return True
    esc = "".join("\\u%04x" % ord(c) for c in s)
    return esc.lower() in bundle.lower()


# 1. login
st, login = call("POST", "/api/auth/login", {"username": USERNAME, "password": PASSWORD})
csrf = (login.get("data") or {}).get("csrf_token", "")
check("登录 admin", st == 200 and bool(csrf), json.dumps(login, ensure_ascii=False)[:160])

# 2. notifications trimmed to exactly the 5 kept channels
st, notif = call("GET", "/api/settings/notifications")
data = notif.get("data") or {}
channels = set(data.keys())
expect = {"telegram", "email", "webhook", "bark", "pushplus"}
check("通知渠道恰好 5 个", st == 200 and channels == expect, ",".join(sorted(channels)))
check("feishu/qq/weixin 已移除", not ({"feishu", "qq", "weixin"} & channels), ",".join(sorted(channels)))

# 3. removed channels rejected by both PUT and the test endpoint
st, resp = call("PUT", "/api/settings/notifications", {"feishu": {"enabled": False}}, csrf)
check("PUT feishu 配置被拒(400)", st == 400, f"status={st} {json.dumps(resp)[:120]}")
for removed in ("feishu", "qq", "weixin"):
    st, _ = call("POST", f"/api/settings/notifications/{removed}/test", {}, csrf)
    check(f"{removed} 测试端点 404", st == 404, f"status={st}")

# 4. new unsaved-config front-proxy probe endpoint (UDP Associate for VoWiFi)
st, probe = call("POST", "/api/upstream-proxy-probe", {"addr": "127.0.0.1:9"}, csrf, timeout=20)
pdata = probe.get("data") or {}
presult = pdata.get("probe") or {}
check("探测端点存在且返回 probe 结构", st == 200 and "probe" in pdata, f"status={st} {json.dumps(probe, ensure_ascii=False)[:200]}")
check(
    "不可达地址正确判定 reachable/udp=false",
    presult.get("reachable") is False and presult.get("udp_associate_ok") is False,
    json.dumps(presult, ensure_ascii=False)[:200],
)
st, _ = call("POST", "/api/upstream-proxy-probe", {"addr": ""}, csrf, timeout=20)
check("空地址探测返回 400", st == 400, f"status={st}")

# 5. frontend bundle reflects the changes (the app code lives in /assets/index-*.js)
st, index = call("GET", "/", raw=True)
m = re.search(r'src="(/assets/index-[^"]+\.js)"', index)
bundle = ""
if m:
    _, bundle = call("GET", m.group(1), raw=True)
check("获取前端 JS bundle", bool(bundle), "no bundle url in index.html")
if bundle:
    check("bundle 含「检测连通性」按钮", bundle_has(bundle, "检测连通性"))
    check("bundle 含 UDP Associate(VoWiFi) 探测", bundle_has(bundle, "UDP Associate"))
    check("bundle 已无「API 文档」按钮", not bundle_has(bundle, "API 文档"))
    check("bundle 已无飞书", not bundle_has(bundle, "飞书"))
    check("bundle 已无 QQ 渠道", not bundle_has(bundle, "QQ 机器人") and not bundle_has(bundle, "QQ Bot"))

failed = [n for n, ok in results if not ok]
print(f"\n==== {len(results) - len(failed)}/{len(results)} 通过 ====")
if failed:
    print("失败项:", "; ".join(failed))
    raise SystemExit(1)
