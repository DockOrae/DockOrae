// 校验 i18n key 完整性:源码用到的 key 必须存在于 5 个语言包
const fs = require('fs')
const path = require('path')
const SRC = path.join(__dirname, 'src')

function collectKeys(text) {
  const keys = new Set()
  const re = /t\(\s*['"]([^'"]+)['"]/g
  let m
  while ((m = re.exec(text))) keys.add(m[1])
  return keys
}

function flatten(obj, prefix = '') {
  const out = {}
  for (const [k, v] of Object.entries(obj)) {
    const key = prefix ? `${prefix}.${k}` : k
    if (v && typeof v === 'object') Object.assign(out, flatten(v, key))
    else out[key] = v
  }
  return out
}

// 收集源码使用的 key
const used = new Set()
function walk(dir) {
  for (const f of fs.readdirSync(dir)) {
    const p = path.join(dir, f)
    if (f === 'locales' || f === 'node_modules' || f === 'dist' || f === '.git') continue
    if (fs.statSync(p).isDirectory()) walk(p)
    else if (f.endsWith('.vue') || f.endsWith('.js')) used.add(...collectKeys(fs.readFileSync(p, 'utf8')))
  }
}
walk(SRC)

// 加载语言包
const langs = {}
for (const f of fs.readdirSync(path.join(SRC, 'locales'))) {
  const code = f.replace(/\.js$/, '')
  const mod = require(path.join(SRC, 'locales', f)).default
  langs[code] = flatten(mod)
}

console.log('使用中的 key 总数:', used.size)
const zh = langs['zh-CN']
const missingZh = [...used].filter((k) => !(k in zh)).sort()
console.log('zh-CN 缺失:', missingZh.length ? missingZh : '无 ✅')

for (const [code, d] of Object.entries(langs)) {
  const missing = Object.keys(zh).filter((k) => !(k in d))
  const extra = Object.keys(d).filter((k) => !(k in zh))
  console.log(`${code}: 缺失 ${missing.length} ${missing.length ? missing : '✅'} | 多余 ${extra.length} ${extra.length ? extra : '✅'}`)
}

// 残留中文(非注释、非属性)
console.log('\n残留中文检查:')
let found = false
function walk2(dir) {
  for (const f of fs.readdirSync(dir)) {
    const p = path.join(dir, f)
    if (f === 'locales' || f === 'node_modules' || f === 'dist' || f === '.git') continue
    if (fs.statSync(p).isDirectory()) walk2(p)
    else if (f.endsWith('.vue') || f.endsWith('.js')) {
      const lines = fs.readFileSync(p, 'utf8').split('\n')
      lines.forEach((line, i) => {
        if (/[\u4e00-\u9fff]/.test(line)) {
          const s = line.trim()
          if (s.startsWith('//') || s.startsWith('/*') || s.startsWith('*') || s.startsWith('<!--')) return
          found = true
          console.log(`  ${path.relative(SRC, p)}:${i + 1}: ${s.slice(0, 90)}`)
        }
      })
    }
  }
}
walk2(SRC)
if (!found) console.log('  无 ✅')
