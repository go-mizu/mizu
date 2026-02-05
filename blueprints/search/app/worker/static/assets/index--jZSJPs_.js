var Ae=Object.defineProperty;var Te=(e,t,s)=>t in e?Ae(e,t,{enumerable:!0,configurable:!0,writable:!0,value:s}):e[t]=s;var K=(e,t,s)=>Te(e,typeof t!="symbol"?t+"":t,s);(function(){const t=document.createElement("link").relList;if(t&&t.supports&&t.supports("modulepreload"))return;for(const a of document.querySelectorAll('link[rel="modulepreload"]'))n(a);new MutationObserver(a=>{for(const r of a)if(r.type==="childList")for(const i of r.addedNodes)i.tagName==="LINK"&&i.rel==="modulepreload"&&n(i)}).observe(document,{childList:!0,subtree:!0});function s(a){const r={};return a.integrity&&(r.integrity=a.integrity),a.referrerPolicy&&(r.referrerPolicy=a.referrerPolicy),a.crossOrigin==="use-credentials"?r.credentials="include":a.crossOrigin==="anonymous"?r.credentials="omit":r.credentials="same-origin",r}function n(a){if(a.ep)return;a.ep=!0;const r=s(a);fetch(a.href,r)}})();class Ne{constructor(){K(this,"routes",[]);K(this,"currentPath","");K(this,"notFoundRenderer",null)}addRoute(t,s){const n=t.split("/").filter(Boolean);this.routes.push({pattern:t,segments:n,renderer:s})}setNotFound(t){this.notFoundRenderer=t}navigate(t,s=!1){t!==this.currentPath&&(s?history.replaceState(null,"",t):history.pushState(null,"",t),this.resolve())}start(){window.addEventListener("popstate",()=>this.resolve()),document.addEventListener("click",t=>{const s=t.target.closest("a[data-link]");if(s){t.preventDefault();const n=s.getAttribute("href");n&&this.navigate(n)}}),this.resolve()}getCurrentPath(){return this.currentPath}resolve(){const t=new URL(window.location.href),s=t.pathname,n=Oe(t.search);this.currentPath=s+t.search;for(const a of this.routes){const r=Re(a.segments,s);if(r!==null){a.renderer(r,n);return}}this.notFoundRenderer&&this.notFoundRenderer({},n)}}function Re(e,t){const s=t.split("/").filter(Boolean);if(e.length===0&&s.length===0)return{};if(e.length!==s.length)return null;const n={};for(let a=0;a<e.length;a++){const r=e[a],i=s[a];if(r.startsWith(":"))n[r.slice(1)]=decodeURIComponent(i);else if(r!==i)return null}return n}function Oe(e){const t={};return new URLSearchParams(e).forEach((n,a)=>{t[a]=n}),t}const X="/api";async function h(e,t){let s=`${X}${e}`;if(t){const a=new URLSearchParams;Object.entries(t).forEach(([i,o])=>{o!==void 0&&o!==""&&o!==null&&a.set(i,o)});const r=a.toString();r&&(s+=`?${r}`)}const n=await fetch(s);if(!n.ok)throw new Error(`API error: ${n.status} ${n.statusText}`);return n.json()}async function L(e,t){const s=await fetch(`${X}${e}`,{method:"POST",headers:{"Content-Type":"application/json"},body:t?JSON.stringify(t):void 0});if(!s.ok)throw new Error(`API error: ${s.status} ${s.statusText}`);return s.json()}async function ge(e,t){const s=await fetch(`${X}${e}`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify(t)});if(!s.ok)throw new Error(`API error: ${s.status} ${s.statusText}`);return s.json()}async function z(e,t){const s=await fetch(`${X}${e}`,{method:"DELETE",headers:t?{"Content-Type":"application/json"}:void 0,body:t?JSON.stringify(t):void 0});if(!s.ok)throw new Error(`API error: ${s.status} ${s.statusText}`);return s.json()}function ae(e,t){const s={q:e};return t&&(t.page!==void 0&&(s.page=String(t.page)),t.per_page!==void 0&&(s.per_page=String(t.per_page)),t.time_range&&(s.time_range=t.time_range),t.region&&(s.region=t.region),t.language&&(s.language=t.language),t.safe_search&&(s.safe_search=t.safe_search),t.site&&(s.site=t.site),t.exclude_site&&(s.exclude_site=t.exclude_site),t.lens&&(s.lens=t.lens)),s}const f={search(e,t){return h("/search",ae(e,t))},searchImages(e,t){const s={q:e};return t&&(t.page!==void 0&&(s.page=String(t.page)),t.per_page!==void 0&&(s.per_page=String(t.per_page)),t.size&&t.size!=="any"&&(s.size=t.size),t.color&&t.color!=="any"&&(s.color=t.color),t.type&&t.type!=="any"&&(s.type=t.type),t.aspect&&t.aspect!=="any"&&(s.aspect=t.aspect),t.time&&t.time!=="any"&&(s.time=t.time),t.rights&&t.rights!=="any"&&(s.rights=t.rights),t.filetype&&t.filetype!=="any"&&(s.filetype=t.filetype),t.safe&&(s.safe=t.safe)),h("/search/images",s)},reverseImageSearch(e){return L("/search/images/reverse",{url:e})},reverseImageSearchByUpload(e){return L("/search/images/reverse",{image_data:e})},searchVideos(e,t){return h("/search/videos",ae(e,t))},searchNews(e,t){return h("/search/news",ae(e,t))},suggest(e){return h("/suggest",{q:e})},trending(){return h("/suggest/trending")},calculate(e){return h("/instant/calculate",{q:e})},convert(e){return h("/instant/convert",{q:e})},currency(e){return h("/instant/currency",{q:e})},weather(e){return h("/instant/weather",{q:e})},define(e){return h("/instant/define",{q:e})},time(e){return h("/instant/time",{q:e})},knowledge(e){return h(`/knowledge/${encodeURIComponent(e)}`)},getPreferences(){return h("/preferences")},setPreference(e,t){return L("/preferences",{domain:e,action:t})},deletePreference(e){return z(`/preferences/${encodeURIComponent(e)}`)},getLenses(){return h("/lenses")},createLens(e){return L("/lenses",e)},deleteLens(e){return z(`/lenses/${encodeURIComponent(e)}`)},getHistory(){return h("/history")},clearHistory(){return z("/history")},deleteHistoryItem(e){return z(`/history/${encodeURIComponent(e)}`)},getSettings(){return h("/settings")},updateSettings(e){return ge("/settings",e)},getBangs(){return h("/bangs")},parseBang(e){return h("/bangs/parse",{q:e})},getRelated(e){return h("/related",{q:e})},newsHome(){return h("/news/home")},newsCategory(e,t=1){return h(`/news/category/${e}`,{page:String(t)})},newsSearch(e,t){const s={q:e};return t!=null&&t.page&&(s.page=String(t.page)),t!=null&&t.time&&(s.time=t.time),t!=null&&t.source&&(s.source=t.source),h("/news/search",s)},newsStory(e){return h(`/news/story/${e}`)},newsLocal(e){const t={};return e&&(t.city=e.city,e.state&&(t.state=e.state),t.country=e.country),h("/news/local",t)},newsFollowing(){return h("/news/following")},newsPreferences(){return h("/news/preferences")},updateNewsPreferences(e){return ge("/news/preferences",e)},followNews(e,t){return L("/news/follow",{type:e,id:t})},unfollowNews(e,t){return z("/news/follow",{type:e,id:t})},hideNewsSource(e){return L("/news/hide",{source:e})},setNewsLocation(e){return L("/news/location",e)},recordNewsRead(e,t){return L("/news/read",{article:e,duration:t})}};function je(e){let t={...e};const s=new Set;return{get(){return t},set(n){t={...t,...n},s.forEach(a=>a(t))},subscribe(n){return s.add(n),()=>{s.delete(n)}}}}const Ee="mizu_search_state";function qe(){try{const e=localStorage.getItem(Ee);if(e)return JSON.parse(e)}catch{}return{recentSearches:[],settings:{safe_search:"moderate",results_per_page:10,region:"auto",language:"en",theme:"light",open_in_new_tab:!1,show_thumbnails:!0}}}const R=je(qe());R.subscribe(e=>{try{localStorage.setItem(Ee,JSON.stringify(e))}catch{}});function ee(e){const t=R.get(),s=[e,...t.recentSearches.filter(n=>n!==e)].slice(0,20);R.set({recentSearches:s})}const Le='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>',Fe='<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>',Pe='<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2a3 3 0 0 0-3 3v7a3 3 0 0 0 6 0V5a3 3 0 0 0-3-3Z"/><path d="M19 10v2a7 7 0 0 1-14 0v-2"/><line x1="12" x2="12" y1="19" y2="22"/></svg>',Ue='<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14.5 4h-5L7 7H4a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2h-3l-2.5-3z"/><circle cx="12" cy="13" r="3"/></svg>',ze='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/><path d="M12 7v5l4 2"/></svg>',Ve='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M13 2 3 14h9l-1 8 10-12h-9l1-8z"/></svg>';function F(e){const t=e.size==="lg"?"search-box-lg":"search-box-sm",s=e.initialValue?Ge(e.initialValue):"",n=e.initialValue?"":"hidden";return`
    <div id="search-box-wrapper" class="relative w-full flex justify-center">
      <div id="search-box" class="search-box ${t}">
        <span class="text-light mr-3 flex-shrink-0">${Le}</span>
        <input
          id="search-input"
          type="text"
          value="${s}"
          placeholder="Search the web"
          autocomplete="off"
          spellcheck="false"
          ${e.autofocus?"autofocus":""}
        />
        <button id="search-clear-btn" class="text-secondary hover:text-primary p-1 flex-shrink-0 ${n}" type="button" aria-label="Clear">
          ${Fe}
        </button>
        <span class="mx-1 w-px h-5 bg-border flex-shrink-0"></span>
        <button id="voice-search-btn" class="text-light hover:text-secondary p-1 flex-shrink-0" type="button" aria-label="Voice search">
          ${Pe}
        </button>
        <button id="camera-search-btn" class="text-light hover:text-secondary p-1 flex-shrink-0" type="button" aria-label="Image search">
          ${Ue}
        </button>
      </div>
      <div id="autocomplete-dropdown" class="autocomplete-dropdown hidden"></div>
    </div>
  `}function P(e){const t=document.getElementById("search-input"),s=document.getElementById("search-clear-btn"),n=document.getElementById("autocomplete-dropdown"),a=document.getElementById("search-box-wrapper");if(!t||!s||!n||!a)return;let r=null,i=[],o=-1,l=!1;function p(d){if(i=d,o=-1,d.length===0){g();return}l=!0,n.innerHTML=d.map((c,m)=>`
        <div class="autocomplete-item ${m===o?"active":""}" data-index="${m}">
          <span class="suggestion-icon">${c.icon}</span>
          ${c.prefix?`<span class="bang-trigger">${ve(c.prefix)}</span>`:""}
          <span>${ve(c.text)}</span>
        </div>
      `).join(""),n.classList.remove("hidden"),n.classList.add("has-items"),n.querySelectorAll(".autocomplete-item").forEach(c=>{c.addEventListener("mousedown",m=>{m.preventDefault();const $=parseInt(c.dataset.index||"0");y($)}),c.addEventListener("mouseenter",()=>{const m=parseInt(c.dataset.index||"0");H(m)})})}function g(){l=!1,n.classList.add("hidden"),n.classList.remove("has-items"),n.innerHTML="",i=[],o=-1}function H(d){o=d,n.querySelectorAll(".autocomplete-item").forEach((c,m)=>{c.classList.toggle("active",m===d)})}function y(d){const c=i[d];c&&(c.type==="bang"&&c.prefix?(t.value=c.prefix+" ",t.focus(),I(c.prefix+" ")):(t.value=c.text,g(),b(c.text)))}function b(d){const c=d.trim();c&&(g(),e(c))}async function I(d){const c=d.trim();if(!c){U();return}if(c.startsWith("!"))try{const $=(await f.getBangs()).filter(A=>A.trigger.startsWith(c)||A.name.toLowerCase().includes(c.slice(1).toLowerCase())).slice(0,8);if($.length>0){p($.map(A=>({text:A.name,type:"bang",icon:Ve,prefix:A.trigger})));return}}catch{}try{const m=await f.suggest(c);if(t.value.trim()!==c)return;const $=m.map(A=>({text:A.text,type:"suggestion",icon:Le}));$.length===0?U(c):p($)}catch{U(c)}}function U(d){let m=R.get().recentSearches;if(d&&(m=m.filter($=>$.toLowerCase().includes(d.toLowerCase()))),m.length===0){g();return}p(m.slice(0,8).map($=>({text:$,type:"recent",icon:ze})))}t.addEventListener("input",()=>{const d=t.value;s.classList.toggle("hidden",d.length===0),r&&clearTimeout(r),r=setTimeout(()=>I(d),150)}),t.addEventListener("focus",()=>{t.value.trim()?I(t.value):U()}),t.addEventListener("keydown",d=>{if(!l){if(d.key==="Enter"){b(t.value);return}if(d.key==="ArrowDown"){I(t.value);return}return}switch(d.key){case"ArrowDown":d.preventDefault(),H(Math.min(o+1,i.length-1));break;case"ArrowUp":d.preventDefault(),H(Math.max(o-1,-1));break;case"Enter":d.preventDefault(),o>=0?y(o):b(t.value);break;case"Escape":g();break;case"Tab":g();break}}),t.addEventListener("blur",()=>{setTimeout(()=>g(),200)}),s.addEventListener("click",()=>{t.value="",s.classList.add("hidden"),t.focus(),U()});const he=document.getElementById("voice-search-btn");he&&De(he,t,d=>{t.value=d,s.classList.remove("hidden"),b(d)});const pe=document.getElementById("camera-search-btn");pe&&pe.addEventListener("click",()=>{const d=document.getElementById("reverse-modal");d?d.classList.remove("hidden"):window.dispatchEvent(new CustomEvent("router:navigate",{detail:{path:"/images?reverse=1"}}))})}function De(e,t,s){const n=window.SpeechRecognition||window.webkitSpeechRecognition;if(!n){e.style.display="none";return}let a=!1,r=null;e.addEventListener("click",()=>{a?o():i()});function i(){r=new n,r.continuous=!1,r.interimResults=!0,r.lang="en-US",r.onstart=()=>{a=!0,e.classList.add("listening"),e.style.color="#ea4335"},r.onresult=l=>{const p=Array.from(l.results).map(g=>g[0].transcript).join("");t.value=p,l.results[0].isFinal&&(o(),s(p))},r.onerror=l=>{console.error("Speech recognition error:",l.error),o(),l.error==="not-allowed"&&alert("Microphone access denied. Please allow microphone access to use voice search.")},r.onend=()=>{o()};try{r.start()}catch(l){console.error("Failed to start speech recognition:",l),o()}}function o(){if(a=!1,e.classList.remove("listening"),e.style.color="",r){try{r.stop()}catch{}r=null}}}function ve(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function Ge(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const We=[{trigger:"!g",label:"Google",color:"#4285F4"},{trigger:"!yt",label:"YouTube",color:"#EA4335"},{trigger:"!gh",label:"GitHub",color:"#24292e"},{trigger:"!w",label:"Wikipedia",color:"#636466"},{trigger:"!r",label:"Reddit",color:"#FF5700"}],Ye=[{label:"Calculator",icon:Qe(),query:"2+2",color:"bg-blue/10 text-blue"},{label:"Conversion",icon:Xe(),query:"10 miles in km",color:"bg-green/10 text-green"},{label:"Currency",icon:et(),query:"100 USD to EUR",color:"bg-yellow/10 text-yellow"},{label:"Weather",icon:tt(),query:"weather New York",color:"bg-blue/10 text-blue"},{label:"Time",icon:st(),query:"time in Tokyo",color:"bg-green/10 text-green"},{label:"Define",icon:nt(),query:"define serendipity",color:"bg-red/10 text-red"}];function Ke(){return`
    <div class="min-h-screen flex flex-col">
      <div class="flex-1 flex flex-col items-center justify-center px-4 -mt-20">
        <!-- Logo -->
        <div class="mb-8 text-center">
          <h1 class="text-6xl font-semibold mb-2 select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </h1>
          <p class="text-secondary text-lg">Privacy-first search</p>
        </div>

        <!-- Search Box -->
        <div class="w-full max-w-2xl mb-6">
          ${F({size:"lg",autofocus:!0})}
        </div>

        <!-- Search Buttons -->
        <div class="flex gap-3 mb-8">
          <button id="home-search-btn" class="px-5 py-2 bg-surface hover:bg-surface-hover border border-border rounded text-sm text-primary cursor-pointer">
            Mizu Search
          </button>
          <button id="home-lucky-btn" class="px-5 py-2 bg-surface hover:bg-surface-hover border border-border rounded text-sm text-primary cursor-pointer">
            I'm Feeling Lucky
          </button>
        </div>

        <!-- Bang Shortcuts -->
        <div class="flex flex-wrap justify-center gap-2 mb-8">
          ${We.map(e=>`
            <button class="bang-shortcut px-3 py-1.5 rounded-full text-xs font-medium border border-border hover:shadow-sm transition-shadow cursor-pointer"
                    data-bang="${e.trigger}"
                    style="color: ${e.color}; border-color: ${e.color}20;">
              <span class="font-semibold">${re(e.trigger)}</span>
              <span class="text-tertiary ml-1">${re(e.label)}</span>
            </button>
          `).join("")}
        </div>

        <!-- Instant Answers Showcase -->
        <div class="mb-8">
          <p class="text-center text-xs text-light mb-3 uppercase tracking-wider">Instant Answers</p>
          <div class="flex flex-wrap justify-center gap-2">
            ${Ye.map(e=>`
              <button class="instant-showcase-btn flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-medium ${e.color} hover:opacity-80 transition-opacity cursor-pointer"
                      data-query="${Je(e.query)}">
                ${e.icon}
                <span>${re(e.label)}</span>
              </button>
            `).join("")}
          </div>
        </div>

        <!-- Category Links -->
        <div class="flex gap-6 text-sm">
          <a href="/images" data-link class="text-tertiary hover:text-primary transition-colors flex items-center gap-1.5">
            ${at()}
            Images
          </a>
          <a href="/news" data-link class="text-tertiary hover:text-primary transition-colors flex items-center gap-1.5">
            ${rt()}
            News
          </a>
        </div>
      </div>

      <!-- Footer -->
      <footer class="py-4 text-center">
        <div class="text-xs text-light space-x-4">
          <span>Use <strong>!bangs</strong> to search other sites directly</span>
          <span>&middot;</span>
          <a href="/settings" data-link class="hover:text-secondary">Settings</a>
          <span>&middot;</span>
          <a href="/history" data-link class="hover:text-secondary">History</a>
        </div>
      </footer>
    </div>
  `}function Ze(e){P(n=>{e.navigate(`/search?q=${encodeURIComponent(n)}`)});const t=document.getElementById("home-search-btn");t==null||t.addEventListener("click",()=>{var r;const n=document.getElementById("search-input"),a=(r=n==null?void 0:n.value)==null?void 0:r.trim();a&&e.navigate(`/search?q=${encodeURIComponent(a)}`)});const s=document.getElementById("home-lucky-btn");s==null||s.addEventListener("click",()=>{var r;const n=document.getElementById("search-input"),a=(r=n==null?void 0:n.value)==null?void 0:r.trim();a&&e.navigate(`/search?q=${encodeURIComponent(a)}&lucky=1`)}),document.querySelectorAll(".bang-shortcut").forEach(n=>{n.addEventListener("click",()=>{const a=n.dataset.bang||"",r=document.getElementById("search-input");r&&(r.value=a+" ",r.focus())})}),document.querySelectorAll(".instant-showcase-btn").forEach(n=>{n.addEventListener("click",()=>{const a=n.dataset.query||"";a&&e.navigate(`/search?q=${encodeURIComponent(a)}`)})})}function re(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function Je(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}function Qe(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="16" height="20" x="4" y="2" rx="2"/><line x1="8" x2="16" y1="6" y2="6"/><line x1="16" x2="16" y1="14" y2="18"/></svg>'}function Xe(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M8 3 4 7l4 4"/><path d="M4 7h16"/><path d="m16 21 4-4-4-4"/><path d="M20 17H4"/></svg>'}function et(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" x2="12" y1="2" y2="22"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>'}function tt(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="M2 12h2"/><path d="M20 12h2"/></svg>'}function st(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>'}function nt(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1 0-5H20"/></svg>'}function at(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="18" height="18" x="3" y="3" rx="2" ry="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"/></svg>'}function rt(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 22h16a2 2 0 0 0 2-2V4a2 2 0 0 0-2-2H8a2 2 0 0 0-2 2v16a2 2 0 0 1-2 2Zm0 0a2 2 0 0 1-2-2v-9c0-1.1.9-2 2-2h2"/></svg>'}const it='<svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="5" r="2"/><circle cx="12" cy="12" r="2"/><circle cx="12" cy="19" r="2"/></svg>',ot='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M7 10v12"/><path d="M15 5.88 14 10h5.83a2 2 0 0 1 1.92 2.56l-2.33 8A2 2 0 0 1 17.5 22H4a2 2 0 0 1-2-2v-8a2 2 0 0 1 2-2h2.76a2 2 0 0 0 1.79-1.11L12 2h0a3.13 3.13 0 0 1 3 3.88Z"/></svg>',lt='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 14V2"/><path d="M9 18.12 10 14H4.17a2 2 0 0 1-1.92-2.56l2.33-8A2 2 0 0 1 6.5 2H20a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2h-2.76a2 2 0 0 0-1.79 1.11L12 22h0a3.13 3.13 0 0 1-3-3.88Z"/></svg>',ct='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="m4.9 4.9 14.2 14.2"/></svg>';function dt(e,t){const s=e.favicon||`https://www.google.com/s2/favicons?domain=${encodeURIComponent(e.domain)}&sz=32`,n=pt(e.url),a=e.published?gt(e.published):"",r=e.snippet||"",i=e.thumbnail?`<img src="${S(e.thumbnail.url)}" alt="" class="w-[120px] h-[80px] rounded-lg object-cover flex-shrink-0 ml-4" loading="lazy" />`:"",o=e.sitelinks&&e.sitelinks.length>0?`<div class="result-sitelinks">
        ${e.sitelinks.map(l=>`<a href="${S(l.url)}" target="_blank" rel="noopener">${_(l.title)}</a>`).join("")}
       </div>`:"";return`
    <div class="search-result" data-result-index="${t}" data-domain="${S(e.domain)}">
      <div class="result-url">
        <img class="favicon" src="${S(s)}" alt="" width="18" height="18" loading="lazy" onerror="this.style.display='none'" />
        <div>
          <span class="text-sm">${_(e.domain)}</span>
          <span class="breadcrumbs">${n}</span>
        </div>
      </div>
      <div class="flex items-start">
        <div class="flex-1">
          <div class="result-title">
            <a href="${S(e.url)}" target="_blank" rel="noopener">${_(e.title)}</a>
          </div>
          ${a?`<span class="result-date">${_(a)} -- </span>`:""}
          <div class="result-snippet">${r}</div>
          ${o}
        </div>
        ${i}
      </div>
      <button class="result-menu-btn" data-menu-index="${t}" aria-label="More options">
        ${it}
      </button>
      <div id="domain-menu-${t}" class="domain-menu hidden"></div>
    </div>
  `}function ut(){document.querySelectorAll(".result-menu-btn").forEach(e=>{e.addEventListener("click",t=>{t.stopPropagation();const s=e.dataset.menuIndex,n=document.getElementById(`domain-menu-${s}`),a=e.closest(".search-result"),r=(a==null?void 0:a.dataset.domain)||"";if(!n)return;if(!n.classList.contains("hidden")){n.classList.add("hidden");return}document.querySelectorAll(".domain-menu").forEach(o=>o.classList.add("hidden")),n.innerHTML=`
        <button class="domain-menu-item boost" data-action="boost" data-domain="${S(r)}">
          ${ot}
          <span>Boost ${_(r)}</span>
        </button>
        <button class="domain-menu-item lower" data-action="lower" data-domain="${S(r)}">
          ${lt}
          <span>Lower ${_(r)}</span>
        </button>
        <button class="domain-menu-item block" data-action="block" data-domain="${S(r)}">
          ${ct}
          <span>Block ${_(r)}</span>
        </button>
      `,n.classList.remove("hidden"),n.querySelectorAll(".domain-menu-item").forEach(o=>{o.addEventListener("click",async()=>{const l=o.dataset.action||"",p=o.dataset.domain||"";try{await f.setPreference(p,l),n.classList.add("hidden"),ht(`${l.charAt(0).toUpperCase()+l.slice(1)}ed ${p}`)}catch(g){console.error("Failed to set preference:",g)}})});const i=o=>{!n.contains(o.target)&&o.target!==e&&(n.classList.add("hidden"),document.removeEventListener("click",i))};setTimeout(()=>document.addEventListener("click",i),0)})})}function ht(e){const t=document.getElementById("toast");t&&t.remove();const s=document.createElement("div");s.id="toast",s.className="fixed bottom-6 left-1/2 -translate-x-1/2 bg-primary text-white px-5 py-3 rounded-lg shadow-lg text-sm z-50 transition-opacity duration-300",s.textContent=e,document.body.appendChild(s),setTimeout(()=>{s.style.opacity="0",setTimeout(()=>s.remove(),300)},2e3)}function pt(e){try{const s=new URL(e).pathname.split("/").filter(Boolean);return s.length===0?"":" > "+s.map(n=>_(decodeURIComponent(n))).join(" > ")}catch{return""}}function gt(e){try{const t=new Date(e),n=new Date().getTime()-t.getTime(),a=Math.floor(n/(1e3*60*60*24));return a===0?"Today":a===1?"1 day ago":a<7?`${a} days ago`:a<30?`${Math.floor(a/7)} weeks ago`:t.toLocaleDateString("en-US",{month:"short",day:"numeric",year:"numeric"})}catch{return e}}function _(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function S(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const vt='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="16" height="20" x="4" y="2" rx="2"/><line x1="8" x2="16" y1="6" y2="6"/><line x1="16" x2="16" y1="14" y2="18"/><path d="M16 10h.01"/><path d="M12 10h.01"/><path d="M8 10h.01"/><path d="M12 14h.01"/><path d="M8 14h.01"/><path d="M12 18h.01"/><path d="M8 18h.01"/></svg>',mt='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M8 3 4 7l4 4"/><path d="M4 7h16"/><path d="m16 21 4-4-4-4"/><path d="M20 17H4"/></svg>',ft='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" x2="12" y1="2" y2="22"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>',yt='<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#FBBC05" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="m4.93 4.93 1.41 1.41"/><path d="m17.66 17.66 1.41 1.41"/><path d="M2 12h2"/><path d="M20 12h2"/><path d="m6.34 17.66-1.41 1.41"/><path d="m19.07 4.93-1.41 1.41"/></svg>',wt='<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#5f6368" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17.5 19H9a7 7 0 1 1 6.71-9h1.79a4.5 4.5 0 1 1 0 9Z"/></svg>',xt='<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#4285F4" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 14.899A7 7 0 1 1 15.71 8h1.79a4.5 4.5 0 0 1 2.5 8.242"/><path d="M16 14v6"/><path d="M8 14v6"/><path d="M12 16v6"/></svg>',bt='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1 0-5H20"/></svg>',$t='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>';function kt(e){switch(e.type){case"calculator":return It(e);case"unit_conversion":return Ct(e);case"currency":return Et(e);case"weather":return Lt(e);case"definition":return St(e);case"time":return _t(e);default:return Bt(e)}}function It(e){const t=e.data||{},s=t.expression||e.query||"",n=t.formatted||t.result||e.result||"";return`
    <div class="instant-card border-l-4 border-l-blue">
      <div class="flex items-center gap-2 mb-2 text-tertiary">
        ${vt}
        <span class="instant-type">Calculator</span>
      </div>
      <div class="instant-result">${u(s)} = ${u(String(n))}</div>
    </div>
  `}function Ct(e){const t=e.data||{},s=t.from_value??"",n=t.from_unit??"",a=t.to_value??"",r=t.to_unit??"",i=t.category??"";return`
    <div class="instant-card border-l-4 border-l-green">
      <div class="flex items-center gap-2 mb-2 text-tertiary">
        ${mt}
        <span class="instant-type">Unit Conversion${i?` -- ${u(i)}`:""}</span>
      </div>
      <div class="instant-result">${u(String(s))} ${u(n)} = ${u(String(a))} ${u(r)}</div>
      ${t.formatted?`<div class="instant-sub">${u(t.formatted)}</div>`:""}
    </div>
  `}function Et(e){const t=e.data||{},s=t.from_value??"",n=t.from_currency??"",a=t.to_value??"",r=t.to_currency??"",i=t.rate??"";return`
    <div class="instant-card border-l-4 border-l-yellow">
      <div class="flex items-center gap-2 mb-2 text-tertiary">
        ${ft}
        <span class="instant-type">Currency</span>
      </div>
      <div class="instant-result">${u(String(s))} ${u(n)} = ${u(String(a))} ${u(r)}</div>
      ${i?`<div class="instant-sub">1 ${u(n)} = ${u(String(i))} ${u(r)}</div>`:""}
    </div>
  `}function Lt(e){const t=e.data||{},s=t.location||"",n=t.temperature??"",a=(t.condition||"").toLowerCase(),r=t.humidity||"",i=t.wind||"";let o=yt;return a.includes("cloud")||a.includes("overcast")?o=wt:(a.includes("rain")||a.includes("drizzle")||a.includes("storm"))&&(o=xt),`
    <div class="instant-card border-l-4 border-l-blue">
      <div class="instant-type mb-2">Weather</div>
      <div class="flex items-center gap-4 mb-3">
        <div>${o}</div>
        <div>
          <div class="text-2xl font-semibold text-primary">${u(String(n))}&deg;</div>
          <div class="text-secondary capitalize">${u(t.condition||"")}</div>
        </div>
      </div>
      <div class="text-sm font-medium text-primary mb-2">${u(s)}</div>
      <div class="flex gap-6 text-sm text-tertiary">
        ${r?`<span>Humidity: ${u(r)}</span>`:""}
        ${i?`<span>Wind: ${u(i)}</span>`:""}
      </div>
    </div>
  `}function St(e){const t=e.data||{},s=t.word||e.query||"",n=t.phonetic||"",a=t.part_of_speech||"",r=t.definitions||[],i=t.synonyms||[];return`
    <div class="instant-card border-l-4 border-l-red">
      <div class="flex items-center gap-2 mb-2 text-tertiary">
        ${bt}
        <span class="instant-type">Definition</span>
      </div>
      <div class="flex items-baseline gap-3 mb-1">
        <span class="text-xl font-semibold text-primary">${u(s)}</span>
        ${n?`<span class="text-tertiary text-sm">${u(n)}</span>`:""}
      </div>
      ${a?`<div class="text-sm italic text-secondary mb-2">${u(a)}</div>`:""}
      ${r.length>0?`<ol class="list-decimal list-inside space-y-1 text-sm text-snippet mb-3">
              ${r.map(o=>`<li>${u(o)}</li>`).join("")}
             </ol>`:""}
      ${i.length>0?`<div class="text-sm">
              <span class="text-tertiary">Synonyms: </span>
              <span class="text-secondary">${i.map(o=>u(o)).join(", ")}</span>
             </div>`:""}
    </div>
  `}function _t(e){const t=e.data||{},s=t.location||"",n=t.time||"",a=t.date||"",r=t.timezone||"";return`
    <div class="instant-card border-l-4 border-l-green">
      <div class="flex items-center gap-2 mb-2 text-tertiary">
        ${$t}
        <span class="instant-type">Time</span>
      </div>
      <div class="text-sm font-medium text-secondary mb-1">${u(s)}</div>
      <div class="text-4xl font-semibold text-primary mb-1">${u(n)}</div>
      <div class="text-sm text-tertiary">${u(a)}</div>
      ${r?`<div class="text-xs text-light mt-1">${u(r)}</div>`:""}
    </div>
  `}function Bt(e){return`
    <div class="instant-card border-l-4 border-l-blue">
      <div class="instant-type mb-2">${u(e.type)}</div>
      <div class="instant-result">${u(e.result)}</div>
    </div>
  `}function u(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}const Mt='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" x2="21" y1="14" y2="3"/></svg>';function Ht(e){const t=e.image?`<img class="kp-image" src="${ie(e.image)}" alt="${ie(e.title)}" loading="lazy" onerror="this.style.display='none'" />`:"",s=e.facts&&e.facts.length>0?`<table class="kp-facts">
          <tbody>
            ${e.facts.map(r=>`
              <tr>
                <td class="fact-label">${T(r.label)}</td>
                <td class="fact-value">${T(r.value)}</td>
              </tr>
            `).join("")}
          </tbody>
        </table>`:"",n=e.links&&e.links.length>0?`<div class="kp-links">
          ${e.links.map(r=>`
            <a class="kp-link" href="${ie(r.url)}" target="_blank" rel="noopener">
              ${Mt}
              <span>${T(r.title)}</span>
            </a>
          `).join("")}
        </div>`:"",a=e.source?`<div class="kp-source">Source: ${T(e.source)}</div>`:"";return`
    <div class="knowledge-panel" id="knowledge-panel">
      ${t}
      <div class="kp-title">${T(e.title)}</div>
      ${e.subtitle?`<div class="kp-subtitle">${T(e.subtitle)}</div>`:""}
      <div class="kp-description">${T(e.description)}</div>
      ${s}
      ${n}
      ${a}
    </div>
  `}function T(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function ie(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const At='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m15 18-6-6 6-6"/></svg>',Tt='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"/></svg>';function Nt(e){const{currentPage:t,hasMore:s,totalResults:n,perPage:a}=e,r=Math.min(Math.ceil(n/a),100);if(r<=1)return"";let i=Math.max(1,t-4),o=Math.min(r,i+9);o-i<9&&(i=Math.max(1,o-9));const l=[];for(let y=i;y<=o;y++)l.push(y);const p=Rt(t),g=t<=1?"disabled":"",H=!s&&t>=r?"disabled":"";return`
    <div class="pagination" id="pagination">
      <div class="flex flex-col items-center gap-3">
        ${p}
        <div class="flex items-center gap-1">
          <button class="pagination-btn ${g}" data-page="${t-1}" ${t<=1?"disabled":""} aria-label="Previous page">
            ${At}
          </button>
          ${l.map(y=>`
            <button class="pagination-btn ${y===t?"active":""}" data-page="${y}">
              ${y}
            </button>
          `).join("")}
          <button class="pagination-btn ${H}" data-page="${t+1}" ${!s&&t>=r?"disabled":""} aria-label="Next page">
            ${Tt}
          </button>
        </div>
      </div>
    </div>
  `}function Rt(e){const t=["#4285F4","#EA4335","#FBBC05","#4285F4","#34A853","#EA4335"],s=["M","i","z","u"],n=Math.min(e-1,6);let a=[s[0]];for(let r=0;r<1+n;r++)a.push("i");a.push("z");for(let r=0;r<1+n;r++)a.push("u");return`
    <div class="flex items-center text-2xl font-semibold tracking-wide select-none">
      ${a.map((r,i)=>`<span style="color: ${t[i%t.length]}">${r}</span>`).join("")}
    </div>
  `}function Ot(e){const t=document.getElementById("pagination");t&&t.querySelectorAll(".pagination-btn").forEach(s=>{s.addEventListener("click",()=>{const n=parseInt(s.dataset.page||"1");isNaN(n)||s.disabled||(e(n),window.scrollTo({top:0,behavior:"smooth"}))})})}const jt='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>',qt='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="18" x="3" y="3" rx="2" ry="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"/></svg>',Ft='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m22 8-6 4 6 4V8Z"/><rect width="14" height="12" x="2" y="6" rx="2" ry="2"/></svg>',Pt='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 22h16a2 2 0 0 0 2-2V4a2 2 0 0 0-2-2H8a2 2 0 0 0-2 2v16a2 2 0 0 1-2 2Zm0 0a2 2 0 0 1-2-2v-9c0-1.1.9-2 2-2h2"/><path d="M18 14h-8"/><path d="M15 18h-5"/><path d="M10 6h8v4h-8V6Z"/></svg>';function te(e){const{query:t,active:s}=e,n=encodeURIComponent(t);return`
    <div class="search-tabs" id="search-tabs">
      ${[{id:"all",label:"All",icon:jt,href:`/search?q=${n}`},{id:"images",label:"Images",icon:qt,href:`/images?q=${n}`},{id:"videos",label:"Videos",icon:Ft,href:`/videos?q=${n}`},{id:"news",label:"News",icon:Pt,href:`/news?q=${n}`}].map(r=>`
        <a class="search-tab ${r.id===s?"active":""}" href="${r.href}" data-link data-tab="${r.id}">
          ${r.icon}
          <span>${r.label}</span>
        </a>
      `).join("")}
    </div>
  `}const Ut='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',zt='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m6 9 6 6 6-6"/></svg>',me=[{value:"",label:"Any time"},{value:"day",label:"Past 24 hours"},{value:"week",label:"Past week"},{value:"month",label:"Past month"},{value:"year",label:"Past year"}];function Vt(e,t){var a;const s=((a=me.find(r=>r.value===t))==null?void 0:a.label)||"Any time",n=t!=="";return`
    <div class="min-h-screen flex flex-col">
      <!-- Header -->
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="search-header-row">
          <a href="/" data-link class="search-logo">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="search-header-box">
            ${F({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors" aria-label="Settings">
            ${Ut}
          </a>
        </div>
        <div class="search-tabs-row">
          <div class="flex items-center gap-2">
            ${te({query:e,active:"all"})}
            <div class="time-filter ml-2" id="time-filter-wrapper">
              <button class="time-filter-btn ${n?"active-filter":""}" id="time-filter-btn" type="button">
                <span id="time-filter-label">${N(s)}</span>
                ${zt}
              </button>
              <div class="time-filter-dropdown hidden" id="time-filter-dropdown">
                ${me.map(r=>`
                  <button class="time-filter-option ${r.value===t?"active":""}" data-time-range="${r.value}">
                    ${N(r.label)}
                  </button>
                `).join("")}
              </div>
            </div>
          </div>
        </div>
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
  `}function Dt(e,t,s){const n=parseInt(s.page||"1"),a=s.time_range||"",r=R.get().settings;P(i=>{e.navigate(`/search?q=${encodeURIComponent(i)}`)}),Gt(e,t),t&&ee(t),Wt(e,t,n,a,r.results_per_page)}function Gt(e,t,s){const n=document.getElementById("time-filter-btn"),a=document.getElementById("time-filter-dropdown");!n||!a||(n.addEventListener("click",r=>{r.stopPropagation(),a.classList.toggle("hidden")}),a.querySelectorAll(".time-filter-option").forEach(r=>{r.addEventListener("click",()=>{const i=r.dataset.timeRange||"";a.classList.add("hidden");let o=`/search?q=${encodeURIComponent(t)}`;i&&(o+=`&time_range=${i}`),e.navigate(o)})}),document.addEventListener("click",r=>{!a.contains(r.target)&&r.target!==n&&a.classList.add("hidden")}))}async function Wt(e,t,s,n,a){const r=document.getElementById("search-content");if(!(!r||!t))try{const i=await f.search(t,{page:s,per_page:a,time_range:n||void 0});if(i.redirect){window.location.href=i.redirect;return}Yt(r,e,i,t,s,n)}catch(i){r.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to load search results. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${N(String(i))}</p>
      </div>
    `}}function Yt(e,t,s,n,a,r){const i=s.corrected_query?`<p class="text-sm text-secondary mb-4">
        Showing results for <a href="/search?q=${encodeURIComponent(s.corrected_query)}" data-link class="text-link font-medium">${N(s.corrected_query)}</a>.
        Search instead for <a href="/search?q=${encodeURIComponent(n)}&exact=1" data-link class="text-link">${N(n)}</a>.
      </p>`:"",o=`
    <div class="text-xs text-tertiary mb-4">
      About ${Kt(s.total_results)} results (${(s.search_time_ms/1e3).toFixed(2)} seconds)
    </div>
  `,l=s.instant_answer?kt(s.instant_answer):"",p=s.results.length>0?s.results.map((b,I)=>dt(b,I)).join(""):`<div class="py-8 text-secondary">No results found for "<strong>${N(n)}</strong>"</div>`,g=s.related_searches&&s.related_searches.length>0?`
      <div class="mt-8 mb-4">
        <h3 class="text-lg font-medium text-primary mb-3">Related searches</h3>
        <div class="grid grid-cols-2 gap-2 max-w-[600px]">
          ${s.related_searches.map(b=>`
            <a href="/search?q=${encodeURIComponent(b)}" data-link class="flex items-center gap-2 p-3 rounded-lg bg-surface hover:bg-surface-hover text-sm text-primary transition-colors">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#9aa0a6" stroke-width="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
              ${N(b)}
            </a>
          `).join("")}
        </div>
      </div>
    `:"",H=Nt({currentPage:a,hasMore:s.has_more,totalResults:s.total_results,perPage:s.per_page}),y=s.knowledge_panel?Ht(s.knowledge_panel):"";e.innerHTML=`
    <div class="search-results-layout">
      <div class="search-results-main">
        ${i}
        ${o}
        ${l}
        ${p}
        ${g}
        ${H}
      </div>
      ${y?`<aside class="search-results-sidebar">${y}</aside>`:""}
    </div>
  `,ut(),Ot(b=>{let I=`/search?q=${encodeURIComponent(n)}&page=${b}`;r&&(I+=`&time_range=${r}`),t.navigate(I)})}function Kt(e){return e.toLocaleString("en-US")}function N(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}const Zt='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',Jt='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14.5 4h-5L7 7H4a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2h-3l-2.5-3z"/><circle cx="12" cy="13" r="3"/></svg>',fe='<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>',ye='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" x2="21" y1="14" y2="3"/></svg>',Qt='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3"/></svg>',Se='<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>',Xt='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6"/></svg>',es='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 18 15 12 9 6"/></svg>';let G="",C={},W=1,q=!1,M=!0,E=[],V=!1,le=[],Z=null;function ts(e){return`
    <div class="min-h-screen flex flex-col bg-white">
      <!-- Header -->
      <header class="sticky top-0 bg-white z-20 shadow-sm">
        <div class="flex items-center gap-4 px-4 py-2">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[600px] flex items-center gap-2">
            ${F({size:"sm",initialValue:e})}
            <button id="reverse-search-btn" class="flex-shrink-0 p-2 text-tertiary hover:text-primary hover:bg-surface-hover rounded-full transition-colors" title="Search by image">
              ${Jt}
            </button>
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors" aria-label="Settings">
            ${Zt}
          </a>
        </div>
        <div class="pl-[56px] flex items-center gap-1">
          ${te({query:e,active:"images"})}
          <button id="tools-btn" class="tools-btn ml-4">
            ${Qt}
            <span>Tools</span>
            ${Se}
          </button>
        </div>
        <!-- Filter toolbar (hidden by default) -->
        <div id="filter-toolbar" class="filter-toolbar hidden">
          ${ss()}
        </div>
      </header>

      <!-- Related searches bar -->
      <div id="related-searches" class="related-searches-bar hidden">
        <div class="related-searches-scroll">
          <button class="related-scroll-btn related-scroll-left hidden">${Xt}</button>
          <div class="related-searches-list"></div>
          <button class="related-scroll-btn related-scroll-right hidden">${es}</button>
        </div>
      </div>

      <!-- Content -->
      <main class="flex-1 flex">
        <div id="images-content" class="flex-1 px-2">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>

        <!-- Preview panel (hidden by default) -->
        <div id="preview-panel" class="preview-panel hidden">
          <div class="preview-overlay"></div>
          <div class="preview-container">
            <button id="preview-close" class="preview-close-btn" aria-label="Close">${fe}</button>
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
            <button id="reverse-modal-close" class="modal-close">${fe}</button>
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
  `}function ss(){return`
    <div class="filter-chips">
      ${[{id:"size",label:"Size",options:["any","large","medium","small","icon"]},{id:"color",label:"Color",options:["any","color","gray","transparent","red","orange","yellow","green","teal","blue","purple","pink","white","black","brown"]},{id:"type",label:"Type",options:["any","photo","clipart","lineart","animated","face"]},{id:"aspect",label:"Aspect",options:["any","tall","square","wide","panoramic"]},{id:"time",label:"Time",options:["any","day","week","month","year"]},{id:"rights",label:"Usage rights",options:["any","creative_commons","commercial"]}].map(t=>`
        <div class="filter-chip-wrapper">
          <button class="filter-chip" data-filter="${t.id}" data-value="any">
            <span class="filter-chip-label">${t.label}</span>
            ${Se}
          </button>
          <div class="filter-dropdown hidden" data-dropdown="${t.id}">
            ${t.options.map(s=>`
              <button class="filter-option${s==="any"?" active":""}" data-value="${s}">
                ${Q(t.id,s)}
              </button>
            `).join("")}
          </div>
        </div>
      `).join("")}
      <button id="clear-filters" class="clear-filters-btn hidden">Clear</button>
    </div>
  `}function Q(e,t){return t==="any"?`Any ${e}`:t.charAt(0).toUpperCase()+t.slice(1).replace("_"," ")}function ns(e,t,s){if(G=t,C={},W=1,E=[],M=!0,V=!1,le=[],P(n=>{e.navigate(`/images?q=${encodeURIComponent(n)}`)}),t&&ee(t),as(),rs(),ls(),us(),is(e),(s==null?void 0:s.reverse)==="1"){const n=document.getElementById("reverse-modal");n&&n.classList.remove("hidden")}ce(t,C)}function as(){const e=document.getElementById("tools-btn"),t=document.getElementById("filter-toolbar");!e||!t||e.addEventListener("click",()=>{V=!V,t.classList.toggle("hidden",!V),e.classList.toggle("active",V)})}function rs(e){const t=document.getElementById("filter-toolbar");if(!t)return;t.querySelectorAll(".filter-chip").forEach(n=>{n.addEventListener("click",a=>{a.stopPropagation();const r=n.dataset.filter,i=t.querySelector(`[data-dropdown="${r}"]`);t.querySelectorAll(".filter-dropdown").forEach(o=>{o!==i&&o.classList.add("hidden")}),i==null||i.classList.toggle("hidden")})}),t.querySelectorAll(".filter-option").forEach(n=>{n.addEventListener("click",()=>{const a=n.closest(".filter-dropdown"),r=a==null?void 0:a.dataset.dropdown,i=n.dataset.value,o=t.querySelector(`[data-filter="${r}"]`);!r||!i||!o||(a.querySelectorAll(".filter-option").forEach(l=>l.classList.remove("active")),n.classList.add("active"),i==="any"?(delete C[r],o.classList.remove("has-value"),o.querySelector(".filter-chip-label").textContent=Q(r,"any").replace("Any ","")):(C[r]=i,o.classList.add("has-value"),o.querySelector(".filter-chip-label").textContent=Q(r,i)),a.classList.add("hidden"),we(),W=1,E=[],M=!0,ce(G,C))})}),document.addEventListener("click",()=>{t.querySelectorAll(".filter-dropdown").forEach(n=>n.classList.add("hidden"))});const s=document.getElementById("clear-filters");s&&s.addEventListener("click",()=>{C={},W=1,E=[],M=!0,t.querySelectorAll(".filter-chip").forEach(n=>{const a=n.dataset.filter;n.classList.remove("has-value"),n.querySelector(".filter-chip-label").textContent=Q(a,"any").replace("Any ","")}),t.querySelectorAll(".filter-dropdown").forEach(n=>{n.querySelectorAll(".filter-option").forEach((a,r)=>{a.classList.toggle("active",r===0)})}),we(),ce(G,C)})}function we(){const e=document.getElementById("clear-filters");e&&e.classList.toggle("hidden",Object.keys(C).length===0)}function is(e){const t=document.getElementById("related-searches");if(!t)return;t.addEventListener("click",r=>{const i=r.target.closest(".related-chip");if(i){const o=i.getAttribute("data-query");o&&e.navigate(`/images?q=${encodeURIComponent(o)}`)}});const s=t.querySelector(".related-scroll-left"),n=t.querySelector(".related-scroll-right"),a=t.querySelector(".related-searches-list");s&&n&&a&&(s.addEventListener("click",()=>{a.scrollBy({left:-200,behavior:"smooth"})}),n.addEventListener("click",()=>{a.scrollBy({left:200,behavior:"smooth"})}),a.addEventListener("scroll",()=>{_e()}))}function _e(){const e=document.getElementById("related-searches");if(!e)return;const t=e.querySelector(".related-searches-list"),s=e.querySelector(".related-scroll-left"),n=e.querySelector(".related-scroll-right");!t||!s||!n||(s.classList.toggle("hidden",t.scrollLeft<=0),n.classList.toggle("hidden",t.scrollLeft>=t.scrollWidth-t.clientWidth-10))}function os(e){const t=document.getElementById("related-searches");if(!t)return;if(!e||e.length===0){t.classList.add("hidden");return}const s=t.querySelector(".related-searches-list");s&&(s.innerHTML=e.map(n=>`
    <button class="related-chip" data-query="${B(n)}">
      <span class="related-chip-text">${O(n)}</span>
    </button>
  `).join(""),t.classList.remove("hidden"),setTimeout(_e,50))}function ls(e){const t=document.getElementById("reverse-search-btn"),s=document.getElementById("reverse-modal"),n=document.getElementById("reverse-modal-close"),a=document.getElementById("drop-zone"),r=document.getElementById("image-upload"),i=document.getElementById("image-url-input"),o=document.getElementById("url-search-btn");!t||!s||(t.addEventListener("click",()=>s.classList.remove("hidden")),n==null||n.addEventListener("click",()=>s.classList.add("hidden")),s.addEventListener("click",l=>{l.target===s&&s.classList.add("hidden")}),a&&(a.addEventListener("dragover",l=>{l.preventDefault(),a.classList.add("drag-over")}),a.addEventListener("dragleave",()=>a.classList.remove("drag-over")),a.addEventListener("drop",l=>{var g;l.preventDefault(),a.classList.remove("drag-over");const p=(g=l.dataTransfer)==null?void 0:g.files;p&&p[0]&&(xe(p[0]),s.classList.add("hidden"))})),r&&r.addEventListener("change",()=>{r.files&&r.files[0]&&(xe(r.files[0]),s.classList.add("hidden"))}),o&&i&&(o.addEventListener("click",()=>{const l=i.value.trim();l&&(be(l),s.classList.add("hidden"))}),i.addEventListener("keydown",l=>{if(l.key==="Enter"){const p=i.value.trim();p&&(be(p),s.classList.add("hidden"))}})))}async function xe(e,t){const s=document.getElementById("images-content");if(s){if(!e.type.startsWith("image/")){alert("Please select an image file");return}if(e.size>10*1024*1024){alert("Image must be smaller than 10MB");return}s.innerHTML=`
    <div class="flex flex-col items-center justify-center py-16">
      <div class="spinner"></div>
      <span class="mt-3 text-secondary">Uploading and searching...</span>
      <div class="w-48 mt-4 h-1 bg-border rounded-full overflow-hidden">
        <div id="upload-progress" class="h-full bg-blue transition-all duration-300" style="width: 0%"></div>
      </div>
    </div>
  `;try{const n=await cs(e),a=document.getElementById("upload-progress");a&&(a.style.width="50%");const r=await f.reverseImageSearchByUpload(n);a&&(a.style.width="100%"),ds(s,n,r)}catch(n){s.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to search by image. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${O(String(n))}</p>
      </div>
    `}}}function cs(e){return new Promise((t,s)=>{const n=new FileReader;n.onload=()=>{const r=n.result.split(",")[1];t(r)},n.onerror=s,n.readAsDataURL(e)})}function ds(e,t,s){const a=!t.startsWith("http")?`data:image/jpeg;base64,${t}`:t;e.innerHTML=`
    <div class="reverse-results">
      <div class="query-image-section">
        <h3>Search image</h3>
        <img src="${a}" alt="Query image" class="query-image" />
      </div>
      ${s.similar_images.length>0?`
        <div class="similar-images-section">
          <h3>Similar images (${s.similar_images.length})</h3>
          <div class="image-grid">
            ${s.similar_images.map((r,i)=>ne(r,i)).join("")}
          </div>
        </div>
      `:'<div class="py-8 text-secondary">No similar images found.</div>'}
    </div>
  `,e.querySelectorAll(".image-card").forEach(r=>{r.addEventListener("click",()=>{const i=parseInt(r.dataset.imageIndex||"0",10);se(s.similar_images[i])})})}async function be(e,t){const s=document.getElementById("images-content");if(s){s.innerHTML=`
    <div class="flex items-center justify-center py-16">
      <div class="spinner"></div>
      <span class="ml-3 text-secondary">Searching for similar images...</span>
    </div>
  `;try{const n=await f.reverseImageSearch(e);s.innerHTML=`
      <div class="reverse-results">
        <div class="query-image-section">
          <h3>Search image</h3>
          <img src="${B(e)}" alt="Query image" class="query-image" />
        </div>
        ${n.similar_images.length>0?`
          <div class="similar-images-section">
            <h3>Similar images (${n.similar_images.length})</h3>
            <div class="image-grid">
              ${n.similar_images.map((a,r)=>ne(a,r)).join("")}
            </div>
          </div>
        `:'<div class="py-8 text-secondary">No similar images found.</div>'}
      </div>
    `,s.querySelectorAll(".image-card").forEach(a=>{a.addEventListener("click",()=>{const r=parseInt(a.dataset.imageIndex||"0",10);se(n.similar_images[r])})})}catch(n){s.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to search by image. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${O(String(n))}</p>
      </div>
    `}}}function us(){const e=document.getElementById("preview-panel"),t=document.getElementById("preview-close"),s=e==null?void 0:e.querySelector(".preview-overlay");t==null||t.addEventListener("click",oe),s==null||s.addEventListener("click",oe),document.addEventListener("keydown",n=>{n.key==="Escape"&&oe()})}function se(e){const t=document.getElementById("preview-panel"),s=document.getElementById("preview-image"),n=document.getElementById("preview-details");if(!t||!s||!n)return;s.src=e.url,s.alt=e.title;const a=e.width&&e.height&&e.width>0&&e.height>0;n.innerHTML=`
    <div class="preview-header">
      <img src="${B(e.thumbnail_url||e.url)}" class="preview-thumb" alt="" />
      <div class="preview-header-info">
        <h3 class="preview-title">${O(e.title||"Untitled")}</h3>
        <a href="${B(e.source_url)}" target="_blank" class="preview-domain">${O(e.source_domain)}</a>
      </div>
    </div>
    <div class="preview-meta">
      ${a?`<div class="preview-meta-item"><span class="preview-meta-label">Size</span><span>${e.width} × ${e.height}</span></div>`:""}
      ${e.format?`<div class="preview-meta-item"><span class="preview-meta-label">Type</span><span>${e.format.toUpperCase()}</span></div>`:""}
    </div>
    <div class="preview-actions">
      <a href="${B(e.source_url)}" target="_blank" class="preview-btn preview-btn-primary">
        Visit page ${ye}
      </a>
      <a href="${B(e.url)}" target="_blank" class="preview-btn">
        View full image ${ye}
      </a>
    </div>
  `,t.classList.remove("hidden"),document.body.style.overflow="hidden"}function oe(){const e=document.getElementById("preview-panel");e&&(e.classList.add("hidden"),document.body.style.overflow="")}function hs(){Z&&Z.disconnect();const e=document.getElementById("images-content");if(!e)return;const t=document.getElementById("scroll-sentinel");t&&t.remove();const s=document.createElement("div");s.id="scroll-sentinel",s.className="scroll-sentinel",e.appendChild(s),Z=new IntersectionObserver(n=>{n[0].isIntersecting&&!q&&M&&G&&ps()},{rootMargin:"400px"}),Z.observe(s)}async function ps(){if(q||!M)return;q=!0,W++;const e=document.getElementById("scroll-sentinel");e&&(e.innerHTML='<div class="loading-more"><div class="spinner-sm"></div></div>');try{const t=await f.searchImages(G,{...C,page:W}),s=t.results;M=t.has_more,E=[...E,...s];const n=document.querySelector(".image-grid");if(n&&s.length>0){const a=E.length-s.length,r=s.map((i,o)=>ne(i,a+o)).join("");n.insertAdjacentHTML("beforeend",r),n.querySelectorAll(".image-card:not([data-initialized])").forEach(i=>{i.setAttribute("data-initialized","true"),i.addEventListener("click",()=>{const o=parseInt(i.dataset.imageIndex||"0",10);se(E[o])})})}e&&(e.innerHTML=M?"":'<div class="no-more-results">No more images</div>')}catch{e&&(e.innerHTML="")}finally{q=!1}}async function ce(e,t){var n;const s=document.getElementById("images-content");if(!(!s||!e)){q=!0,s.innerHTML='<div class="flex items-center justify-center py-16"><div class="spinner"></div></div>';try{const a=await f.searchImages(e,{...t,page:1,per_page:50}),r=a.results;if(M=a.has_more,E=r,le=(n=a.related_searches)!=null&&n.length?a.related_searches:gs(e),os(le),r.length===0){s.innerHTML=`<div class="py-8 text-secondary">No image results found for "<strong>${O(e)}</strong>"</div>`;return}s.innerHTML=`<div class="image-grid">${r.map((i,o)=>ne(i,o)).join("")}</div>`,s.querySelectorAll(".image-card").forEach(i=>{i.setAttribute("data-initialized","true"),i.addEventListener("click",()=>{const o=parseInt(i.dataset.imageIndex||"0",10);se(E[o])})}),hs()}catch(a){s.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to load image results. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${O(String(a))}</p>
      </div>
    `}finally{q=!1}}}function ne(e,t){return`
    <div class="image-card" data-image-index="${t}">
      <div class="image-card-img">
        <img
          src="${B(e.thumbnail_url||e.url)}"
          alt="${B(e.title)}"
          loading="lazy"
          onerror="this.closest('.image-card').style.display='none'"
        />
      </div>
    </div>
  `}function O(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function B(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}function gs(e){const t=e.toLowerCase().trim().split(/\s+/).filter(i=>i.length>1);if(t.length===0)return[];const s=[],n=["wallpaper","hd","4k","aesthetic","cute","beautiful","background","art","photography","design","illustration","vintage","modern","minimalist","colorful","dark","light"],a={cat:["kitten","cats playing","black cat","tabby cat","cat meme"],dog:["puppy","dogs playing","golden retriever","german shepherd","dog meme"],nature:["forest","mountains","ocean","sunset nature","flowers"],food:["dessert","healthy food","breakfast","dinner","food photography"],car:["sports car","luxury car","vintage car","car interior","supercar"],house:["modern house","interior design","living room","bedroom design","architecture"],city:["skyline","night city","urban photography","street photography","downtown"]},r=t.slice(0,2).join(" ");for(const i of n)!e.includes(i)&&s.length<4&&s.push(`${r} ${i}`);for(const[i,o]of Object.entries(a))if(t.some(l=>l.includes(i)||i.includes(l))){for(const l of o)!s.includes(l)&&s.length<8&&s.push(l);break}return t.length>=2&&s.length<8&&s.push(t.reverse().join(" ")),s.length<4&&s.push(`${r} images`,`${r} photos`,`best ${r}`),s.slice(0,8)}const vs='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>';function ms(e){return`
    <div class="min-h-screen flex flex-col">
      <!-- Header -->
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="flex items-center gap-4 px-4 lg:px-8 py-3">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[692px]">
            ${F({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors" aria-label="Settings">
            ${vs}
          </a>
        </div>
        <div class="px-4 lg:px-8 pl-[170px]">
          ${te({query:e,active:"videos"})}
        </div>
      </header>

      <!-- Content -->
      <main class="flex-1">
        <div id="videos-content" class="px-4 lg:px-8 py-6">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `}function fs(e,t){P(s=>{e.navigate(`/videos?q=${encodeURIComponent(s)}`)}),t&&ee(t),ys(t)}async function ys(e){const t=document.getElementById("videos-content");if(!(!t||!e))try{const s=await f.searchVideos(e),n=s.results;if(n.length===0){t.innerHTML=`
        <div class="py-8 text-secondary">No video results found for "<strong>${j(e)}</strong>"</div>
      `;return}t.innerHTML=`
      <div class="text-xs text-tertiary mb-4">
        About ${s.total_results.toLocaleString()} video results (${(s.search_time_ms/1e3).toFixed(2)} seconds)
      </div>
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
        ${n.map(a=>ws(a)).join("")}
      </div>
    `}catch(s){t.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to load video results. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${j(String(s))}</p>
      </div>
    `}}function ws(e){var r;const t=((r=e.thumbnail)==null?void 0:r.url)||"",s=e.views?xs(e.views):"",n=e.published?bs(e.published):"",a=[e.channel,s,n].filter(Boolean).join(" · ");return`
    <div class="video-card">
      <a href="${J(e.url)}" target="_blank" rel="noopener" class="block">
        <div class="video-thumb">
          ${t?`<img src="${J(t)}" alt="${J(e.title)}" loading="lazy" onerror="this.style.display='none'" />`:`<div class="w-full h-full flex items-center justify-center bg-surface">
                  <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="#dadce0" stroke-width="1.5"><path d="m22 8-6 4 6 4V8Z"/><rect width="14" height="12" x="2" y="6" rx="2" ry="2"/></svg>
                </div>`}
          ${e.duration?`<span class="video-duration">${j(e.duration)}</span>`:""}
        </div>
      </a>
      <div class="video-info">
        <div class="video-title">
          <a href="${J(e.url)}" target="_blank" rel="noopener">${j(e.title)}</a>
        </div>
        <div class="video-meta">${j(a)}</div>
        ${e.platform?`<div class="text-xs text-light mt-1">${j(e.platform)}</div>`:""}
      </div>
    </div>
  `}function xs(e){return e>=1e6?`${(e/1e6).toFixed(1)}M views`:e>=1e3?`${(e/1e3).toFixed(1)}K views`:`${e} views`}function bs(e){try{const t=new Date(e),n=new Date().getTime()-t.getTime(),a=Math.floor(n/(1e3*60*60*24));return a===0?"Today":a===1?"1 day ago":a<7?`${a} days ago`:a<30?`${Math.floor(a/7)} weeks ago`:a<365?`${Math.floor(a/30)} months ago`:`${Math.floor(a/365)} years ago`}catch{return e}}function j(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function J(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const $s='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>';function ks(e){return`
    <div class="min-h-screen flex flex-col">
      <!-- Header -->
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="flex items-center gap-4 px-4 lg:px-8 py-3">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[692px]">
            ${F({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors" aria-label="Settings">
            ${$s}
          </a>
        </div>
        <div class="px-4 lg:px-8 pl-[170px]">
          ${te({query:e,active:"news"})}
        </div>
      </header>

      <!-- Content -->
      <main class="flex-1">
        <div id="news-content" class="px-4 lg:px-8 py-6">
          <div class="max-w-[900px]">
            <div class="flex items-center justify-center py-16">
              <div class="spinner"></div>
            </div>
          </div>
        </div>
      </main>
    </div>
  `}function Is(e,t){P(s=>{e.navigate(`/news?q=${encodeURIComponent(s)}`)}),t&&ee(t),Cs(t)}async function Cs(e){const t=document.getElementById("news-content");if(!(!t||!e))try{const s=await f.searchNews(e),n=s.results;if(n.length===0){t.innerHTML=`
        <div class="max-w-[900px]">
          <div class="py-8 text-secondary">No news results found for "<strong>${D(e)}</strong>"</div>
        </div>
      `;return}t.innerHTML=`
      <div class="max-w-[900px]">
        <div class="text-xs text-tertiary mb-6">
          About ${s.total_results.toLocaleString()} news results (${(s.search_time_ms/1e3).toFixed(2)} seconds)
        </div>
        <div class="space-y-4">
          ${n.map(a=>Es(a)).join("")}
        </div>
      </div>
    `}catch(s){t.innerHTML=`
      <div class="max-w-[900px]">
        <div class="py-8">
          <p class="text-red text-sm">Failed to load news results. Please try again.</p>
          <p class="text-tertiary text-xs mt-2">${D(String(s))}</p>
        </div>
      </div>
    `}}function Es(e){var n;const t=((n=e.thumbnail)==null?void 0:n.url)||"",s=e.published_date?Ls(e.published_date):"";return`
    <div class="news-card">
      <div class="flex-1 min-w-0">
        <div class="news-source">
          ${D(e.source||e.domain)}
          ${s?` · ${D(s)}`:""}
        </div>
        <div class="news-title">
          <a href="${$e(e.url)}" target="_blank" rel="noopener">${D(e.title)}</a>
        </div>
        <div class="news-snippet">${e.snippet||""}</div>
      </div>
      ${t?`<img class="news-image" src="${$e(t)}" alt="" loading="lazy" onerror="this.style.display='none'" />`:""}
    </div>
  `}function Ls(e){try{const t=new Date(e),n=new Date().getTime()-t.getTime(),a=Math.floor(n/(1e3*60*60)),r=Math.floor(n/(1e3*60*60*24));return a<1?"Just now":a<24?`${a}h ago`:r===1?"1 day ago":r<7?`${r} days ago`:r<30?`${Math.floor(r/7)} weeks ago`:t.toLocaleDateString("en-US",{month:"short",day:"numeric",year:"numeric"})}catch{return e}}function D(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function $e(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const Ss='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',Be='<svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M10 20v-6h4v6h5v-8h3L12 3 2 12h3v8z"/></svg>',_s='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>',Bs='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z"/></svg>',ke='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/><circle cx="12" cy="10" r="3"/></svg>',Ms='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>',Hs='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="7" width="20" height="14" rx="2" ry="2"/><path d="M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16"/></svg>',As='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="4" y="4" width="16" height="16" rx="2" ry="2"/><rect x="9" y="9" width="6" height="6"/><line x1="9" y1="1" x2="9" y2="4"/><line x1="15" y1="1" x2="15" y2="4"/><line x1="9" y1="20" x2="9" y2="23"/><line x1="15" y1="20" x2="15" y2="23"/><line x1="20" y1="9" x2="23" y2="9"/><line x1="20" y1="14" x2="23" y2="14"/><line x1="1" y1="9" x2="4" y2="9"/><line x1="1" y1="14" x2="4" y2="14"/></svg>',Ts='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="2" width="20" height="20" rx="2.18" ry="2.18"/><line x1="7" y1="2" x2="7" y2="22"/><line x1="17" y1="2" x2="17" y2="22"/><line x1="2" y1="12" x2="22" y2="12"/><line x1="2" y1="7" x2="7" y2="7"/><line x1="2" y1="17" x2="7" y2="17"/><line x1="17" y1="17" x2="22" y2="17"/><line x1="17" y1="7" x2="22" y2="7"/></svg>',Ns='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>',Rs='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2v6.5a.5.5 0 0 0 .5.5h3a.5.5 0 0 1 .5.5v1a.5.5 0 0 1-.5.5H14.5a.5.5 0 0 0-.5.5V22H10V11.5a.5.5 0 0 0-.5-.5H6.5a.5.5 0 0 1-.5-.5v-1a.5.5 0 0 1 .5-.5h3a.5.5 0 0 0 .5-.5V2"/></svg>',Os='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"/></svg>',js='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>',Me=[{id:"top",label:"Top Stories",icon:js},{id:"world",label:"World",icon:Ms},{id:"nation",label:"U.S.",icon:Be},{id:"business",label:"Business",icon:Hs},{id:"technology",label:"Technology",icon:As},{id:"entertainment",label:"Entertainment",icon:Ts},{id:"sports",label:"Sports",icon:Ns},{id:"science",label:"Science",icon:Rs},{id:"health",label:"Health",icon:Os}];function qs(){const e=new Date().toLocaleDateString("en-US",{weekday:"long",month:"long",day:"numeric"});return`
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
            ${Be}
            <span>Home</span>
          </a>
          <a href="/news-home?section=for-you" data-link class="news-nav-item">
            ${_s}
            <span>For you</span>
          </a>
          <a href="/news-home?section=following" data-link class="news-nav-item">
            ${Bs}
            <span>Following</span>
          </a>
        </div>

        <div class="news-nav-divider"></div>

        <div class="news-nav-section">
          ${Me.map(t=>`
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
            ${F({size:"sm"})}
          </div>

          <div class="news-header-right">
            <button class="news-icon-btn" id="location-btn" title="Change location">
              ${ke}
            </button>
            <a href="/settings" data-link class="news-icon-btn" title="Settings">
              ${Ss}
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
                ${ke}
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
  `}function Fs(e,t){P(a=>{e.navigate(`/news?q=${encodeURIComponent(a)}`)});const s=document.getElementById("menu-toggle"),n=document.querySelector(".news-sidebar");s&&n&&s.addEventListener("click",()=>{n.classList.toggle("open")}),Ps()}async function Ps(e){const t=document.getElementById("news-loading"),s=document.getElementById("top-stories-section"),n=document.getElementById("for-you-section"),a=document.getElementById("local-section");try{const r=await f.newsHome();if(t&&(t.style.display="none"),s&&r.topStories.length>0){s.style.display="block";const o=document.getElementById("top-stories-grid");o&&(o.innerHTML=Us(r.topStories))}if(n&&r.forYou.length>0){n.style.display="block";const o=document.getElementById("for-you-list");o&&(o.innerHTML=r.forYou.slice(0,10).map(l=>Gs(l)).join(""))}if(a&&r.localNews.length>0){a.style.display="block";const o=document.getElementById("local-news-scroll");o&&(o.innerHTML=r.localNews.map(l=>He(l)).join(""))}const i=document.getElementById("category-sections");if(i&&r.categories){const o=Object.entries(r.categories).filter(([l,p])=>p&&p.length>0).map(([l,p])=>Ws(l,p)).join("");i.innerHTML=o}Ys()}catch(r){t&&(t.innerHTML=`
        <div class="news-error">
          <p>Failed to load news. Please try again.</p>
          <button class="news-btn" onclick="location.reload()">Retry</button>
        </div>
      `),console.error("Failed to load news:",r)}}function Us(e){if(e.length===0)return"";const t=e[0],s=e.slice(1,3),n=e.slice(3,9);return`
    <div class="news-featured-row">
      ${zs(t)}
      <div class="news-secondary-col">
        ${s.map(a=>Vs(a)).join("")}
      </div>
    </div>
    <div class="news-grid-row">
      ${n.map(a=>Ds(a)).join("")}
    </div>
  `}function zs(e){const t=Y(e.publishedAt);return`
    <article class="news-card news-card-featured">
      ${e.imageUrl?`<img class="news-card-image" src="${x(e.imageUrl)}" alt="" loading="lazy" onerror="this.style.display='none'" />`:""}
      <div class="news-card-content">
        <div class="news-card-meta">
          <img class="news-source-icon" src="${x(e.sourceIcon||"")}" alt="" onerror="this.style.display='none'" />
          <span class="news-source-name">${w(e.source)}</span>
          <span class="news-time">${t}</span>
        </div>
        <h3 class="news-card-title">
          <a href="${x(e.url)}" target="_blank" rel="noopener" onclick="trackArticleClick('${e.id}')">${w(e.title)}</a>
        </h3>
        <p class="news-card-snippet">${w(e.snippet)}</p>
        ${e.clusterId?`<a href="/news-home?story=${e.clusterId}" data-link class="news-full-coverage">Full coverage</a>`:""}
      </div>
    </article>
  `}function Vs(e){const t=Y(e.publishedAt);return`
    <article class="news-card news-card-medium">
      <div class="news-card-content">
        <div class="news-card-meta">
          <img class="news-source-icon" src="${x(e.sourceIcon||"")}" alt="" onerror="this.style.display='none'" />
          <span class="news-source-name">${w(e.source)}</span>
          <span class="news-time">${t}</span>
        </div>
        <h3 class="news-card-title">
          <a href="${x(e.url)}" target="_blank" rel="noopener">${w(e.title)}</a>
        </h3>
      </div>
      ${e.imageUrl?`<img class="news-card-thumb" src="${x(e.imageUrl)}" alt="" loading="lazy" onerror="this.style.display='none'" />`:""}
    </article>
  `}function Ds(e){const t=Y(e.publishedAt);return`
    <article class="news-card news-card-small">
      <div class="news-card-content">
        <div class="news-card-meta">
          <span class="news-source-name">${w(e.source)}</span>
          <span class="news-time">${t}</span>
        </div>
        <h3 class="news-card-title">
          <a href="${x(e.url)}" target="_blank" rel="noopener">${w(e.title)}</a>
        </h3>
      </div>
    </article>
  `}function He(e){const t=Y(e.publishedAt);return`
    <article class="news-card news-card-compact">
      ${e.imageUrl?`<img class="news-card-thumb-sm" src="${x(e.imageUrl)}" alt="" loading="lazy" onerror="this.style.display='none'" />`:'<div class="news-card-thumb-placeholder"></div>'}
      <div class="news-card-content">
        <span class="news-source-name">${w(e.source)}</span>
        <h4 class="news-card-title-sm">
          <a href="${x(e.url)}" target="_blank" rel="noopener">${w(e.title)}</a>
        </h4>
        <span class="news-time">${t}</span>
      </div>
    </article>
  `}function Gs(e){const t=Y(e.publishedAt);return`
    <article class="news-list-item">
      <div class="news-list-content">
        <div class="news-card-meta">
          <img class="news-source-icon" src="${x(e.sourceIcon||"")}" alt="" onerror="this.style.display='none'" />
          <span class="news-source-name">${w(e.source)}</span>
          <span class="news-time">${t}</span>
        </div>
        <h3 class="news-list-title">
          <a href="${x(e.url)}" target="_blank" rel="noopener">${w(e.title)}</a>
        </h3>
        <p class="news-list-snippet">${w(e.snippet)}</p>
      </div>
      ${e.imageUrl?`<img class="news-list-thumb" src="${x(e.imageUrl)}" alt="" loading="lazy" onerror="this.style.display='none'" />`:""}
    </article>
  `}function Ws(e,t){const s=Me.find(n=>n.id===e);return s?`
    <section class="news-section">
      <div class="news-section-header">
        <h2 class="news-section-title">
          ${s.icon}
          <span>${s.label}</span>
        </h2>
        <a href="/news-home?category=${e}" data-link class="news-text-btn">More ${s.label.toLowerCase()}</a>
      </div>
      <div class="news-horizontal-scroll">
        ${t.slice(0,5).map(n=>He(n)).join("")}
      </div>
    </section>
  `:""}function Y(e){try{const t=new Date(e),n=new Date().getTime()-t.getTime(),a=Math.floor(n/(1e3*60*60)),r=Math.floor(n/(1e3*60*60*24));return a<1?"Just now":a<24?`${a}h ago`:r===1?"1 day ago":r<7?`${r} days ago`:t.toLocaleDateString("en-US",{month:"short",day:"numeric"})}catch{return""}}function Ys(){document.querySelectorAll(".news-card a, .news-list-item a").forEach(e=>{e.addEventListener("click",function(){const t=this.closest(".news-card, .news-list-item");if(t){const s=t.getAttribute("data-article-id");s&&f.recordNewsRead({id:s,url:this.href,title:this.textContent||"",snippet:"",source:"",sourceUrl:"",publishedAt:"",category:"top",engines:[],score:1}).catch(()=>{})}})})}function w(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function x(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const Ks='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m12 19-7-7 7-7"/><path d="M19 12H5"/></svg>',Zs=[{value:"auto",label:"Auto-detect"},{value:"us",label:"United States"},{value:"gb",label:"United Kingdom"},{value:"de",label:"Germany"},{value:"fr",label:"France"},{value:"es",label:"Spain"},{value:"it",label:"Italy"},{value:"nl",label:"Netherlands"},{value:"pl",label:"Poland"},{value:"br",label:"Brazil"},{value:"ca",label:"Canada"},{value:"au",label:"Australia"},{value:"in",label:"India"},{value:"jp",label:"Japan"},{value:"kr",label:"South Korea"},{value:"cn",label:"China"},{value:"ru",label:"Russia"}],Js=[{value:"en",label:"English"},{value:"de",label:"German (Deutsch)"},{value:"fr",label:"French (Français)"},{value:"es",label:"Spanish (Español)"},{value:"it",label:"Italian (Italiano)"},{value:"pt",label:"Portuguese (Português)"},{value:"nl",label:"Dutch (Nederlands)"},{value:"pl",label:"Polish (Polski)"},{value:"ja",label:"Japanese"},{value:"ko",label:"Korean"},{value:"zh",label:"Chinese"},{value:"ru",label:"Russian"},{value:"ar",label:"Arabic"},{value:"hi",label:"Hindi"}];function Qs(){const e=R.get().settings;return`
    <div class="min-h-screen bg-white">
      <!-- Header -->
      <header class="border-b border-border">
        <div class="max-w-[700px] mx-auto px-4 py-4 flex items-center gap-4">
          <a href="/" data-link class="text-tertiary hover:text-primary transition-colors" aria-label="Back">
            ${Ks}
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
              ${Zs.map(t=>`<option value="${t.value}" ${e.region===t.value?"selected":""}>${Ie(t.label)}</option>`).join("")}
            </select>
          </div>

          <!-- Language -->
          <div class="settings-section">
            <h3>Language</h3>
            <select name="language" class="settings-select">
              ${Js.map(t=>`<option value="${t.value}" ${e.language===t.value?"selected":""}>${Ie(t.label)}</option>`).join("")}
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
  `}function Xs(e){const t=document.getElementById("settings-form"),s=document.getElementById("settings-status");t&&t.addEventListener("submit",async n=>{n.preventDefault();const a=new FormData(t),r={safe_search:a.get("safe_search")||"moderate",results_per_page:parseInt(a.get("results_per_page"))||10,region:a.get("region")||"auto",language:a.get("language")||"en",theme:a.get("theme")||"light",open_in_new_tab:a.has("open_in_new_tab"),show_thumbnails:a.has("show_thumbnails")};R.set({settings:r});try{await f.updateSettings(r)}catch{}s&&(s.classList.remove("hidden"),setTimeout(()=>{s.classList.add("hidden")},2e3))})}function Ie(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}const en='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m12 19-7-7 7-7"/><path d="M19 12H5"/></svg>',tn='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/></svg>',sn='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>',nn='<svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="#dadce0" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/><path d="M12 7v5l4 2"/></svg>';function an(){return`
    <div class="min-h-screen bg-white">
      <!-- Header -->
      <header class="border-b border-border">
        <div class="max-w-[700px] mx-auto px-4 py-4 flex items-center justify-between">
          <div class="flex items-center gap-4">
            <a href="/" data-link class="text-tertiary hover:text-primary transition-colors" aria-label="Back">
              ${en}
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
  `}function rn(e){const t=document.getElementById("clear-all-btn");on(e),t==null||t.addEventListener("click",async()=>{if(confirm("Are you sure you want to clear all search history?"))try{await f.clearHistory(),ue(),t.classList.add("hidden")}catch(s){console.error("Failed to clear history:",s)}})}async function on(e){const t=document.getElementById("history-content"),s=document.getElementById("clear-all-btn");if(t)try{const n=await f.getHistory();if(n.length===0){ue();return}s&&s.classList.remove("hidden"),t.innerHTML=`
      <div id="history-list">
        ${n.map(a=>ln(a)).join("")}
      </div>
    `,cn(e)}catch(n){t.innerHTML=`
      <div class="py-8 text-center">
        <p class="text-red text-sm">Failed to load search history.</p>
        <p class="text-tertiary text-xs mt-2">${de(String(n))}</p>
      </div>
    `}}function ln(e){const t=dn(e.searched_at);return`
    <div class="history-item flex items-center gap-3 py-3 px-2 border-b border-border hover:bg-surface-hover rounded transition-colors group" data-history-id="${Ce(e.id)}">
      <span class="text-light flex-shrink-0">${sn}</span>
      <div class="flex-1 min-w-0">
        <a href="/search?q=${encodeURIComponent(e.query)}" data-link class="text-sm text-primary hover:text-link font-medium truncate block">
          ${de(e.query)}
        </a>
        <div class="flex items-center gap-2 text-xs text-light mt-0.5">
          <span>${de(t)}</span>
          ${e.results>0?`<span>&middot; ${e.results} results</span>`:""}
          ${e.clicked_url?"<span>&middot; visited</span>":""}
        </div>
      </div>
      <button class="history-delete-btn text-light hover:text-red p-1.5 rounded-full hover:bg-red/10 opacity-0 group-hover:opacity-100 transition-opacity flex-shrink-0 cursor-pointer"
              data-delete-id="${Ce(e.id)}" aria-label="Delete">
        ${tn}
      </button>
    </div>
  `}function cn(e){document.querySelectorAll(".history-delete-btn").forEach(t=>{t.addEventListener("click",async s=>{s.preventDefault(),s.stopPropagation();const n=t.dataset.deleteId||"",a=t.closest(".history-item");try{await f.deleteHistoryItem(n),a&&a.remove();const r=document.getElementById("history-list");if(r&&r.children.length===0){ue();const i=document.getElementById("clear-all-btn");i&&i.classList.add("hidden")}}catch(r){console.error("Failed to delete history item:",r)}})})}function ue(){const e=document.getElementById("history-content");e&&(e.innerHTML=`
    <div class="py-16 flex flex-col items-center text-center">
      ${nn}
      <h2 class="text-lg font-medium text-primary mt-4 mb-2">No search history</h2>
      <p class="text-sm text-tertiary max-w-[300px]">
        Your recent searches will appear here. Start searching to build your history.
      </p>
      <a href="/" data-link class="mt-4 text-sm text-blue hover:underline">Go to search</a>
    </div>
  `)}function dn(e){try{const t=new Date(e),s=new Date,n=s.getTime()-t.getTime(),a=Math.floor(n/(1e3*60)),r=Math.floor(n/(1e3*60*60)),i=Math.floor(n/(1e3*60*60*24));return a<1?"Just now":a<60?`${a}m ago`:r<24?`${r}h ago`:i===1?"Yesterday":i<7?`${i} days ago`:t.toLocaleDateString("en-US",{month:"short",day:"numeric",year:t.getFullYear()!==s.getFullYear()?"numeric":void 0})}catch{return e}}function de(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function Ce(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const k=document.getElementById("app");if(!k)throw new Error("App container not found");const v=new Ne;v.addRoute("",(e,t)=>{k.innerHTML=Ke(),Ze(v)});v.addRoute("search",(e,t)=>{const s=t.q||"",n=t.time_range||"";k.innerHTML=Vt(s,n),Dt(v,s,t)});v.addRoute("images",(e,t)=>{const s=t.q||"";k.innerHTML=ts(s),ns(v,s,t)});v.addRoute("videos",(e,t)=>{const s=t.q||"";k.innerHTML=ms(s),fs(v,s)});v.addRoute("news",(e,t)=>{const s=t.q||"";k.innerHTML=ks(s),Is(v,s)});v.addRoute("news-home",(e,t)=>{k.innerHTML=qs(),Fs(v)});v.addRoute("settings",(e,t)=>{k.innerHTML=Qs(),Xs()});v.addRoute("history",(e,t)=>{k.innerHTML=an(),rn(v)});v.setNotFound((e,t)=>{k.innerHTML=`
    <div class="min-h-screen flex flex-col items-center justify-center px-4">
      <h1 class="text-4xl font-semibold mb-4">
        <span style="color: #4285F4">4</span><span style="color: #EA4335">0</span><span style="color: #FBBC05">4</span>
      </h1>
      <p class="text-secondary mb-6">Page not found</p>
      <a href="/" data-link class="text-blue hover:underline">Go home</a>
    </div>
  `});window.addEventListener("router:navigate",e=>{const t=e;v.navigate(t.detail.path)});v.start();
