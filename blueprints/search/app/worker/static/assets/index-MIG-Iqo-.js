(function(){const t=document.createElement("link").relList;if(t&&t.supports&&t.supports("modulepreload"))return;for(const n of document.querySelectorAll('link[rel="modulepreload"]'))a(n);new MutationObserver(n=>{for(const r of n)if(r.type==="childList")for(const i of r.addedNodes)i.tagName==="LINK"&&i.rel==="modulepreload"&&a(i)}).observe(document,{childList:!0,subtree:!0});function s(n){const r={};return n.integrity&&(r.integrity=n.integrity),n.referrerPolicy&&(r.referrerPolicy=n.referrerPolicy),n.crossOrigin==="use-credentials"?r.credentials="include":n.crossOrigin==="anonymous"?r.credentials="omit":r.credentials="same-origin",r}function a(n){if(n.ep)return;n.ep=!0;const r=s(n);fetch(n.href,r)}})();class yt{routes=[];currentPath="";notFoundRenderer=null;addRoute(t,s){const a=t.split("/").filter(Boolean);this.routes.push({pattern:t,segments:a,renderer:s})}setNotFound(t){this.notFoundRenderer=t}navigate(t,s=!1){t!==this.currentPath&&(s?history.replaceState(null,"",t):history.pushState(null,"",t),this.resolve())}start(){window.addEventListener("popstate",()=>this.resolve()),document.addEventListener("click",t=>{const s=t.target.closest("a[data-link]");if(s){t.preventDefault();const a=s.getAttribute("href");a&&this.navigate(a)}}),this.resolve()}getCurrentPath(){return this.currentPath}resolve(){const t=new URL(window.location.href),s=t.pathname,a=bt(t.search);this.currentPath=s+t.search;for(const n of this.routes){const r=wt(n.segments,s);if(r!==null){n.renderer(r,a);return}}this.notFoundRenderer&&this.notFoundRenderer({},a)}}function wt(e,t){const s=t.split("/").filter(Boolean);if(e.length===0&&s.length===0)return{};if(e.length!==s.length)return null;const a={};for(let n=0;n<e.length;n++){const r=e[n],i=s[n];if(r.startsWith(":"))a[r.slice(1)]=decodeURIComponent(i);else if(r!==i)return null}return a}function bt(e){const t={};return new URLSearchParams(e).forEach((a,n)=>{t[n]=a}),t}const xe="/api";async function p(e,t){let s=`${xe}${e}`;if(t){const n=new URLSearchParams;Object.entries(t).forEach(([i,o])=>{o!==void 0&&o!==""&&o!==null&&n.set(i,o)});const r=n.toString();r&&(s+=`?${r}`)}const a=await fetch(s);if(!a.ok)throw new Error(`API error: ${a.status} ${a.statusText}`);return a.json()}async function U(e,t){const s=await fetch(`${xe}${e}`,{method:"POST",headers:{"Content-Type":"application/json"},body:t?JSON.stringify(t):void 0});if(!s.ok)throw new Error(`API error: ${s.status} ${s.statusText}`);return s.json()}async function je(e,t){const s=await fetch(`${xe}${e}`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify(t)});if(!s.ok)throw new Error(`API error: ${s.status} ${s.statusText}`);return s.json()}async function ae(e,t){const s=await fetch(`${xe}${e}`,{method:"DELETE",headers:t?{"Content-Type":"application/json"}:void 0,body:t?JSON.stringify(t):void 0});if(!s.ok)throw new Error(`API error: ${s.status} ${s.statusText}`);return s.json()}function Pe(e,t){const s={q:e};return t&&(t.page!==void 0&&(s.page=String(t.page)),t.per_page!==void 0&&(s.per_page=String(t.per_page)),t.time_range&&(s.time_range=t.time_range),t.region&&(s.region=t.region),t.language&&(s.language=t.language),t.safe_search&&(s.safe_search=t.safe_search),t.site&&(s.site=t.site),t.exclude_site&&(s.exclude_site=t.exclude_site),t.lens&&(s.lens=t.lens),t.verbatim&&(s.verbatim="1")),s}const f={search(e,t){return p("/search",Pe(e,t))},searchImages(e,t){const s={q:e};return t&&(t.page!==void 0&&(s.page=String(t.page)),t.per_page!==void 0&&(s.per_page=String(t.per_page)),t.size&&t.size!=="any"&&(s.size=t.size),t.color&&t.color!=="any"&&(s.color=t.color),t.type&&t.type!=="any"&&(s.type=t.type),t.aspect&&t.aspect!=="any"&&(s.aspect=t.aspect),t.time&&t.time!=="any"&&(s.time=t.time),t.rights&&t.rights!=="any"&&(s.rights=t.rights),t.filetype&&t.filetype!=="any"&&(s.filetype=t.filetype),t.safe&&(s.safe=t.safe)),p("/search/images",s)},reverseImageSearch(e){return U("/search/images/reverse",{url:e})},reverseImageSearchByUpload(e){return U("/search/images/reverse",{image_data:e})},searchVideos(e,t){const s={q:e};if(t){t.page!==void 0&&(s.page=String(t.page)),t.per_page!==void 0&&(s.per_page=String(t.per_page));const a=t.time??t.time_range;a&&(s.time=a),t.duration&&(s.duration=t.duration),t.quality&&(s.quality=t.quality),t.source&&(s.source=t.source),t.sort&&(s.sort=t.sort),t.cc!==void 0&&(s.cc=t.cc?"1":"0"),t.region&&(s.region=t.region),t.language&&(s.language=t.language),t.safe_search&&(s.safe_search=t.safe_search),t.site&&(s.site=t.site),t.exclude_site&&(s.exclude_site=t.exclude_site),t.lens&&(s.lens=t.lens),t.verbatim&&(s.verbatim="1")}return p("/search/videos",s)},searchNews(e,t){return p("/search/news",Pe(e,t))},searchMusic(e,t){const s=new URLSearchParams({q:e});return t?.page&&s.set("page",String(t.page)),p(`/search/music?${s}`)},searchScience(e,t){const s=new URLSearchParams({q:e});return t?.page&&s.set("page",String(t.page)),t?.per_page&&s.set("per_page",String(t.per_page)),p(`/search/science?${s}`)},searchMaps(e){const t=new URLSearchParams({q:e});return p(`/search/maps?${t}`)},searchCode(e,t){const s=new URLSearchParams({q:e});return t?.page&&s.set("page",String(t.page)),t?.per_page&&s.set("per_page",String(t.per_page)),p(`/search/code?${s}`)},searchSocial(e,t){const s=new URLSearchParams({q:e});return t?.page&&s.set("page",String(t.page)),p(`/search/social?${s}`)},suggest(e){return p("/suggest",{q:e})},trending(){return p("/suggest/trending")},calculate(e){return p("/instant/calculate",{q:e})},convert(e){return p("/instant/convert",{q:e})},currency(e){return p("/instant/currency",{q:e})},weather(e){return p("/instant/weather",{q:e})},define(e){return p("/instant/define",{q:e})},time(e){return p("/instant/time",{q:e})},knowledge(e){return p(`/knowledge/${encodeURIComponent(e)}`)},getPreferences(){return p("/preferences")},setPreference(e,t){return U("/preferences",{domain:e,action:t})},deletePreference(e){return ae(`/preferences/${encodeURIComponent(e)}`)},getLenses(){return p("/lenses")},createLens(e){return U("/lenses",e)},deleteLens(e){return ae(`/lenses/${encodeURIComponent(e)}`)},getHistory(){return p("/history")},clearHistory(){return ae("/history")},deleteHistoryItem(e){return ae(`/history/${encodeURIComponent(e)}`)},getSettings(){return p("/settings")},updateSettings(e){return je("/settings",e)},getBangs(){return p("/bangs")},parseBang(e){return p("/bangs/parse",{q:e})},getRelated(e){return p("/related",{q:e})},newsHome(){return p("/news/home")},newsCategory(e,t=1){return p(`/news/category/${e}`,{page:String(t)})},newsSearch(e,t){const s={q:e};return t?.page&&(s.page=String(t.page)),t?.time&&(s.time=t.time),t?.source&&(s.source=t.source),p("/news/search",s)},newsStory(e){return p(`/news/story/${e}`)},newsLocal(e){const t={};return e&&(t.city=e.city,e.state&&(t.state=e.state),t.country=e.country),p("/news/local",t)},newsFollowing(){return p("/news/following")},newsPreferences(){return p("/news/preferences")},updateNewsPreferences(e){return je("/news/preferences",e)},followNews(e,t){return U("/news/follow",{type:e,id:t})},unfollowNews(e,t){return ae("/news/follow",{type:e,id:t})},hideNewsSource(e){return U("/news/hide",{source:e})},setNewsLocation(e){return U("/news/location",e)},recordNewsRead(e,t){return U("/news/read",{article:e,duration:t})}};async function xt(){try{const e=await fetch("/api/suggest/trending");if(!e.ok)return[];const t=await e.json();return Array.isArray(t)?t.map(s=>typeof s=="string"?s:s.text):t.suggestions||[]}catch{return[]}}function $t(e){let t={...e};const s=new Set;return{get(){return t},set(a){t={...t,...a},s.forEach(n=>n(t))},subscribe(a){return s.add(a),()=>{s.delete(a)}}}}const ot="mizu_search_state";function kt(){try{const e=localStorage.getItem(ot);if(e)return JSON.parse(e)}catch{}return{recentSearches:[],settings:{safe_search:"moderate",results_per_page:10,region:"auto",language:"en",theme:"light",open_in_new_tab:!1,show_thumbnails:!0}}}const Q=$t(kt());Q.subscribe(e=>{try{localStorage.setItem(ot,JSON.stringify(e))}catch{}});function O(e){const t=Q.get(),s=[e,...t.recentSearches.filter(a=>a!==e)].slice(0,20);Q.set({recentSearches:s})}const Ct='<svg class="search-icon" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>',St='<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>',Lt='<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2a3 3 0 0 0-3 3v7a3 3 0 0 0 6 0V5a3 3 0 0 0-3-3Z"/><path d="M19 10v2a7 7 0 0 1-14 0v-2"/><line x1="12" x2="12" y1="19" y2="22"/></svg>',It='<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14.5 4h-5L7 7H4a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2h-3l-2.5-3z"/><circle cx="12" cy="13" r="3"/></svg>',Et='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/><path d="M12 7v5l4 2"/></svg>',_t='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M13 2 3 14h9l-1 8 10-12h-9l1-8z"/></svg>',Bt='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>';function E(e){const t=e.size==="lg"?"search-box-lg":"search-box-sm",s=e.initialValue?Fe(e.initialValue):"",a=e.initialValue?"":"hidden",n=e.placeholder||"Search the web...";return`
    <div class="search-box-wrapper" id="search-box-wrapper">
      <div class="search-box ${t}" id="search-box">
        ${Ct}
        <input
          id="search-input"
          type="text"
          value="${s}"
          placeholder="${Fe(n)}"
          autocomplete="off"
          spellcheck="false"
          ${e.autofocus?"autofocus":""}
        />
        <div class="search-box-actions">
          <button id="search-clear-btn" class="search-box-btn ${a}" type="button" aria-label="Clear search">
            ${St}
          </button>
          <button id="voice-search-btn" class="search-box-btn" type="button" aria-label="Voice search">
            ${Lt}
          </button>
          <button id="camera-search-btn" class="search-box-btn" type="button" aria-label="Search by image">
            ${It}
          </button>
        </div>
      </div>
      <div id="autocomplete-dropdown" class="autocomplete-dropdown hidden"></div>
    </div>
  `}function _(e){const t=document.getElementById("search-input"),s=document.getElementById("search-clear-btn"),a=document.getElementById("autocomplete-dropdown"),n=document.getElementById("search-box-wrapper");if(!t||!s||!a||!n)return;let r=null,i=[],o=-1,l=!1;function c(d){if(i=d,o=-1,d.length===0){h();return}l=!0;const u=d.filter(b=>b.type==="recent"),C=d.filter(b=>b.type==="suggestion"),S=d.filter(b=>b.type==="bang");let w="";if(u.length>0&&(w+='<div class="autocomplete-section">',w+='<div class="autocomplete-section-title">Recent</div>',w+=u.map((b,M)=>v(b,M)).join(""),w+="</div>"),C.length>0){const b=u.length;w+=C.map((M,se)=>v(M,b+se)).join("")}if(S.length>0){const b=u.length+C.length;w+='<div class="autocomplete-section">',w+='<div class="autocomplete-section-title">Quick Actions</div>',w+=S.map((M,se)=>v(M,b+se)).join(""),w+="</div>"}a.innerHTML=w,a.classList.remove("hidden"),a.querySelectorAll(".autocomplete-item").forEach(b=>{b.addEventListener("mousedown",M=>{M.preventDefault();const se=parseInt(b.dataset.index||"0");k(se)}),b.addEventListener("mouseenter",()=>{const M=parseInt(b.dataset.index||"0");y(M)})})}function v(d,u){return`
      <div class="autocomplete-item" data-index="${u}">
        <span class="item-icon">${d.icon}</span>
        ${d.prefix?`<span class="bang-trigger">${Ue(d.prefix)}</span>`:""}
        <span>${Ue(d.text)}</span>
      </div>
    `}function h(){l=!1,a.classList.add("hidden"),a.innerHTML="",i=[],o=-1}function y(d){o=d,a.querySelectorAll(".autocomplete-item").forEach((u,C)=>{u.classList.toggle("active",C===d)})}function k(d){const u=i[d];u&&(u.type==="bang"&&u.prefix?(t.value=u.prefix+" ",t.focus(),B(u.prefix+" ")):(t.value=u.text,h(),P(u.text)))}function P(d){const u=d.trim();u&&(h(),e(u))}async function B(d){const u=d.trim();if(!u){$();return}if(u.startsWith("!"))try{const S=(await f.getBangs()).filter(w=>w.trigger.startsWith(u)||w.name.toLowerCase().includes(u.slice(1).toLowerCase())).slice(0,8);if(S.length>0){c(S.map(w=>({text:w.name,type:"bang",icon:_t,prefix:w.trigger})));return}}catch{}try{const C=await f.suggest(u);if(t.value.trim()!==u)return;const S=C.map(w=>({text:w.text,type:"suggestion",icon:Bt}));S.length===0?$(u):c(S)}catch{$(u)}}function $(d){let C=Q.get().recentSearches;if(d&&(C=C.filter(S=>S.toLowerCase().includes(d.toLowerCase()))),C.length===0){h();return}c(C.slice(0,8).map(S=>({text:S,type:"recent",icon:Et})))}t.addEventListener("input",()=>{const d=t.value;s.classList.toggle("hidden",d.length===0),r&&clearTimeout(r),r=setTimeout(()=>B(d),150)}),t.addEventListener("focus",()=>{t.value.trim()?B(t.value):$()}),t.addEventListener("keydown",d=>{if(!l){if(d.key==="Enter"){P(t.value);return}if(d.key==="ArrowDown"){B(t.value);return}return}switch(d.key){case"ArrowDown":d.preventDefault(),y(Math.min(o+1,i.length-1));break;case"ArrowUp":d.preventDefault(),y(Math.max(o-1,-1));break;case"Enter":d.preventDefault(),o>=0?k(o):P(t.value);break;case"Escape":h();break;case"Tab":h();break}}),t.addEventListener("blur",()=>{setTimeout(()=>h(),200)}),s.addEventListener("click",()=>{t.value="",s.classList.add("hidden"),t.focus(),$()});const pe=document.getElementById("voice-search-btn");pe&&Mt(pe,t,d=>{t.value=d,s.classList.remove("hidden"),P(d)});const te=document.getElementById("camera-search-btn");te&&te.addEventListener("click",()=>{const d=document.getElementById("reverse-modal");d?d.classList.remove("hidden"):window.dispatchEvent(new CustomEvent("router:navigate",{detail:{path:"/images?reverse=1"}}))})}function Mt(e,t,s){const a=window.SpeechRecognition||window.webkitSpeechRecognition;if(!a){e.style.display="none";return}let n=!1,r=null;e.addEventListener("click",()=>{n?o():i()});function i(){r=new a,r.continuous=!1,r.interimResults=!0,r.lang="en-US",r.onstart=()=>{n=!0,e.classList.add("listening")},r.onresult=l=>{const c=Array.from(l.results).map(v=>v[0].transcript).join("");t.value=c,l.results[0].isFinal&&(o(),s(c))},r.onerror=l=>{console.error("Speech recognition error:",l.error),o(),l.error==="not-allowed"&&alert("Microphone access denied. Please allow microphone access to use voice search.")},r.onend=()=>{o()};try{r.start()}catch(l){console.error("Failed to start speech recognition:",l),o()}}function o(){if(n=!1,e.classList.remove("listening"),r){try{r.stop()}catch{}r=null}}}function Ue(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function Fe(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const Tt='<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 6 13.5 15.5 8.5 10.5 1 18"/><polyline points="17 6 23 6 23 12"/></svg>',At='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="18" height="18" x="3" y="3" rx="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"/></svg>',Nt='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 22h16a2 2 0 0 0 2-2V4a2 2 0 0 0-2-2H8a2 2 0 0 0-2 2v16a2 2 0 0 1-2 2Zm0 0a2 2 0 0 1-2-2v-9c0-1.1.9-2 2-2h2"/></svg>',Ht='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="16" height="20" x="4" y="2" rx="2"/><line x1="8" x2="16" y1="6" y2="6"/><line x1="16" x2="16" y1="14" y2="18"/></svg>',Rt='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M8 3 4 7l4 4"/><path d="M4 7h16"/><path d="m16 21 4-4-4-4"/><path d="M20 17H4"/></svg>',Ot='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" x2="12" y1="2" y2="22"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>',qt='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="M2 12h2"/><path d="M20 12h2"/></svg>',jt='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>',Pt='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1 0-5H20"/></svg>',Ut=[{trigger:"!g",label:"Google"},{trigger:"!yt",label:"YouTube"},{trigger:"!gh",label:"GitHub"},{trigger:"!w",label:"Wikipedia"},{trigger:"!r",label:"Reddit"}],Ft=[{label:"Calculator",icon:Ht,query:"2+2",colorClass:"blue"},{label:"Conversion",icon:Rt,query:"10 miles in km",colorClass:"green"},{label:"Currency",icon:Ot,query:"100 USD to EUR",colorClass:"yellow"},{label:"Weather",icon:qt,query:"weather New York",colorClass:"blue"},{label:"Time",icon:jt,query:"time in Tokyo",colorClass:"green"},{label:"Define",icon:Pt,query:"define serendipity",colorClass:"red"}];function Vt(){return`
    <div class="home-container">
      <main class="home-main">
        <!-- Logo -->
        <h1 class="home-logo">
          <span style="color: #2563eb">M</span><span style="color: #ef4444">i</span><span style="color: #f59e0b">z</span><span style="color: #22c55e">u</span>
        </h1>
        <p class="home-tagline">Privacy-first search</p>

        <!-- Search Box -->
        <div class="home-search">
          ${E({size:"lg",autofocus:!0})}
        </div>

        <!-- Search Buttons -->
        <div class="home-buttons">
          <button id="home-search-btn" class="btn btn-secondary">
            Mizu Search
          </button>
          <button id="home-lucky-btn" class="btn btn-ghost">
            I'm Feeling Lucky
          </button>
        </div>

        <!-- Trending Searches -->
        <div class="home-trending hidden" id="trending-container">
          <p class="home-trending-title">Trending</p>
          <div class="home-trending-chips" id="trending-chips"></div>
        </div>

        <!-- Bang Shortcuts -->
        <div class="home-bangs">
          ${Ut.map(e=>`
            <button class="bang-chip" data-bang="${e.trigger}">
              <span class="bang-trigger">${fe(e.trigger)}</span>
              <span class="bang-label">${fe(e.label)}</span>
            </button>
          `).join("")}
        </div>

        <!-- Instant Answers -->
        <div class="home-instant">
          <p class="home-instant-title">Instant Answers</p>
          <div class="home-instant-buttons">
            ${Ft.map(e=>`
              <button class="instant-btn ${e.colorClass}" data-query="${Gt(e.query)}">
                ${e.icon}
                <span>${fe(e.label)}</span>
              </button>
            `).join("")}
          </div>
        </div>

        <!-- Quick Links -->
        <div class="home-links">
          <a href="/images" data-link>
            ${At}
            Images
          </a>
          <a href="/news" data-link>
            ${Nt}
            News
          </a>
        </div>
      </main>

      <!-- Footer -->
      <footer class="home-footer">
        <div class="home-footer-links">
          <span>Use <strong>!bangs</strong> to search other sites</span>
          <span>&middot;</span>
          <a href="/settings" data-link>Settings</a>
          <span>&middot;</span>
          <a href="/history" data-link>History</a>
        </div>
      </footer>
    </div>
  `}function zt(e){_(a=>{e.navigate(`/search?q=${encodeURIComponent(a)}`)}),document.getElementById("home-search-btn")?.addEventListener("click",()=>{const n=document.getElementById("search-input")?.value?.trim();n&&e.navigate(`/search?q=${encodeURIComponent(n)}`)}),document.getElementById("home-lucky-btn")?.addEventListener("click",()=>{const n=document.getElementById("search-input")?.value?.trim();n&&e.navigate(`/search?q=${encodeURIComponent(n)}&lucky=1`)}),document.querySelectorAll(".bang-chip").forEach(a=>{a.addEventListener("click",()=>{const n=a.dataset.bang||"",r=document.getElementById("search-input");r&&(r.value=n+" ",r.focus())})}),document.querySelectorAll(".instant-btn").forEach(a=>{a.addEventListener("click",()=>{const n=a.dataset.query||"";n&&e.navigate(`/search?q=${encodeURIComponent(n)}`)})}),Dt()}async function Dt(){try{const e=await xt(),t=document.getElementById("trending-container"),s=document.getElementById("trending-chips");t&&s&&e.length>0&&(s.innerHTML=e.slice(0,8).map(a=>`
        <a href="/search?q=${encodeURIComponent(a)}" data-link class="trending-chip">
          ${Tt}
          ${fe(a)}
        </a>
      `).join(""),t.classList.remove("hidden"))}catch{}}function fe(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function Gt(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const Wt='<svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="5" r="2"/><circle cx="12" cy="12" r="2"/><circle cx="12" cy="19" r="2"/></svg>',Kt='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M7 10v12"/><path d="M15 5.88 14 10h5.83a2 2 0 0 1 1.92 2.56l-2.33 8A2 2 0 0 1 17.5 22H4a2 2 0 0 1-2-2v-8a2 2 0 0 1 2-2h2.76a2 2 0 0 0 1.79-1.11L12 2h0a3.13 3.13 0 0 1 3 3.88Z"/></svg>',Yt='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 14V2"/><path d="M9 18.12 10 14H4.17a2 2 0 0 1-1.92-2.56l2.33-8A2 2 0 0 1 6.5 2H20a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2h-2.76a2 2 0 0 0-1.79 1.11L12 22h0a3.13 3.13 0 0 1-3-3.88Z"/></svg>',Qt='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="m4.9 4.9 14.2 14.2"/></svg>',Jt='<svg class="favicon-fallback" style="display:none" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M2 12h20M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>';function Ve(e,t){const s=e.favicon||`https://www.google.com/s2/favicons?domain=${encodeURIComponent(e.domain)}&sz=32`,a=ts(e.url),n=e.published?ss(e.published):"",r=e.snippet||"",i=e.thumbnail?`<img src="${V(e.thumbnail.url)}" alt="" class="w-[120px] h-[80px] rounded-lg object-cover flex-shrink-0 ml-4" loading="lazy" />`:"",o=e.sitelinks&&e.sitelinks.length>0?`<div class="sitelinks-grid">
        ${e.sitelinks.slice(0,4).map(h=>`
          <div class="sitelink">
            <a href="${V(h.url)}" target="_blank" rel="noopener">${A(h.title)}</a>
          </div>
        `).join("")}
       </div>`:"",l=e.metadata||{},c=typeof l.rating=="number"?l.rating:null,v=c!==null?`
    <div class="rich-snippet">
      <span class="rating-stars">${"★".repeat(Math.round(c))}${"☆".repeat(5-Math.round(c))}</span>
      <span class="rating-value">${c.toFixed(1)}</span>
      ${l.reviewCount?`<span class="rating-count">(${Zt(l.reviewCount)} reviews)</span>`:""}
      ${l.price?`<span class="price">${A(String(l.price))}</span>`:""}
    </div>
  `:"";return`
    <div class="search-result" data-result-index="${t}" data-domain="${V(e.domain)}">
      <div class="result-url">
        <div class="favicon">
          <img src="${V(s)}" alt="" loading="lazy" onerror="this.style.display='none'; this.nextElementSibling.style.display='block';" />
          ${Jt}
        </div>
        <div>
          <span class="text-sm">${A(e.domain)}</span>
          <span class="breadcrumbs">${a}</span>
        </div>
      </div>
      <div class="flex items-start">
        <div class="flex-1">
          <div class="result-title">
            <a href="${V(e.url)}" target="_blank" rel="noopener">${A(e.title)}</a>
          </div>
          ${v}
          <div class="result-snippet">
            ${n?`<span class="result-date">${A(n)} — </span>`:""}${r}
          </div>
          ${o}
        </div>
        ${i}
      </div>
      <button class="result-menu-btn" data-menu-index="${t}" aria-label="More options">
        ${Wt}
      </button>
      <div id="domain-menu-${t}" class="domain-menu hidden"></div>
    </div>
  `}function Zt(e){return e>=1e6?(e/1e6).toFixed(1).replace(/\.0$/,"")+"M":e>=1e3?(e/1e3).toFixed(1).replace(/\.0$/,"")+"K":e.toLocaleString()}function Xt(){document.querySelectorAll(".result-menu-btn").forEach(e=>{e.addEventListener("click",t=>{t.stopPropagation();const s=e.dataset.menuIndex,a=document.getElementById(`domain-menu-${s}`),r=e.closest(".search-result")?.dataset.domain||"";if(!a)return;if(!a.classList.contains("hidden")){a.classList.add("hidden");return}document.querySelectorAll(".domain-menu").forEach(o=>o.classList.add("hidden")),a.innerHTML=`
        <button class="domain-menu-item boost" data-action="boost" data-domain="${V(r)}">
          ${Kt}
          <span>Boost ${A(r)}</span>
        </button>
        <button class="domain-menu-item lower" data-action="lower" data-domain="${V(r)}">
          ${Yt}
          <span>Lower ${A(r)}</span>
        </button>
        <button class="domain-menu-item block" data-action="block" data-domain="${V(r)}">
          ${Qt}
          <span>Block ${A(r)}</span>
        </button>
      `,a.classList.remove("hidden"),a.querySelectorAll(".domain-menu-item").forEach(o=>{o.addEventListener("click",async()=>{const l=o.dataset.action||"",c=o.dataset.domain||"";try{await f.setPreference(c,l),a.classList.add("hidden"),es(`${l.charAt(0).toUpperCase()+l.slice(1)}ed ${c}`)}catch(v){console.error("Failed to set preference:",v)}})});const i=o=>{!a.contains(o.target)&&o.target!==e&&(a.classList.add("hidden"),document.removeEventListener("click",i))};setTimeout(()=>document.addEventListener("click",i),0)})})}function es(e){const t=document.getElementById("toast");t&&t.remove();const s=document.createElement("div");s.id="toast",s.className="fixed bottom-6 left-1/2 -translate-x-1/2 bg-primary text-white px-5 py-3 rounded-lg shadow-lg text-sm z-50 transition-opacity duration-300",s.textContent=e,document.body.appendChild(s),setTimeout(()=>{s.style.opacity="0",setTimeout(()=>s.remove(),300)},2e3)}function ts(e){try{const s=new URL(e).pathname.split("/").filter(Boolean);return s.length===0?"":" > "+s.map(a=>A(decodeURIComponent(a))).join(" > ")}catch{return""}}function ss(e){try{const t=new Date(e),a=new Date().getTime()-t.getTime(),n=Math.floor(a/(1e3*60*60*24));return n===0?"Today":n===1?"1 day ago":n<7?`${n} days ago`:n<30?`${Math.floor(n/7)} weeks ago`:t.toLocaleDateString("en-US",{month:"short",day:"numeric",year:"numeric"})}catch{return e}}function A(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function V(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const as='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="16" height="20" x="4" y="2" rx="2"/><line x1="8" x2="16" y1="6" y2="6"/><line x1="16" x2="16" y1="14" y2="18"/><path d="M16 10h.01"/><path d="M12 10h.01"/><path d="M8 10h.01"/><path d="M12 14h.01"/><path d="M8 14h.01"/><path d="M12 18h.01"/><path d="M8 18h.01"/></svg>',ns='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M8 3 4 7l4 4"/><path d="M4 7h16"/><path d="m16 21 4-4-4-4"/><path d="M20 17H4"/></svg>',rs='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" x2="12" y1="2" y2="22"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>',is='<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#FBBC05" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="m4.93 4.93 1.41 1.41"/><path d="m17.66 17.66 1.41 1.41"/><path d="M2 12h2"/><path d="M20 12h2"/><path d="m6.34 17.66-1.41 1.41"/><path d="m19.07 4.93-1.41 1.41"/></svg>',os='<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#5f6368" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17.5 19H9a7 7 0 1 1 6.71-9h1.79a4.5 4.5 0 1 1 0 9Z"/></svg>',ls='<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#4285F4" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 14.899A7 7 0 1 1 15.71 8h1.79a4.5 4.5 0 0 1 2.5 8.242"/><path d="M16 14v6"/><path d="M8 14v6"/><path d="M12 16v6"/></svg>',cs='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1 0-5H20"/></svg>',ds='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>';function us(e){switch(e.type){case"calculator":return hs(e);case"unit_conversion":return ps(e);case"currency":return gs(e);case"weather":return vs(e);case"definition":return ms(e);case"time":return fs(e);default:return ys(e)}}function hs(e){const t=e.data||{},s=t.expression||e.query||"",a=t.formatted||t.result||e.result||"";return`
    <div class="instant-card calculator">
      <div class="flex items-center gap-2 text-tertiary">
        ${as}
        <span class="instant-type">Calculator</span>
      </div>
      <div class="instant-result">${g(String(a))}</div>
      <div class="instant-sub">${g(s)}</div>
    </div>
  `}function ps(e){const t=e.data||{},s=t.from_value??"",a=t.from_unit??"",n=t.to_value??"",r=t.to_unit??"",i=t.category??"";return`
    <div class="instant-card conversion">
      <div class="flex items-center gap-2 text-tertiary">
        ${ns}
        <span class="instant-type">Unit Conversion${i?` - ${g(i)}`:""}</span>
      </div>
      <div class="instant-result">${g(String(n))} ${g(r)}</div>
      <div class="instant-sub">${g(String(s))} ${g(a)}</div>
    </div>
  `}function gs(e){const t=e.data||{},s=t.from_value??"",a=t.from_currency??"",n=t.to_value??"",r=t.to_currency??"",i=t.rate??"";return`
    <div class="instant-card currency">
      <div class="flex items-center gap-2 text-tertiary">
        ${rs}
        <span class="instant-type">Currency</span>
      </div>
      <div class="instant-result">${g(String(n))} ${g(r)}</div>
      ${i?`<div class="currency-rate">1 ${g(a)} = ${g(String(i))} ${g(r)}</div>`:""}
      <div class="currency-updated">${g(String(s))} ${g(a)}</div>
    </div>
  `}function vs(e){const t=e.data||{},s=t.location||"",a=t.temperature??"",n=(t.condition||"").toLowerCase(),r=t.humidity||"",i=t.wind||"";let o=is;n.includes("cloud")||n.includes("overcast")?o=os:(n.includes("rain")||n.includes("drizzle")||n.includes("storm"))&&(o=ls);const l=[];return r&&l.push(`Humidity: ${g(r)}`),i&&l.push(`Wind: ${g(i)}`),`
    <div class="instant-card weather">
      <div class="weather-main">
        <div class="weather-icon">${o}</div>
        <div class="weather-temp">${g(String(a))}<sup>°</sup></div>
      </div>
      <div class="weather-details">
        <div class="weather-condition">${g(t.condition||"")}</div>
        <div class="weather-location">${g(s)}</div>
        ${l.length>0?`<div class="weather-meta">${l.join(" · ")}</div>`:""}
      </div>
    </div>
  `}function ms(e){const t=e.data||{},s=t.word||e.query||"",a=t.phonetic||"",n=t.part_of_speech||"",r=t.definitions||[],i=t.synonyms||[],o=t.example||"";return`
    <div class="instant-card definition">
      <div class="flex items-center gap-2 text-tertiary">
        ${cs}
        <span class="instant-type">Definition</span>
      </div>
      <div class="word">
        <span>${g(s)}</span>
        <button class="pronunciation-btn" title="Listen to pronunciation" aria-label="Listen to pronunciation">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"/><path d="M15.54 8.46a5 5 0 0 1 0 7.07"/></svg>
        </button>
      </div>
      ${a?`<div class="phonetic">${g(a)}</div>`:""}
      ${n?`<div class="part-of-speech">${g(n)}</div>`:""}
      ${r.length>0?r.map((c,v)=>`<div class="definition-text">${v+1}. ${g(c)}</div>`).join(""):""}
      ${o?`<div class="definition-example">"${g(o)}"</div>`:""}
      ${i.length>0?`<div class="mt-3 text-sm">
              <span class="text-tertiary">Synonyms: </span>
              <span class="text-secondary">${i.map(c=>g(c)).join(", ")}</span>
             </div>`:""}
    </div>
  `}function fs(e){const t=e.data||{},s=t.location||"",a=t.time||"",n=t.date||"",r=t.timezone||"";return`
    <div class="instant-card time">
      <div class="flex items-center gap-2 text-tertiary">
        ${ds}
        <span class="instant-type">Time</span>
      </div>
      <div class="time-display">${g(a)}</div>
      <div class="time-location">${g(s)}</div>
      <div class="time-date">${g(n)}</div>
      ${r?`<div class="time-timezone">${g(r)}</div>`:""}
    </div>
  `}function ys(e){return`
    <div class="instant-card">
      <div class="instant-type">${g(e.type)}</div>
      <div class="instant-result">${g(e.result)}</div>
    </div>
  `}function g(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}const ws='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" x2="21" y1="14" y2="3"/></svg>';function bs(e){const t=e.image?`<img class="kp-image" src="${Ce(e.image)}" alt="${Ce(e.title)}" loading="lazy" onerror="this.style.display='none'" />`:"",s=e.facts&&e.facts.length>0?`<table class="kp-facts">
          <tbody>
            ${e.facts.map(r=>`
              <tr>
                <td class="fact-label">${K(r.label)}</td>
                <td class="fact-value">${K(r.value)}</td>
              </tr>
            `).join("")}
          </tbody>
        </table>`:"",a=e.links&&e.links.length>0?`<div class="kp-links">
          ${e.links.map(r=>`
            <a class="kp-link" href="${Ce(r.url)}" target="_blank" rel="noopener">
              ${ws}
              <span>${K(r.title)}</span>
            </a>
          `).join("")}
        </div>`:"",n=e.source?`<div class="kp-source">Source: ${K(e.source)}</div>`:"";return`
    <div class="knowledge-panel" id="knowledge-panel">
      ${t}
      <div class="kp-title">${K(e.title)}</div>
      ${e.subtitle?`<div class="kp-subtitle">${K(e.subtitle)}</div>`:""}
      <div class="kp-description">${K(e.description)}</div>
      ${s}
      ${a}
      ${n}
    </div>
  `}function K(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function Ce(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const xs='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m15 18-6-6 6-6"/></svg>',$s='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"/></svg>';function ks(e){const{currentPage:t,hasMore:s,totalResults:a,perPage:n}=e,r=Math.min(Math.ceil(a/n),100);if(r<=1)return"";let i=Math.max(1,t-4),o=Math.min(r,i+9);o-i<9&&(i=Math.max(1,o-9));const l=[];for(let y=i;y<=o;y++)l.push(y);const c=Cs(t),v=t<=1?"disabled":"",h=!s&&t>=r?"disabled":"";return`
    <div class="pagination" id="pagination">
      <div class="flex flex-col items-center gap-3">
        ${c}
        <div class="flex items-center gap-1">
          <button class="pagination-btn ${v}" data-page="${t-1}" ${t<=1?"disabled":""} aria-label="Previous page">
            ${xs}
          </button>
          ${l.map(y=>`
            <button class="pagination-btn ${y===t?"active":""}" data-page="${y}">
              ${y}
            </button>
          `).join("")}
          <button class="pagination-btn ${h}" data-page="${t+1}" ${!s&&t>=r?"disabled":""} aria-label="Next page">
            ${$s}
          </button>
        </div>
      </div>
    </div>
  `}function Cs(e){const t=["#4285F4","#EA4335","#FBBC05","#4285F4","#34A853","#EA4335"],s=["M","i","z","u"],a=Math.min(e-1,6);let n=[s[0]];for(let r=0;r<1+a;r++)n.push("i");n.push("z");for(let r=0;r<1+a;r++)n.push("u");return`
    <div class="flex items-center text-2xl font-semibold tracking-wide select-none">
      ${n.map((r,i)=>`<span style="color: ${t[i%t.length]}">${r}</span>`).join("")}
    </div>
  `}function Ss(e){const t=document.getElementById("pagination");t&&t.querySelectorAll(".pagination-btn").forEach(s=>{s.addEventListener("click",()=>{const a=parseInt(s.dataset.page||"1");isNaN(a)||s.disabled||(e(a),window.scrollTo({top:0,behavior:"smooth"}))})})}const Ls='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>',Is='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="18" x="3" y="3" rx="2" ry="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"/></svg>',Es='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m22 8-6 4 6 4V8Z"/><rect width="14" height="12" x="2" y="6" rx="2" ry="2"/></svg>',_s='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 22h16a2 2 0 0 0 2-2V4a2 2 0 0 0-2-2H8a2 2 0 0 0-2 2v16a2 2 0 0 1-2 2Zm0 0a2 2 0 0 1-2-2v-9c0-1.1.9-2 2-2h2"/><path d="M18 14h-8"/><path d="M15 18h-5"/><path d="M10 6h8v4h-8V6Z"/></svg>',Bs='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10 2v7.527a2 2 0 0 1-.211.896L4.72 20.55a1 1 0 0 0 .9 1.45h12.76a1 1 0 0 0 .9-1.45l-5.069-10.127A2 2 0 0 1 14 9.527V2"/><path d="M8.5 2h7"/><path d="M7 16h10"/></svg>',Ms='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>',Ts='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/></svg>',As='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>',Ns='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/><circle cx="12" cy="10" r="3"/></svg>',Hs=[{id:"all",label:"All",icon:Ls,href:e=>`/search?q=${e}`},{id:"images",label:"Images",icon:Is,href:e=>`/images?q=${e}`},{id:"videos",label:"Videos",icon:Es,href:e=>`/videos?q=${e}`},{id:"news",label:"News",icon:_s,href:e=>`/news?q=${e}`},{id:"code",label:"Code",icon:Ms,href:e=>`/code?q=${e}`},{id:"science",label:"Science",icon:Bs,href:e=>`/science?q=${e}`},{id:"music",label:"Music",icon:Ts,href:e=>`/music?q=${e}`},{id:"social",label:"Social",icon:As,href:e=>`/social?q=${e}`},{id:"maps",label:"Maps",icon:Ns,href:e=>`/maps?q=${e}`}];function q(e){const{query:t,active:s}=e,a=encodeURIComponent(t);return`
    <div class="search-tabs-container" id="tabs-container">
      <nav class="search-tabs" id="search-tabs" role="tablist">
        ${Hs.map(n=>`
          <a class="search-tab ${n.id===s?"active":""}"
             href="${n.href(a)}"
             data-link
             data-tab="${n.id}"
             role="tab"
             aria-selected="${n.id===s}">
            ${n.icon}
            <span>${n.label}</span>
          </a>
        `).join("")}
      </nav>
    </div>
  `}function j(){const e=document.getElementById("tabs-container"),t=e?.closest(".search-tabs-row");if(e&&t){const s=()=>{const n=e.scrollWidth>e.clientWidth,r=e.scrollLeft+e.clientWidth>=e.scrollWidth-10;n&&!r?t.classList.add("has-scroll"):t.classList.remove("has-scroll")};s(),e.addEventListener("scroll",s),window.addEventListener("resize",s);const a=e.querySelector(".search-tab.active");a&&a.scrollIntoView({behavior:"smooth",block:"nearest",inline:"center"})}}const Rs='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m6 9 6 6 6-6"/></svg>';function Os(e){return!e||e.length===0?"":`
    <div class="paa-container">
      <h3 class="paa-title">People also ask</h3>
      <div class="paa-list">
        ${e.map((t,s)=>`
          <div class="paa-item" data-index="${s}">
            <button class="paa-question" aria-expanded="false">
              <span>${Se(t.question)}</span>
              <span class="paa-chevron">${Rs}</span>
            </button>
            <div class="paa-answer hidden">
              ${t.answer?`<p class="paa-answer-text">${Se(t.answer)}</p>`:'<p class="paa-loading">Loading...</p>'}
              ${t.source&&t.url?`
                <a href="${js(t.url)}" target="_blank" class="paa-source">
                  ${Se(t.source)}
                </a>
              `:""}
            </div>
          </div>
        `).join("")}
      </div>
    </div>
  `}function qs(){const e=document.querySelector(".paa-container");e&&e.querySelectorAll(".paa-item").forEach(t=>{const s=t.querySelector(".paa-question"),a=t.querySelector(".paa-answer");s?.addEventListener("click",()=>{const n=s.getAttribute("aria-expanded")==="true";e.querySelectorAll(".paa-item").forEach(r=>{r!==t&&(r.querySelector(".paa-question")?.setAttribute("aria-expanded","false"),r.querySelector(".paa-answer")?.classList.add("hidden"),r.querySelector(".paa-chevron")?.classList.remove("rotated"))}),s.setAttribute("aria-expanded",String(!n)),a?.classList.toggle("hidden",n),t.querySelector(".paa-chevron")?.classList.toggle("rotated",!n)})})}function Se(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function js(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const ze='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m6 9 6 6 6-6"/></svg>',Ps='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>',Us='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M2 12h20"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>',Fs='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 21c3 0 7-1 7-8V5c0-1.25-.756-2.017-2-2H4c-1.25 0-2 .75-2 1.972V11c0 1.25.75 2 2 2 1 0 1 0 1 1v1c0 1-1 2-2 2s-1 .008-1 1.031V21c0 1 0 1 1 1z"/><path d="M15 21c3 0 7-1 7-8V5c0-1.25-.757-2.017-2-2h-4c-1.25 0-2 .75-2 1.972V11c0 1.25.75 2 2 2h.75c0 2.25.25 4-2.75 4v3c0 1 0 1 1 1z"/></svg>',Vs='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>',De='<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>',_e=[{value:"",label:"Any time"},{value:"hour",label:"Past hour"},{value:"day",label:"Past 24 hours"},{value:"week",label:"Past week"},{value:"month",label:"Past month"},{value:"year",label:"Past year"}],Be=[{value:"",label:"Any region"},{value:"us",label:"United States"},{value:"gb",label:"United Kingdom"},{value:"ca",label:"Canada"},{value:"au",label:"Australia"},{value:"de",label:"Germany"},{value:"fr",label:"France"},{value:"jp",label:"Japan"},{value:"in",label:"India"},{value:"br",label:"Brazil"}];function zs(e={}){const{timeRange:t="",region:s="",verbatim:a=!1,site:n=""}=e,r=_e.find(h=>h.value===t)?.label||"Any time",i=Be.find(h=>h.value===s)?.label||"Any region",o=t!=="",l=s!=="",c=n!=="",v=o||l||a||c;return`
    <div class="search-tools" id="search-tools">
      <div class="search-tools-row">
        <!-- Time Filter -->
        <div class="search-tool-dropdown" data-tool="time">
          <button class="search-tool-btn ${o?"active":""}" type="button">
            ${Ps}
            <span class="search-tool-label">${ne(r)}</span>
            ${ze}
          </button>
          <div class="search-tool-menu hidden">
            ${_e.map(h=>`
              <button class="search-tool-option ${h.value===t?"selected":""}" data-value="${h.value}">
                ${ne(h.label)}
              </button>
            `).join("")}
          </div>
        </div>

        <!-- Region Filter -->
        <div class="search-tool-dropdown" data-tool="region">
          <button class="search-tool-btn ${l?"active":""}" type="button">
            ${Us}
            <span class="search-tool-label">${ne(i)}</span>
            ${ze}
          </button>
          <div class="search-tool-menu hidden">
            ${Be.map(h=>`
              <button class="search-tool-option ${h.value===s?"selected":""}" data-value="${h.value}">
                ${ne(h.label)}
              </button>
            `).join("")}
          </div>
        </div>

        <!-- Verbatim Toggle -->
        <button class="search-tool-toggle ${a?"active":""}" data-tool="verbatim" type="button">
          ${Fs}
          <span>Verbatim</span>
        </button>

        <!-- Site Search -->
        <div class="search-tool-site" data-tool="site">
          <div class="search-tool-site-input ${c?"has-value":""}">
            ${Vs}
            <input
              type="text"
              id="site-filter-input"
              placeholder="Filter by site..."
              value="${ne(n)}"
              autocomplete="off"
              spellcheck="false"
            />
            ${c?`
              <button class="search-tool-site-clear" type="button" aria-label="Clear site filter">
                ${De}
              </button>
            `:""}
          </div>
        </div>

        <!-- Clear All Filters -->
        ${v?`
          <button class="search-tool-clear" id="clear-all-filters" type="button">
            ${De}
            <span>Clear filters</span>
          </button>
        `:""}
      </div>
    </div>
  `}function Ds(e){const t=document.getElementById("search-tools");if(!t)return;const s={timeRange:"",region:"",verbatim:!1,site:""},a=t.querySelector('[data-tool="time"]'),n=t.querySelector('[data-tool="region"]'),r=t.querySelector('[data-tool="verbatim"]'),i=t.querySelector("#site-filter-input");if(a){const l=a.querySelector(".search-tool-option.selected");s.timeRange=l?.dataset.value||""}if(n){const l=n.querySelector(".search-tool-option.selected");s.region=l?.dataset.value||""}r?.classList.contains("active")&&(s.verbatim=!0),i&&(s.site=i.value),t.querySelectorAll(".search-tool-dropdown").forEach(l=>{const c=l.querySelector(".search-tool-btn"),v=l.querySelector(".search-tool-menu"),h=l.dataset.tool;c?.addEventListener("click",y=>{y.stopPropagation(),t.querySelectorAll(".search-tool-menu").forEach(k=>{k!==v&&k.classList.add("hidden")}),v?.classList.toggle("hidden")}),v?.querySelectorAll(".search-tool-option").forEach(y=>{y.addEventListener("click",()=>{const k=y.dataset.value||"";v.querySelectorAll(".search-tool-option").forEach($=>{$.classList.toggle("selected",$===y)});const P=k!=="";c?.classList.toggle("active",P);const B=c?.querySelector(".search-tool-label");B&&(h==="time"?(B.textContent=_e.find($=>$.value===k)?.label||"Any time",s.timeRange=k):h==="region"&&(B.textContent=Be.find($=>$.value===k)?.label||"Any region",s.region=k)),v?.classList.add("hidden"),e({...s})})})}),r?.addEventListener("click",()=>{s.verbatim=!s.verbatim,r.classList.toggle("active",s.verbatim),e({...s})});let o;i?.addEventListener("input",()=>{clearTimeout(o),o=setTimeout(()=>{s.site=i.value.trim(),Le(t,s.site),e({...s})},500)}),i?.addEventListener("keydown",l=>{l.key==="Enter"&&(l.preventDefault(),clearTimeout(o),s.site=i.value.trim(),Le(t,s.site),e({...s}))}),t.querySelector(".search-tool-site-clear")?.addEventListener("click",()=>{s.site="",i&&(i.value=""),Le(t,""),e({...s})}),t.querySelector("#clear-all-filters")?.addEventListener("click",()=>{s.timeRange="",s.region="",s.verbatim=!1,s.site="",e({...s})}),document.addEventListener("click",()=>{t.querySelectorAll(".search-tool-menu").forEach(l=>l.classList.add("hidden"))})}function Le(e,t){e.querySelector(".search-tool-site-input")?.classList.toggle("has-value",t!=="")}function ne(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}const Gs='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="18" height="18" x="3" y="3" rx="2" ry="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"/></svg>',Ws='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M5 12h14"/><path d="m12 5 7 7-7 7"/></svg>',Ks='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>';function Ys(e,t={}){return`
    <div class="min-h-screen flex flex-col">
      <!-- Header -->
      <header class="search-header">
        <div class="search-header-row">
          <a href="/" data-link class="search-logo">
            <span style="color: #2563eb">M</span><span style="color: #ef4444">i</span><span style="color: #f59e0b">z</span><span style="color: #22c55e">u</span>
          </a>
          <div class="search-header-box">
            ${E({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="search-box-btn" aria-label="Settings">
            ${Ks}
          </a>
        </div>
        <div class="search-tabs-row">
          ${q({query:e,active:"all"})}
        </div>
        <!-- Search Tools Bar -->
        ${zs(t)}
      </header>

      <!-- Content -->
      <main class="flex-1">
        <div id="search-content" class="search-content-area">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `}function Qs(e,t,s){const a=parseInt(s.page||"1"),n={timeRange:s.time_range||"",region:s.region||"",verbatim:s.verbatim==="1",site:s.site||""},r=Q.get().settings;_(i=>{e.navigate(`/search?q=${encodeURIComponent(i)}`)}),j(),Ds(i=>{const o=lt(t,i);e.navigate(o)}),t&&O(t),Js(e,t,a,n,r.results_per_page)}function lt(e,t,s){const a=new URLSearchParams;return a.set("q",e),s&&s>1&&a.set("page",String(s)),t.timeRange&&a.set("time_range",t.timeRange),t.region&&a.set("region",t.region),t.verbatim&&a.set("verbatim","1"),t.site&&a.set("site",t.site),`/search?${a.toString()}`}async function Js(e,t,s,a,n){const r=document.getElementById("search-content");if(!r||!t)return;let i=t;a.site&&(i=`site:${a.site} ${t}`);try{const o=await f.search(i,{page:s,per_page:n,time_range:a.timeRange||void 0,region:a.region||void 0,verbatim:a.verbatim||void 0,site:a.site||void 0});if(o.redirect){window.location.href=o.redirect;return}Zs(r,e,o,t,s,a)}catch(o){r.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to load search results. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${Z(String(o))}</p>
      </div>
    `}}function Zs(e,t,s,a,n,r){const i=s.corrected_query?`<p class="text-sm text-secondary mb-4">
        Showing results for <a href="/search?q=${encodeURIComponent(s.corrected_query)}" data-link class="text-link font-medium">${Z(s.corrected_query)}</a>.
        Search instead for <a href="/search?q=${encodeURIComponent(a)}&exact=1" data-link class="text-link">${Z(a)}</a>.
      </p>`:"",o=`
    <div class="text-xs text-tertiary mb-4">
      About ${ta(s.total_results)} results (${(s.search_time_ms/1e3).toFixed(2)} seconds)
    </div>
  `,l=s.instant_answer?us(s.instant_answer):"",c=s.related_searches?.slice(0,4).map(d=>({question:d,answer:void 0}))||[],v=c.length>0?Os(c):"",h=s.results.slice(0,3),y=s.results.slice(3),k=h.length>0?h.map((d,u)=>Ve(d,u)).join(""):"",P=y.length>0?y.map((d,u)=>Ve(d,u+3)).join(""):"",B=s.results.length===0?`<div class="py-8 text-secondary">No results found for "<strong>${Z(a)}</strong>"</div>`:"",$=s.related_searches&&s.related_searches.length>0?`
      <div class="related-searches-section">
        <h3 class="related-title">Related searches</h3>
        <div class="related-grid">
          ${s.related_searches.map(d=>`
            <a href="/search?q=${encodeURIComponent(d)}" data-link class="related-item">
              <span class="related-icon">${Xs}</span>
              <span class="related-text">${Z(d)}</span>
            </a>
          `).join("")}
        </div>
      </div>
    `:"",pe=ks({currentPage:n,hasMore:s.has_more,totalResults:s.total_results,perPage:s.per_page}),te=s.knowledge_panel?bs(s.knowledge_panel):"";e.innerHTML=`
    <div class="search-results-layout">
      <div class="search-results-main">
        ${i}
        ${o}
        ${l}
        ${B}
        ${k}
        <div id="images-carousel-slot"></div>
        ${v}
        ${P}
        ${$}
        ${pe}
      </div>
      ${te?`<aside class="search-results-sidebar">${te}</aside>`:""}
    </div>
  `,Xt(),qs(),Ss(d=>{const u=lt(a,r,d);t.navigate(u)}),n===1&&s.results.length>0&&ea(a,t)}const Xs='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>';async function ea(e,t){const s=document.getElementById("images-carousel-slot");if(s)try{const n=(await f.searchImages(e,{per_page:8})).results;if(n.length<4){s.remove();return}s.innerHTML=`
      <div class="image-preview-carousel">
        <div class="carousel-header">
          <div class="carousel-title">
            ${Gs}
            <span>Images for "${Z(e)}"</span>
          </div>
          <a href="/images?q=${encodeURIComponent(e)}" data-link class="carousel-more">
            View all ${Ws}
          </a>
        </div>
        <div class="carousel-images">
          ${n.map((r,i)=>`
            <a href="/images?q=${encodeURIComponent(e)}" data-link class="carousel-image" data-index="${i}">
              <img src="${Ge(r.thumbnail_url||r.url)}" alt="${Ge(r.title)}" loading="lazy" />
            </a>
          `).join("")}
        </div>
      </div>
    `,s.querySelectorAll(".carousel-image").forEach(r=>{r.addEventListener("click",i=>{i.preventDefault(),t.navigate(`/images?q=${encodeURIComponent(e)}`)})})}catch{s.remove()}}function ta(e){return e.toLocaleString("en-US")}function Z(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function Ge(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const sa='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',aa='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14.5 4h-5L7 7H4a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2h-3l-2.5-3z"/><circle cx="12" cy="13" r="3"/></svg>',We='<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>',Ke='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" x2="21" y1="14" y2="3"/></svg>',na='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3"/></svg>',ct='<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>',ra='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6"/></svg>',ia='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 18 15 12 9 6"/></svg>';let de="",N={},ue=1,ee=!1,W=!0,H=[],re=!1,Me=[],ge=null;function oa(e){return`
    <div class="min-h-screen flex flex-col">
      <header class="search-header">
        <div class="search-header-row">
          <a href="/" data-link class="search-logo">
            <span style="color: #2563eb">M</span><span style="color: #ef4444">i</span><span style="color: #f59e0b">z</span><span style="color: #22c55e">u</span>
          </a>
          <div class="search-header-box flex items-center gap-2">
            ${E({size:"sm",initialValue:e})}
            <button id="reverse-search-btn" class="search-box-btn" title="Search by image">
              ${aa}
            </button>
          </div>
          <a href="/settings" data-link class="search-box-btn" aria-label="Settings">
            ${sa}
          </a>
        </div>
        <div class="search-tabs-row flex items-center gap-1">
          ${q({query:e,active:"images"})}
          <button id="tools-btn" class="filter-btn ml-4">
            ${na}
            <span class="hidden sm:inline">Tools</span>
            ${ct}
          </button>
        </div>
        <!-- Filter toolbar (hidden by default) -->
        <div id="filter-toolbar" class="filter-toolbar hidden">
          ${la()}
        </div>
      </header>

      <!-- Related searches bar -->
      <div id="related-searches" class="related-searches-bar hidden">
        <div class="related-searches-scroll">
          <button class="related-scroll-btn related-scroll-left hidden">${ra}</button>
          <div class="related-searches-list"></div>
          <button class="related-scroll-btn related-scroll-right hidden">${ia}</button>
        </div>
      </div>

      <!-- Content - Full width -->
      <main class="flex-1 flex">
        <div id="images-content" class="flex-1 px-2 sm:px-4 lg:px-6 xl:px-8 py-4">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>

        <!-- Preview panel (hidden by default) -->
        <div id="preview-panel" class="preview-panel hidden">
          <div class="preview-overlay"></div>
          <div class="preview-container">
            <button id="preview-close" class="preview-close-btn" aria-label="Close">${We}</button>
            <div class="preview-main">
              <div class="preview-image-wrap">
                <img id="preview-image" src="" alt="" />
              </div>
              <div class="preview-sidebar">
                <div id="preview-details" class="preview-info"></div>
              </div>
            </div>
          </div>
        </div>
      </main>

      <!-- Reverse image search modal -->
      <div id="reverse-modal" class="modal hidden">
        <div class="modal-content">
          <div class="modal-header">
            <h2>Search by image</h2>
            <button id="reverse-modal-close" class="modal-close">${We}</button>
          </div>
          <div class="modal-body">
            <div id="drop-zone" class="drop-zone">
              <p>Drag and drop an image here or</p>
              <label class="file-upload-btn">
                Upload a file
                <input type="file" id="image-upload" accept="image/*" hidden />
              </label>
            </div>
            <div class="url-input-section">
              <p>Or paste an image URL:</p>
              <div class="url-input-container">
                <input type="text" id="image-url-input" placeholder="https://example.com/image.jpg" />
                <button id="url-search-btn">Search</button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  `}function la(){return`
    <div class="filter-chips">
      ${[{id:"size",label:"Size",options:["any","large","medium","small","icon"]},{id:"color",label:"Color",options:["any","color","gray","transparent","red","orange","yellow","green","teal","blue","purple","pink","white","black","brown"]},{id:"type",label:"Type",options:["any","photo","clipart","lineart","animated","face"]},{id:"aspect",label:"Aspect",options:["any","tall","square","wide","panoramic"]},{id:"time",label:"Time",options:["any","day","week","month","year"]},{id:"rights",label:"Usage rights",options:["any","creative_commons","commercial"]}].map(t=>`
        <div class="filter-chip-wrapper">
          <button class="filter-chip" data-filter="${t.id}" data-value="any">
            <span class="filter-chip-label">${t.label}</span>
            ${ct}
          </button>
          <div class="filter-dropdown hidden" data-dropdown="${t.id}">
            ${t.options.map(s=>`
              <button class="filter-option${s==="any"?" active":""}" data-value="${s}">
                ${ye(t.id,s)}
              </button>
            `).join("")}
          </div>
        </div>
      `).join("")}
      <button id="clear-filters" class="clear-filters-btn hidden">Clear</button>
    </div>
  `}function ye(e,t){return t==="any"?`Any ${e}`:t.charAt(0).toUpperCase()+t.slice(1).replace("_"," ")}function ca(e,t,s){if(de=t,N={},ue=1,H=[],W=!0,re=!1,Me=[],_(a=>{e.navigate(`/images?q=${encodeURIComponent(a)}`)}),j(),t&&O(t),da(),ua(),ga(),fa(),ha(e),s?.reverse==="1"){const a=document.getElementById("reverse-modal");a&&a.classList.remove("hidden")}Te(t,N)}function da(){const e=document.getElementById("tools-btn"),t=document.getElementById("filter-toolbar");!e||!t||e.addEventListener("click",()=>{re=!re,t.classList.toggle("hidden",!re),e.classList.toggle("active",re)})}function ua(e){const t=document.getElementById("filter-toolbar");if(!t)return;t.querySelectorAll(".filter-chip").forEach(a=>{a.addEventListener("click",n=>{n.stopPropagation();const r=a.dataset.filter,i=t.querySelector(`[data-dropdown="${r}"]`);t.querySelectorAll(".filter-dropdown").forEach(o=>{o!==i&&o.classList.add("hidden")}),i?.classList.toggle("hidden")})}),t.querySelectorAll(".filter-option").forEach(a=>{a.addEventListener("click",()=>{const n=a.closest(".filter-dropdown"),r=n?.dataset.dropdown,i=a.dataset.value,o=t.querySelector(`[data-filter="${r}"]`);!r||!i||!o||(n.querySelectorAll(".filter-option").forEach(l=>l.classList.remove("active")),a.classList.add("active"),i==="any"?(delete N[r],o.classList.remove("has-value"),o.querySelector(".filter-chip-label").textContent=ye(r,"any").replace("Any ","")):(N[r]=i,o.classList.add("has-value"),o.querySelector(".filter-chip-label").textContent=ye(r,i)),n.classList.add("hidden"),Ye(),ue=1,H=[],W=!0,Te(de,N))})}),document.addEventListener("click",()=>{t.querySelectorAll(".filter-dropdown").forEach(a=>a.classList.add("hidden"))});const s=document.getElementById("clear-filters");s&&s.addEventListener("click",()=>{N={},ue=1,H=[],W=!0,t.querySelectorAll(".filter-chip").forEach(a=>{const n=a.dataset.filter;a.classList.remove("has-value"),a.querySelector(".filter-chip-label").textContent=ye(n,"any").replace("Any ","")}),t.querySelectorAll(".filter-dropdown").forEach(a=>{a.querySelectorAll(".filter-option").forEach((n,r)=>{n.classList.toggle("active",r===0)})}),Ye(),Te(de,N)})}function Ye(){const e=document.getElementById("clear-filters");e&&e.classList.toggle("hidden",Object.keys(N).length===0)}function ha(e){const t=document.getElementById("related-searches");if(!t)return;t.addEventListener("click",r=>{const i=r.target.closest(".related-chip");if(i){const o=i.getAttribute("data-query");o&&e.navigate(`/images?q=${encodeURIComponent(o)}`)}});const s=t.querySelector(".related-scroll-left"),a=t.querySelector(".related-scroll-right"),n=t.querySelector(".related-searches-list");s&&a&&n&&(s.addEventListener("click",()=>{n.scrollBy({left:-200,behavior:"smooth"})}),a.addEventListener("click",()=>{n.scrollBy({left:200,behavior:"smooth"})}),n.addEventListener("scroll",()=>{dt()}))}function dt(){const e=document.getElementById("related-searches");if(!e)return;const t=e.querySelector(".related-searches-list"),s=e.querySelector(".related-scroll-left"),a=e.querySelector(".related-scroll-right");!t||!s||!a||(s.classList.toggle("hidden",t.scrollLeft<=0),a.classList.toggle("hidden",t.scrollLeft>=t.scrollWidth-t.clientWidth-10))}function pa(e){const t=document.getElementById("related-searches");if(!t)return;if(!e||e.length===0){t.classList.add("hidden");return}const s=t.querySelector(".related-searches-list");s&&(s.innerHTML=e.map(a=>`
    <button class="related-chip" data-query="${G(a)}">
      <span class="related-chip-text">${J(a)}</span>
    </button>
  `).join(""),t.classList.remove("hidden"),setTimeout(dt,50))}function ga(e){const t=document.getElementById("reverse-search-btn"),s=document.getElementById("reverse-modal"),a=document.getElementById("reverse-modal-close"),n=document.getElementById("drop-zone"),r=document.getElementById("image-upload"),i=document.getElementById("image-url-input"),o=document.getElementById("url-search-btn");!t||!s||(t.addEventListener("click",()=>s.classList.remove("hidden")),a?.addEventListener("click",()=>s.classList.add("hidden")),s.addEventListener("click",l=>{l.target===s&&s.classList.add("hidden")}),n&&(n.addEventListener("dragover",l=>{l.preventDefault(),n.classList.add("drag-over")}),n.addEventListener("dragleave",()=>n.classList.remove("drag-over")),n.addEventListener("drop",l=>{l.preventDefault(),n.classList.remove("drag-over");const c=l.dataTransfer?.files;c&&c[0]&&(Qe(c[0]),s.classList.add("hidden"))})),r&&r.addEventListener("change",()=>{r.files&&r.files[0]&&(Qe(r.files[0]),s.classList.add("hidden"))}),o&&i&&(o.addEventListener("click",()=>{const l=i.value.trim();l&&(Je(l),s.classList.add("hidden"))}),i.addEventListener("keydown",l=>{if(l.key==="Enter"){const c=i.value.trim();c&&(Je(c),s.classList.add("hidden"))}})))}async function Qe(e,t){const s=document.getElementById("images-content");if(s){if(!e.type.startsWith("image/")){alert("Please select an image file");return}if(e.size>10*1024*1024){alert("Image must be smaller than 10MB");return}s.innerHTML=`
    <div class="flex flex-col items-center justify-center py-16">
      <div class="spinner"></div>
      <span class="mt-3 text-secondary">Uploading and searching...</span>
      <div class="w-48 mt-4 h-1 bg-border rounded-full overflow-hidden">
        <div id="upload-progress" class="h-full bg-blue transition-all duration-300" style="width: 0%"></div>
      </div>
    </div>
  `;try{const a=await va(e),n=document.getElementById("upload-progress");n&&(n.style.width="50%");const r=await f.reverseImageSearchByUpload(a);n&&(n.style.width="100%"),ma(s,a,r)}catch(a){s.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to search by image. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${J(String(a))}</p>
      </div>
    `}}}function va(e){return new Promise((t,s)=>{const a=new FileReader;a.onload=()=>{const r=a.result.split(",")[1];t(r)},a.onerror=s,a.readAsDataURL(e)})}function ma(e,t,s){const n=!t.startsWith("http")?`data:image/jpeg;base64,${t}`:t;e.innerHTML=`
    <div class="reverse-results">
      <div class="query-image-section">
        <h3>Search image</h3>
        <img src="${n}" alt="Query image" class="query-image" />
      </div>
      ${s.similar_images.length>0?`
        <div class="similar-images-section">
          <h3>Similar images (${s.similar_images.length})</h3>
          <div class="image-grid">
            ${s.similar_images.map((r,i)=>ke(r,i)).join("")}
          </div>
        </div>
      `:'<div class="py-8 text-secondary">No similar images found.</div>'}
    </div>
  `,e.querySelectorAll(".image-card").forEach(r=>{r.addEventListener("click",()=>{const i=parseInt(r.dataset.imageIndex||"0",10);$e(s.similar_images[i])})})}async function Je(e,t){const s=document.getElementById("images-content");if(s){s.innerHTML=`
    <div class="flex items-center justify-center py-16">
      <div class="spinner"></div>
      <span class="ml-3 text-secondary">Searching for similar images...</span>
    </div>
  `;try{const a=await f.reverseImageSearch(e);s.innerHTML=`
      <div class="reverse-results">
        <div class="query-image-section">
          <h3>Search image</h3>
          <img src="${G(e)}" alt="Query image" class="query-image" />
        </div>
        ${a.similar_images.length>0?`
          <div class="similar-images-section">
            <h3>Similar images (${a.similar_images.length})</h3>
            <div class="image-grid">
              ${a.similar_images.map((n,r)=>ke(n,r)).join("")}
            </div>
          </div>
        `:'<div class="py-8 text-secondary">No similar images found.</div>'}
      </div>
    `,s.querySelectorAll(".image-card").forEach(n=>{n.addEventListener("click",()=>{const r=parseInt(n.dataset.imageIndex||"0",10);$e(a.similar_images[r])})})}catch(a){s.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to search by image. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${J(String(a))}</p>
      </div>
    `}}}function fa(){const e=document.getElementById("preview-panel"),t=document.getElementById("preview-close"),s=e?.querySelector(".preview-overlay");t?.addEventListener("click",Ie),s?.addEventListener("click",Ie),document.addEventListener("keydown",a=>{a.key==="Escape"&&Ie()})}function $e(e){const t=document.getElementById("preview-panel"),s=document.getElementById("preview-image"),a=document.getElementById("preview-details");if(!t||!s||!a)return;s.src=e.url,s.alt=e.title;const n=e.width&&e.height&&e.width>0&&e.height>0;a.innerHTML=`
    <div class="preview-header">
      <img src="${G(e.thumbnail_url||e.url)}" class="preview-thumb" alt="" />
      <div class="preview-header-info">
        <h3 class="preview-title">${J(e.title||"Untitled")}</h3>
        <a href="${G(e.source_url)}" target="_blank" class="preview-domain">${J(e.source_domain)}</a>
      </div>
    </div>
    <div class="preview-meta">
      ${n?`<div class="preview-meta-item"><span class="preview-meta-label">Size</span><span>${e.width} × ${e.height}</span></div>`:""}
      ${e.format?`<div class="preview-meta-item"><span class="preview-meta-label">Type</span><span>${e.format.toUpperCase()}</span></div>`:""}
    </div>
    <div class="preview-actions">
      <a href="${G(e.source_url)}" target="_blank" class="preview-btn preview-btn-primary">
        Visit page ${Ke}
      </a>
      <a href="${G(e.url)}" target="_blank" class="preview-btn">
        View full image ${Ke}
      </a>
    </div>
  `,t.classList.remove("hidden"),document.body.style.overflow="hidden"}function Ie(){const e=document.getElementById("preview-panel");e&&(e.classList.add("hidden"),document.body.style.overflow="")}function ya(){ge&&ge.disconnect();const e=document.getElementById("images-content");if(!e)return;const t=document.getElementById("scroll-sentinel");t&&t.remove();const s=document.createElement("div");s.id="scroll-sentinel",s.className="scroll-sentinel",e.appendChild(s),ge=new IntersectionObserver(a=>{a[0].isIntersecting&&!ee&&W&&de&&wa()},{rootMargin:"400px"}),ge.observe(s)}async function wa(){if(ee||!W)return;ee=!0,ue++;const e=document.getElementById("scroll-sentinel");e&&(e.innerHTML='<div class="loading-more"><div class="spinner-sm"></div></div>');try{const t=await f.searchImages(de,{...N,page:ue}),s=t.results;W=t.has_more,H=[...H,...s];const a=document.querySelector(".image-grid");if(a&&s.length>0){const n=H.length-s.length,r=s.map((i,o)=>ke(i,n+o)).join("");a.insertAdjacentHTML("beforeend",r),a.querySelectorAll(".image-card:not([data-initialized])").forEach(i=>{i.setAttribute("data-initialized","true"),i.addEventListener("click",()=>{const o=parseInt(i.dataset.imageIndex||"0",10);$e(H[o])})})}e&&(e.innerHTML=W?"":'<div class="no-more-results">No more images</div>')}catch{e&&(e.innerHTML="")}finally{ee=!1}}async function Te(e,t){const s=document.getElementById("images-content");if(!(!s||!e)){ee=!0,s.innerHTML='<div class="flex items-center justify-center py-16"><div class="spinner"></div></div>';try{const a=await f.searchImages(e,{...t,page:1,per_page:50}),n=a.results;if(W=a.has_more,H=n,Me=a.related_searches?.length?a.related_searches:ba(e),pa(Me),n.length===0){s.innerHTML=`<div class="py-8 text-secondary">No image results found for "<strong>${J(e)}</strong>"</div>`;return}s.innerHTML=`<div class="image-grid">${n.map((r,i)=>ke(r,i)).join("")}</div>`,s.querySelectorAll(".image-card").forEach(r=>{r.setAttribute("data-initialized","true"),r.addEventListener("click",()=>{const i=parseInt(r.dataset.imageIndex||"0",10);$e(H[i])})}),ya()}catch(a){s.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to load image results. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${J(String(a))}</p>
      </div>
    `}finally{ee=!1}}}function ke(e,t){return`
    <div class="image-card" data-image-index="${t}">
      <div class="image-card-img">
        <div class="image-placeholder" style="background: #e0e0e0;"></div>
        <img
          src="${G(e.thumbnail_url||e.url)}"
          alt="${G(e.title)}"
          loading="lazy"
          class="image-lazy"
          onload="this.classList.add('loaded'); this.previousElementSibling.style.display='none';"
          onerror="this.closest('.image-card').style.display='none'"
        />
      </div>
    </div>
  `}function J(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function G(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}function ba(e){const t=e.toLowerCase().trim().split(/\s+/).filter(i=>i.length>1);if(t.length===0)return[];const s=[],a=["wallpaper","hd","4k","aesthetic","cute","beautiful","background","art","photography","design","illustration","vintage","modern","minimalist","colorful","dark","light"],n={cat:["kitten","cats playing","black cat","tabby cat","cat meme"],dog:["puppy","dogs playing","golden retriever","german shepherd","dog meme"],nature:["forest","mountains","ocean","sunset nature","flowers"],food:["dessert","healthy food","breakfast","dinner","food photography"],car:["sports car","luxury car","vintage car","car interior","supercar"],house:["modern house","interior design","living room","bedroom design","architecture"],city:["skyline","night city","urban photography","street photography","downtown"]},r=t.slice(0,2).join(" ");for(const i of a)!e.includes(i)&&s.length<4&&s.push(`${r} ${i}`);for(const[i,o]of Object.entries(n))if(t.some(l=>l.includes(i)||i.includes(l))){for(const l of o)!s.includes(l)&&s.length<8&&s.push(l);break}return t.length>=2&&s.length<8&&s.push(t.reverse().join(" ")),s.length<4&&s.push(`${r} images`,`${r} photos`,`best ${r}`),s.slice(0,8)}const xa='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',$a='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3"/></svg>',ut='<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>';let Ae="",z={},ie=!1;function ka(e){return`
    <div class="min-h-screen flex flex-col">
      <header class="search-header">
        <div class="search-header-row">
          <a href="/" data-link class="search-logo">
            <span style="color: #2563eb">M</span><span style="color: #ef4444">i</span><span style="color: #f59e0b">z</span><span style="color: #22c55e">u</span>
          </a>
          <div class="search-header-box">
            ${E({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="search-box-btn" aria-label="Settings">
            ${xa}
          </a>
        </div>
        <div class="search-tabs-row flex items-center gap-1">
          ${q({query:e,active:"videos"})}
          <button id="tools-btn" class="filter-btn ml-4">
            ${$a}
            <span class="hidden sm:inline">Tools</span>
            ${ut}
          </button>
        </div>
        <!-- Filter toolbar (hidden by default) -->
        <div id="filter-toolbar" class="filter-toolbar hidden">
          ${Ca()}
        </div>
      </header>

      <!-- Content - Full width -->
      <main class="flex-1">
        <div id="videos-content" class="px-2 sm:px-4 lg:px-6 xl:px-8 py-4">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `}function Ca(){return`
    <div class="filter-chips">
      ${[{id:"duration",label:"Duration",options:["any","short","medium","long"]},{id:"time",label:"Upload date",options:["any","hour","day","week","month","year"]},{id:"quality",label:"Quality",options:["any","hd","4k"]},{id:"sort",label:"Sort by",options:["relevance","date","views"]}].map(t=>`
        <div class="filter-chip-wrapper">
          <button class="filter-chip" data-filter="${t.id}" data-value="any">
            <span class="filter-chip-label">${t.label}</span>
            ${ut}
          </button>
          <div class="filter-dropdown hidden" data-dropdown="${t.id}">
            ${t.options.map(s=>`
              <button class="filter-option${s==="any"||s==="relevance"?" active":""}" data-value="${s}">
                ${we(t.id,s)}
              </button>
            `).join("")}
          </div>
        </div>
      `).join("")}
      <button id="clear-filters" class="clear-filters-btn hidden">Clear</button>
    </div>
  `}function we(e,t){return{duration:{any:"Any duration",short:"Under 4 min",medium:"4-20 min",long:"Over 20 min"},time:{any:"Any time",hour:"Last hour",day:"Today",week:"This week",month:"This month",year:"This year"},quality:{any:"Any quality",hd:"HD","4k":"4K"},sort:{relevance:"Relevance",date:"Upload date",views:"View count"}}[e]?.[t]||t.charAt(0).toUpperCase()+t.slice(1)}function Sa(e,t){Ae=t,z={},ie=!1,_(s=>{e.navigate(`/videos?q=${encodeURIComponent(s)}`)}),j(),La(),Ia(),t&&O(t),Ne(t,z)}function La(){const e=document.getElementById("tools-btn"),t=document.getElementById("filter-toolbar");!e||!t||e.addEventListener("click",()=>{ie=!ie,t.classList.toggle("hidden",!ie),e.classList.toggle("active",ie)})}function Ia(e){const t=document.getElementById("filter-toolbar");if(!t)return;t.querySelectorAll(".filter-chip").forEach(a=>{a.addEventListener("click",n=>{n.stopPropagation();const r=a.dataset.filter,i=t.querySelector(`[data-dropdown="${r}"]`);t.querySelectorAll(".filter-dropdown").forEach(o=>{o!==i&&o.classList.add("hidden")}),i?.classList.toggle("hidden")})}),t.querySelectorAll(".filter-option").forEach(a=>{a.addEventListener("click",()=>{const n=a.closest(".filter-dropdown"),r=n?.dataset.dropdown,i=a.dataset.value,o=t.querySelector(`[data-filter="${r}"]`);if(!r||!i||!o)return;n.querySelectorAll(".filter-option").forEach(c=>c.classList.remove("active")),a.classList.add("active");const l=i==="any"||r==="sort"&&i==="relevance";l?(delete z[r],o.classList.remove("has-value"),o.querySelector(".filter-chip-label").textContent=we(r,l?"any":i).replace(/^Any /,"")):(z[r]=i,o.classList.add("has-value"),o.querySelector(".filter-chip-label").textContent=we(r,i)),n.classList.add("hidden"),Ze(),Ne(Ae,z)})}),document.addEventListener("click",()=>{t.querySelectorAll(".filter-dropdown").forEach(a=>a.classList.add("hidden"))});const s=document.getElementById("clear-filters");s&&s.addEventListener("click",()=>{z={},t.querySelectorAll(".filter-chip").forEach(a=>{const n=a.dataset.filter;a.classList.remove("has-value");const r=n==="sort"?"Sort by":we(n,"any").replace(/^Any /,"");a.querySelector(".filter-chip-label").textContent=r}),t.querySelectorAll(".filter-dropdown").forEach(a=>{a.querySelectorAll(".filter-option").forEach((n,r)=>{n.classList.toggle("active",r===0)})}),Ze(),Ne(Ae,z)})}function Ze(){const e=document.getElementById("clear-filters");e&&e.classList.toggle("hidden",Object.keys(z).length===0)}async function Ne(e,t){const s=document.getElementById("videos-content");if(!(!s||!e)){s.innerHTML='<div class="flex items-center justify-center py-16"><div class="spinner"></div></div>';try{const a=await f.searchVideos(e,{page:1,per_page:24,...t}),n=a.results;if(n.length===0){s.innerHTML=`
        <div class="py-8 text-secondary">No video results found for "<strong>${X(e)}</strong>"</div>
      `;return}s.innerHTML=`
      <div class="text-xs text-tertiary mb-4">
        About ${a.total_results.toLocaleString()} video results (${(a.search_time_ms/1e3).toFixed(2)} seconds)
      </div>
      <div class="video-grid">
        ${n.map(r=>Ea(r)).join("")}
      </div>
    `}catch(a){s.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to load video results. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${X(String(a))}</p>
      </div>
    `}}}function Ea(e){const t=e.thumbnail?.url||"",s=e.views?_a(e.views):"",a=e.published?Ba(e.published):"",n=[e.channel,s,a].filter(Boolean).join(" · ");return`
    <div class="video-card">
      <a href="${ve(e.url)}" target="_blank" rel="noopener" class="block">
        <div class="video-thumb">
          ${t?`<img src="${ve(t)}" alt="${ve(e.title)}" loading="lazy" onerror="this.style.display='none'; this.nextElementSibling.style.display='flex'" />`:""}
          <div class="video-thumb-placeholder" ${t?'style="display:none"':""}>
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="#dadce0" stroke-width="1.5"><path d="m22 8-6 4 6 4V8Z"/><rect width="14" height="12" x="2" y="6" rx="2" ry="2"/></svg>
          </div>
          ${e.duration?`<span class="video-duration">${X(e.duration)}</span>`:""}
        </div>
      </a>
      <div class="video-info">
        <div class="video-title">
          <a href="${ve(e.url)}" target="_blank" rel="noopener">${X(e.title)}</a>
        </div>
        <div class="video-meta">${X(n)}</div>
        ${e.platform?`<div class="text-xs text-light mt-1">${X(e.platform)}</div>`:""}
      </div>
    </div>
  `}function _a(e){return e>=1e6?`${(e/1e6).toFixed(1)}M views`:e>=1e3?`${(e/1e3).toFixed(1)}K views`:`${e} views`}function Ba(e){try{const t=new Date(e),a=new Date().getTime()-t.getTime(),n=Math.floor(a/(1e3*60*60*24));return n===0?"Today":n===1?"1 day ago":n<7?`${n} days ago`:n<30?`${Math.floor(n/7)} weeks ago`:n<365?`${Math.floor(n/30)} months ago`:`${Math.floor(n/365)} years ago`}catch{return e}}function X(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function ve(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const Ma='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',Ta='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3"/></svg>',ht='<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>';let He="",D={},oe=!1;function Aa(e){return`
    <div class="min-h-screen flex flex-col">
      <header class="search-header">
        <div class="search-header-row">
          <a href="/" data-link class="search-logo">
            <span style="color: #2563eb">M</span><span style="color: #ef4444">i</span><span style="color: #f59e0b">z</span><span style="color: #22c55e">u</span>
          </a>
          <div class="search-header-box">
            ${E({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="search-box-btn" aria-label="Settings">
            ${Ma}
          </a>
        </div>
        <div class="search-tabs-row flex items-center gap-1">
          ${q({query:e,active:"news"})}
          <button id="tools-btn" class="filter-btn ml-4">
            ${Ta}
            <span class="hidden sm:inline">Tools</span>
            ${ht}
          </button>
        </div>
        <!-- Filter toolbar (hidden by default) -->
        <div id="filter-toolbar" class="filter-toolbar hidden">
          ${Na()}
        </div>
      </header>

      <!-- Content - Full width -->
      <main class="flex-1">
        <div id="news-content" class="px-2 sm:px-4 lg:px-6 xl:px-8 py-4">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `}function Na(){return`
    <div class="filter-chips">
      ${[{id:"time",label:"Time",options:["any","hour","day","week","month","year"]},{id:"sort",label:"Sort by",options:["relevance","date"]}].map(t=>`
        <div class="filter-chip-wrapper">
          <button class="filter-chip" data-filter="${t.id}" data-value="any">
            <span class="filter-chip-label">${t.label}</span>
            ${ht}
          </button>
          <div class="filter-dropdown hidden" data-dropdown="${t.id}">
            ${t.options.map(s=>`
              <button class="filter-option${s==="any"||s==="relevance"?" active":""}" data-value="${s}">
                ${pt(t.id,s)}
              </button>
            `).join("")}
          </div>
        </div>
      `).join("")}
      <button id="clear-filters" class="clear-filters-btn hidden">Clear</button>
    </div>
  `}function pt(e,t){return{time:{any:"Any time",hour:"Past hour",day:"Past 24 hours",week:"Past week",month:"Past month",year:"Past year"},sort:{relevance:"Relevance",date:"Date"}}[e]?.[t]||t.charAt(0).toUpperCase()+t.slice(1)}function Ha(e,t){He=t,D={},oe=!1,_(s=>{e.navigate(`/news?q=${encodeURIComponent(s)}`)}),j(),Ra(),Oa(),t&&O(t),Re(t,D)}function Ra(){const e=document.getElementById("tools-btn"),t=document.getElementById("filter-toolbar");!e||!t||e.addEventListener("click",()=>{oe=!oe,t.classList.toggle("hidden",!oe),e.classList.toggle("active",oe)})}function Oa(){const e=document.getElementById("filter-toolbar");if(!e)return;e.querySelectorAll(".filter-chip").forEach(s=>{s.addEventListener("click",a=>{a.stopPropagation();const n=s.dataset.filter,r=e.querySelector(`[data-dropdown="${n}"]`);e.querySelectorAll(".filter-dropdown").forEach(i=>{i!==r&&i.classList.add("hidden")}),r?.classList.toggle("hidden")})}),e.querySelectorAll(".filter-option").forEach(s=>{s.addEventListener("click",()=>{const a=s.closest(".filter-dropdown"),n=a?.dataset.dropdown,r=s.dataset.value,i=e.querySelector(`[data-filter="${n}"]`);if(!n||!r||!i)return;a.querySelectorAll(".filter-option").forEach(l=>l.classList.remove("active")),s.classList.add("active"),r==="any"||n==="sort"&&r==="relevance"?(delete D[n],i.classList.remove("has-value"),i.querySelector(".filter-chip-label").textContent=n==="sort"?"Sort by":"Time"):(D[n]=r,i.classList.add("has-value"),i.querySelector(".filter-chip-label").textContent=pt(n,r)),a.classList.add("hidden"),Xe(),Re(He,D)})}),document.addEventListener("click",()=>{e.querySelectorAll(".filter-dropdown").forEach(s=>s.classList.add("hidden"))});const t=document.getElementById("clear-filters");t&&t.addEventListener("click",()=>{D={},e.querySelectorAll(".filter-chip").forEach(s=>{const a=s.dataset.filter;s.classList.remove("has-value"),s.querySelector(".filter-chip-label").textContent=a==="sort"?"Sort by":"Time"}),e.querySelectorAll(".filter-dropdown").forEach(s=>{s.querySelectorAll(".filter-option").forEach((a,n)=>{a.classList.toggle("active",n===0)})}),Xe(),Re(He,D)})}function Xe(){const e=document.getElementById("clear-filters");e&&e.classList.toggle("hidden",Object.keys(D).length===0)}async function Re(e,t){const s=document.getElementById("news-content");if(!(!s||!e)){s.innerHTML='<div class="flex items-center justify-center py-16"><div class="spinner"></div></div>';try{const a=await f.searchNews(e,{page:1,per_page:20,time_range:t.time}),n=a.results;if(n.length===0){s.innerHTML=`
        <div class="py-8 text-secondary">No news results found for "<strong>${R(e)}</strong>"</div>
      `;return}const r=n.slice(0,5),i=n.slice(5),o=r.length>0?`
      <div class="news-top-stories mb-8">
        <h2 class="text-lg font-medium text-primary mb-4">Top Stories</h2>
        <div class="news-carousel">
          ${r.map(l=>qa(l)).join("")}
        </div>
      </div>
    `:"";s.innerHTML=`
      <div class="news-results-container">
        <div class="text-xs text-tertiary mb-6">
          About ${a.total_results.toLocaleString()} news results (${(a.search_time_ms/1e3).toFixed(2)} seconds)
        </div>
        ${o}
        ${i.length>0?`
          <div class="news-list">
            ${i.map(l=>ja(l)).join("")}
          </div>
        `:""}
      </div>
    `}catch(a){s.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to load news results. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${R(String(a))}</p>
      </div>
    `}}}function qa(e){const t=e.thumbnail?.url||"",s=e.published_date?gt(e.published_date):"";return`
    <a href="${be(e.url)}" target="_blank" rel="noopener" class="news-top-card">
      <div class="news-top-card-image">
        ${t?`<img src="${be(t)}" alt="" loading="lazy" onerror="this.style.display='none'; this.nextElementSibling.style.display='flex'" />`:""}
        <div class="news-image-placeholder" ${t?'style="display:none"':""}>
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="#9aa0a6" stroke-width="1.5">
            <rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect>
            <circle cx="9" cy="9" r="2"></circle>
            <path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"></path>
          </svg>
        </div>
      </div>
      <div class="news-top-card-content">
        <div class="news-source">
          <img class="news-source-icon" src="https://www.google.com/s2/favicons?domain=${encodeURIComponent(e.domain)}&sz=16" alt="" onerror="this.style.display='none'" />
          <span>${R(e.source||e.domain)}</span>
          ${s?`<span class="news-time">· ${R(s)}</span>`:""}
        </div>
        <h3 class="news-top-card-title">${R(e.title)}</h3>
      </div>
    </a>
  `}function ja(e){const t=e.thumbnail?.url||"",s=e.published_date?gt(e.published_date):"";return`
    <article class="news-card-item">
      <div class="news-card-main">
        <div class="news-card-source">
          <img class="news-favicon" src="https://www.google.com/s2/favicons?domain=${encodeURIComponent(e.domain)}&sz=16" alt="" onerror="this.style.display='none'" />
          <span>${R(e.source||e.domain)}</span>
          ${s?`<span class="news-card-time">· ${R(s)}</span>`:""}
        </div>
        <h3 class="news-card-headline">
          <a href="${be(e.url)}" target="_blank" rel="noopener">${R(e.title)}</a>
        </h3>
        <p class="news-card-snippet">${R(e.snippet||"")}</p>
      </div>
      <div class="news-card-thumb">
        ${t?`<img src="${be(t)}" alt="" loading="lazy" onerror="this.style.display='none'; this.nextElementSibling.style.display='flex'" />`:""}
        <div class="news-thumb-placeholder" ${t?'style="display:none"':""}>
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#9aa0a6" stroke-width="1.5">
            <rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect>
            <circle cx="9" cy="9" r="2"></circle>
            <path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"></path>
          </svg>
        </div>
      </div>
    </article>
  `}function gt(e){try{const t=new Date(e),a=new Date().getTime()-t.getTime(),n=Math.floor(a/(1e3*60*60)),r=Math.floor(a/(1e3*60*60*24));return n<1?"Just now":n<24?`${n}h ago`:r===1?"1 day ago":r<7?`${r} days ago`:r<30?`${Math.floor(r/7)} weeks ago`:t.toLocaleDateString("en-US",{month:"short",day:"numeric",year:"numeric"})}catch{return e}}function R(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function be(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const Pa='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',vt='<svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M10 20v-6h4v6h5v-8h3L12 3 2 12h3v8z"/></svg>',Ua='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>',Fa='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z"/></svg>',et='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/><circle cx="12" cy="10" r="3"/></svg>',Va='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>',za='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="7" width="20" height="14" rx="2" ry="2"/><path d="M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16"/></svg>',Da='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="4" y="4" width="16" height="16" rx="2" ry="2"/><rect x="9" y="9" width="6" height="6"/><line x1="9" y1="1" x2="9" y2="4"/><line x1="15" y1="1" x2="15" y2="4"/><line x1="9" y1="20" x2="9" y2="23"/><line x1="15" y1="20" x2="15" y2="23"/><line x1="20" y1="9" x2="23" y2="9"/><line x1="20" y1="14" x2="23" y2="14"/><line x1="1" y1="9" x2="4" y2="9"/><line x1="1" y1="14" x2="4" y2="14"/></svg>',Ga='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="2" width="20" height="20" rx="2.18" ry="2.18"/><line x1="7" y1="2" x2="7" y2="22"/><line x1="17" y1="2" x2="17" y2="22"/><line x1="2" y1="12" x2="22" y2="12"/><line x1="2" y1="7" x2="7" y2="7"/><line x1="2" y1="17" x2="7" y2="17"/><line x1="17" y1="17" x2="22" y2="17"/><line x1="17" y1="7" x2="22" y2="7"/></svg>',Wa='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>',Ka='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2v6.5a.5.5 0 0 0 .5.5h3a.5.5 0 0 1 .5.5v1a.5.5 0 0 1-.5.5H14.5a.5.5 0 0 0-.5.5V22H10V11.5a.5.5 0 0 0-.5-.5H6.5a.5.5 0 0 1-.5-.5v-1a.5.5 0 0 1 .5-.5h3a.5.5 0 0 0 .5-.5V2"/></svg>',Ya='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"/></svg>',Qa='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>',mt=[{id:"top",label:"Top Stories",icon:Qa},{id:"world",label:"World",icon:Va},{id:"nation",label:"U.S.",icon:vt},{id:"business",label:"Business",icon:za},{id:"technology",label:"Technology",icon:Da},{id:"entertainment",label:"Entertainment",icon:Ga},{id:"sports",label:"Sports",icon:Wa},{id:"science",label:"Science",icon:Ka},{id:"health",label:"Health",icon:Ya}];function Ja(){const e=new Date().toLocaleDateString("en-US",{weekday:"long",month:"long",day:"numeric"});return`
    <div class="news-layout">
      <!-- Sidebar Navigation -->
      <nav class="news-sidebar">
        <div class="news-sidebar-header">
          <a href="/" data-link class="news-logo">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
            <span class="news-logo-suffix">News</span>
          </a>
        </div>

        <div class="news-nav-section">
          <a href="/news-home" data-link class="news-nav-item active">
            ${vt}
            <span>Home</span>
          </a>
          <a href="/news-home?section=for-you" data-link class="news-nav-item">
            ${Ua}
            <span>For you</span>
          </a>
          <a href="/news-home?section=following" data-link class="news-nav-item">
            ${Fa}
            <span>Following</span>
          </a>
        </div>

        <div class="news-nav-divider"></div>

        <div class="news-nav-section">
          ${mt.map(t=>`
            <a href="/news-home?category=${t.id}" data-link class="news-nav-item" data-category="${t.id}">
              ${t.icon}
              <span>${t.label}</span>
            </a>
          `).join("")}
        </div>
      </nav>

      <!-- Main Content -->
      <main class="news-main">
        <!-- Header -->
        <header class="news-header">
          <div class="news-header-left">
            <button class="news-menu-btn" id="menu-toggle">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="3" y1="12" x2="21" y2="12"></line>
                <line x1="3" y1="6" x2="21" y2="6"></line>
                <line x1="3" y1="18" x2="21" y2="18"></line>
              </svg>
            </button>
            <a href="/" data-link class="news-logo-mobile">
              <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
              <span class="news-logo-suffix">News</span>
            </a>
          </div>

          <div class="news-search-container">
            ${E({size:"sm",placeholder:"Search for topics, locations & sources"})}
          </div>

          <div class="news-header-right">
            <button class="news-icon-btn" id="location-btn" title="Change location">
              ${et}
            </button>
            <a href="/settings" data-link class="news-icon-btn" title="Settings">
              ${Pa}
            </a>
          </div>
        </header>

        <!-- Content Area -->
        <div class="news-content" id="news-content">
          <div class="news-briefing">
            <h1 class="news-briefing-title">Your briefing</h1>
            <p class="news-briefing-date">${e}</p>
          </div>

          <!-- Loading State -->
          <div class="news-loading" id="news-loading">
            <div class="spinner"></div>
            <p>Loading your news...</p>
          </div>

          <!-- Top Stories Section -->
          <section class="news-section" id="top-stories-section" style="display: none;">
            <h2 class="news-section-title">Top stories</h2>
            <div class="news-grid" id="top-stories-grid"></div>
          </section>

          <!-- For You Section -->
          <section class="news-section" id="for-you-section" style="display: none;">
            <h2 class="news-section-title">For you</h2>
            <div class="news-list" id="for-you-list"></div>
          </section>

          <!-- Local News Section -->
          <section class="news-section" id="local-section" style="display: none;">
            <div class="news-section-header">
              <h2 class="news-section-title">
                ${et}
                <span id="local-title">Local news</span>
              </h2>
              <button class="news-text-btn" id="change-location-btn">Change location</button>
            </div>
            <div class="news-horizontal-scroll" id="local-news-scroll"></div>
          </section>

          <!-- Category Sections -->
          <div id="category-sections"></div>
        </div>
      </main>
    </div>
  `}function Za(e,t){_(n=>{e.navigate(`/news?q=${encodeURIComponent(n)}`)});const s=document.getElementById("menu-toggle"),a=document.querySelector(".news-sidebar");s&&a&&s.addEventListener("click",()=>{a.classList.toggle("open")}),Xa()}async function Xa(e){const t=document.getElementById("news-loading"),s=document.getElementById("top-stories-section"),a=document.getElementById("for-you-section"),n=document.getElementById("local-section");try{const r=await f.newsHome();if(t&&(t.style.display="none"),s&&r.topStories.length>0){s.style.display="block";const o=document.getElementById("top-stories-grid");o&&(o.innerHTML=en(r.topStories))}if(a&&r.forYou.length>0){a.style.display="block";const o=document.getElementById("for-you-list");o&&(o.innerHTML=r.forYou.slice(0,10).map(l=>nn(l)).join(""))}if(n&&r.localNews.length>0){n.style.display="block";const o=document.getElementById("local-news-scroll");o&&(o.innerHTML=r.localNews.map(l=>ft(l)).join(""))}const i=document.getElementById("category-sections");if(i&&r.categories){const o=Object.entries(r.categories).filter(([l,c])=>c&&c.length>0).map(([l,c])=>rn(l,c)).join("");i.innerHTML=o}on()}catch(r){t&&(t.innerHTML=`
        <div class="news-error">
          <p>Failed to load news. Please try again.</p>
          <button class="news-btn" onclick="location.reload()">Retry</button>
        </div>
      `),console.error("Failed to load news:",r)}}function en(e){if(e.length===0)return"";const t=e[0],s=e.slice(1,3),a=e.slice(3,9);return`
    <div class="news-featured-row">
      ${tn(t)}
      <div class="news-secondary-col">
        ${s.map(n=>sn(n)).join("")}
      </div>
    </div>
    <div class="news-grid-row">
      ${a.map(n=>an(n)).join("")}
    </div>
  `}function tn(e){const t=he(e.publishedAt);return`
    <article class="news-card news-card-featured">
      ${e.imageUrl?`<img class="news-card-image" src="${I(e.imageUrl)}" alt="" loading="lazy" onerror="this.style.display='none'" />`:""}
      <div class="news-card-content">
        <div class="news-card-meta">
          <img class="news-source-icon" src="${I(e.sourceIcon||"")}" alt="" onerror="this.style.display='none'" />
          <span class="news-source-name">${L(e.source)}</span>
          <span class="news-time">${t}</span>
        </div>
        <h3 class="news-card-title">
          <a href="${I(e.url)}" target="_blank" rel="noopener" onclick="trackArticleClick('${e.id}')">${L(e.title)}</a>
        </h3>
        <p class="news-card-snippet">${L(e.snippet)}</p>
        ${e.clusterId?`<a href="/news-home?story=${e.clusterId}" data-link class="news-full-coverage">Full coverage</a>`:""}
      </div>
    </article>
  `}function sn(e){const t=he(e.publishedAt);return`
    <article class="news-card news-card-medium">
      <div class="news-card-content">
        <div class="news-card-meta">
          <img class="news-source-icon" src="${I(e.sourceIcon||"")}" alt="" onerror="this.style.display='none'" />
          <span class="news-source-name">${L(e.source)}</span>
          <span class="news-time">${t}</span>
        </div>
        <h3 class="news-card-title">
          <a href="${I(e.url)}" target="_blank" rel="noopener">${L(e.title)}</a>
        </h3>
      </div>
      ${e.imageUrl?`<img class="news-card-thumb" src="${I(e.imageUrl)}" alt="" loading="lazy" onerror="this.style.display='none'" />`:""}
    </article>
  `}function an(e){const t=he(e.publishedAt);return`
    <article class="news-card news-card-small">
      <div class="news-card-content">
        <div class="news-card-meta">
          <span class="news-source-name">${L(e.source)}</span>
          <span class="news-time">${t}</span>
        </div>
        <h3 class="news-card-title">
          <a href="${I(e.url)}" target="_blank" rel="noopener">${L(e.title)}</a>
        </h3>
      </div>
    </article>
  `}function ft(e){const t=he(e.publishedAt);return`
    <article class="news-card news-card-compact">
      ${e.imageUrl?`<img class="news-card-thumb-sm" src="${I(e.imageUrl)}" alt="" loading="lazy" onerror="this.style.display='none'" />`:'<div class="news-card-thumb-placeholder"></div>'}
      <div class="news-card-content">
        <span class="news-source-name">${L(e.source)}</span>
        <h4 class="news-card-title-sm">
          <a href="${I(e.url)}" target="_blank" rel="noopener">${L(e.title)}</a>
        </h4>
        <span class="news-time">${t}</span>
      </div>
    </article>
  `}function nn(e){const t=he(e.publishedAt);return`
    <article class="news-list-item">
      <div class="news-list-content">
        <div class="news-card-meta">
          <img class="news-source-icon" src="${I(e.sourceIcon||"")}" alt="" onerror="this.style.display='none'" />
          <span class="news-source-name">${L(e.source)}</span>
          <span class="news-time">${t}</span>
        </div>
        <h3 class="news-list-title">
          <a href="${I(e.url)}" target="_blank" rel="noopener">${L(e.title)}</a>
        </h3>
        <p class="news-list-snippet">${L(e.snippet)}</p>
      </div>
      ${e.imageUrl?`<img class="news-list-thumb" src="${I(e.imageUrl)}" alt="" loading="lazy" onerror="this.style.display='none'" />`:""}
    </article>
  `}function rn(e,t){const s=mt.find(a=>a.id===e);return s?`
    <section class="news-section">
      <div class="news-section-header">
        <h2 class="news-section-title">
          ${s.icon}
          <span>${s.label}</span>
        </h2>
        <a href="/news-home?category=${e}" data-link class="news-text-btn">More ${s.label.toLowerCase()}</a>
      </div>
      <div class="news-horizontal-scroll">
        ${t.slice(0,5).map(a=>ft(a)).join("")}
      </div>
    </section>
  `:""}function he(e){try{const t=new Date(e),a=new Date().getTime()-t.getTime(),n=Math.floor(a/(1e3*60*60)),r=Math.floor(a/(1e3*60*60*24));return n<1?"Just now":n<24?`${n}h ago`:r===1?"1 day ago":r<7?`${r} days ago`:t.toLocaleDateString("en-US",{month:"short",day:"numeric"})}catch{return""}}function on(){document.querySelectorAll(".news-card a, .news-list-item a").forEach(e=>{e.addEventListener("click",function(){const t=this.closest(".news-card, .news-list-item");if(t){const s=t.getAttribute("data-article-id");s&&f.recordNewsRead({id:s,url:this.href,title:this.textContent||"",snippet:"",source:"",sourceUrl:"",publishedAt:"",category:"top",engines:[],score:1}).catch(()=>{})}})})}function L(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function I(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const ln='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',cn='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><path d="M12 18v-6"/><path d="m9 15 3 3 3-3"/></svg>',dn='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 21c3 0 7-1 7-8V5c0-1.25-.756-2.017-2-2H4c-1.25 0-2 .75-2 1.972V11c0 1.25.75 2 2 2 1 0 1 0 1 1v1c0 1-1 2-2 2s-1 .008-1 1.031V21"/><path d="M15 21c3 0 7-1 7-8V5c0-1.25-.757-2.017-2-2h-4c-1.25 0-2 .75-2 1.972V11c0 1.25.75 2 2 2h.75c0 2.25.25 4-2.75 4v3"/></svg>';function un(e){return`
    <div class="min-h-screen flex flex-col">
      <header class="search-header">
        <div class="search-header-row">
          <a href="/" data-link class="search-logo">
            <span style="color: #2563eb">M</span><span style="color: #ef4444">i</span><span style="color: #f59e0b">z</span><span style="color: #22c55e">u</span>
          </a>
          <div class="search-header-box">
            ${E({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="search-box-btn" aria-label="Settings">
            ${ln}
          </a>
        </div>
        <div class="search-tabs-row">
          ${q({query:e,active:"science"})}
        </div>
      </header>
      <main class="flex-1">
        <div id="science-content" class="search-content-area">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `}function hn(e,t){_(s=>{e.navigate(`/science?q=${encodeURIComponent(s)}`)}),j(),t&&O(t),pn(t)}async function pn(e){const t=document.getElementById("science-content");if(!(!t||!e))try{const s=await f.searchScience(e),a=s.results;if(a.length===0){t.innerHTML=`
        <div class="w-full">
          <div class="py-8 text-secondary">No academic results found for "<strong>${T(e)}</strong>"</div>
        </div>
      `;return}t.innerHTML=`
      <div class="w-full">
        <div class="text-xs text-tertiary mb-4">
          About ${s.total_results.toLocaleString()} results (${(s.search_time_ms/1e3).toFixed(2)} seconds)
        </div>
        <div class="space-y-6">
          ${a.map(gn).join("")}
        </div>
      </div>
    `}catch(s){t.innerHTML=`
      <div class="w-full">
        <div class="py-8">
          <p class="text-red text-sm">Failed to load academic results. Please try again.</p>
          <p class="text-tertiary text-xs mt-2">${T(String(s))}</p>
        </div>
      </div>
    `}}function gn(e){const t=e,s=t.metadata?.authors||"",a=t.metadata?.year||"",n=t.metadata?.citations,r=t.metadata?.doi||"",i=t.metadata?.pdf_url||"",o=t.metadata?.source||vn(e.url);return`
    <article class="paper-card bg-white border border-border rounded-xl p-5 hover:shadow-md transition-shadow">
      <div class="flex items-start gap-3 mb-2">
        <span class="text-xs px-2 py-0.5 bg-blue/10 text-blue rounded-full font-medium">${T(o)}</span>
        ${a?`<span class="text-xs text-tertiary">${T(a)}</span>`:""}
      </div>
      <h3 class="text-lg font-medium text-primary mb-2">
        <a href="${Ee(e.url)}" target="_blank" rel="noopener" class="hover:text-blue hover:underline">${T(e.title)}</a>
      </h3>
      ${s?`<p class="text-sm text-secondary mb-2">${T(s)}</p>`:""}
      <p class="text-sm text-snippet line-clamp-3 mb-3">${T(e.snippet??"")}</p>
      <div class="flex items-center gap-4 text-xs">
        ${n!=null?`<span class="flex items-center gap-1 text-tertiary">${dn} ${T(String(n))} citations</span>`:""}
        ${r?`<a href="https://doi.org/${Ee(r)}" target="_blank" rel="noopener" class="text-tertiary hover:text-blue">DOI: ${T(r)}</a>`:""}
        ${i?`<a href="${Ee(i)}" target="_blank" rel="noopener" class="flex items-center gap-1 text-blue hover:underline">${cn} PDF</a>`:""}
      </div>
    </article>
  `}function vn(e){try{return new URL(e).hostname.replace("www.","")}catch{return""}}function T(e){return(e==null?"":String(e)).replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function Ee(e){return(e==null?"":String(e)).replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const mn='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',fn='<svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>',yn='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="18" r="3"/><circle cx="6" cy="6" r="3"/><circle cx="18" cy="6" r="3"/><path d="M18 9v1a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2V9"/><path d="M12 12v3"/></svg>',wn='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" x2="12" y1="15" y2="3"/></svg>';function bn(e){return`
    <div class="min-h-screen flex flex-col">
      <header class="search-header">
        <div class="search-header-row">
          <a href="/" data-link class="search-logo">
            <span style="color: #2563eb">M</span><span style="color: #ef4444">i</span><span style="color: #f59e0b">z</span><span style="color: #22c55e">u</span>
          </a>
          <div class="search-header-box">
            ${E({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="search-box-btn" aria-label="Settings">
            ${mn}
          </a>
        </div>
        <div class="search-tabs-row">
          ${q({query:e,active:"code"})}
        </div>
      </header>
      <main class="flex-1">
        <div id="code-content" class="search-content-area">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `}function xn(e,t){_(s=>e.navigate(`/code?q=${encodeURIComponent(s)}`)),j(),t&&O(t),$n(t)}async function $n(e){const t=document.getElementById("code-content");if(!(!t||!e))try{const s=await f.searchCode(e),a=s.results;if(a.length===0){t.innerHTML=`<div class="py-8 text-secondary">No code results found for "${le(e)}"</div>`;return}t.innerHTML=`
      <div class="text-xs text-tertiary mb-4">
        About ${s.total_results.toLocaleString()} results (${(s.search_time_ms/1e3).toFixed(2)} seconds)
      </div>
      <div class="w-full space-y-4">
        ${a.map(kn).join("")}
      </div>
    `}catch(s){t.innerHTML=`<div class="py-8 text-red text-sm">Failed to load results. ${le(String(s))}</div>`}}function kn(e){const t=e.metadata?.source||Sn(e.url),s=e.metadata?.stars,a=e.metadata?.forks,n=e.metadata?.downloads,r=e.metadata?.language||"",i=e.metadata?.votes,o=e.metadata?.answers;return`
    <article class="code-card bg-white border border-border rounded-xl p-4 hover:shadow-md transition-shadow">
      <div class="flex items-start gap-3 mb-2">
        <span class="text-xs px-2 py-0.5 rounded-full font-medium ${Cn(t)}">${le(t)}</span>
        ${r?`<span class="text-xs px-2 py-0.5 bg-surface text-secondary rounded-full">${le(r)}</span>`:""}
      </div>
      <h3 class="text-base font-medium text-primary mb-1">
        <a href="${Ln(e.url)}" target="_blank" rel="noopener" class="hover:text-blue hover:underline">${le(e.title)}</a>
      </h3>
      <p class="text-sm text-snippet line-clamp-2 mb-3">${e.content||""}</p>
      <div class="flex items-center gap-4 text-xs text-tertiary">
        ${s!==void 0?`<span class="flex items-center gap-1">${fn} <span class="text-yellow-500">${me(s)}</span></span>`:""}
        ${a!==void 0?`<span class="flex items-center gap-1">${yn} ${me(a)}</span>`:""}
        ${n!==void 0?`<span class="flex items-center gap-1">${wn} ${me(n)}</span>`:""}
        ${i!==void 0?`<span class="flex items-center gap-1">▲ ${me(i)}</span>`:""}
        ${o!==void 0?`<span class="flex items-center gap-1">${o} answers</span>`:""}
      </div>
    </article>
  `}function Cn(e){return e.includes("github")?"bg-gray-900 text-white":e.includes("gitlab")?"bg-orange-500 text-white":e.includes("stackoverflow")?"bg-orange-400 text-white":e.includes("npm")?"bg-red-500 text-white":e.includes("pypi")?"bg-blue-500 text-white":e.includes("crates")?"bg-orange-600 text-white":"bg-surface text-secondary"}function me(e){return e>=1e6?(e/1e6).toFixed(1)+"M":e>=1e3?(e/1e3).toFixed(1)+"K":String(e)}function Sn(e){try{return new URL(e).hostname.replace("www.","")}catch{return""}}function le(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function Ln(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const In='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',En='<svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg>',_n='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/></svg>';function Bn(e){return`
    <div class="min-h-screen flex flex-col">
      <header class="search-header">
        <div class="search-header-row">
          <a href="/" data-link class="search-logo">
            <span style="color: #2563eb">M</span><span style="color: #ef4444">i</span><span style="color: #f59e0b">z</span><span style="color: #22c55e">u</span>
          </a>
          <div class="search-header-box">
            ${E({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="search-box-btn" aria-label="Settings">
            ${In}
          </a>
        </div>
        <div class="search-tabs-row">
          ${q({query:e,active:"music"})}
        </div>
      </header>
      <main class="flex-1">
        <div id="music-content" class="search-content-area">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `}function Mn(e,t){_(s=>e.navigate(`/music?q=${encodeURIComponent(s)}`)),j(),t&&O(t),Tn(t)}async function Tn(e){const t=document.getElementById("music-content");if(!(!t||!e))try{const s=await f.searchMusic(e),a=s.results;if(a.length===0){t.innerHTML=`<div class="py-8 text-secondary">No music results found for "${F(e)}"</div>`;return}t.innerHTML=`
      <div class="text-xs text-tertiary mb-4">
        About ${s.total_results.toLocaleString()} results (${(s.search_time_ms/1e3).toFixed(2)} seconds)
      </div>
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
        ${a.map(An).join("")}
      </div>
    `}catch(s){t.innerHTML=`<div class="py-8 text-red text-sm">Failed to load results. ${F(String(s))}</div>`}}function An(e){const t=e.metadata?.source||Hn(e.url),s=e.metadata?.artist||"",a=e.metadata?.album||"",n=e.metadata?.duration||"",r=e.thumbnail?.url||"",i=t.toLowerCase().includes("genius");return`
    <article class="music-card bg-white border border-border rounded-xl overflow-hidden hover:shadow-md transition-shadow">
      <a href="${tt(e.url)}" target="_blank" rel="noopener" class="block">
        <div class="relative aspect-square bg-surface">
          ${r?`<img src="${tt(r)}" alt="" class="w-full h-full object-cover" loading="lazy" onerror="this.style.display='none'" />`:`<div class="w-full h-full flex items-center justify-center text-border">${_n}</div>`}
          <div class="absolute inset-0 bg-black/40 opacity-0 hover:opacity-100 transition-opacity flex items-center justify-center">
            <span class="w-12 h-12 rounded-full bg-white flex items-center justify-center text-primary">${En}</span>
          </div>
        </div>
        <div class="p-3">
          <span class="text-xs px-2 py-0.5 rounded-full font-medium ${Nn(t)}">${F(t)}</span>
          <h3 class="text-sm font-medium text-primary mt-2 line-clamp-2">${F(e.title)}</h3>
          ${s?`<p class="text-xs text-secondary mt-1">${F(s)}</p>`:""}
          ${a?`<p class="text-xs text-tertiary">${F(a)}</p>`:""}
          ${n?`<p class="text-xs text-tertiary mt-1">${F(n)}</p>`:""}
          ${i&&e.snippet?`<p class="text-xs text-snippet mt-2 line-clamp-2 italic">"${F(e.snippet.slice(0,100))}..."</p>`:""}
        </div>
      </a>
    </article>
  `}function Nn(e){return e.toLowerCase().includes("soundcloud")?"bg-orange-500 text-white":e.toLowerCase().includes("bandcamp")?"bg-teal-500 text-white":e.toLowerCase().includes("genius")?"bg-yellow-400 text-black":"bg-surface text-secondary"}function Hn(e){try{return new URL(e).hostname.replace("www.","")}catch{return""}}function F(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function tt(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const Rn='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',On='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m18 15-6-6-6 6"/></svg>',qn='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>';function jn(e){return`
    <div class="min-h-screen flex flex-col">
      <header class="search-header">
        <div class="search-header-row">
          <a href="/" data-link class="search-logo">
            <span style="color: #2563eb">M</span><span style="color: #ef4444">i</span><span style="color: #f59e0b">z</span><span style="color: #22c55e">u</span>
          </a>
          <div class="search-header-box">
            ${E({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="search-box-btn" aria-label="Settings">
            ${Rn}
          </a>
        </div>
        <div class="search-tabs-row">
          ${q({query:e,active:"social"})}
        </div>
      </header>
      <main class="flex-1">
        <div id="social-content" class="search-content-area">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `}function Pn(e,t){_(s=>e.navigate(`/social?q=${encodeURIComponent(s)}`)),j(),t&&O(t),Un(t)}async function Un(e){const t=document.getElementById("social-content");if(!(!t||!e))try{const s=await f.searchSocial(e),a=s.results;if(a.length===0){t.innerHTML=`<div class="py-8 text-secondary">No social results found for "${Y(e)}"</div>`;return}t.innerHTML=`
      <div class="text-xs text-tertiary mb-4">
        About ${s.total_results.toLocaleString()} results (${(s.search_time_ms/1e3).toFixed(2)} seconds)
      </div>
      <div class="w-full space-y-4">
        ${a.map(Fn).join("")}
      </div>
    `}catch(s){t.innerHTML=`<div class="py-8 text-red text-sm">Failed to load results. ${Y(String(s))}</div>`}}function Fn(e){const t=e.metadata||{},s=t.source||Dn(e.url),a=t.upvotes||t.score||0,n=t.comments||0,r=t.author||"",i=t.subreddit||"",o=t.published||"",l=e.thumbnail?.url||"",c=e.snippet||"";return`
    <article class="social-card bg-white border border-border rounded-xl p-4 hover:shadow-md transition-shadow">
      <div class="flex items-start gap-3">
        <!-- Upvote column -->
        <div class="flex flex-col items-center text-tertiary text-sm">
          ${On}
          <span class="font-medium ${a>0?"text-orange-500":""}">${st(a)}</span>
        </div>
        <!-- Content -->
        <div class="flex-1 min-w-0">
          <div class="flex items-center gap-2 mb-1 flex-wrap">
            <span class="text-xs px-2 py-0.5 rounded-full font-medium ${Vn(s)}">${Y(s)}</span>
            ${i?`<span class="text-xs text-blue">r/${Y(i)}</span>`:""}
            ${r?`<span class="text-xs text-tertiary">by ${Y(r)}</span>`:""}
            ${o?`<span class="text-xs text-tertiary">${zn(o)}</span>`:""}
          </div>
          <h3 class="text-base font-medium text-primary mb-1">
            <a href="${at(e.url)}" target="_blank" rel="noopener" class="hover:text-blue hover:underline">${Y(e.title)}</a>
          </h3>
          ${c?`<p class="text-sm text-snippet line-clamp-3 mb-2">${Y(c)}</p>`:""}
          <div class="flex items-center gap-4 text-xs text-tertiary">
            <span class="flex items-center gap-1">${qn} ${st(n)} comments</span>
          </div>
        </div>
        <!-- Thumbnail if available -->
        ${l?`
          <img src="${at(l)}" alt="" class="w-20 h-20 rounded-lg object-cover flex-shrink-0" loading="lazy" onerror="this.style.display='none'" />
        `:""}
      </div>
    </article>
  `}function Vn(e){const t=e.toLowerCase();return t.includes("reddit")?"bg-orange-500 text-white":t.includes("hacker")||t.includes("hn")?"bg-orange-600 text-white":t.includes("mastodon")?"bg-purple-500 text-white":t.includes("lemmy")?"bg-green-500 text-white":"bg-surface text-secondary"}function st(e){return e>=1e6?(e/1e6).toFixed(1)+"M":e>=1e3?(e/1e3).toFixed(1)+"K":String(e)}function zn(e){try{const t=new Date(e),a=new Date().getTime()-t.getTime(),n=Math.floor(a/(1e3*60*60)),r=Math.floor(a/(1e3*60*60*24));return n<1?"just now":n<24?`${n}h ago`:r<7?`${r}d ago`:r<30?`${Math.floor(r/7)}w ago`:`${Math.floor(r/30)}mo ago`}catch{return""}}function Dn(e){try{return new URL(e).hostname.replace("www.","")}catch{return""}}function Y(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function at(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const Gn='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',nt='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" x2="21" y1="14" y2="3"/></svg>';function Wn(e){return`
    <div class="min-h-screen flex flex-col">
      <header class="search-header">
        <div class="search-header-row">
          <a href="/" data-link class="search-logo">
            <span style="color: #2563eb">M</span><span style="color: #ef4444">i</span><span style="color: #f59e0b">z</span><span style="color: #22c55e">u</span>
          </a>
          <div class="search-header-box">
            ${E({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="search-box-btn" aria-label="Settings">
            ${Gn}
          </a>
        </div>
        <div class="search-tabs-row">
          ${q({query:e,active:"maps"})}
        </div>
      </header>
      <main class="flex-1 flex flex-col lg:flex-row">
        <!-- Map area -->
        <div id="map-container" class="h-[300px] lg:h-auto lg:flex-1 bg-surface relative">
          <iframe id="map-iframe" class="w-full h-full border-0" src="" title="Map"></iframe>
        </div>
        <!-- Results sidebar -->
        <div id="maps-content" class="lg:w-[400px] lg:border-l border-border overflow-y-auto">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `}function Kn(e,t){_(s=>e.navigate(`/maps?q=${encodeURIComponent(s)}`)),j(),t&&O(t),Yn(t)}async function Yn(e){const t=document.getElementById("maps-content"),s=document.getElementById("map-iframe");if(!(!t||!e)){if(s){const a="https://www.openstreetmap.org/export/embed.html?bbox=-180,-90,180,90&layer=mapnik&marker=0,0";s.src=a}try{const n=(await f.searchMaps(e)).results;if(n.length===0){t.innerHTML=`<div class="p-4 text-secondary">No locations found for "${ce(e)}"</div>`;return}const r=n[0],i=r.metadata?.lat||0,o=r.metadata?.lon||0;if(s&&i&&o){const l=`${o-.1},${i-.1},${o+.1},${i+.1}`;s.src=`https://www.openstreetmap.org/export/embed.html?bbox=${l}&layer=mapnik&marker=${i},${o}`}t.innerHTML=`
      <div class="p-4">
        <div class="text-xs text-tertiary mb-4">${n.length} locations found</div>
        <div class="space-y-3">
          ${n.map((l,c)=>Qn(l,c)).join("")}
        </div>
      </div>
    `,t.querySelectorAll(".location-card").forEach(l=>{l.addEventListener("click",()=>{const c=l.dataset.lat,v=l.dataset.lon;if(c&&v&&s){const h=`${parseFloat(v)-.05},${parseFloat(c)-.05},${parseFloat(v)+.05},${parseFloat(c)+.05}`;s.src=`https://www.openstreetmap.org/export/embed.html?bbox=${h}&layer=mapnik&marker=${c},${v}`}})})}catch(a){t.innerHTML=`<div class="p-4 text-red text-sm">Failed to load results. ${ce(String(a))}</div>`}}}function Qn(e,t){const s=e.metadata?.lat||0,a=e.metadata?.lon||0,n=e.metadata?.type||"place",r=e.content||"";return`
    <article class="location-card bg-white border border-border rounded-lg p-3 cursor-pointer hover:shadow-md transition-shadow"
             data-lat="${s}" data-lon="${a}">
      <div class="flex items-start gap-3">
        <span class="flex-shrink-0 w-8 h-8 rounded-full bg-red-500 text-white flex items-center justify-center text-sm font-medium">
          ${t+1}
        </span>
        <div class="flex-1 min-w-0">
          <h3 class="font-medium text-primary text-sm">${ce(e.title)}</h3>
          <p class="text-xs text-tertiary mt-0.5 capitalize">${ce(n)}</p>
          ${r?`<p class="text-xs text-secondary mt-1 line-clamp-2">${ce(r)}</p>`:""}
          <p class="text-xs text-tertiary mt-1">${s.toFixed(5)}, ${a.toFixed(5)}</p>
          <div class="flex items-center gap-2 mt-2">
            <a href="${Jn(e.url)}" target="_blank" rel="noopener"
               class="text-xs text-blue hover:underline flex items-center gap-1"
               onclick="event.stopPropagation()">
              View on OSM ${nt}
            </a>
            <a href="https://www.google.com/maps?q=${s},${a}" target="_blank" rel="noopener"
               class="text-xs text-blue hover:underline flex items-center gap-1"
               onclick="event.stopPropagation()">
              Google Maps ${nt}
            </a>
          </div>
        </div>
      </div>
    </article>
  `}function ce(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function Jn(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const Zn='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m12 19-7-7 7-7"/><path d="M19 12H5"/></svg>',Xn=[{value:"auto",label:"Auto-detect"},{value:"us",label:"United States"},{value:"gb",label:"United Kingdom"},{value:"de",label:"Germany"},{value:"fr",label:"France"},{value:"es",label:"Spain"},{value:"it",label:"Italy"},{value:"nl",label:"Netherlands"},{value:"pl",label:"Poland"},{value:"br",label:"Brazil"},{value:"ca",label:"Canada"},{value:"au",label:"Australia"},{value:"in",label:"India"},{value:"jp",label:"Japan"},{value:"kr",label:"South Korea"},{value:"cn",label:"China"},{value:"ru",label:"Russia"}],er=[{value:"en",label:"English"},{value:"de",label:"German (Deutsch)"},{value:"fr",label:"French (Français)"},{value:"es",label:"Spanish (Español)"},{value:"it",label:"Italian (Italiano)"},{value:"pt",label:"Portuguese (Português)"},{value:"nl",label:"Dutch (Nederlands)"},{value:"pl",label:"Polish (Polski)"},{value:"ja",label:"Japanese"},{value:"ko",label:"Korean"},{value:"zh",label:"Chinese"},{value:"ru",label:"Russian"},{value:"ar",label:"Arabic"},{value:"hi",label:"Hindi"}];function tr(){const e=Q.get().settings;return`
    <div class="min-h-screen bg-white">
      <!-- Header -->
      <header class="border-b border-border">
        <div class="max-w-[700px] mx-auto px-4 py-4 flex items-center gap-4">
          <a href="/" data-link class="text-tertiary hover:text-primary transition-colors" aria-label="Back">
            ${Zn}
          </a>
          <h1 class="text-xl font-semibold text-primary">Settings</h1>
        </div>
      </header>

      <!-- Form -->
      <main class="max-w-[700px] mx-auto px-4 py-8">
        <form id="settings-form">
          <!-- Safe Search -->
          <div class="settings-section">
            <h3>Safe Search</h3>
            <div class="space-y-1">
              <label class="settings-label">
                <input type="radio" name="safe_search" value="off" ${e.safe_search==="off"?"checked":""} />
                <span>Off</span>
              </label>
              <label class="settings-label">
                <input type="radio" name="safe_search" value="moderate" ${e.safe_search==="moderate"?"checked":""} />
                <span>Moderate</span>
              </label>
              <label class="settings-label">
                <input type="radio" name="safe_search" value="strict" ${e.safe_search==="strict"?"checked":""} />
                <span>Strict</span>
              </label>
            </div>
          </div>

          <!-- Results per page -->
          <div class="settings-section">
            <h3>Results per page</h3>
            <select name="results_per_page" class="settings-select">
              ${[10,20,30,50].map(t=>`<option value="${t}" ${e.results_per_page===t?"selected":""}>${t}</option>`).join("")}
            </select>
          </div>

          <!-- Region -->
          <div class="settings-section">
            <h3>Region</h3>
            <select name="region" class="settings-select">
              ${Xn.map(t=>`<option value="${t.value}" ${e.region===t.value?"selected":""}>${rt(t.label)}</option>`).join("")}
            </select>
          </div>

          <!-- Language -->
          <div class="settings-section">
            <h3>Language</h3>
            <select name="language" class="settings-select">
              ${er.map(t=>`<option value="${t.value}" ${e.language===t.value?"selected":""}>${rt(t.label)}</option>`).join("")}
            </select>
          </div>

          <!-- Theme -->
          <div class="settings-section">
            <h3>Theme</h3>
            <div class="space-y-1">
              <label class="settings-label">
                <input type="radio" name="theme" value="light" ${e.theme==="light"?"checked":""} />
                <span>Light</span>
              </label>
              <label class="settings-label">
                <input type="radio" name="theme" value="dark" ${e.theme==="dark"?"checked":""} />
                <span>Dark</span>
              </label>
              <label class="settings-label">
                <input type="radio" name="theme" value="system" ${e.theme==="system"?"checked":""} />
                <span>System</span>
              </label>
            </div>
          </div>

          <!-- Open in new tab -->
          <div class="settings-section">
            <h3>Behavior</h3>
            <label class="settings-label">
              <input type="checkbox" name="open_in_new_tab" ${e.open_in_new_tab?"checked":""} />
              <span>Open results in new tab</span>
            </label>
            <label class="settings-label">
              <input type="checkbox" name="show_thumbnails" ${e.show_thumbnails?"checked":""} />
              <span>Show thumbnails in results</span>
            </label>
          </div>

          <!-- Save Button -->
          <div class="flex items-center gap-4 pt-4">
            <button type="submit" id="settings-save-btn"
                    class="px-6 py-2.5 bg-blue text-white rounded-lg font-medium text-sm hover:bg-blue/90 transition-colors cursor-pointer">
              Save settings
            </button>
            <span id="settings-status" class="text-sm text-green hidden">Settings saved!</span>
          </div>
        </form>
      </main>
    </div>
  `}function sr(e){const t=document.getElementById("settings-form"),s=document.getElementById("settings-status");t&&t.addEventListener("submit",async a=>{a.preventDefault();const n=new FormData(t),r={safe_search:n.get("safe_search")||"moderate",results_per_page:parseInt(n.get("results_per_page"))||10,region:n.get("region")||"auto",language:n.get("language")||"en",theme:n.get("theme")||"light",open_in_new_tab:n.has("open_in_new_tab"),show_thumbnails:n.has("show_thumbnails")};Q.set({settings:r});try{await f.updateSettings(r)}catch{}s&&(s.classList.remove("hidden"),setTimeout(()=>{s.classList.add("hidden")},2e3))})}function rt(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}const ar='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m12 19-7-7 7-7"/><path d="M19 12H5"/></svg>',nr='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/></svg>',rr='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>',ir='<svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="#dadce0" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/><path d="M12 7v5l4 2"/></svg>';function or(){return`
    <div class="min-h-screen bg-white">
      <!-- Header -->
      <header class="border-b border-border">
        <div class="max-w-[700px] mx-auto px-4 py-4 flex items-center justify-between">
          <div class="flex items-center gap-4">
            <a href="/" data-link class="text-tertiary hover:text-primary transition-colors" aria-label="Back">
              ${ar}
            </a>
            <h1 class="text-xl font-semibold text-primary">Search History</h1>
          </div>
          <button id="clear-all-btn" class="text-sm text-red hover:text-red/80 font-medium cursor-pointer hidden">
            Clear all
          </button>
        </div>
      </header>

      <!-- Content -->
      <main class="max-w-[700px] mx-auto px-4 py-6">
        <div id="history-content">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `}function lr(e){const t=document.getElementById("clear-all-btn");cr(e),t?.addEventListener("click",async()=>{if(confirm("Are you sure you want to clear all search history?"))try{await f.clearHistory(),qe(),t.classList.add("hidden")}catch(s){console.error("Failed to clear history:",s)}})}async function cr(e){const t=document.getElementById("history-content"),s=document.getElementById("clear-all-btn");if(t)try{const a=await f.getHistory();if(a.length===0){qe();return}s&&s.classList.remove("hidden"),t.innerHTML=`
      <div id="history-list">
        ${a.map(n=>dr(n)).join("")}
      </div>
    `,ur(e)}catch(a){t.innerHTML=`
      <div class="py-8 text-center">
        <p class="text-red text-sm">Failed to load search history.</p>
        <p class="text-tertiary text-xs mt-2">${Oe(String(a))}</p>
      </div>
    `}}function dr(e){const t=hr(e.searched_at);return`
    <div class="history-item flex items-center gap-3 py-3 px-2 border-b border-border hover:bg-surface-hover rounded transition-colors group" data-history-id="${it(e.id)}">
      <span class="text-light flex-shrink-0">${rr}</span>
      <div class="flex-1 min-w-0">
        <a href="/search?q=${encodeURIComponent(e.query)}" data-link class="text-sm text-primary hover:text-link font-medium truncate block">
          ${Oe(e.query)}
        </a>
        <div class="flex items-center gap-2 text-xs text-light mt-0.5">
          <span>${Oe(t)}</span>
          ${e.results>0?`<span>&middot; ${e.results} results</span>`:""}
          ${e.clicked_url?"<span>&middot; visited</span>":""}
        </div>
      </div>
      <button class="history-delete-btn text-light hover:text-red p-1.5 rounded-full hover:bg-red/10 opacity-0 group-hover:opacity-100 transition-opacity flex-shrink-0 cursor-pointer"
              data-delete-id="${it(e.id)}" aria-label="Delete">
        ${nr}
      </button>
    </div>
  `}function ur(e){document.querySelectorAll(".history-delete-btn").forEach(t=>{t.addEventListener("click",async s=>{s.preventDefault(),s.stopPropagation();const a=t.dataset.deleteId||"",n=t.closest(".history-item");try{await f.deleteHistoryItem(a),n&&n.remove();const r=document.getElementById("history-list");if(r&&r.children.length===0){qe();const i=document.getElementById("clear-all-btn");i&&i.classList.add("hidden")}}catch(r){console.error("Failed to delete history item:",r)}})})}function qe(){const e=document.getElementById("history-content");e&&(e.innerHTML=`
    <div class="py-16 flex flex-col items-center text-center">
      ${ir}
      <h2 class="text-lg font-medium text-primary mt-4 mb-2">No search history</h2>
      <p class="text-sm text-tertiary max-w-[300px]">
        Your recent searches will appear here. Start searching to build your history.
      </p>
      <a href="/" data-link class="mt-4 text-sm text-blue hover:underline">Go to search</a>
    </div>
  `)}function hr(e){try{const t=new Date(e),s=new Date,a=s.getTime()-t.getTime(),n=Math.floor(a/(1e3*60)),r=Math.floor(a/(1e3*60*60)),i=Math.floor(a/(1e3*60*60*24));return n<1?"Just now":n<60?`${n}m ago`:r<24?`${r}h ago`:i===1?"Yesterday":i<7?`${i} days ago`:t.toLocaleDateString("en-US",{month:"short",day:"numeric",year:t.getFullYear()!==s.getFullYear()?"numeric":void 0})}catch{return e}}function Oe(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function it(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const x=document.getElementById("app");if(!x)throw new Error("App container not found");const m=new yt;m.addRoute("",(e,t)=>{x.innerHTML=Vt(),zt(m)});m.addRoute("search",(e,t)=>{const s=t.q||"",a={timeRange:t.time_range||"",region:t.region||"",verbatim:t.verbatim==="1",site:t.site||""};x.innerHTML=Ys(s,a),Qs(m,s,t)});m.addRoute("images",(e,t)=>{const s=t.q||"";x.innerHTML=oa(s),ca(m,s,t)});m.addRoute("videos",(e,t)=>{const s=t.q||"";x.innerHTML=ka(s),Sa(m,s)});m.addRoute("news",(e,t)=>{const s=t.q||"";x.innerHTML=Aa(s),Ha(m,s)});m.addRoute("news-home",(e,t)=>{x.innerHTML=Ja(),Za(m)});m.addRoute("science",(e,t)=>{const s=t.q||"";x.innerHTML=un(s),hn(m,s)});m.addRoute("code",(e,t)=>{const s=t.q||"";x.innerHTML=bn(s),xn(m,s)});m.addRoute("music",(e,t)=>{const s=t.q||"";x.innerHTML=Bn(s),Mn(m,s)});m.addRoute("social",(e,t)=>{const s=t.q||"";x.innerHTML=jn(s),Pn(m,s)});m.addRoute("maps",(e,t)=>{const s=t.q||"";x.innerHTML=Wn(s),Kn(m,s)});m.addRoute("settings",(e,t)=>{x.innerHTML=tr(),sr()});m.addRoute("history",(e,t)=>{x.innerHTML=or(),lr(m)});m.setNotFound((e,t)=>{x.innerHTML=`
    <div class="min-h-screen flex flex-col items-center justify-center px-4">
      <h1 class="text-4xl font-semibold mb-4">
        <span style="color: #4285F4">4</span><span style="color: #EA4335">0</span><span style="color: #FBBC05">4</span>
      </h1>
      <p class="text-secondary mb-6">Page not found</p>
      <a href="/" data-link class="text-blue hover:underline">Go home</a>
    </div>
  `});window.addEventListener("router:navigate",e=>{const t=e;m.navigate(t.detail.path)});m.start();function pr(){document.addEventListener("keydown",e=>{const t=e.target,s=t.tagName==="INPUT"||t.tagName==="TEXTAREA"||t.isContentEditable;if(e.key==="/"&&!s){e.preventDefault();const a=document.getElementById("search-input");a&&(a.focus(),a.select())}e.key==="Escape"&&(document.querySelectorAll(".modal:not(.hidden), .preview-panel:not(.hidden), .lightbox:not(.hidden)").forEach(r=>{r.classList.add("hidden")}),document.querySelectorAll(".autocomplete-dropdown:not(.hidden), .filter-dropdown:not(.hidden), .filter-pill-dropdown:not(.hidden), .more-tabs-dropdown:not(.hidden), .time-filter-dropdown:not(.hidden), .search-tool-menu:not(.hidden)").forEach(r=>{r.classList.add("hidden")}),document.body.style.overflow="",document.activeElement?.id==="search-input"&&document.activeElement.blur()),e.key==="?"&&!s&&gr()})}function gr(){let e=document.getElementById("keyboard-shortcuts-help");if(e){e.classList.toggle("hidden");return}e=document.createElement("div"),e.id="keyboard-shortcuts-help",e.className="modal",e.innerHTML=`
    <div class="modal-content" style="max-width: 400px;">
      <div class="modal-header">
        <h2>Keyboard Shortcuts</h2>
        <button class="modal-close" id="shortcuts-close">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>
        </button>
      </div>
      <div class="modal-body">
        <div class="shortcuts-list">
          <div class="shortcut-item">
            <kbd>/</kbd>
            <span>Focus search box</span>
          </div>
          <div class="shortcut-item">
            <kbd>Escape</kbd>
            <span>Close modal / unfocus</span>
          </div>
          <div class="shortcut-item">
            <kbd>?</kbd>
            <span>Show this help</span>
          </div>
          <div class="shortcut-item">
            <kbd>&#8593;</kbd> <kbd>&#8595;</kbd>
            <span>Navigate suggestions</span>
          </div>
          <div class="shortcut-item">
            <kbd>Enter</kbd>
            <span>Select / submit</span>
          </div>
        </div>
      </div>
    </div>
  `,document.body.appendChild(e),e.addEventListener("click",t=>{t.target===e&&e.classList.add("hidden")}),document.getElementById("shortcuts-close")?.addEventListener("click",()=>{e.classList.add("hidden")})}pr();"serviceWorker"in navigator&&(navigator.serviceWorker.getRegistrations().then(e=>{for(const t of e)t.unregister()}),"caches"in window&&caches.keys().then(e=>{for(const t of e)caches.delete(t)}));
