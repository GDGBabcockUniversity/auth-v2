// Chapter register — a browser client for the GDG Babcock auth service.
// Sign-in goes through Firebase; everything after that uses platform JWTs
// against the same origin, so no CORS entry is needed for this page.

const SDK = "https://www.gstatic.com/firebasejs/10.14.1";
const STORE = { access: "gdg.access", refresh: "gdg.refresh", expires: "gdg.expires" };

const TRACKS = ["mobile", "web", "cloud", "ml", "design", "devops"];
const LEVELS = ["beginner", "intermediate", "advanced"];
// Matches the vocabulary in database/auth.sql; new users default to "user".
const KNOWN_ROLES = ["user", "admin", "moderator", "lead"];
const KNOWN_TEAMS = ["core", "android", "web", "cloud", "ml", "design", "media"];

const $ = (sel, root = document) => root.querySelector(sel);
const $$ = (sel, root = document) => [...root.querySelectorAll(sel)];

const state = {
  firebaseAuth: null,
  me: null,
  page: 1,
  limit: 20,
  search: "",
  role: "",
  archived: false,
  total: 0,
  rows: [],
  editing: null,
  draft: { roles: [], teams: [], isActive: true },
  newDraft: { roles: ["user"], teams: [] },
  teamVocabulary: new Set(KNOWN_TEAMS),
};

/* ---------------- tokens ---------------- */

const tokens = {
  get access() { return localStorage.getItem(STORE.access); },
  get refresh() { return localStorage.getItem(STORE.refresh); },
  get expiresAt() { return Number(localStorage.getItem(STORE.expires)) || 0; },
  save(pair) {
    localStorage.setItem(STORE.access, pair.access_token);
    localStorage.setItem(STORE.refresh, pair.refresh_token);
    localStorage.setItem(STORE.expires, String(Date.now() + (pair.expires_in || 0) * 1000));
  },
  clear() { Object.values(STORE).forEach((k) => localStorage.removeItem(k)); },
};

/* ---------------- api ---------------- */

class ApiError extends Error {
  constructor(status, message) { super(message); this.status = status; }
}

async function request(path, { method = "GET", body, auth = true } = {}) {
  const headers = {};
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (auth && tokens.access) headers.Authorization = `Bearer ${tokens.access}`;

  const response = await fetch(path, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  });

  let payload = null;
  try { payload = await response.json(); } catch { /* empty or non-JSON body */ }

  if (!response.ok) {
    throw new ApiError(response.status, payload?.error || `Request failed (${response.status})`);
  }
  return payload;
}

// One retry after a silent refresh, so an expired access token never shows up
// to the person as a failure.
async function api(path, options = {}) {
  try {
    return await request(path, options);
  } catch (error) {
    if (error.status !== 401 || options.auth === false || !tokens.refresh || options.retried) throw error;
    await refreshSession();
    return request(path, { ...options, retried: true });
  }
}

async function refreshSession() {
  try {
    const data = await request("/auth/refresh", {
      method: "POST",
      auth: false,
      body: { refresh_token: tokens.refresh },
    });
    tokens.save(data.tokens);
  } catch (error) {
    tokens.clear();
    throw error;
  }
}

/* ---------------- formatting ---------------- */

const dateFmt = new Intl.DateTimeFormat(undefined, { day: "numeric", month: "short", year: "numeric" });

function shortDate(value) {
  return value ? dateFmt.format(new Date(value)) : "—";
}

function sinceLabel(value) {
  if (!value) return "Never";
  const days = Math.floor((Date.now() - new Date(value).getTime()) / 86400000);
  if (days <= 0) return "Today";
  if (days === 1) return "Yesterday";
  if (days < 30) return `${days} days ago`;
  return shortDate(value);
}

function initials(name, email) {
  const source = (name || email || "?").trim();
  return source.split(/\s+/).slice(0, 2).map((part) => part[0]).join("").toUpperCase();
}

// Google avatars come and go; initials keep the row from collapsing into a hole.
function paintAvatar(node, user) {
  node.replaceChildren();
  if (user.avatar_url) {
    const image = new Image();
    image.src = user.avatar_url;
    image.alt = "";
    image.addEventListener("error", () => { node.textContent = initials(user.full_name, user.email); });
    node.append(image);
  } else {
    node.textContent = initials(user.full_name, user.email);
  }
}

function toast(message, tone = "ok") {
  const node = $("#toast");
  node.textContent = message;
  node.dataset.tone = tone;
  node.hidden = false;
  clearTimeout(toast.timer);
  toast.timer = setTimeout(() => { node.hidden = true; }, 4000);
}

function setView(view) { document.body.dataset.view = view; }

/* ---------------- sign in ---------------- */

async function bootFirebase(config) {
  const [{ initializeApp }, auth] = await Promise.all([
    import(`${SDK}/firebase-app.js`),
    import(`${SDK}/firebase-auth.js`),
  ]);
  const app = initializeApp(config);
  state.firebaseAuth = { instance: auth.getAuth(app), ...auth };
}

async function signInWithGoogle() {
  const button = $("#google-signin");
  const error = $("#signin-error");
  error.hidden = true;
  button.disabled = true;
  button.textContent = "Opening Google…";
  try {
    const { instance, GoogleAuthProvider, signInWithPopup } = state.firebaseAuth;
    const provider = new GoogleAuthProvider();
    provider.setCustomParameters({ prompt: "select_account" });
    const credential = await signInWithPopup(instance, provider);
    const firebaseToken = await credential.user.getIdToken();
    const data = await request("/auth/login", { method: "POST", auth: false, body: { firebase_token: firebaseToken } });
    tokens.save(data.tokens);
    await instance.signOut();
    await startSession();
  } catch (failure) {
    if (failure?.code === "auth/popup-closed-by-user" || failure?.code === "auth/cancelled-popup-request") {
      // The person backed out; no message needed.
    } else if (failure?.status === 403) {
      error.textContent = "This account is deactivated. Ask a chapter lead to reactivate it.";
      error.hidden = false;
    } else if (failure?.code === "auth/unauthorized-domain") {
      error.textContent = `Add ${location.host} to Firebase → Authentication → Settings → Authorized domains, then try again.`;
      error.hidden = false;
    } else {
      error.textContent = failure?.message || "Sign-in failed. Try again.";
      error.hidden = false;
    }
  } finally {
    button.disabled = false;
    button.innerHTML = button.dataset.label;
  }
}

async function signOut() {
  try {
    await api("/auth/logout", { method: "POST", body: {} });
  } catch { /* the local session goes either way */ }
  tokens.clear();
  state.me = null;
  setView("signin");
}

/* ---------------- session ---------------- */

async function startSession() {
  const data = await api("/auth/me");
  state.me = data.user;
  paintIdentity();
  setView("app");
  const isAdmin = state.me.roles?.includes("admin");
  $('.rail-link[data-page="members"]').hidden = !isAdmin;
  fillProfileForm(state.me);
  if (isAdmin) {
    showPage("members");
    refreshRoster();
  } else {
    showPage("profile");
  }
}

function paintIdentity() {
  $("#me-name").textContent = state.me.full_name || "Unnamed member";
  $("#me-email").textContent = state.me.email;
  paintAvatar($("#me-avatar"), state.me);
  paintExpiry();
  clearInterval(paintIdentity.timer);
  paintIdentity.timer = setInterval(paintExpiry, 30000);
}

function paintExpiry() {
  const label = $("#session-expiry");
  const minutes = Math.round((tokens.expiresAt - Date.now()) / 60000);
  if (!tokens.expiresAt) { label.textContent = ""; return; }
  label.textContent = minutes > 60
    ? `Session good for ${Math.round(minutes / 60)} h`
    : minutes > 0 ? `Session renews in ${minutes} min` : "Session renews on next action";
}

/* ---------------- navigation ---------------- */

function showPage(name) {
  $$(".page").forEach((page) => { page.hidden = page.dataset.page !== name; });
  $$(".rail-link").forEach((link) => link.classList.toggle("is-current", link.dataset.page === name));
}

/* ---------------- roster ---------------- */

async function loadRoster() {
  const body = $("#roster-body");
  const empty = $("#roster-empty");
  const params = new URLSearchParams({ page: state.page, limit: state.limit });
  if (state.search) params.set("search", state.search);
  if (state.role) params.set("role", state.role);
  if (state.archived) params.set("archived", "true");

  try {
    const data = await api(`/admin/users?${params}`);
    state.rows = data.users || [];
    state.total = data.total || 0;
    state.rows.forEach((user) => (user.teams || []).forEach((team) => state.teamVocabulary.add(team)));
    body.replaceChildren(...state.rows.map(rosterRow));
    empty.hidden = state.rows.length > 0;
    if (!state.rows.length) {
      empty.textContent = state.archived
        ? "Nothing archived. Members you archive land here and can be restored."
        : state.search || state.role
          ? "No member matches this search. Try a different name, email or role."
          : "No one on the register yet. Add a member, or wait for the first Google sign-in.";
    }
    paintPager();
  } catch (error) {
    body.replaceChildren();
    empty.hidden = false;
    empty.textContent = error.message;
  }
}

function rosterRow(user) {
  const row = document.createElement("tr");
  row.tabIndex = 0;
  row.classList.toggle("is-inactive", !user.is_active);

  const person = document.createElement("td");
  const wrap = document.createElement("div");
  wrap.className = "person";
  const badge = document.createElement("span");
  badge.className = "avatar";
  badge.setAttribute("aria-hidden", "true");
  paintAvatar(badge, user);
  const text = document.createElement("div");
  text.innerHTML = "<strong></strong><span class='muted'></span>";
  text.querySelector("strong").textContent = user.full_name || initials(user.full_name, user.email);
  text.querySelector("span").textContent = user.email;
  wrap.append(badge, text);
  person.append(wrap);

  const roles = document.createElement("td");
  if (user.roles?.length) roles.append(...user.roles.map((role) => chip(role, role === "admin" ? "admin" : "role")));
  else roles.append(document.createTextNode("—"));

  const teams = document.createElement("td");
  if (user.teams?.length) teams.append(...user.teams.map((team) => chip(team, "team")));
  else teams.append(document.createTextNode("—"));

  const seen = document.createElement("td");
  seen.className = "when";
  seen.textContent = user.pending ? "Not yet" : sinceLabel(user.last_login_at);

  const joined = document.createElement("td");
  joined.className = "when";
  joined.textContent = shortDate(user.created_at);

  const status = document.createElement("td");
  status.className = "statuscell";
  if (user.deleted_at) status.append(chip("Archived", "off"));
  else if (user.pending) status.append(chip("Invited", "pending"));
  else status.append(chip(user.is_active ? "Active" : "Deactivated", user.is_active ? "on" : "off"));

  row.append(person, roles, teams, seen, joined, status);
  row.addEventListener("click", () => openSheet(user));
  row.addEventListener("keydown", (event) => {
    if (event.key === "Enter" || event.key === " ") { event.preventDefault(); openSheet(user); }
  });
  return row;
}

function chip(label, tone) {
  const node = document.createElement("span");
  node.className = "chip";
  node.dataset.tone = tone;
  node.textContent = label;
  return node;
}

function paintPager() {
  const pages = Math.max(1, Math.ceil(state.total / state.limit));
  const first = state.total === 0 ? 0 : (state.page - 1) * state.limit + 1;
  const last = Math.min(state.page * state.limit, state.total);
  $("#pager-label").textContent = `${first}–${last} of ${state.total}`;
  $("#prev").disabled = state.page <= 1;
  $("#next").disabled = state.page >= pages;
}

async function loadLedger() {
  const ledger = $("#ledger");
  try {
    const [everyone, admins, archived] = await Promise.all([
      api("/admin/users?page=1&limit=1"),
      api("/admin/users?page=1&limit=1&role=admin"),
      api("/admin/users?page=1&limit=1&archived=true"),
    ]);
    const figures = [
      { value: everyone.total, label: "Members on the register" },
      { value: admins.total, label: "Chapter admins", tone: "admin" },
    ];
    if (archived.total) {
      figures.push({ value: archived.total, label: "Archived" });
    }
    if (state.search || state.role) {
      figures.push({ value: state.total, label: "Matching this search" });
    }
    ledger.replaceChildren(...figures.map(({ value, label, tone }) => {
      const item = document.createElement("div");
      item.className = "ledger-item";
      if (tone) item.dataset.tone = tone;
      item.innerHTML = "<span class='ledger-value'></span><span class='ledger-label'></span>";
      item.querySelector(".ledger-value").textContent = value ?? 0;
      item.querySelector(".ledger-label").textContent = label;
      return item;
    }));
  } catch {
    ledger.replaceChildren();
  }
}

/* ---------------- member editor ---------------- */

// The roster row carries only a summary, so the editor fetches the full record.
async function openSheet(summary) {
  $("#sheet").hidden = false;
  $("#sheet-backdrop").hidden = false;
  $("#sheet-error").hidden = true;
  $("#sheet-close").focus();

  let user = summary;
  try {
    const data = await api(`/admin/users/${summary.id}`);
    // Fall back to the roster summary rather than crashing on a thin body.
    user = data?.user ?? summary;
  } catch (error) {
    showSheetError(error.message);
  }

  state.editing = user;
  state.draft = {
    roles: [...(user.roles || [])],
    teams: [...(user.teams || [])],
    isActive: Boolean(user.is_active),
  };

  $("#sheet-name").textContent = user.full_name || "Unnamed member";
  $("#sheet-email").textContent = user.email;
  paintAvatar($("#sheet-avatar"), user);

  const form = $("#sheet-form");
  SHEET_PROFILE_FIELDS.forEach((name) => {
    const field = form.elements[name];
    if (!field) return;
    let value = user[name] ?? "";
    if (name === "birthday" && value) value = String(value).slice(0, 10);
    field.value = value;
  });

  $("#sheet-id").textContent = user.id;
  $("#sheet-email-fact").textContent = user.email;
  $("#sheet-verified").textContent = user.email_verified ? "Yes" : "No";
  $("#sheet-tos").textContent = user.tos_agreed
    ? `Agreed ${shortDate(user.tos_agreed_at)}${user.tos_version ? ` (v${user.tos_version})` : ""}`
    : "Not agreed";
  $("#sheet-joined").textContent = shortDate(user.created_at);
  $("#sheet-seen").textContent = user.pending
    ? "Never — invitation pending"
    : user.last_login_at ? shortDate(user.last_login_at) : "Never";
  $("#sheet-updated").textContent = user.updated_at ? shortDate(user.updated_at) : "—";

  const archived = Boolean(user.deleted_at);
  const self = user.id === state.me.id;
  const banner = $("#sheet-banner");
  banner.hidden = !archived && !user.pending;
  banner.dataset.tone = archived ? "archived" : "pending";
  if (archived) {
    banner.textContent = `Archived ${shortDate(user.deleted_at)}. This person cannot sign in until you restore them.`;
  } else if (user.pending) {
    banner.textContent = "Invited, not signed in yet. The record is claimed when they sign in with Google using this address.";
  }

  $("#is-active").checked = state.draft.isActive;
  $("#is-active").disabled = self || archived;
  $("#self-note").hidden = !self;
  $("#sheet-archive").hidden = archived || self;
  $("#sheet-restore").hidden = !archived;
  form.querySelectorAll("input, select").forEach((field) => {
    if (field.id !== "is-active") field.disabled = archived;
  });
  // Nothing on an archived record is editable, so don't leave live controls.
  $("#team-add").disabled = archived;
  $('[form="sheet-form"]').disabled = archived;

  paintRoleChips(self, archived);
  paintTeamChips(archived);
}

function showSheetError(message) {
  const error = $("#sheet-error");
  error.textContent = message;
  error.hidden = false;
}

function closeSheet() {
  $("#sheet").hidden = true;
  $("#sheet-backdrop").hidden = true;
  state.editing = null;
}

function paintRoleChips(self, locked) {
  const box = $("#roles-box");
  const roles = [...new Set([...KNOWN_ROLES, ...state.draft.roles])];
  box.replaceChildren(...roles.map((role) => {
    const on = state.draft.roles.includes(role);
    const button = toggleChip(role, on, role === "admin" ? "admin" : "role");
    if (locked) {
      button.disabled = true;
    } else if (self && role === "admin") {
      button.disabled = true;
      button.title = "You cannot remove your own admin role.";
    }
    button.addEventListener("click", () => {
      state.draft.roles = on ? state.draft.roles.filter((r) => r !== role) : [...state.draft.roles, role];
      paintRoleChips(self, locked);
    });
    return button;
  }));
}

function paintTeamChips(locked) {
  const box = $("#teams-box");
  const teams = [...new Set([...state.teamVocabulary, ...state.draft.teams])].sort();
  box.replaceChildren(...teams.map((team) => {
    const on = state.draft.teams.includes(team);
    const button = toggleChip(team, on, "team");
    button.disabled = Boolean(locked);
    button.addEventListener("click", () => {
      state.draft.teams = on ? state.draft.teams.filter((t) => t !== team) : [...state.draft.teams, team];
      paintTeamChips(locked);
    });
    return button;
  }));
}

function toggleChip(label, pressed, tone) {
  const button = document.createElement("button");
  button.type = "button";
  button.className = "togglechip";
  button.dataset.tone = tone;
  button.setAttribute("aria-pressed", String(pressed));
  button.textContent = label;
  return button;
}

const SHEET_PROFILE_FIELDS = [
  "full_name", "whatsapp_number", "matric_no", "student_status",
  "department", "faculty", "primary_track", "primary_skill_level",
  "secondary_track", "secondary_skill_level", "gender", "birthday",
];

async function saveMember(event) {
  event.preventDefault();
  const user = state.editing;
  if (!user) return;
  $("#sheet-error").hidden = true;

  if (!state.draft.roles.length) {
    showSheetError("A member needs at least one role.");
    return;
  }

  const form = $("#sheet-form");
  const body = { roles: state.draft.roles, teams: state.draft.teams, is_active: $("#is-active").checked };
  SHEET_PROFILE_FIELDS.forEach((name) => {
    const field = form.elements[name];
    if (!field) return;
    const current = name === "birthday"
      ? String(user[name] ?? "").slice(0, 10)
      : String(user[name] ?? "");
    if (field.value !== current) body[name] = field.value;
  });

  try {
    const data = await api(`/admin/users/${user.id}`, { method: "PUT", body });
    closeSheet();
    toast(`Saved ${data.user.full_name || data.user.email}`);
    await refreshRoster();
  } catch (failure) {
    showSheetError(failure.status === 403
      ? "That change isn't allowed. Admins cannot demote themselves."
      : failure.message);
  }
}

async function archiveMember() {
  const user = state.editing;
  if (!user) return;
  const name = user.full_name || user.email;
  if (!confirm(`Archive ${name}? They are signed out everywhere and cannot sign in again until you restore them. Their record is kept.`)) return;

  try {
    await api(`/admin/users/${user.id}`, { method: "DELETE" });
    closeSheet();
    toast(`Archived ${name}`);
    await refreshRoster();
  } catch (error) {
    showSheetError(error.message);
  }
}

async function restoreMember() {
  const user = state.editing;
  if (!user) return;
  const name = user.full_name || user.email;
  try {
    await api(`/admin/users/${user.id}/restore`, { method: "POST", body: {} });
    closeSheet();
    toast(`Restored ${name}`);
    await refreshRoster();
  } catch (error) {
    showSheetError(error.message);
  }
}

/* ---------------- add a member ---------------- */

function openCreateSheet() {
  state.newDraft = { roles: ["user"], teams: [] };
  const form = $("#create-form");
  form.reset();
  $("#create-error").hidden = true;
  paintCreateChips();
  $("#create-sheet").hidden = false;
  $("#create-backdrop").hidden = false;
  form.elements.email.focus();
}

function closeCreateSheet() {
  $("#create-sheet").hidden = true;
  $("#create-backdrop").hidden = true;
}

function paintCreateChips() {
  const roles = $("#create-roles");
  roles.replaceChildren(...KNOWN_ROLES.map((role) => {
    const on = state.newDraft.roles.includes(role);
    const button = toggleChip(role, on, role === "admin" ? "admin" : "role");
    button.addEventListener("click", () => {
      state.newDraft.roles = on
        ? state.newDraft.roles.filter((r) => r !== role)
        : [...state.newDraft.roles, role];
      paintCreateChips();
    });
    return button;
  }));

  const teams = $("#create-teams");
  const vocabulary = [...new Set([...state.teamVocabulary, ...state.newDraft.teams])].sort();
  teams.replaceChildren(...vocabulary.map((team) => {
    const on = state.newDraft.teams.includes(team);
    const button = toggleChip(team, on, "team");
    button.addEventListener("click", () => {
      state.newDraft.teams = on
        ? state.newDraft.teams.filter((t) => t !== team)
        : [...state.newDraft.teams, team];
      paintCreateChips();
    });
    return button;
  }));
}

async function createMember(event) {
  event.preventDefault();
  const form = event.target;
  const error = $("#create-error");
  error.hidden = true;

  const email = form.elements.email.value.trim();
  if (!email.includes("@")) {
    error.textContent = "Enter the Google address this person will sign in with.";
    error.hidden = false;
    return;
  }
  if (!state.newDraft.roles.length) {
    error.textContent = "Pick at least one role.";
    error.hidden = false;
    return;
  }

  const submit = form.querySelector('button[type="submit"]') || $('[form="create-form"]');
  submit.disabled = true;
  try {
    const data = await api("/admin/users", {
      method: "POST",
      body: {
        email,
        full_name: form.elements.full_name.value.trim(),
        roles: state.newDraft.roles,
        teams: state.newDraft.teams,
      },
    });
    closeCreateSheet();
    toast(`${data.user.email} added — pending their first sign-in`);
    await refreshRoster();
  } catch (failure) {
    error.textContent = failure.status === 409
      ? "Someone with that email is already on the register."
      : failure.message;
    error.hidden = false;
  } finally {
    submit.disabled = false;
  }
}

async function refreshRoster() {
  await loadRoster();
  await loadLedger();
}

/* ---------------- profile ---------------- */

function fillOptions() {
  $$("select[data-tracks]").forEach((select) => {
    select.replaceChildren(option("", "Not set"), ...TRACKS.map((track) => option(track, track)));
  });
  $$("select[data-levels]").forEach((select) => {
    select.replaceChildren(option("", "Not set"), ...LEVELS.map((level) => option(level, level)));
  });
}

function option(value, label) {
  const node = document.createElement("option");
  node.value = value;
  node.textContent = label.charAt(0).toUpperCase() + label.slice(1);
  return node;
}

const PROFILE_TEXT_FIELDS = [
  "full_name", "whatsapp_number", "avatar_url", "gender", "birthday",
  "matric_no", "student_status", "department", "faculty",
  "primary_track", "secondary_track", "primary_skill_level", "secondary_skill_level",
];

function fillProfileForm(user) {
  const form = $("#profile-form");
  PROFILE_TEXT_FIELDS.forEach((name) => {
    const field = form.elements[name];
    if (!field) return;
    let value = user[name] ?? "";
    if (name === "birthday" && value) value = String(value).slice(0, 10);
    field.value = value;
  });
  form.elements.tos_agreed.checked = Boolean(user.tos_agreed);
  form.elements.tos_agreed.disabled = Boolean(user.tos_agreed);

  $("#tos-state").textContent = user.tos_agreed
    ? `Agreed ${shortDate(user.tos_agreed_at)}${user.tos_version ? ` to version ${user.tos_version}` : ""}.`
    : "Not agreed yet.";
  $("#profile-meta").textContent = `Last updated ${shortDate(user.updated_at)}`;
}

async function saveProfile(event) {
  event.preventDefault();
  const form = event.target;
  const submit = form.querySelector('button[type="submit"]');
  const body = {};

  PROFILE_TEXT_FIELDS.forEach((name) => {
    const field = form.elements[name];
    if (!field) return;
    const current = String(state.me[name] ?? "").slice(0, name === "birthday" ? 10 : undefined);
    if (field.value !== current) body[name] = field.value;
  });
  if (form.elements.tos_agreed.checked && !state.me.tos_agreed) body.tos_agreed = true;

  if (!Object.keys(body).length) {
    toast("Nothing to save yet");
    return;
  }

  submit.disabled = true;
  try {
    const data = await api("/auth/profile", { method: "PUT", body });
    state.me = data.user;
    fillProfileForm(state.me);
    paintIdentity();
    toast("Profile saved");
  } catch (error) {
    toast(error.message, "error");
  } finally {
    submit.disabled = false;
  }
}

/* ---------------- wiring ---------------- */

function debounce(fn, wait) {
  let timer;
  return (...args) => { clearTimeout(timer); timer = setTimeout(() => fn(...args), wait); };
}

function wire() {
  fillOptions();

  const signin = $("#google-signin");
  signin.dataset.label = signin.innerHTML;
  signin.addEventListener("click", signInWithGoogle);
  $("#signout").addEventListener("click", signOut);

  $$(".rail-link").forEach((link) => link.addEventListener("click", () => showPage(link.dataset.page)));

  const refilter = debounce(() => { state.page = 1; refreshRoster(); }, 250);
  $("#q").addEventListener("input", (event) => { state.search = event.target.value.trim(); refilter(); });
  $("#role-filter").addEventListener("change", (event) => { state.role = event.target.value; state.page = 1; refreshRoster(); });
  $("#limit").addEventListener("change", (event) => { state.limit = Number(event.target.value); state.page = 1; loadRoster(); });
  $("#filters").addEventListener("submit", (event) => event.preventDefault());
  $("#prev").addEventListener("click", () => { if (state.page > 1) { state.page -= 1; loadRoster(); } });
  $("#next").addEventListener("click", () => { state.page += 1; loadRoster(); });

  $("#sheet-form").addEventListener("submit", saveMember);
  $("#sheet-close").addEventListener("click", closeSheet);
  $("#sheet-cancel").addEventListener("click", closeSheet);
  $("#sheet-backdrop").addEventListener("click", closeSheet);
  $("#sheet-archive").addEventListener("click", archiveMember);
  $("#sheet-restore").addEventListener("click", restoreMember);

  $("#add-member").addEventListener("click", openCreateSheet);
  $("#create-form").addEventListener("submit", createMember);
  $("#create-close").addEventListener("click", closeCreateSheet);
  $("#create-cancel").addEventListener("click", closeCreateSheet);
  $("#create-backdrop").addEventListener("click", closeCreateSheet);

  $("#status-filter").addEventListener("change", (event) => {
    state.archived = event.target.value === "true";
    state.page = 1;
    refreshRoster();
  });
  $("#team-add").addEventListener("click", addTeam);
  $("#team-new").addEventListener("keydown", (event) => {
    if (event.key === "Enter") { event.preventDefault(); addTeam(); }
  });
  document.addEventListener("keydown", (event) => {
    if (event.key !== "Escape") return;
    if (!$("#create-sheet").hidden) closeCreateSheet();
    else if (!$("#sheet").hidden) closeSheet();
  });

  $("#profile-form").addEventListener("submit", saveProfile);
  $("#profile-reset").addEventListener("click", (event) => {
    event.preventDefault();
    fillProfileForm(state.me);
  });
}

function addTeam() {
  const input = $("#team-new");
  const name = input.value.trim().toLowerCase();
  if (!name) return;
  state.teamVocabulary.add(name);
  if (!state.draft.teams.includes(name)) state.draft.teams.push(name);
  input.value = "";
  paintTeamChips();
}

async function main() {
  wire();

  let settings;
  try {
    settings = await request("/dashboard/config.json", { auth: false });
  } catch {
    setView("setup");
    return;
  }
  if (!settings.configured) {
    setView("setup");
    return;
  }

  try {
    await bootFirebase(settings.firebase);
  } catch {
    setView("setup");
    return;
  }

  if (tokens.access || tokens.refresh) {
    try {
      await startSession();
      return;
    } catch {
      tokens.clear();
    }
  }
  setView("signin");
}

main();
