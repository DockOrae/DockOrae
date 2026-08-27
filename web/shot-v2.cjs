// 截图:登录页(暗色+亮色+语言菜单)+ 系统状态页 + 侧边栏悬停
const puppeteer = require('puppeteer-core');
const fs = require('fs');

const sleep = (ms) => new Promise(r => setTimeout(r, ms));

async function gotoLogin(page) {
  await page.goto('http://localhost:8080/login', { waitUntil: 'load', timeout: 20000 });
  for (let i = 0; i < 20; i++) {
    const n = await page.$$('input').then(l => l.length);
    if (n >= 2) break;
    await sleep(400);
  }
  await sleep(800);
}

(async () => {
  const browser = await puppeteer.launch({
    executablePath: 'C:/Program Files/Google/Chrome/Application/chrome.exe',
    headless: 'new',
    args: ['--no-sandbox', '--disable-gpu'],
    defaultViewport: { width: 1440, height: 900 },
  });
  const page = await browser.newPage();
  page.on('pageerror', e => console.log('[pageerror]', e.message.slice(0, 200)));
  const out = 'C:/Users/zj181/AppData/Local/Temp/dm-shots';

  // 1. 登录页(暗色默认)
  await gotoLogin(page);
  await page.screenshot({ path: out + '/v2-login-dark.png' });
  console.log('1 login dark');

  // 2. 打开语言菜单
  await page.evaluate(() => {
    const btns = [...document.querySelectorAll('.toolbar-btn')];
    if (btns[1]) btns[1].click();
  });
  await sleep(400);
  await page.screenshot({ path: out + '/v2-login-langmenu.png' });
  console.log('2 lang menu');

  // 3. 切亮色
  await page.evaluate(() => {
    const btns = [...document.querySelectorAll('.toolbar-btn')];
    if (btns[0]) btns[0].click();
  });
  await sleep(500);
  await page.screenshot({ path: out + '/v2-login-light.png' });
  console.log('3 login light');

  // 4. 登录(用 evaluate 原生 setter)
  await page.evaluate(() => {
    const inputs = document.querySelectorAll('input');
    const setVal = (el, v) => {
      const s = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value').set;
      s.call(el, v);
      el.dispatchEvent(new Event('input', { bubbles: true }));
    };
    setVal(inputs[0], 'admin');
    setVal(inputs[1], '123456');
  });
  await sleep(300);
  await page.evaluate(() => {
    const btn = [...document.querySelectorAll('button')].find(b => (b.textContent || '').replace(/\s/g, '') === '登录');
    if (btn) btn.click();
  });
  await sleep(3000);
  console.log('after login:', page.url());
  await page.screenshot({ path: out + '/v2-status-page.png' });
  await sleep(3000);
  await page.screenshot({ path: out + '/v2-status-page-loaded.png' });
  console.log('4 status page');

  // 5. 侧边栏折叠态 + 悬停展开
  await page.screenshot({ path: out + '/v2-sidebar-collapsed.png' });
  await page.hover('.app-sider');
  await sleep(600);
  await page.screenshot({ path: out + '/v2-sidebar-expanded.png' });
  console.log('5 sidebar');

  // 6. 容器页
  await page.goto('http://localhost:8080/containers', { waitUntil: 'load', timeout: 20000 });
  await sleep(2500);
  await page.screenshot({ path: out + '/v2-containers.png' });
  console.log('6 containers');

  await browser.close();
  console.log('DONE');
})();
