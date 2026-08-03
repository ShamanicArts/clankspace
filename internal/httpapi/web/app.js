const $ = (selector) => document.querySelector(selector)

let authMode = 'bootstrap'
let accessMode = 'session'
let bootstrapToken = sessionStorage.getItem('clank-token') || ''
let account = null
let workspaceList = []
let currentWorkspace = null
let projectList = []
let currentProject = null
let projectData = null
let members = []
let invites = []
let replicas = []
let canShareHumans = true
let canManageReplicas = false

const requestKey = () => crypto.randomUUID()
const csrfToken = () => document.cookie.split('; ').find((item) => item.startsWith('clank_csrf='))?.split('=').slice(1).join('=') || ''

async function request(path, options = {}, mode = accessMode) {
  const method = options.method || 'GET'
  const headers = { 'Content-Type': 'application/json', ...options.headers }
  if (method !== 'GET' && method !== 'HEAD') headers['Idempotency-Key'] = requestKey()
  if (mode === 'legacy') headers.Authorization = 'Bearer ' + bootstrapToken
  if (mode === 'session' && method !== 'GET' && method !== 'HEAD') headers['X-CSRF-Token'] = csrfToken()
  const response = await fetch('/api/v1' + path, { credentials: 'same-origin', ...options, method, headers })
  const body = await response.json().catch(() => ({}))
  if (!response.ok) {
    const error = new Error(body.error || response.statusText)
    error.status = response.status
    throw error
  }
  return body
}

function esc(value = '') {
  const element = document.createElement('div')
  element.textContent = String(value)
  return element.innerHTML
}

function safeURL(value) {
  try {
    const parsed = new URL(value)
    return ['http:', 'https:'].includes(parsed.protocol) ? esc(parsed.href) : ''
  } catch { return '' }
}

function toast(message) {
  const element = $('#toast')
  element.textContent = message
  element.classList.add('show')
  setTimeout(() => element.classList.remove('show'), 2600)
}

function showSignedIn(show) {
  $('#auth').classList.toggle('hidden', show)
  $('#app').classList.toggle('hidden', !show)
  $('#logout').classList.toggle('hidden', !show)
}

function showView(id) {
  document.querySelectorAll('.content-view').forEach((view) => view.classList.add('hidden'))
  $('#' + id).classList.remove('hidden')
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

async function boot() {
  const meta = await request('/meta', {}, 'public').catch(() => ({ authMode: 'bootstrap' }))
  authMode = meta.authMode || 'bootstrap'
  $('#email-form').classList.toggle('hidden', authMode === 'bootstrap')
  $('#auth-request > p').classList.toggle('hidden', authMode === 'bootstrap')
  const authToken = new URLSearchParams(location.search).get('token')
  if (authToken) {
    const returning = authToken.startsWith('login_')
    $('#consume-title').textContent = returning ? 'Sign in to ClankSpace' : 'Accept invitation'
    $('#consume-copy').textContent = returning ? 'This one-time link will open your account.' : 'Confirm the name collaborators should see beside your project context.'
    $('#display-name-label').classList.toggle('hidden', returning)
    $('#auth-request').classList.add('hidden')
    $('#auth-consume').classList.remove('hidden')
    showSignedIn(false)
    return
  }
  try {
    const status = await request('/session-status', {}, 'public')
    if (!status.authenticated) throw new Error('not signed in')
    account = status
    accessMode = 'session'
    await openAccount()
    return
  } catch {}
  if (bootstrapToken) {
    try {
      const identity = await request('/whoami', {}, 'legacy')
      accessMode = 'legacy'
      account = { user: { displayName: identity.principal.displayName }, workspaces: [] }
      await openLegacy()
      return
    } catch {
      sessionStorage.removeItem('clank-token')
      bootstrapToken = ''
    }
  }
  showSignedIn(false)
}

async function openAccount() {
  showSignedIn(true)
  $('#account-name').textContent = account.user.displayName
  workspaceList = account.workspaces || []
  currentWorkspace = null
  currentProject = null
  projectList = []
  renderWorkspaceNav()
  renderWorkspaceHome()
  $('#project-nav-wrap').classList.add('hidden')
  $('#legacy-claim').classList.add('hidden')
  showView('home-view')
}

async function openLegacy() {
  showSignedIn(true)
  $('#account-name').textContent = account.user.displayName + ' · token access'
  projectList = (await request('/projects', {}, 'legacy')).projects || []
  workspaceList = [{ id: 'legacy', name: 'Self-hosted workspace', role: 'token' }]
  currentWorkspace = workspaceList[0]
  renderWorkspaceNav()
  renderProjectNav()
  $('#project-nav-wrap').classList.remove('hidden')
  $('#legacy-claim').classList.toggle('hidden', authMode === 'bootstrap')
  if (projectList[0]) await openProject(projectList[0].id)
  else showView('workspace-view')
}

function renderWorkspaceNav() {
  $('#workspaces').innerHTML = workspaceList.map((workspace) => `<button data-workspace="${esc(workspace.id)}" class="${currentWorkspace?.id === workspace.id ? 'active' : ''}"><span>${esc(workspace.name)}</span><small>${esc(workspace.role || '')}</small></button>`).join('') || '<span class="rail-empty">No workspaces</span>'
  document.querySelectorAll('[data-workspace]').forEach((button) => { button.onclick = () => selectWorkspace(button.dataset.workspace) })
}

function renderWorkspaceHome() {
  $('#workspace-list').innerHTML = workspaceList.length ? workspaceList.map((workspace) => `<button class="list-row workspace-row" data-home-workspace="${esc(workspace.id)}"><span><strong>${esc(workspace.name)}</strong><small>${esc(workspace.slug || workspace.id)}</small></span><span class="row-state">${esc(workspace.role || '')} →</span></button>`).join('') : '<div class="empty-row"><strong>No workspaces yet.</strong><span>Create one for a project or collaboration.</span></div>'
  document.querySelectorAll('[data-home-workspace]').forEach((button) => { button.onclick = () => selectWorkspace(button.dataset.homeWorkspace) })
}

async function selectWorkspace(id) {
  if (accessMode === 'legacy') return
  currentWorkspace = workspaceList.find((workspace) => workspace.id === id)
  currentProject = null
  projectData = null
  const base = `/account/workspaces/${id}`
  const [projectsResponse, membersResponse, replicasResponse] = await Promise.all([request(base + '/projects'), request(base + '/members'), request(base + '/replicas').catch(() => ({ replicas: [] }))])
  projectList = projectsResponse.projects || []
  members = membersResponse.members || []
  canShareHumans = membersResponse.canShareHumans !== false
  replicas = replicasResponse.replicas || []
	canManageReplicas = replicasResponse.isAuthority === true
  invites = currentWorkspace.role === 'owner' ? ((await request(base + '/invites')).invites || []) : []
  renderWorkspaceNav()
  renderProjectNav()
  renderWorkspace()
  $('#project-nav-wrap').classList.remove('hidden')
  showView('workspace-view')
}

function renderProjectNav() {
  $('#projects').innerHTML = projectList.map((project) => `<button data-project="${esc(project.id)}" class="${currentProject?.id === project.id ? 'active' : ''}">${esc(project.name)}</button>`).join('') || '<span class="rail-empty">No projects</span>'
  document.querySelectorAll('[data-project]').forEach((button) => { button.onclick = () => openProject(button.dataset.project) })
}

function renderWorkspace() {
  $('#workspace-name').textContent = currentWorkspace.name
  $('#workspace-role').textContent = currentWorkspace.role + ' · workspace'
  $('#workspace-project-count').textContent = `${projectList.length} ${projectList.length === 1 ? 'project' : 'projects'}`
  $('#workspace-project-list').innerHTML = projectList.length ? projectList.map((project) => `<button class="list-row" data-list-project="${esc(project.id)}"><span><strong>${esc(project.name)}</strong><small>${esc(project.description || 'No description')}</small></span><span class="row-state">Open log →</span></button>`).join('') : '<div class="empty-row"><strong>No projects yet.</strong><span>Create one, then point agents at it.</span></div>'
  document.querySelectorAll('[data-list-project]').forEach((button) => { button.onclick = () => openProject(button.dataset.listProject) })
  renderMembers()
  renderReplicas()
}

function renderMembers() {
  $('#member-list').innerHTML = members.map((membership) => `<div class="list-row"><span><strong>${esc(membership.user?.displayName || 'Member')}</strong><small>${esc(membership.user?.email || '')}</small></span><span class="member-controls"><span class="row-state">${esc(membership.role)} · ${esc(membership.status)}</span>${currentWorkspace.role === 'owner' ? `<button class="text-button member-edit" data-membership="${esc(membership.id)}" data-role="${esc(membership.role)}" data-status="${esc(membership.status)}">Edit</button>` : ''}</span></div>`).join('')
  $('#invite-list').innerHTML = invites.length ? invites.map((invite) => `<div class="list-row"><span><strong>${esc(invite.email)}</strong><small>${esc(invite.role)} · expires ${esc(formatDate(invite.expiresAt))}</small></span><span>${!invite.acceptedAt && !invite.revokedAt ? `<button class="text-button revoke-invite" data-invite="${esc(invite.id)}">Revoke</button>` : `<span class="row-state">${invite.acceptedAt ? 'accepted' : 'revoked'}</span>`}</span></div>`).join('') : '<div class="empty-row">No pending invitations.</div>'
  document.querySelectorAll('.member-edit').forEach((button) => { button.onclick = () => editMember(button) })
  document.querySelectorAll('.revoke-invite').forEach((button) => { button.onclick = () => revokeInvite(button.dataset.invite) })
  $('#invite-person').classList.toggle('hidden', currentWorkspace.role !== 'owner' || !canShareHumans)
}

function renderReplicas() {
  $('#replica-list').innerHTML = replicas.length ? replicas.map((replica) => `<div class="list-row"><span><strong>${esc(replica.displayName)}</strong><small>${esc(replica.baseUrl || 'Private endpoint')} · ${esc(replica.capabilities.join(' + ') || 'read only')}</small></span><span class="member-controls"><span class="row-state">${esc(replica.role)} · ${esc(replica.status)}${replica.lastSuccessAt ? ` · ${esc(formatDate(replica.lastSuccessAt))}` : ''}</span>${canManageReplicas && currentWorkspace.role === 'owner' && replica.role !== 'authority' && replica.status === 'active' ? `<button class="text-button revoke-replica" data-replica="${esc(replica.id)}">Revoke</button>` : ''}</span></div>`).join('') : '<div class="empty-row">This server is the only copy.</div>'
  document.querySelectorAll('.revoke-replica').forEach((button) => { button.onclick = () => revokeReplica(button.dataset.replica) })
	$('#connect-replica').classList.toggle('hidden', !canManageReplicas)
}

async function openProject(id) {
  currentProject = projectList.find((project) => project.id === id) || { id }
  if (accessMode === 'legacy') projectData = await request('/projects/' + id, {}, 'legacy')
  else projectData = await request(`/account/workspaces/${currentWorkspace.id}/projects/${id}`)
  currentProject = projectData.project
  renderProjectNav()
  $('#project-name').textContent = currentProject.name
  $('#project-slug').textContent = currentProject.slug
  $('#project-description').textContent = currentProject.description || 'No project description.'
  renderRepositories()
  renderLog()
  showView('project-view')
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
  return run ? [run.harness, run.harnessVersion, run.provider, run.model, run.reasoning, run.role, run.runType].filter(Boolean).join(' · ') : ''
}

function normalizedEntries() {
  const notes = (projectData.notes || []).map((note) => ({ type: 'note', kind: note.kind, status: note.status, timestamp: note.updatedAt, item: note, searchable: [note.kind, note.status, note.title, note.summary, note.rationale, noteActor(note), runtimeText(note.run), note.directionBasis, note.verification, note.sourceRef, note.pullRequestUrl, ...(note.paths || [])].join(' ').toLowerCase() }))
  const trajectories = (projectData.trajectories || []).map((trajectory) => ({ type: 'trajectory', kind: 'trajectory', status: trajectory.status, timestamp: trajectory.updatedAt, item: trajectory, searchable: ['work started', trajectory.status, trajectory.objective, trajectory.rationale, actor(trajectory.run), runtimeText(trajectory.run), trajectory.branch, ...(trajectory.paths || [])].join(' ').toLowerCase() }))
  return [...notes, ...trajectories].sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp))
}

function renderLog() {
  const query = $('#log-search').value.trim().toLowerCase()
  const kind = $('#kind-filter').value
  const status = $('#status-filter').value
  const all = normalizedEntries()
  const visible = all.filter((entry) => {
    const current = entry.type === 'trajectory' ? entry.status === 'active' : entry.status === 'current'
    return (kind === 'all' || entry.kind === kind) && (status === 'all' || (status === 'current' ? current : !current)) && (!query || entry.searchable.includes(query))
  })
  $('#log-count').textContent = visible.length === all.length ? `${all.length} ${all.length === 1 ? 'entry' : 'entries'}` : `${visible.length} of ${all.length}`
  $('#log').innerHTML = visible.length ? visible.map(logEntry).join('') : `<div class="log-empty"><strong>${all.length ? 'No entries match.' : 'No material context recorded yet.'}</strong><p>${all.length ? 'Try a different search or filter.' : 'Agents append context through the skill, CLI, or MCP when it could change another collaborator’s work.'}</p></div>`
  document.querySelectorAll('.supersede').forEach((button) => { button.onclick = () => supersede(button.dataset.id, Number(button.dataset.rev)) })
}

function logEntry(entry) {
  const item = entry.item
  const historical = entry.type === 'trajectory' ? item.status !== 'active' : item.status !== 'current'
  const title = entry.type === 'trajectory' ? item.objective : item.title
  const run = item.run
  const who = entry.type === 'trajectory' ? actor(run) : noteActor(item)
  const meta = [who, runtimeText(run), run?.branch || item.branch].filter(Boolean).map(esc).join(' · ')
  const paths = (item.paths || []).map((path) => `<code>${esc(path)}</code>`).join('')
  const pullURL = safeURL(item.pullRequestUrl)
  const sourceURL = safeURL(item.sourceRef)
  return `<article class="log-entry ${historical ? 'historical' : ''}"><time datetime="${esc(entry.timestamp)}"><strong>${esc(formatDate(entry.timestamp))}</strong><span>${esc(formatTime(entry.timestamp))}</span></time><div class="entry-body"><div class="entry-labels"><span class="entry-kind">${esc(entry.type === 'trajectory' ? 'work started' : item.kind)}</span><span class="entry-status">${esc(item.status)}</span>${entry.type === 'note' && item.status === 'current' ? `<button class="supersede" data-id="${esc(item.id)}" data-rev="${item.revision}">Supersede</button>` : ''}</div><h3>${esc(title)}</h3>${entry.type === 'note' && item.summary ? `<p class="entry-summary">${esc(item.summary)}</p>` : ''}${item.rationale ? `<p class="entry-rationale"><strong>Why:</strong> ${esc(item.rationale)}</p>` : ''}<div class="entry-provenance"><span class="actor">${meta}</span>${item.directionBasis ? `<span>${esc(item.directionBasis.replaceAll('_', ' '))}</span>` : ''}${item.verification ? `<span>${esc(item.verification)}</span>` : ''}</div>${paths || pullURL || sourceURL ? `<div class="entry-evidence"><div class="entry-paths">${paths}</div>${pullURL ? `<a href="${pullURL}" target="_blank" rel="noreferrer">Pull request ↗</a>` : ''}${sourceURL ? `<a href="${sourceURL}" target="_blank" rel="noreferrer">Source ↗</a>` : ''}</div>` : ''}</div></article>`
}

function renderRepositories() {
  const repositories = projectData.repositories || []
  $('#repository-context').innerHTML = repositories.length ? repositories.map((repository) => `<span><a href="${esc(repository.url)}" target="_blank" rel="noreferrer">${esc(repository.owner + '/' + repository.name)} ↗</a></span>`).join('') : '<span>No repository connected</span>'
}

function modal(title, fields, onSave, saveLabel = 'Save') {
  $('#dialog-fields').innerHTML = `<h2>${esc(title)}</h2>` + fields
  $('#dialog-submit').textContent = saveLabel
  const dialog = $('#dialog')
  dialog.showModal()
  $('#dialog-form').onsubmit = async (event) => {
    event.preventDefault()
    try { await onSave(new FormData(event.target)); dialog.close() } catch (error) { toast(error.message) }
  }
}

$('#email-form').onsubmit = async (event) => {
  event.preventDefault()
  const response = await request('/auth/magic-link', { method: 'POST', body: JSON.stringify({ email: $('#email').value.trim() }) }, 'public')
  $('#email-status').textContent = response.status
}

$('#consume-form').onsubmit = async (event) => {
  event.preventDefault()
  const token = new URLSearchParams(location.search).get('token')
  await request('/auth/consume', { method: 'POST', body: JSON.stringify({ token, displayName: $('#display-name').value.trim() }) }, 'public')
  history.replaceState({}, '', '/')
  account = await request('/account', {}, 'session')
  accessMode = 'session'
  await openAccount()
}

$('#token-form').onsubmit = async (event) => {
  event.preventDefault()
  bootstrapToken = $('#token').value.trim()
  sessionStorage.setItem('clank-token', bootstrapToken)
  accessMode = 'legacy'
  const identity = await request('/whoami', {}, 'legacy')
  account = { user: { displayName: identity.principal.displayName }, workspaces: [] }
  await openLegacy()
}

$('#logout').onclick = async () => {
  if (accessMode === 'session') await request('/auth/logout', { method: 'POST', body: '{}' }).catch(() => {})
  sessionStorage.removeItem('clank-token')
  location.href = '/'
}

$('#brand-home').onclick = (event) => { if (account && accessMode === 'session') { event.preventDefault(); openAccount() } }
$('#new-workspace').onclick = () => modal('New workspace', '<label>Name<input name="name" required></label><label>Slug<input name="slug" pattern="[a-z0-9]+(?:-[a-z0-9]+)*" required></label>', async (form) => { const response = await request('/account/workspaces', { method: 'POST', body: JSON.stringify(Object.fromEntries(form)) }); account = await request('/account'); workspaceList = account.workspaces; renderWorkspaceNav(); renderWorkspaceHome(); await selectWorkspace(response.workspace.id); toast('Workspace created') }, 'Create workspace')
$('#import-self-host').onclick = () => modal('Mirror a self-hosted workspace', '<p>Create a one-time code, then run the shown command on the self-hosted authority. The self-host stays in control.</p>', async () => { const response = await request('/account/mirror-offers', { method: 'POST', body: '{}' }); const command = `clank replica mirror --remote ${location.origin} --workspace &lt;workspace-id&gt; --code ${esc(response.code)}`; setTimeout(() => modal('Run this on the self-host', `<p>${esc(response.notice)}</p><label>Command<input value="${command}" readonly></label><p>Expires ${esc(formatTime(response.expiresAt))}.</p>`, async () => {}, 'Done'), 80) }, 'Create mirror code')

$('#claim-account').onclick = () => modal('Create hosted account', '<p>Link this bootstrap owner to an email address. We will send a one-time sign-in link next.</p><label>Display name<input name="displayName" required></label><label>Email address<input name="email" type="email" required></label>', async (form) => {
  const input = Object.fromEntries(form)
  await request('/admin/claim', { method: 'POST', body: JSON.stringify(input) }, 'legacy')
  await request('/auth/magic-link', { method: 'POST', body: JSON.stringify({ email: input.email }) }, 'public')
  $('#legacy-claim').classList.add('hidden')
  setTimeout(() => modal('Check your email', '<p>Your hosted account is ready. Open the one-time link to enter it; bootstrap token access remains available.</p>', async () => {}, 'Done'), 80)
}, 'Create account')

function newProject() {
  if (!currentWorkspace || accessMode === 'legacy') return toast('Create projects with the workspace owner token in self-hosted mode.')
  modal('New project', '<label>Name<input name="name" required></label><label>Slug<input name="slug" pattern="[a-z0-9]+(?:-[a-z0-9]+)*" required></label><label>Description<textarea name="description"></textarea></label>', async (form) => { const response = await request(`/account/workspaces/${currentWorkspace.id}/projects`, { method: 'POST', body: JSON.stringify(Object.fromEntries(form)) }); await selectWorkspace(currentWorkspace.id); await openProject(response.project.id); toast('Project created') }, 'Create project')
}
$('#new-project').onclick = newProject
$('#workspace-new-project').onclick = newProject
$('#workspace-settings').onclick = () => { $('#access-section').classList.toggle('hidden'); $('#access-section').scrollIntoView({ behavior: 'smooth' }) }
$('#invite-person').onclick = () => modal('Invite person', '<p>They will receive a one-time email link for this workspace.</p><label>Email<input name="email" type="email" required></label><label>Role<select name="role"><option value="member">Member</option><option value="owner">Owner</option></select></label>', async (form) => { await request(`/account/workspaces/${currentWorkspace.id}/invites`, { method: 'POST', body: JSON.stringify(Object.fromEntries(form)) }); invites = (await request(`/account/workspaces/${currentWorkspace.id}/invites`)).invites || []; renderMembers(); toast('Invitation sent') }, 'Send invitation')

async function revokeReplica(id) { await request(`/account/workspaces/${currentWorkspace.id}/replicas/${id}/revoke`, { method: 'POST', body: '{}' }); replicas = (await request(`/account/workspaces/${currentWorkspace.id}/replicas`)).replicas || []; renderReplicas(); toast('Replica revoked; later offline events will be rejected') }

async function revokeInvite(id) { await request(`/account/workspaces/${currentWorkspace.id}/invites/${id}/revoke`, { method: 'POST', body: '{}' }); invites = (await request(`/account/workspaces/${currentWorkspace.id}/invites`)).invites || []; renderMembers(); toast('Invitation revoked') }
function editMember(button) { modal('Edit member', `<label>Role<select name="role"><option value="member" ${button.dataset.role === 'member' ? 'selected' : ''}>Member</option><option value="owner" ${button.dataset.role === 'owner' ? 'selected' : ''}>Owner</option></select></label><label>Status<select name="status"><option value="active" ${button.dataset.status === 'active' ? 'selected' : ''}>Active</option><option value="suspended" ${button.dataset.status === 'suspended' ? 'selected' : ''}>Suspended</option></select></label>`, async (form) => { await request(`/account/workspaces/${currentWorkspace.id}/members/${button.dataset.membership}`, { method: 'POST', body: JSON.stringify(Object.fromEntries(form)) }); members = (await request(`/account/workspaces/${currentWorkspace.id}/members`)).members || []; renderMembers(); toast('Member updated') }) }

$('#back-workspace').onclick = () => { renderWorkspace(); showView('workspace-view') }
$('#refresh-project').onclick = () => openProject(currentProject.id)
$('#log-search').oninput = renderLog
$('#kind-filter').onchange = renderLog
$('#status-filter').onchange = renderLog
$('#dialog-cancel').onclick = () => $('#dialog').close()

$('#add-note').onclick = () => modal('Add log entry', '<p>Record only context that could change another collaborator’s next move.</p><label>Kind<select name="kind"><option>intent</option><option>decision</option><option>understanding</option><option>observation</option><option>checkpoint</option></select></label><label>Title<input name="title" maxlength="180" required></label><label>Project implication<textarea name="summary" maxlength="1200" required></textarea></label><label>Why it matters<textarea name="rationale" maxlength="2400"></textarea></label><label>Led by<select name="ledBy"><option>human</option><option>joint</option><option>agent</option><option>external</option></select></label><label>Direction basis<select name="directionBasis"><option>explicit_human_direction</option><option>joint_reasoning</option><option>interpreted_human_intent</option><option>autonomous_agent_judgment</option><option>external_evidence</option></select></label><label>Paths<input name="paths" placeholder="Comma-separated"></label>', async (form) => { const input = Object.fromEntries(form); input.paths = input.paths.split(',').map((path) => path.trim()).filter(Boolean); const path = accessMode === 'legacy' ? `/projects/${currentProject.id}/notes` : `/account/workspaces/${currentWorkspace.id}/projects/${currentProject.id}/notes`; await request(path, { method: 'POST', body: JSON.stringify(input) }); await openProject(currentProject.id); toast('Entry appended') }, 'Append entry')

function supersede(id, revision) { modal('Supersede entry', '<p>The original remains visible as historical context.</p><label>What changed?<textarea name="reason" maxlength="1200" required></textarea></label>', async (form) => { const path = accessMode === 'legacy' ? `/projects/${currentProject.id}/notes/${id}/supersede` : `/account/workspaces/${currentWorkspace.id}/projects/${currentProject.id}/notes/${id}/supersede`; await request(path, { method: 'POST', body: JSON.stringify({ expectedRevision: revision, reason: form.get('reason') }) }); await openProject(currentProject.id); toast('Entry superseded') }, 'Supersede') }

$('#agent-key').onclick = () => { closeMenu(); modal('Issue project agent key', '<p>The key acts only inside this project and is shown once.</p><label>Identity name<input name="displayName" placeholder="Shuv agents" required></label>', async (form) => { const path = accessMode === 'legacy' ? `/projects/${currentProject.id}/tokens` : `/account/workspaces/${currentWorkspace.id}/projects/${currentProject.id}/tokens`; const credential = await request(path, { method: 'POST', body: JSON.stringify(Object.fromEntries(form)) }); setTimeout(() => modal('Copy this key now', `<p>${esc(credential.notice)}</p><label>Project agent token<input value="${esc(credential.token)}" readonly></label>`, async () => {}, 'Done'), 80) }, 'Issue key') }

$('#manage-keys').onclick = async () => { closeMenu(); if (accessMode === 'legacy') return toast('Token listing is available from a signed-in workspace.'); const response = await request(`/account/workspaces/${currentWorkspace.id}/projects/${currentProject.id}/tokens`); const rows = (response.tokens || []).map((token) => `<div class="token-row"><span><strong>${esc(token.displayName)}</strong><small>${esc(token.prefix)}… · ${esc(token.scopes.join(', '))}</small></span>${token.revokedAt ? '<span>revoked</span>' : `<button type="button" class="text-button revoke-token" data-token="${esc(token.id)}">Revoke</button>`}</div>`).join('') || '<p>No agent keys.</p>'; modal('Agent keys', `<div class="token-list">${rows}</div>`, async () => {}, 'Done'); setTimeout(() => document.querySelectorAll('.revoke-token').forEach((button) => { button.onclick = async () => { await request(`/account/workspaces/${currentWorkspace.id}/projects/${currentProject.id}/tokens/${button.dataset.token}/revoke`, { method: 'POST', body: '{}' }); $('#dialog').close(); toast('Agent key revoked') } }), 0) }

$('#attach-repo').onclick = () => { closeMenu(); modal('Connect GitHub repository', '<p>Public repository data is read-only evidence, not project intent.</p><label>Public repository URL<input name="url" type="url" required></label>', async (form) => { const path = accessMode === 'legacy' ? `/projects/${currentProject.id}/repositories` : `/account/workspaces/${currentWorkspace.id}/projects/${currentProject.id}/repositories`; await request(path, { method: 'POST', body: JSON.stringify(Object.fromEntries(form)) }); await openProject(currentProject.id); toast('Repository connected') }, 'Connect') }
$('#export-project').onclick = async () => { closeMenu(); const path = accessMode === 'legacy' ? `/api/v1/projects/${currentProject.id}/export` : `/api/v1/account/workspaces/${currentWorkspace.id}/projects/${currentProject.id}/export`; const headers = accessMode === 'legacy' ? { Authorization: 'Bearer ' + bootstrapToken } : {}; const response = await fetch(path, { credentials: 'same-origin', headers }); if (!response.ok) return toast('Export failed'); const blob = await response.blob(); const download = URL.createObjectURL(blob); const link = document.createElement('a'); link.href = download; link.download = `${currentProject.slug}.clankspace.json`; link.click(); URL.revokeObjectURL(download); toast('Project log exported') }
$('#connect-replica').onclick = () => modal('Connect another server', '<p>Create a short-lived code. Run the resulting command on the server that should receive this workspace.</p>', async () => { const response = await request(`/account/workspaces/${currentWorkspace.id}/replicas/offers`, { method: 'POST', body: JSON.stringify({ capabilities: ['pull', 'push'] }) }); const command = `clank replica join --remote ${location.origin} --code ${esc(response.code)}`; setTimeout(() => modal('Run this on the new replica', `<p>${esc(response.notice)}</p><label>Command<input value="${command}" readonly></label><p>Expires ${esc(formatTime(response.expiresAt))}.</p>`, async () => {}, 'Done'), 80) }, 'Create pairing code')
function closeMenu() { $('.action-menu')?.removeAttribute('open') }

boot().catch((error) => { showSignedIn(false); toast(error.message) })
