const $ = (selector) => document.querySelector(selector)

let token = sessionStorage.getItem('clank-token') || ''
let current = null
let data = null

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
  element.textContent = value
  return element.innerHTML
}

function fmt(timestamp) {
  return new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(timestamp))
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
    await projects()
  } catch {
    sessionStorage.removeItem('clank-token')
    token = ''
    showApp(false)
  }
}

async function projects() {
  const response = await api('/projects')
  const items = response.projects || []
  const nav = $('#projects')
  nav.innerHTML = items.map((project) => `<button data-id="${project.id}" class="${current === project.id ? 'active' : ''}">${esc(project.name)}</button>`).join('')
  nav.querySelectorAll('button').forEach((button) => { button.onclick = () => openProject(button.dataset.id) })
  if (!current && items[0]) openProject(items[0].id)
}

async function openProject(id) {
  current = id
  data = await api('/projects/' + id)
  $('#empty').classList.add('hidden')
  $('#project-view').classList.remove('hidden')
  $('#project-slug').textContent = data.project.slug
  $('#project-name').textContent = data.project.name
  $('#project-description').textContent = data.project.description || 'No project description yet.'
  render()
  projects()
}

function actor(run) {
  if (!run) return 'Unattributed entry'
  const agent = run.agentName || 'Agent'
  return run.principalName && run.principalName !== agent ? `${run.principalName} · ${agent}` : agent
}

function runtime(run) {
  if (!run) return ''
  return [run.harness, run.provider, run.model, run.role, run.runType]
    .filter(Boolean)
    .map((value) => `<span class="pill">${esc(value)}</span>`)
    .join('')
}

function noteActor(note) {
  if (note.run) return actor(note.run)
  if (note.ledBy === 'human') return 'Human entry'
  if (note.ledBy === 'joint') return 'Joint entry'
  return 'Agent entry'
}

function render() {
  const notes = data.notes || []
  const trajectories = data.trajectories || []
  const repositories = data.repositories || []
  const currentNotes = notes.filter((note) => note.status === 'current')
  const actors = new Set(trajectories.map((trajectory) => actor(trajectory.run)))

  $('#pulse-summary').textContent = trajectories.length
    ? `${trajectories.length} active ${trajectories.length === 1 ? 'trajectory' : 'trajectories'} across ${actors.size} ${actors.size === 1 ? 'agent identity' : 'agent identities'}.`
    : 'No agent has declared material work yet.'

  $('#note-count').textContent = `${currentNotes.length} current · ${notes.length} total`
  $('#notes').innerHTML = notes.length ? notes.map(intentEntry).join('') : intentEmpty()
  $('#notes').querySelectorAll('.supersede').forEach((button) => { button.onclick = () => supersede(button.dataset.id, +button.dataset.rev) })

  $('#trajectories').innerHTML = trajectories.length ? trajectories.map(trajectoryRow).join('') : trajectoryEmpty()
  $('#agent-brief').innerHTML = agentBrief(trajectories, currentNotes)
  $('#repositories').innerHTML = repositories.length ? repositories.map(repositoryRow).join('') : repositoryEmpty()
}

function trajectoryRow(trajectory) {
  const run = trajectory.run
  return `<article class="trajectory" id="trajectory-${trajectory.id}">
    <div class="trajectory-who"><span class="avatar" aria-hidden="true">${esc((run?.agentName || 'A').slice(0, 1).toUpperCase())}</span><div><strong>${esc(actor(run))}</strong><span>${esc(run?.branch || trajectory.branch || 'Working trajectory')}</span></div></div>
    <div class="trajectory-direction"><span class="kind">Active direction</span><h3>${esc(trajectory.objective)}</h3>${trajectory.rationale ? `<p>${esc(trajectory.rationale)}</p>` : ''}</div>
    <div class="trajectory-scope">${(trajectory.paths || []).map((path) => `<code>${esc(path)}</code>`).join('') || '<span>No path scope declared</span>'}<div class="meta">${runtime(run)}</div></div>
  </article>`
}

function trajectoryEmpty() {
  return `<div class="guided-empty"><strong>No work declared yet.</strong><p>Issue a project agent key, then have the agent start a run and publish its trajectory. Direction and rationale will appear here before code collides.</p><code>clank trajectory start --objective "…" --paths "…"</code></div>`
}

function intentEntry(note) {
  const lifecycle = note.status === 'current'
    ? `<button class="supersede" data-id="${note.id}" data-rev="${note.revision}">Supersede</button>`
    : `<span class="status">${esc(note.status)}</span>`
  const source = noteActor(note)
  return `<article class="intent-entry ${note.status !== 'current' ? 'is-past' : ''}">
    <div class="intent-marker" aria-hidden="true"></div>
    <div class="intent-time"><strong>${esc(source)}</strong><span>${fmt(note.updatedAt)}</span>${runtime(note.run)}</div>
    <div class="intent-body">${lifecycle}<span class="kind">${esc(note.kind)} · ${esc(note.directionBasis.replaceAll('_', ' '))}</span><h3>${esc(note.title)}</h3><p>${esc(note.summary)}</p>${note.rationale ? `<blockquote><strong>Why</strong>${esc(note.rationale)}</blockquote>` : ''}<div class="path-list">${(note.paths || []).map((path) => `<code>${esc(path)}</code>`).join('')}</div></div>
  </article>`
}

function intentEmpty() {
  return `<div class="guided-empty"><strong>No intent has been carried forward.</strong><p>This should stay empty until a choice, constraint, reversal, or rationale could genuinely change another collaborator's work.</p></div>`
}

function agentBrief(trajectories, notes) {
  if (!trajectories.length && !notes.length) {
    return `<p class="brief-lead">The next agent currently receives a quiet board.</p><ol class="onboarding-loop"><li><span>1</span><div><strong>Declare work</strong><p>An agent publishes its objective, paths, and reason.</p></div></li><li><span>2</span><div><strong>Accrue intent</strong><p>Only material choices and corrections survive the session.</p></div></li><li><span>3</span><div><strong>Catch divergence</strong><p>The next agent checks its proposed move before changing code.</p></div></li></ol>`
  }
  const directions = trajectories.slice(0, 3).map((trajectory) => `<li><span>Direction</span><strong>${esc(trajectory.objective)}</strong><small>${esc(actor(trajectory.run))}</small></li>`).join('')
  const context = notes.slice(0, 3).map((note) => `<li><span>${esc(note.kind)}</span><strong>${esc(note.title)}</strong><small>${esc(noteActor(note))}</small></li>`).join('')
  return `<p class="brief-lead">A bounded brief, not a rulebook. The agent sees current direction beside the reason it exists.</p><ul class="brief-items">${directions}${context}</ul><p class="brief-notice">Current human direction and repository evidence still win.</p>`
}

function repositoryRow(repository) {
  return `<article class="repository-row"><div><a href="${repository.url}" target="_blank" rel="noreferrer">${esc(repository.owner + '/' + repository.name)} ↗</a><p>${esc(repository.description || 'Public GitHub repository')}</p></div><div class="pulls">${(repository.pullRequests || []).map((pull) => `<a href="${pull.url}" target="_blank" rel="noreferrer"><span>#${pull.externalId}</span>${esc(pull.title)}</a>`).join('') || '<span>No open pull requests</span>'}</div></article>`
}

function repositoryEmpty() {
  return `<div class="guided-empty compact"><strong>No repository evidence attached.</strong><p>Link a public GitHub repository to keep open pull requests beside the intent they support.</p></div>`
}

function modal(title, fields, onSave) {
  $('#dialog-fields').innerHTML = `<h2>${esc(title)}</h2>` + fields
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

$('#login-form').onsubmit = async (event) => {
  event.preventDefault()
  token = $('#token').value.trim()
  sessionStorage.setItem('clank-token', token)
  boot()
}

$('#logout').onclick = () => { sessionStorage.clear(); location.reload() }

$('#new-project').onclick = () => modal('New project', '<label>Name<input name="name" required></label><label>Slug<input name="slug" pattern="[a-z0-9]+(?:-[a-z0-9]+)*" required></label><label>Description<textarea name="description"></textarea></label>', async (form) => {
  await api('/projects', { method: 'POST', body: JSON.stringify(Object.fromEntries(form)) })
  await projects()
  toast('Project created')
})

$('#add-note').onclick = () => modal('Add project context', '<p>Record only what could change another collaborator\'s next move.</p><label>Kind<select name="kind"><option>intent</option><option>decision</option><option>understanding</option><option>observation</option><option>checkpoint</option></select></label><label>Title<input name="title" maxlength="180" required></label><label>Project implication<textarea name="summary" maxlength="1200" required></textarea></label><label>Why it matters<textarea name="rationale" maxlength="2400"></textarea></label><label>Led by<select name="ledBy"><option>human</option><option>joint</option><option>agent</option><option>external</option></select></label><label>Direction basis<select name="directionBasis"><option>explicit_human_direction</option><option>interpreted_human_intent</option><option>joint_reasoning</option><option>autonomous_agent_judgment</option><option>external_evidence</option></select></label><label>Paths<input name="paths" placeholder="Comma-separated"></label>', async (form) => {
  const input = Object.fromEntries(form)
  input.paths = input.paths.split(',').map((path) => path.trim()).filter(Boolean)
  await api(`/projects/${current}/notes`, { method: 'POST', body: JSON.stringify(input) })
  await openProject(current)
  toast('Context added')
})

async function supersede(id, revision) {
  const reason = prompt('Why is this context no longer current?')
  if (!reason) return
  await api(`/projects/${current}/notes/${id}/supersede`, { method: 'POST', body: JSON.stringify({ expectedRevision: revision, reason }) })
  await openProject(current)
  toast('Context superseded')
}

$('#brief-form').onsubmit = async (event) => {
  event.preventDefault()
  const objective = $('#brief-objective').value.trim()
  const paths = $('#brief-paths').value.split(',').map((path) => path.trim()).filter(Boolean)
  if (!objective && !paths.length) return toast('Describe the proposed move or add a path first')
  const brief = await api(`/projects/${current}/brief`, { method: 'POST', body: JSON.stringify({ objective, paths, checkConflicts: true }) })
  renderWarnings(brief.coordinationWarnings || [], objective, paths)
}

function renderWarnings(warnings, objective, paths) {
  const container = $('#warnings')
  if (!warnings.length) {
    container.innerHTML = `<div class="clear-result"><span aria-hidden="true">✓</span><div><strong>No intersecting trajectory found.</strong><p>This is a context check, not proof that the change is safe.</p></div></div>`
  } else {
    container.innerHTML = warnings.map((warning) => collisionView(warning, objective, paths)).join('')
    container.querySelectorAll('[data-action="continue"]').forEach((button) => { button.onclick = () => { button.closest('.collision').remove(); toast('Context acknowledged; no project state changed') } })
    container.querySelectorAll('[data-action="inspect"]').forEach((button) => { button.onclick = () => document.querySelector(`#trajectory-${button.dataset.trajectory}`)?.scrollIntoView({ behavior: 'smooth', block: 'center' }) })
    container.querySelectorAll('[data-action="realign"]').forEach((button) => { button.onclick = () => { $('#brief-objective').focus(); toast('Adjust the proposed move, then check again') } })
  }
  container.scrollIntoView({ behavior: 'smooth', block: 'center' })
}

function collisionView(warning, objective, paths) {
  const trajectory = warning.trajectory
  return `<article class="collision">
    <div class="collision-title"><span>Possible divergence</span><strong>Two directions touch the same work.</strong></div>
    <div class="direction-compare">
      <div><span>Your proposed move</span><h3>${esc(objective || paths.join(', '))}</h3><div class="path-list">${paths.map((path) => `<code>${esc(path)}</code>`).join('')}</div></div>
      <div class="intersection"><span aria-hidden="true">↔</span><strong>${esc(warning.reason)}</strong></div>
      <div><span>${esc(actor(trajectory?.run))} is moving toward</span><h3>${esc(trajectory?.objective || 'Related work')}</h3>${trajectory?.rationale ? `<p>${esc(trajectory.rationale)}</p>` : ''}</div>
    </div>
    <div class="resolution"><p>This context does not decide for you.</p><div><button class="quiet" data-action="continue">Continue anyway</button><button class="quiet" data-action="inspect" data-trajectory="${trajectory?.id || ''}">Inspect rationale</button><button data-action="realign">Realign the move</button></div></div>
  </article>`
}

$('#attach-repo').onclick = () => modal('Attach public GitHub repository', '<label>Repository URL<input name="url" placeholder="https://github.com/owner/repo" required></label>', async (form) => {
  await api(`/projects/${current}/repositories`, { method: 'POST', body: JSON.stringify(Object.fromEntries(form)) })
  await openProject(current)
  toast('Repository attached')
})

$('#agent-key').onclick = () => modal('Issue a project agent key', '<p>This identity represents agents working for this project. The secret is shown once.</p><label>Identity name<input name="displayName" placeholder="shuv2code agents"></label>', async (form) => {
  const credential = await api(`/projects/${current}/tokens`, { method: 'POST', body: JSON.stringify(Object.fromEntries(form)) })
  setTimeout(() => modal('Copy this key now', `<p>${esc(credential.notice)}</p><label>Project agent token<input value="${esc(credential.token)}" readonly></label>`, async () => {}), 80)
})

boot()
