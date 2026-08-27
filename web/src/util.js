export function formatBytes(n, digits = 1) {
  if (n == null || isNaN(n)) return '-'
  if (n === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.min(units.length - 1, Math.floor(Math.log(n) / Math.log(1024)))
  return `${(n / Math.pow(1024, i)).toFixed(digits)} ${units[i]}`
}

export function formatDate(ts) {
  if (!ts) return '-'
  const d = new Date(ts * 1000)
  const p = (x) => String(x).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

export function containerName(c) {
  if (!c || !c.Names || !c.Names.length) return c?.Id?.slice(0, 12) || '-'
  return c.Names[0].replace(/^\//, '')
}

export function shortId(id, n = 12) {
  return (id || '').slice(0, n)
}

export function humanPorts(ports) {
  if (!ports || !ports.length) return '-'
  return ports
    .map((p) => {
      const ip = p.IP || '0.0.0.0'
      if (p.PublicPort) return `${ip}:${p.PublicPort}->${p.PrivatePort}/${p.Type || 'tcp'}`
      return `${p.PrivatePort}/${p.Type || 'tcp'}`
    })
    .join(', ')
}

export function imageTag(repoTags, id) {
  if (repoTags && repoTags.length) return repoTags[0]
  return shortId(id, 19)
}

export const STATE_COLOR = {
  running: 'ok',
  exited: 'muted',
  stopped: 'muted',
  created: 'info',
  restarting: 'warn',
  paused: 'warn',
  dead: 'danger',
  removing: 'warn',
}
