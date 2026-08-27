// 最终截图:登录页(暗/亮/语言菜单)+ 系统状态 + 侧边栏 + 容器页
const puppeteer = require('puppeteer-core');
const fs = require('fs');
const sleep = (ms) => new Promise(r => setTimeout(r, ms));

(async () => {
  const browser = await puppeteer.launch({
    executablePath: 'C:/Program Files/Google/Chrome/Application/chrome.exe',
    headless: 'new',
    args: ['--no-sandbox', '--disable-gpu'],
    defaultViewport: { width: 1440, height: 900 },
  });
  const page = await browser.newPage();
  const out = 'C:/Users/zj181/AppData/Local/Temp/dm-shots';

  // 登录页(暗色)
  await page.goto('http://localhost:8080/login', { waitUntil: 'load', timeout: 20000 });
  for (let i = 0; i < 20; i++) { const n = await page.$$('input').then(l => l.length); if (n >= 2) break; await sleep(400); }
  await sleep(1000);
  await page.screenshot({ path: out + '/v3-login-dark.png' });

  // 语言菜单
  await page.evaluate(() => { const b = [...document.querySelectorAll('.toolbar-btn')][1]; if (b) b.click(); });
  await sleep(400);
  await page.screenshot({ path: out + '/v3-login-langmenu.png' });

  // 切亮色
  await page.evaluate(() => { const b = [...document.querySelectorAll('.toolbar-btn')][0]; if (b) b.click(); });
  await sleep(600);
  await page.screenshot({ path: out + '/v3-login-light.png' });

  // 登录
  await page.evaluate(() => {
    const inputs = document.querySelectorAll('input');
    const setVal = (el, v) => { const s = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value').set; s.call(el, v); el.dispatchEvent(new Event('input', { bubbles: true })); };
    setVal(inputs[0], 'admin'); setVal(inputs[1], '123456');
  });
  await sleep(300);
  await page.evaluate(() => { const b = [...document.querySelectorAll('button')].find(x => (x.textContent || '').replace(/\s/g, '') === '登录'); if (b) b.click(); });
  await sleep(4000);
  await page.screenshot({ path: out + '/v3-status.png' });

  // 侧边栏悬停展开
  await page.hover('.app-sider');
  await sleep(700);
  await page.screenshot({ path: out + '/v3-sidebar-expanded.png' });
  await page.hover('.app-main');
  await sleep(500);

  // 容器页(折叠侧边栏)
  await page.goto('http://localhost:8080/containers', { waitUntil: 'load', timeout: 20000 });
  await sleep(2500);
  await page.screenshot({ path: out + '/v3-containers.png' });

  await browser.close();
  console.log('DONE');
})();
