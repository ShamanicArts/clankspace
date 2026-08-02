const $ = (selector) => document.querySelector(selector)

let token = sessionStorage.getItem('clank-token') || ''
let current = null
let data = null
let projectList = []

const key = () => crypto.randomUUID()

async function api(path, options = {}) {
  const response = await fetch('/api/v1' + path, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      'Authorization': 'Bearer ' + token,
      ...(options.method && options.method !== 'GET' ? { 'Idempotency-Key': key() } : {}),
      ...options.headers,
    },
  })
  const body = await response.json().catch(() => ({}))
  if (!response.ok) throw new Error(body.error || response.statusText)
  return body
}

function toast(message) {
  const element = $('#toast')
  element.textContent = message
  element.classList.add('show')
  setTimeout(() => element.classList.remove('show'), 2400)
}

function esc(value = '') {
  const element = document.createElement('div')
  element.textContent = String(value)
  return element.innerHTML
}

function safeURL(value) {
  try {
    const url = new URL(value)
    return ['http:', 'https:'].includes(url.protocol) ? esc(url.href) : ''
  } catch {
    return ''
  }
}

function formatDate(timestamp) {
  const date = new Date(timestamp)
  const today = new Date()
  if (date.toDateString() === today.toDateString()) return 'Today'
  return new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric', year: date.getFullYear() === today.getFullYear() ? undefined : 'numeric' }).format(date)
}

function formatTime(timestamp) {
  return new Intl.DateTimeFormat(undefined, { hour: 'numeric', minute: '2-digit' }).format(new Date(timestamp))
}

function showApp(show) {
  $('#login').classList.toggle('hidden', show)
  $('#app').classList.toggle('hidden', !show)
  $('#logout').classList.toggle('hidden', !show)
}

async function boot() {
  if (!token) return showApp(false)
  try {
    await api('/whoami')
    showApp(true)
    await loadProjects()
  } catch (error) {
    sessionStorage.removeItem('clank-token')
    token = ''
    showApp(false)
    toast(error.message)
  }
}

async function loadProjects() {
  const response = await api('/projects')
  projectList = response.projects || []
  renderProjectNav()
  if (!current && projectList[0]) await openProject(projectList[0].id)
}

function renderProjectNav() {
  const nav = $('#projects')
  nav.innerHTML = projectList.map((project) => `<button data-id="${esc(project.id)}" class="${current === project.id ? 'active' : ''}">${esc(project.name)}</button>`).join('')
  nav.querySelectorAll('button').forEach((button) => { button.onclick = () => openProject(button.dataset.id) })
}

async function openProject(id) {
  current = id
  data = await api('/projects/' + id)
  $('#empty').classList.add('hidden')
  $('#project-view').classList.remove('hidden')
  const slug = $('#project-slug')
  const comparableName = data.project.name.toLowerCase().trim().replace(/\s+/g, '-')
  slug.textContent = data.project.slug
  slug.classList.toggle('hidden', comparableName === data.project.slug)
  $('#project-name').textContent = data.project.name
  $('#project-description').textContent = data.project.description || 'No project description.'
  renderProjectNav()
  renderRepositories()
  renderLog()
}

function actor(run, fallback = 'Unattributed entry') {
  if (!run) return fallback
  const agent = run.agentName || 'Agent'
  return run.principalName && run.principalName !== agent ? `${run.principalName} · ${agent}` : agent
}

function noteActor(note) {
  if (note.run) return actor(note.run)
  if (note.ledBy === 'human') return 'Human entry'
  if (note.ledBy === 'joint') return 'Human and agent'
  if (note.ledBy === 'external') return 'External evidence'
  return 'Agent entry'
}

function runtimeText(run) {
  if (!run) return ''
  return [run.harness, run.harnessVersion, run.provider, run.model, run.reasoning, run.role, run.runType]
    .filter(Boolean)
    .join(' · ')
}

function normalizedEntries() {
  const notes = (data.notes || []).map((note) => ({
    type: 'note',
    kind: note.kind,
    status: note.status,
    timestamp: note.updatedAt,
    item: note,
    searchable: [note.kind, note.status, note.title, note.summary, note.rationale, noteActor(note), runtimeText(note.run), note.directionBasis, note.verification, note.sourceRef, note.pullRequestUrl, ...(note.paths || [])].join(' ').toLowerCase(),
  }))
  const trajectories = (data.trajectories || []).map((trajectory) => ({
    type: 'trajectory',
    kind: 'trajectory',
    status: trajectory.status,
    timestamp: trajectory.updatedAt,
    item: trajectory,
    searchable: ['work started', trajectory.status, trajectory.objective, trajectory.rationale, actor(trajectory.run), runtimeText(trajectory.run), trajectory.branch, ...(trajectory.paths || [])].join(' ').toLowerCase(),
  }))
  return [...notes, ...trajectories].sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp))
}

function renderLog() {
  if (!data) return
  const query = $('#log-search').value.trim().toLowerCase()
  const kind = $('#kind-filter').value
  const status = $('#status-filter').value
  const all = normalizedEntries()
  const visible = all.filter((entry) => {
    const kindMatches = kind === 'all' || entry.kind === kind
    const isCurrent = entry.type === 'trajectory' ? entry.status === 'active' : entry.status === 'current'
    const statusMatches = status === 'all' || (status === 'current' ? isCurrent : !isCurrent)
    return kindMatches && statusMatches && (!query || entry.searchable.includes(query))
  })

  $('#log-count').textContent = visible.length === all.length
    ? `${all.length} ${all.length === 1 ? 'entry' : 'entries'}`
    : `${visible.length} of ${all.length}`
  $('#log').innerHTML = visible.length ? visible.map(logEntry).join('') : logEmpty(all.length)
  $('#log').querySelectorAll('.supersede').forEach((button) => { button.onclick = () => supersede(button.dataset.id, Number(button.dataset.rev)) })
  $('#reset-filters')?.addEventListener('click', resetFilters)
}

function logEntry(entry) {
  const item = entry.item
  const historical = entry.type === 'trajectory' ? item.status !== 'active' : item.status !== 'current'
  const kind = entry.type === 'trajectory' ? 'work started' : item.kind
  const title = entry.type === 'trajectory' ? item.objective : item.title
  const summary = entry.type === 'trajectory' ? '' : item.summary
  const rationale = item.rationale || ''
  const run = item.run
  const who = entry.type === 'trajectory' ? actor(run) : noteActor(item)
  const basis = entry.type === 'note' && item.directionBasis ? item.directionBasis.replaceAll('_', ' ') : ''
  const lifecycle = entry.type === 'trajectory' ? item.status : item.status
  const branch = run?.branch || item.branch || ''
  const runtime = runtimeText(run)
  const paths = (item.paths || []).map((path) => `<code>${esc(path)}</code>`).join('')
  const pullURL = safeURL(item.pullRequestUrl)
  const sourceURL = safeURL(item.sourceRef)
  const evidence = [
    pullURL ? `<a href="${pullURL}" target="_blank" rel="noreferrer">Pull request ↗</a>` : '',
    sourceURL ? `<a href="${sourceURL}" target="_blank" rel="noreferrer">Source ↗</a>` : item.sourceRef ? `<span>${esc(item.sourceRef)}</span>` : '',
  ].filter(Boolean).join('')
  const supersedeAction = entry.type === 'note' && item.status === 'current'
    ? `<button class="supersede" data-id="${esc(item.id)}" data-rev="${item.revision}">Supersede</button>`
    : ''
  const meta = [who, runtime, branch].filter(Boolean).map(esc).join(' · ')

  return `<article class="log-entry ${historical ? 'historical' : ''}">
    <time datetime="${esc(entry.timestamp)}"><strong>${esc(formatDate(entry.timestamp))}</strong><span>${esc(formatTime(entry.timestamp))}</span></time>
    <div class="entry-body">
      <div class="entry-labels"><span class="entry-kind">${esc(kind)}</span><span class="entry-status">${esc(lifecycle)}</span>${supersedeAction}</div>
      <h3>${esc(title)}</h3>
      ${summary ? `<p class="entry-summary">${esc(summary)}</p>` : ''}
      ${rationale ? `<p class="entry-rationale"><strong>Why:</strong> ${esc(rationale)}</p>` : ''}
      <div class="entry-provenance"><span class="actor">${meta}</span>${basis ? `<span>${esc(basis)}</span>` : ''}${item.verification ? `<span>${esc(item.verification)}</span>` : ''}</div>
      ${paths || evidence ? `<div class="entry-evidence"><div class="entry-paths">${paths}</div>${evidence}</div>` : ''}
    </div>
  </article>`
}

function logEmpty(total) {
  if (total) return `<div class="log-empty"><strong>No entries match.</strong><p>Try a different search or filter.</p><button id="reset-filters" class="text-button">Clear filters</button></div>`
  return `<div class="log-empty"><strong>No material context recorded yet.</strong><p>Agents append entries through the ClankSpace skill, CLI, or MCP tools when context could change another collaborator's work.</p><code>clank context</code></div>`
}

function resetFilters() {
  $('#log-search').value = ''
  $('#kind-filter').value = 'all'
  $('#status-filter').value = 'all'
  renderLog()
}

function renderRepositories() {
  const repositories = data.repositories || []
  const container = $('#repository-context')
  if (!repositories.length) {
    container.innerHTML = '<span>No repository connected</span>'
    return
  }
  container.innerHTML = repositories.map((repository) => {
    const count = (repository.pullRequests || []).length
    return `<span><a href="${esc(repository.url)}" target="_blank" rel="noreferrer">${esc(repository.owner + '/' + repository.name)} ↗</a>${count ? ` · ${count} open ${count === 1 ? 'PR' : 'PRs'}` : ''}</span>`
  }).join('')
}

function closeActionMenu() {
  $('.action-menu')?.removeAttribute('open')
}

function modal(title, fields, onSave, saveLabel = 'Save') {
  $('#dialog-fields').innerHTML = `<h2>${esc(title)}</h2>` + fields
  $('#dialog-submit').textContent = saveLabel
  const dialog = $('#dialog')
  dialog.showModal()
  $('#dialog-form').onsubmit = async (event) => {
    event.preventDefault()
    try {
      await onSave(new FormData(event.target))
      dialog.close()
    } catch (error) {
      toast(error.message)
    }
  }
}

$('#dialog-cancel').onclick = () => $('#dialog').close()

$('#login-form').onsubmit = async (event) => {
  event.preventDefault()
  token = $('#token').value.trim()
  sessionStorage.setItem('clank-token', token)
  await boot()
}

$('#logout').onclick = () => { sessionStorage.clear(); location.reload() }

$('#new-project').onclick = () => modal('New project', '<label>Name<input name="name" required></label><label>Slug<input name="slug" pattern="[a-z0-9]+(?:-[a-z0-9]+)*" required></label><label>Description<textarea name="description"></textarea></label>', async (form) => {
  const response = await api('/projects', { method: 'POST', body: JSON.stringify(Object.fromEntries(form)) })
  await loadProjects()
  await openProject(response.project.id)
  toast('Project created')
}, 'Create project')

$('#add-note').onclick = () => modal('Add log entry', '<p>Record only context that could change another collaborator\'s next move.</p><label>Kind<select name="kind"><option>intent</option><option>decision</option><option>understanding</option><option>observation</option><option>checkpoint</option></select></label><label>Title<input name="title" maxlength="180" required></label><label>Project implication<textarea name="summary" maxlength="1200" required></textarea></label><label>Why it matters<textarea name="rationale" maxlength="2400"></textarea></label><label>Led by<select name="ledBy"><option>human</option><option>joint</option><option>agent</option><option>external</option></select></label><label>Direction basis<select name="directionBasis"><option>explicit_human_direction</option><option>interpreted_human_intent</option><option>joint_reasoning</option><option>autonomous_agent_judgment</option><option>external_evidence</option></select></label><label>Paths<input name="paths" placeholder="Comma-separated"></label>', async (form) => {
  const input = Object.fromEntries(form)
  input.paths = input.paths.split(',').map((path) => path.trim()).filter(Boolean)
  await api(`/projects/${current}/notes`, { method: 'POST', body: JSON.stringify(input) })
  await openProject(current)
  toast('Entry appended')
}, 'Append entry')

async function supersede(id, revision) {
  modal('Supersede entry', '<p>The original entry remains in the log as historical context.</p><label>What changed?<textarea name="reason" maxlength="1200" required autofocus></textarea></label>', async (form) => {
    await api(`/projects/${current}/notes/${id}/supersede`, { method: 'POST', body: JSON.stringify({ expectedRevision: revision, reason: form.get('reason') }) })
    await openProject(current)
    toast('Entry superseded')
  }, 'Supersede')
}

$('#attach-repo').onclick = () => {
  closeActionMenu()
  modal('Connect GitHub repository', '<p>Public repositories are read-only evidence. Their content does not become project intent.</p><label>Repository URL<input name="url" type="url" placeholder="https://github.com/owner/repo" required></label>', async (form) => {
    await api(`/projects/${current}/repositories`, { method: 'POST', body: JSON.stringify(Object.fromEntries(form)) })
    await openProject(current)
    toast('Repository connected')
  }, 'Connect')
}

$('#agent-key').onclick = () => {
  closeActionMenu()
  modal('Issue project agent key', '<p>This identity represents agents working on this project. The secret is shown once.</p><label>Identity name<input name="displayName" placeholder="Shamanic agents" required></label>', async (form) => {
    const credential = await api(`/projects/${current}/tokens`, { method: 'POST', body: JSON.stringify(Object.fromEntries(form)) })
    setTimeout(() => modal('Copy this key now', `<p>${esc(credential.notice)}</p><label>Project agent token<input value="${esc(credential.token)}" readonly></label>`, async () => {}, 'Done'), 80)
  }, 'Issue key')
}

$('#export-project').onclick = async () => {
  closeActionMenu()
  try {
    const response = await fetch(`/api/v1/projects/${current}/export`, { headers: { 'Authorization': 'Bearer ' + token } })
    if (!response.ok) throw new Error('Export failed')
    const blob = await response.blob()
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `${data.project.slug}.clankspace.json`
    link.click()
    URL.revokeObjectURL(url)
    toast('Project log exported')
  } catch (error) {
    toast(error.message)
  }
}

$('#refresh-project').onclick = async () => {
  await openProject(current)
  toast('Log refreshed')
}

$('#log-search').addEventListener('input', renderLog)
$('#kind-filter').addEventListener('change', renderLog)
$('#status-filter').addEventListener('change', renderLog)

boot()
