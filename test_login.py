from playwright.sync_api import sync_playwright

BASE = 'http://localhost:3301'
USERNAME = 'admin@atius.com.br'
PASSWORD = 'Bkfigt!546'

def run():
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        context = browser.new_context()
        page = context.new_page()

        console_messages = []
        page.on('console', lambda msg: console_messages.append(f"[{msg.type}] {msg.text}"))

        try:
            print('=== Step 1: Navigate to /sign-in ===')
            page.goto(f'{BASE}/sign-in', wait_until='networkidle')
            print(f"Page title: {page.title}")

            print('\n=== Step 2: Fill login form ===')
            page.wait_for_selector('input[name="username"]', timeout=10000)
            page.wait_for_selector('input[name="password"]', timeout=5000)

            page.fill('input[name="username"]', USERNAME)
            page.fill('input[name="password"]', PASSWORD)

            print(f'Filled username and password')

            print('\n=== Step 3: Submit form ===')
            page.click('button[type="submit"]')

            # Wait for network response
            page.wait_for_response(lambda r: '/api/user/login' in r.url, timeout=10000)
            page.wait_for_timeout(2000)

            print(f'URL after submit: {page.url}')

            # Check API response by inspecting network
            print('\n=== Checking login response ===')
            response = page.request.get(f'{BASE}/api/user/login', headers={'Content-Type': 'application/json'})
            print(f'GET /api/user/login status: {response.status}')
            print(f'GET /api/user/login body: {response.text}')

            print('\n=== Try POST /api/user/login directly ===')
            response2 = page.request.post(f'{BASE}/api/user/login', 
                headers={'Content-Type': 'application/json'},
                data='{"username":"admin@atius.com.br","password":"Bkfigt!546"}'
            )
            print(f'POST /api/user/login status: {response2.status}')
            print(f'POST /api/user/login body: {response2.text}')

            print('\n=== Cookies ===')
            cookies = context.cookies()
            print(f'Cookies received: {len(cookies)}')
            for cookie in cookies:
                print(f'  {cookie["name"]}: {cookie.get("value", "")[:50]}...')

            print('\n=== Step 4: Navigate directly to /docs/ ===')
            page.goto(f'{BASE}/docs/', wait_until='networkidle')
            print(f'Docs URL: {page.url}')
            print(f'Docs HTML (first 1500 chars):')
            body_html = page.evaluate('document.body.innerHTML')
            print(body_html[:1500])

            print('\n=== Console messages (last 10) ===')
            for msg in console_messages[-10:]:
                print(msg)

        except Exception as err:
            print(f'Error: {err}')
            import traceback
            traceback.print_exc()
        finally:
            browser.close()

if __name__ == '__main__':
    run()