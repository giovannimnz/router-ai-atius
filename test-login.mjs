import { chromium } from 'playwright';

const BASE = 'http://localhost:3301';
const USERNAME = 'admin@atius.com.br';
const PASSWORD = 'Bkfigt!546';

async function run() {
  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext();
  const page = await context.newPage();

  const consoleMessages = [];
  const consoleErrors = [];
  page.on('console', msg => {
    consoleMessages.push(`[${msg.type()}] ${msg.text()}`);
    if (msg.type() === 'error') consoleErrors.push(msg.text());
  });

  try {
    console.log('=== Step 1: Navigate to /sign-in ===');
    await page.goto(`${BASE}/sign-in`, { waitUntil: 'networkidle' });
    const signInTitle = await page.title();
    console.log(`Page title: ${signInTitle}`);
    const url = page.url();
    console.log(`URL: ${url}`);

    console.log('\n=== Step 2: Fill login form ===');
    // Wait for form to be visible
    await page.waitForSelector('form', { timeout: 10000 }).catch(() => {
      console.log('Form not found via "form" selector, checking inputs...');
    });

    // Try to find username/password fields
    const usernameInput = await page.$('input[type="text"], input[name="username"], input[id*="username" i]');
    const passwordInput = await page.$('input[type="password"]');
    const submitBtn = await page.$('button[type="submit"], button:has-text("Sign in"), button:has-text("Login"), button:has-text("Entrar")');

    console.log(`Username input found: ${!!usernameInput}`);
    console.log(`Password input found: ${!!passwordInput}`);
    console.log(`Submit button found: ${!!submitBtn}`);

    // Get all input fields
    const inputs = await page.$$('input');
    console.log(`Total input fields: ${inputs.length}`);
    for (const inp of inputs) {
      const type = await inp.getAttribute('type');
      const name = await inp.getAttribute('name');
      const id = await inp.getAttribute('id');
      const placeholder = await inp.getAttribute('placeholder');
      console.log(`  Input: type=${type}, name=${name}, id=${id}, placeholder=${placeholder}`);
    }

    if (usernameInput) await usernameInput.fill(USERNAME);
    if (passwordInput) await passwordInput.fill(PASSWORD);

    console.log('\n=== Step 3: Submit form ===');
    if (submitBtn) {
      await submitBtn.click();
    } else {
      await page.keyboard.press('Enter');
    }

    // Wait for response
    await page.waitForTimeout(3000);
    const afterSubmitUrl = page.url();
    console.log(`URL after submit: ${afterSubmitUrl}`);

    console.log('\n=== Step 4: Capture session cookies ===');
    const cookies = await context.cookies();
    console.log('Cookies received:');
    for (const cookie of cookies) {
      console.log(`  ${cookie.name}: ${cookie.value.substring(0, 50)}... (httpOnly=${cookie.httpOnly}, secure=${cookie.secure}, sameSite=${cookie.sameSite})`);
    }

    console.log('\n=== Step 5: Visit /docs/ with cookies ===');
    await page.goto(`${BASE}/docs/`, { waitUntil: 'networkidle' });
    const docsUrl = page.url();
    const docsTitle = await page.title();
    console.log(`Docs URL: ${docsUrl}`);
    console.log(`Docs title: ${docsTitle}`);

    // Get page content
    const bodyHTML = await page.evaluate(() => document.body.innerHTML);
    console.log('\n/docs/ HTML (first 2000 chars):');
    console.log(bodyHTML.substring(0, 2000));

    console.log('\n=== Console messages from /docs/ ===');
    console.log(consoleMessages.slice(-20).join('\n'));

    if (consoleErrors.length > 0) {
      console.log('\n=== Console ERRORS ===');
      console.log(consoleErrors.join('\n'));
    }

    // Check if Swagger UI (Scalar) rendered
    const scalarApp = await page.$('#scalar-app');
    const loadingScreen = await page.$('#loading');
    const authGate = await page.$('#auth-gate');
    console.log(`\nScalar app element: ${!!scalarApp}`);
    console.log(`Loading screen: ${!!loadingScreen}`);
    console.log(`Auth gate: ${!!authGate}`);

    const authGateDisplay = authGate ? await authGate.evaluate(el => getComputedStyle(el).display) : 'N/A';
    console.log(`Auth gate display: ${authGateDisplay}`);

    const scalarAppDisplay = scalarApp ? await scalarApp.evaluate(el => getComputedStyle(el).display) : 'N/A';
    console.log(`Scalar app display: ${scalarAppDisplay}`);

  } catch (err) {
    console.error('Error:', err.message);
  } finally {
    await browser.close();
  }
}

run().catch(console.error);