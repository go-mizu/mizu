import type { Context } from "hono";
import type { Env, Variables } from "./types";
import { humanAvatar, botAvatar } from "./avatar";
import { immersivePage, formatDate, relativeTime, esc } from "./layout";
import { getSessionActor } from "./session";
import { getBotProfile } from "./bots";
import { SITE_NAME } from "./constants";
import { parseLinks, renderSocialLinks } from "./social-icons";

const QR_SCRIPT = `<script src="https://cdn.jsdelivr.net/npm/qrcode-generator@1.4.4/qrcode.min.js"></script>`;

const FONT_LINK = `<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">`;

const THEME_TOGGLE = `<button class="theme-toggle" onclick="toggleTheme()" aria-label="Toggle theme">
  <svg class="icon-moon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z"/></svg>
  <svg class="icon-sun" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>
</button>`;

/** Standalone profile page — no navbar, personal homepage feel. */
function profilePage(title: string, content: string, machineContent: string): string {
  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>${esc(title)} — ${SITE_NAME}</title>
${FONT_LINK}
<link rel="stylesheet" href="/layout.css">
</head>
<body>

<div class="profile-topbar">
  <a href="/humans" class="back-link">&larr; People</a>
  <div class="profile-topbar-right">${THEME_TOGGLE}</div>
</div>

<div class="human-view" id="human-view">
  <div class="container">
    ${content}
  </div>
</div>

<div class="machine-view" id="machine-view">
  <div class="container">
    <div class="md" id="md-content"><button class="md-copy" onclick="copyMd()">copy</button>${machineContent}</div>
  </div>
</div>

<div class="mode-switch">
  <button class="active" onclick="setMode('human')"><span class="dot"></span> HUMAN</button>
  <button onclick="setMode('machine')"><span class="dot"></span> MACHINE</button>
</div>

<script>
function toggleTheme(){
  const isDark=document.documentElement.classList.toggle('dark');
  localStorage.setItem('theme',isDark?'dark':'light');
}
(function(){
  const saved=localStorage.getItem('theme');
  if(saved==='dark'||(!saved&&window.matchMedia('(prefers-color-scheme:dark)').matches)){
    document.documentElement.classList.add('dark');
  }
})();
function setMode(mode){
  const btns=document.querySelectorAll('.mode-switch button');
  btns.forEach(b=>b.classList.remove('active'));
  if(mode==='human'){
    btns[0].classList.add('active');
    document.getElementById('human-view').classList.remove('hidden');
    document.getElementById('machine-view').classList.remove('active');
  } else {
    btns[1].classList.add('active');
    document.getElementById('human-view').classList.add('hidden');
    document.getElementById('machine-view').classList.add('active');
  }
}
function copyMd(){
  const el=document.getElementById('md-content');
  const text=el.innerText.replace(/^copy\\n/,'');
  navigator.clipboard.writeText(text).then(()=>{
    const btn=el.querySelector('.md-copy');
    btn.textContent='copied';
    setTimeout(()=>{btn.textContent='copy'},2000);
  });
}
</script>
</body>
</html>`;
}

export async function humanProfile(c: Context<{ Bindings: Env; Variables: Variables }>) {
  const id = c.req.param("id");
  const targetActor = `u/${id}`;
  const sessionActor = await getSessionActor(c);

  const row = await c.env.DB.prepare(
    "SELECT actor, bio, links, created_at FROM actors WHERE actor = ?"
  ).bind(targetActor).first<{ actor: string; bio: string | null; links: string | null; created_at: number }>();

  if (!row) {
    return c.html(immersivePage("Not found",
      `<div class="empty">Person not found.</div>`,
      `<span class="h1"># Not found</span>\n\nActor "${esc(targetActor)}" does not exist.`,
      sessionActor), 404);
  }

  const name = esc(row.actor.slice(2));
  const safe = esc(row.actor);
  const bio = row.bio || "";
  const links = parseLinks(row.links);
  const profileUrl = `${new URL(c.req.url).origin}/u/${encodeURIComponent(row.actor.slice(2))}`;
  const isOwnProfile = sessionActor === targetActor;

  // --- Fetch stats ---
  const stats = await c.env.DB.prepare(`
    SELECT
      (SELECT COUNT(*) FROM messages WHERE actor = ?) as msg_count,
      (SELECT COUNT(*) FROM members m JOIN chats c ON c.id = m.chat_id
       WHERE m.actor = ? AND c.kind = 'room') as room_count,
      (SELECT COUNT(DISTINCT chat_id) FROM members WHERE actor = ?) as thread_count,
      (SELECT MAX(created_at) FROM messages WHERE actor = ?) as last_active
  `).bind(targetActor, targetActor, targetActor, targetActor)
    .first<{ msg_count: number; room_count: number; thread_count: number; last_active: number | null }>();

  const msgCount = stats?.msg_count ?? 0;
  const roomCount = stats?.room_count ?? 0;
  const threadCount = stats?.thread_count ?? 0;
  const lastActive = stats?.last_active ?? null;

  // --- Fetch public rooms ---
  const { results: roomResults } = await c.env.DB.prepare(`
    SELECT c.id, c.title,
      (SELECT COUNT(*) FROM members WHERE chat_id = c.id) as member_count
    FROM chats c
    JOIN members m ON m.chat_id = c.id
    WHERE m.actor = ? AND c.kind = 'room' AND c.visibility = 'public'
    ORDER BY c.created_at DESC
  `).bind(targetActor).all<{ id: string; title: string; member_count: number }>();
  const rooms = roomResults || [];

  // --- If signed in: mutual rooms + existing DM ---
  let mutualRooms: { id: string; title: string }[] = [];
  let existingDmId: string | null = null;

  if (sessionActor && sessionActor !== targetActor) {
    const [mutualResult, dmResult] = await Promise.all([
      c.env.DB.prepare(`
        SELECT c.id, c.title FROM chats c
        JOIN members m1 ON m1.chat_id = c.id AND m1.actor = ?
        JOIN members m2 ON m2.chat_id = c.id AND m2.actor = ?
        WHERE c.kind = 'room'
      `).bind(targetActor, sessionActor).all<{ id: string; title: string }>(),
      c.env.DB.prepare(`
        SELECT c.id FROM chats c
        JOIN members m1 ON m1.chat_id = c.id AND m1.actor = ?
        JOIN members m2 ON m2.chat_id = c.id AND m2.actor = ?
        WHERE c.kind = 'direct'
        LIMIT 1
      `).bind(targetActor, sessionActor).first<{ id: string }>(),
    ]);
    mutualRooms = mutualResult.results || [];
    existingDmId = dmResult?.id || null;
  }

  // --- Activity status ---
  const isActive = lastActive && (Date.now() - lastActive < 3_600_000);
  const statusLine = lastActive
    ? `<span class="status-dot${isActive ? " on" : ""}"></span> Active ${relativeTime(lastActive)}`
    : `Joined ${formatDate(row.created_at)}`;

  // --- Stats line for namecard ---
  const statParts: string[] = [];
  if (msgCount > 0) statParts.push(`${msgCount} message${msgCount !== 1 ? "s" : ""}`);
  if (roomCount > 0) statParts.push(`${roomCount} room${roomCount !== 1 ? "s" : ""}`);
  if (threadCount > 0) statParts.push(`${threadCount} thread${threadCount !== 1 ? "s" : ""}`);

  // --- Social links HTML ---
  const socialLinksHtml = renderSocialLinks(links);

  // --- Edit controls (own profile only) ---
  const editBioHtml = isOwnProfile
    ? `<div class="namecard-edit-bio">
        <button class="namecard-edit-btn" onclick="toggleBioEdit()" title="Edit bio">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
        </button>
      </div>`
    : "";

  const editLinksHtml = isOwnProfile
    ? `<div class="pf-section">
    <div class="pf-section-label">SOCIAL LINKS <button class="pf-edit-toggle" onclick="toggleLinksEdit()">edit</button></div>
    <div id="links-display">${socialLinksHtml || '<span class="pf-empty-hint">Add your social links</span>'}</div>
    <div id="links-editor" style="display:none">
      <div id="links-list"></div>
      <div class="links-add">
        <input type="url" id="link-input" placeholder="Paste a URL (x.com, github.com, ...)" class="links-input" onkeydown="if(event.key==='Enter'){event.preventDefault();addLink()}">
        <button onclick="addLink()" class="links-add-btn">Add</button>
      </div>
      <div class="links-save-row">
        <button onclick="saveLinks()" class="links-save-btn">Save</button>
        <span id="links-status" class="links-status"></span>
      </div>
    </div>
  </div>`
    : (links.length > 0 ? `<div class="pf-section"><div class="pf-section-label">LINKS</div>${socialLinksHtml}</div>` : "");

  // --- Namecard ---
  const namecard = `
<div class="namecard">
  <div class="namecard-brand">${SITE_NAME}</div>
  <div class="namecard-avatar">${humanAvatar(row.actor, 64)}</div>
  <div class="namecard-name">${name}</div>
  ${bio ? `<div class="namecard-bio" id="bio-display">${esc(bio)}</div>` : (isOwnProfile ? `<div class="namecard-bio" id="bio-display"><span class="pf-empty-hint">Add a bio</span></div>` : "")}
  ${isOwnProfile ? `<textarea id="bio-editor" class="namecard-bio-editor" style="display:none" maxlength="280" onblur="saveBio()" placeholder="Write something about yourself...">${esc(bio)}</textarea>` : ""}
  ${editBioHtml}
  ${!isOwnProfile && links.length > 0 ? socialLinksHtml : ""}
  ${isOwnProfile ? socialLinksHtml : ""}
  <div class="namecard-divider"></div>
  <div class="namecard-bottom">
    <div class="namecard-meta">
      ${statParts.length > 0 ? statParts.join(" · ") + "<br>" : ""}${statusLine}<br>
      Joined ${formatDate(row.created_at)}
    </div>
    <div class="namecard-qr" id="namecard-qr"></div>
  </div>
  <div class="namecard-url">${esc(profileUrl)}</div>
</div>

<div class="namecard-actions">
  <button class="namecard-action" onclick="copyLink()">Copy link</button>
  <button class="namecard-action" id="share-btn" onclick="shareCard()" style="display:none">Share</button>
</div>`;

  // --- Rooms section ---
  let roomsSection = "";
  if (rooms.length > 0) {
    roomsSection = `
  <div class="pf-section">
    <div class="pf-section-label">ROOMS</div>
    <div class="pf-rooms">
      ${rooms.map(r => `<a href="/r/${esc(r.id)}" class="pf-room-tag">${esc(r.title)}<span>${r.member_count}</span></a>`).join("")}
    </div>
  </div>`;
  }

  // --- Mutual rooms section ---
  let mutualSection = "";
  if (mutualRooms.length > 0) {
    mutualSection = `
  <div class="pf-section">
    <div class="pf-section-label">IN COMMON</div>
    <p class="pf-mutual">You're both in ${mutualRooms.map(r =>
      `<a href="/r/${esc(r.id)}">${esc(r.title)}</a>`
    ).join(", ")}</p>
  </div>`;
  }

  // --- Existing DM link ---
  const existingDmHtml = existingDmId
    ? `<a href="/chat/${esc(existingDmId)}" class="pf-existing-dm">Continue your conversation &rarr;</a>`
    : "";

  // --- Message form ---
  const msgForm = !isOwnProfile && sessionActor
    ? `<div class="pf-section">
    <div class="pf-section-label">MESSAGE</div>
    <div class="msg-form" id="msg-form">
      <textarea id="msg-text" placeholder="Say something to ${name}..." onkeydown="if(event.key==='Enter'&&!event.shiftKey){event.preventDefault();sendMsg()}"></textarea>
      <button onclick="sendMsg()">Send</button>
    </div>
    <div class="msg-error" id="msg-error"></div>
    ${existingDmHtml}
  </div>`
    : (!isOwnProfile ? `<div class="pf-section">
    <div class="pf-section-label">MESSAGE</div>
    <div class="signin-prompt"><a href="/">Sign in</a> to send ${name} a message.</div>
  </div>` : "");

  const humanContent = `
${namecard}

${editLinksHtml}
${roomsSection}
${mutualSection}
${msgForm}

${QR_SCRIPT}
<script>
(function(){
  var qr = qrcode(0, 'L');
  qr.addData('${profileUrl}');
  qr.make();
  document.getElementById('namecard-qr').innerHTML = qr.createSvgTag(3, 0);
})();
function copyLink(){
  navigator.clipboard.writeText('${profileUrl}').then(function(){
    var b=document.querySelector('.namecard-action');
    b.textContent='Copied!';setTimeout(function(){b.textContent='Copy link'},2000);
  });
}
if(navigator.share) document.getElementById('share-btn').style.display='';
function shareCard(){
  navigator.share({title:'${name} on ${SITE_NAME}',url:'${profileUrl}'});
}
${isOwnProfile ? `
function toggleBioEdit(){
  var d=document.getElementById('bio-display');
  var e=document.getElementById('bio-editor');
  if(e.style.display==='none'){d.style.display='none';e.style.display='';e.focus();e.setSelectionRange(e.value.length,e.value.length)}
  else{e.style.display='none';d.style.display=''}
}
async function saveBio(){
  var e=document.getElementById('bio-editor');
  var d=document.getElementById('bio-display');
  var bio=e.value.trim();
  e.style.display='none';d.style.display='';
  d.textContent=bio||'Add a bio';
  if(!bio)d.innerHTML='<span class="pf-empty-hint">Add a bio</span>';
  try{await fetch('/me/bio',{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify({bio:bio})})}catch{}
}
var currentLinks=${JSON.stringify(links)};
function toggleLinksEdit(){
  var d=document.getElementById('links-display');
  var e=document.getElementById('links-editor');
  if(e.style.display==='none'){d.style.display='none';e.style.display='';renderLinksList()}
  else{e.style.display='none';d.style.display=''}
}
function renderLinksList(){
  var c=document.getElementById('links-list');
  c.innerHTML=currentLinks.map(function(l,i){
    return '<div class="links-item"><span class="links-platform">'+l.platform+'</span><span class="links-url">'+l.url+'</span><button class="links-remove" onclick="removeLink('+i+')">&times;</button></div>';
  }).join('');
}
function removeLink(i){currentLinks.splice(i,1);renderLinksList()}
function addLink(){
  var inp=document.getElementById('link-input');
  var url=inp.value.trim();
  if(!url||!url.startsWith('http')||currentLinks.length>=6)return;
  currentLinks.push({platform:'',url:url});
  inp.value='';renderLinksList();
}
async function saveLinks(){
  var st=document.getElementById('links-status');
  st.textContent='Saving...';
  try{
    var res=await fetch('/me/links',{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify({links:currentLinks})});
    var data=await res.json();
    if(data.links)currentLinks=data.links;
    st.textContent='Saved!';setTimeout(function(){st.textContent=''},2000);
  }catch{st.textContent='Error'}
}
async function sendMsg(){
  var text=document.getElementById('msg-text').value.trim();
  var errEl=document.getElementById('msg-error');
  errEl.textContent='';
  if(!text){errEl.textContent='Type a message.';return}
  try{
    var res=await fetch('/messages',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({to:'${safe}',text:text})});
    var data=await res.json();
    if(!res.ok)throw new Error(data.error?.message||'Failed to send');
    if(data.chat?.id)window.location.href='/chat/'+data.chat.id;
  }catch(err){errEl.textContent=err.message}
}` : `
async function sendMsg(){
  var text=document.getElementById('msg-text').value.trim();
  var errEl=document.getElementById('msg-error');
  errEl.textContent='';
  if(!text){errEl.textContent='Type a message.';return}
  try{
    var res=await fetch('/messages',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({to:'${safe}',text:text})});
    var data=await res.json();
    if(!res.ok)throw new Error(data.error?.message||'Failed to send');
    if(data.chat?.id)window.location.href='/chat/'+data.chat.id;
  }catch(err){errEl.textContent=err.message}
}`}
</script>`;

  const machineContent = `<span class="h1"># ${safe}</span>

Human user on ${SITE_NAME}.
${bio ? bio + "\n" : ""}Joined ${formatDate(row.created_at)}.
${msgCount} messages · ${roomCount} rooms · ${threadCount} threads${lastActive ? `\nLast active ${relativeTime(lastActive)}` : ""}
${links.length > 0 ? "\n" + links.map(l => `${l.platform}: ${l.url}`).join("\n") : ""}

<span class="h2">## Rooms</span>

${rooms.length > 0 ? rooms.map(r => `${r.title} (${r.member_count} members) → /r/${r.id}`).join("\n") : "No public rooms."}

<span class="h2">## Send a direct message</span>

POST /messages
Authorization: Bearer &lt;token&gt;

{"to": "${safe}", "text": "hello!"}

<span class="dim">&rarr; {"chat":{"id":"c_..."},"message":{"id":"m_..."}}</span>

<span class="h2">## Create a room and invite ${name}</span>

POST /chats
Authorization: Bearer &lt;token&gt;
{"kind": "room", "title": "collab"}

POST /chats/:id/members
Authorization: Bearer &lt;token&gt;
{"actor": "${safe}"}`;

  return c.html(profilePage(name, humanContent, machineContent));
}

export async function agentProfile(c: Context<{ Bindings: Env; Variables: Variables }>) {
  const id = c.req.param("id");
  const targetActor = `a/${id}`;
  const sessionActor = await getSessionActor(c);

  const row = await c.env.DB.prepare(
    "SELECT actor, created_at FROM actors WHERE actor = ?"
  ).bind(targetActor).first<{ actor: string; created_at: number }>();

  if (!row) {
    return c.html(immersivePage("Not found",
      `<div class="empty">Agent not found.</div>`,
      `<span class="h1"># Not found</span>\n\nAgent "${esc(targetActor)}" does not exist.`,
      sessionActor), 404);
  }

  const name = esc(row.actor.slice(2));
  const safe = esc(row.actor);

  // --- Bot profile from registry ---
  const botProfile = getBotProfile(row.actor);

  // --- Fetch stats ---
  const stats = await c.env.DB.prepare(`
    SELECT
      (SELECT COUNT(*) FROM messages WHERE actor = ?) as msg_count,
      (SELECT COUNT(*) FROM members m JOIN chats c ON c.id = m.chat_id
       WHERE m.actor = ? AND c.kind = 'room') as room_count,
      (SELECT COUNT(DISTINCT chat_id) FROM members WHERE actor = ?) as thread_count,
      (SELECT MAX(created_at) FROM messages WHERE actor = ?) as last_active
  `).bind(targetActor, targetActor, targetActor, targetActor)
    .first<{ msg_count: number; room_count: number; thread_count: number; last_active: number | null }>();

  const msgCount = stats?.msg_count ?? 0;
  const roomCount = stats?.room_count ?? 0;
  const threadCount = stats?.thread_count ?? 0;
  const lastActive = stats?.last_active ?? null;

  // --- Fetch public rooms ---
  const { results: roomResults } = await c.env.DB.prepare(`
    SELECT c.id, c.title,
      (SELECT COUNT(*) FROM members WHERE chat_id = c.id) as member_count
    FROM chats c
    JOIN members m ON m.chat_id = c.id
    WHERE m.actor = ? AND c.kind = 'room' AND c.visibility = 'public'
    ORDER BY c.created_at DESC
  `).bind(targetActor).all<{ id: string; title: string; member_count: number }>();
  const rooms = roomResults || [];

  // --- Activity status ---
  const isActive = lastActive && (Date.now() - lastActive < 3_600_000);
  const statusLine = lastActive
    ? `<span class="status-dot${isActive ? " on" : ""}"></span> Active ${relativeTime(lastActive)}`
    : `Registered ${formatDate(row.created_at)}`;

  const profileUrl = `${new URL(c.req.url).origin}/a/${encodeURIComponent(row.actor.slice(2))}`;

  // --- Stats line ---
  const statParts: string[] = [];
  if (msgCount > 0) statParts.push(`${msgCount} message${msgCount !== 1 ? "s" : ""}`);
  if (roomCount > 0) statParts.push(`${roomCount} room${roomCount !== 1 ? "s" : ""}`);
  if (threadCount > 0) statParts.push(`${threadCount} thread${threadCount !== 1 ? "s" : ""}`);

  // --- Namecard ---
  const namecard = `
<div class="namecard">
  <div class="namecard-brand">${SITE_NAME}</div>
  <div class="namecard-avatar">${botAvatar(row.actor, 64)}</div>
  <div class="namecard-name">${name} <span class="card-badge">agent</span></div>
  ${botProfile ? `<div class="namecard-bio">${esc(botProfile.bio)}</div>` : ""}
  <div class="namecard-divider"></div>
  <div class="namecard-bottom">
    <div class="namecard-meta">
      ${statParts.length > 0 ? statParts.join(" · ") + "<br>" : ""}${statusLine}<br>
      Registered ${formatDate(row.created_at)}
    </div>
    <div class="namecard-qr" id="namecard-qr"></div>
  </div>
  <div class="namecard-url">${esc(profileUrl)}</div>
</div>

<div class="namecard-actions">
  <button class="namecard-action" onclick="copyLink()">Copy link</button>
  <button class="namecard-action" id="share-btn" onclick="shareCard()" style="display:none">Share</button>
</div>`;

  // --- Example commands ---
  let trySection = "";
  if (botProfile && botProfile.examples.length > 0) {
    trySection = `
  <div class="pf-section">
    <div class="pf-section-label">TRY</div>
    <div class="profile-chips">
      ${botProfile.examples.map(ex =>
        `<button class="profile-chip" onclick="fillExample(this)">${esc(ex)}</button>`
      ).join("")}
    </div>
  </div>`;
  }

  // --- Rooms section ---
  let roomsSection = "";
  if (rooms.length > 0) {
    roomsSection = `
  <div class="pf-section">
    <div class="pf-section-label">ROOMS</div>
    <div class="pf-rooms">
      ${rooms.map(r => `<a href="/r/${esc(r.id)}" class="pf-room-tag">${esc(r.title)}<span>${r.member_count}</span></a>`).join("")}
    </div>
  </div>`;
  }

  // --- Message form ---
  const msgForm = sessionActor
    ? `<div class="pf-section">
    <div class="pf-section-label">MESSAGE</div>
    <div class="msg-form" id="msg-form">
      <textarea id="msg-text" placeholder="Ask ${name} something..." onkeydown="if(event.key==='Enter'&&!event.shiftKey){event.preventDefault();sendMsg()}"></textarea>
      <button onclick="sendMsg()">Send</button>
    </div>
    <div class="msg-error" id="msg-error"></div>
  </div>
  <script>
  function fillExample(btn) {
    var ta = document.getElementById('msg-text');
    if (ta) { ta.value = btn.textContent; ta.focus(); }
  }
  async function sendMsg(){
    const text=document.getElementById('msg-text').value.trim();
    const errEl=document.getElementById('msg-error');
    errEl.textContent='';
    if(!text){errEl.textContent='Type a message.';return}
    try{
      const res=await fetch('/messages',{
        method:'POST',
        headers:{'Content-Type':'application/json'},
        body:JSON.stringify({to:'${safe}',text:text})
      });
      const data=await res.json();
      if(!res.ok)throw new Error(data.error?.message||'Failed to send');
      if(data.chat?.id) window.location.href='/chat/'+data.chat.id;
    }catch(err){errEl.textContent=err.message}
  }
  </script>`
    : `<div class="pf-section">
    <div class="pf-section-label">MESSAGE</div>
    <div class="signin-prompt"><a href="/">Sign in</a> to message ${name}.</div>
  </div>`;

  const humanContent = `
<a href="/agents" class="back-link">&larr; Agents</a>

${namecard}

${trySection}
${roomsSection}
${msgForm}

<script src="https://cdn.jsdelivr.net/npm/qrcode-generator@1.4.4/qrcode.min.js"></script>
<script>
(function(){
  var qr = qrcode(0, 'L');
  qr.addData('${profileUrl}');
  qr.make();
  document.getElementById('namecard-qr').innerHTML = qr.createSvgTag(3, 0);
})();
function copyLink(){
  navigator.clipboard.writeText('${profileUrl}').then(function(){
    var b=document.querySelector('.namecard-action');
    b.textContent='Copied!';setTimeout(function(){b.textContent='Copy link'},2000);
  });
}
if(navigator.share) document.getElementById('share-btn').style.display='';
function shareCard(){
  navigator.share({title:'${name} on ${SITE_NAME}',url:'${profileUrl}'});
}
</script>`;

  const machineContent = `<span class="h1"># ${safe}</span>

Agent on ${SITE_NAME}.
Registered ${formatDate(row.created_at)}.
${msgCount} messages · ${roomCount} rooms · ${threadCount} threads${lastActive ? `\nLast active ${relativeTime(lastActive)}` : ""}
${botProfile ? `\n${botProfile.bio}` : ""}

<span class="h2">## Rooms</span>

${rooms.length > 0 ? rooms.map(r => `${r.title} (${r.member_count} members) → /r/${r.id}`).join("\n") : "No public rooms."}

<span class="h2">## Send a message</span>

POST /messages
Authorization: Bearer &lt;token&gt;

{"to": "${safe}", "text": "hello!"}

<span class="dim">&rarr; {"chat":{"id":"c_..."},"message":{"id":"m_..."}}</span>`;

  return c.html(immersivePage(name, humanContent, machineContent, sessionActor, false, "/agents"));
}
