var Ve=Object.defineProperty;var De=(e,t,s)=>t in e?Ve(e,t,{enumerable:!0,configurable:!0,writable:!0,value:s}):e[t]=s;var se=(e,t,s)=>De(e,typeof t!="symbol"?t+"":t,s);(function(){const t=document.createElement("link").relList;if(t&&t.supports&&t.supports("modulepreload"))return;for(const r of document.querySelectorAll('link[rel="modulepreload"]'))a(r);new MutationObserver(r=>{for(const n of r)if(n.type==="childList")for(const i of n.addedNodes)i.tagName==="LINK"&&i.rel==="modulepreload"&&a(i)}).observe(document,{childList:!0,subtree:!0});function s(r){const n={};return r.integrity&&(n.integrity=r.integrity),r.referrerPolicy&&(n.referrerPolicy=r.referrerPolicy),r.crossOrigin==="use-credentials"?n.credentials="include":r.crossOrigin==="anonymous"?n.credentials="omit":n.credentials="same-origin",n}function a(r){if(r.ep)return;r.ep=!0;const n=s(r);fetch(r.href,n)}})();class Ge{constructor(){se(this,"routes",[]);se(this,"currentPath","");se(this,"notFoundRenderer",null)}addRoute(t,s){const a=t.split("/").filter(Boolean);this.routes.push({pattern:t,segments:a,renderer:s})}setNotFound(t){this.notFoundRenderer=t}navigate(t,s=!1){t!==this.currentPath&&(s?history.replaceState(null,"",t):history.pushState(null,"",t),this.resolve())}start(){window.addEventListener("popstate",()=>this.resolve()),document.addEventListener("click",t=>{const s=t.target.closest("a[data-link]");if(s){t.preventDefault();const a=s.getAttribute("href");a&&this.navigate(a)}}),this.resolve()}getCurrentPath(){return this.currentPath}resolve(){const t=new URL(window.location.href),s=t.pathname,a=Ye(t.search);this.currentPath=s+t.search;for(const r of this.routes){const n=We(r.segments,s);if(n!==null){r.renderer(n,a);return}}this.notFoundRenderer&&this.notFoundRenderer({},a)}}function We(e,t){const s=t.split("/").filter(Boolean);if(e.length===0&&s.length===0)return{};if(e.length!==s.length)return null;const a={};for(let r=0;r<e.length;r++){const n=e[r],i=s[r];if(n.startsWith(":"))a[n.slice(1)]=decodeURIComponent(i);else if(n!==i)return null}return a}function Ye(e){const t={};return new URLSearchParams(e).forEach((a,r)=>{t[r]=a}),t}const le="/api";async function p(e,t){let s=`${le}${e}`;if(t){const r=new URLSearchParams;Object.entries(t).forEach(([i,l])=>{l!==void 0&&l!==""&&l!==null&&r.set(i,l)});const n=r.toString();n&&(s+=`?${n}`)}const a=await fetch(s);if(!a.ok)throw new Error(`API error: ${a.status} ${a.statusText}`);return a.json()}async function T(e,t){const s=await fetch(`${le}${e}`,{method:"POST",headers:{"Content-Type":"application/json"},body:t?JSON.stringify(t):void 0});if(!s.ok)throw new Error(`API error: ${s.status} ${s.statusText}`);return s.json()}async function be(e,t){const s=await fetch(`${le}${e}`,{method:"PUT",headers:{"Content-Type":"application/json"},body:JSON.stringify(t)});if(!s.ok)throw new Error(`API error: ${s.status} ${s.statusText}`);return s.json()}async function Y(e,t){const s=await fetch(`${le}${e}`,{method:"DELETE",headers:t?{"Content-Type":"application/json"}:void 0,body:t?JSON.stringify(t):void 0});if(!s.ok)throw new Error(`API error: ${s.status} ${s.statusText}`);return s.json()}function de(e,t){const s={q:e};return t&&(t.page!==void 0&&(s.page=String(t.page)),t.per_page!==void 0&&(s.per_page=String(t.per_page)),t.time_range&&(s.time_range=t.time_range),t.region&&(s.region=t.region),t.language&&(s.language=t.language),t.safe_search&&(s.safe_search=t.safe_search),t.site&&(s.site=t.site),t.exclude_site&&(s.exclude_site=t.exclude_site),t.lens&&(s.lens=t.lens)),s}const x={search(e,t){return p("/search",de(e,t))},searchImages(e,t){const s={q:e};return t&&(t.page!==void 0&&(s.page=String(t.page)),t.per_page!==void 0&&(s.per_page=String(t.per_page)),t.size&&t.size!=="any"&&(s.size=t.size),t.color&&t.color!=="any"&&(s.color=t.color),t.type&&t.type!=="any"&&(s.type=t.type),t.aspect&&t.aspect!=="any"&&(s.aspect=t.aspect),t.time&&t.time!=="any"&&(s.time=t.time),t.rights&&t.rights!=="any"&&(s.rights=t.rights),t.filetype&&t.filetype!=="any"&&(s.filetype=t.filetype),t.safe&&(s.safe=t.safe)),p("/search/images",s)},reverseImageSearch(e){return T("/search/images/reverse",{url:e})},reverseImageSearchByUpload(e){return T("/search/images/reverse",{image_data:e})},searchVideos(e,t){return p("/search/videos",de(e,t))},searchNews(e,t){return p("/search/news",de(e,t))},searchMusic(e,t){const s=new URLSearchParams({q:e});return t!=null&&t.page&&s.set("page",String(t.page)),p(`/search/music?${s}`)},searchScience(e,t){const s=new URLSearchParams({q:e});return t!=null&&t.page&&s.set("page",String(t.page)),t!=null&&t.per_page&&s.set("per_page",String(t.per_page)),p(`/search/science?${s}`)},searchMaps(e){const t=new URLSearchParams({q:e});return p(`/search/maps?${t}`)},searchCode(e,t){const s=new URLSearchParams({q:e});return t!=null&&t.page&&s.set("page",String(t.page)),t!=null&&t.per_page&&s.set("per_page",String(t.per_page)),p(`/search/code?${s}`)},searchSocial(e,t){const s=new URLSearchParams({q:e});return t!=null&&t.page&&s.set("page",String(t.page)),p(`/search/social?${s}`)},suggest(e){return p("/suggest",{q:e})},trending(){return p("/suggest/trending")},calculate(e){return p("/instant/calculate",{q:e})},convert(e){return p("/instant/convert",{q:e})},currency(e){return p("/instant/currency",{q:e})},weather(e){return p("/instant/weather",{q:e})},define(e){return p("/instant/define",{q:e})},time(e){return p("/instant/time",{q:e})},knowledge(e){return p(`/knowledge/${encodeURIComponent(e)}`)},getPreferences(){return p("/preferences")},setPreference(e,t){return T("/preferences",{domain:e,action:t})},deletePreference(e){return Y(`/preferences/${encodeURIComponent(e)}`)},getLenses(){return p("/lenses")},createLens(e){return T("/lenses",e)},deleteLens(e){return Y(`/lenses/${encodeURIComponent(e)}`)},getHistory(){return p("/history")},clearHistory(){return Y("/history")},deleteHistoryItem(e){return Y(`/history/${encodeURIComponent(e)}`)},getSettings(){return p("/settings")},updateSettings(e){return be("/settings",e)},getBangs(){return p("/bangs")},parseBang(e){return p("/bangs/parse",{q:e})},getRelated(e){return p("/related",{q:e})},newsHome(){return p("/news/home")},newsCategory(e,t=1){return p(`/news/category/${e}`,{page:String(t)})},newsSearch(e,t){const s={q:e};return t!=null&&t.page&&(s.page=String(t.page)),t!=null&&t.time&&(s.time=t.time),t!=null&&t.source&&(s.source=t.source),p("/news/search",s)},newsStory(e){return p(`/news/story/${e}`)},newsLocal(e){const t={};return e&&(t.city=e.city,e.state&&(t.state=e.state),t.country=e.country),p("/news/local",t)},newsFollowing(){return p("/news/following")},newsPreferences(){return p("/news/preferences")},updateNewsPreferences(e){return be("/news/preferences",e)},followNews(e,t){return T("/news/follow",{type:e,id:t})},unfollowNews(e,t){return Y("/news/follow",{type:e,id:t})},hideNewsSource(e){return T("/news/hide",{source:e})},setNewsLocation(e){return T("/news/location",e)},recordNewsRead(e,t){return T("/news/read",{article:e,duration:t})}};function Ke(e){let t={...e};const s=new Set;return{get(){return t},set(a){t={...t,...a},s.forEach(r=>r(t))},subscribe(a){return s.add(a),()=>{s.delete(a)}}}}const Oe="mizu_search_state";function Ze(){try{const e=localStorage.getItem(Oe);if(e)return JSON.parse(e)}catch{}return{recentSearches:[],settings:{safe_search:"moderate",results_per_page:10,region:"auto",language:"en",theme:"light",open_in_new_tab:!1,show_thumbnails:!0}}}const z=Ke(Ze());z.subscribe(e=>{try{localStorage.setItem(Oe,JSON.stringify(e))}catch{}});function M(e){const t=z.get(),s=[e,...t.recentSearches.filter(a=>a!==e)].slice(0,20);z.set({recentSearches:s})}const je='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>',Je='<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>',Qe='<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2a3 3 0 0 0-3 3v7a3 3 0 0 0 6 0V5a3 3 0 0 0-3-3Z"/><path d="M19 10v2a7 7 0 0 1-14 0v-2"/><line x1="12" x2="12" y1="19" y2="22"/></svg>',Xe='<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14.5 4h-5L7 7H4a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2h-3l-2.5-3z"/><circle cx="12" cy="13" r="3"/></svg>',et='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/><path d="M12 7v5l4 2"/></svg>',tt='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M13 2 3 14h9l-1 8 10-12h-9l1-8z"/></svg>';function S(e){const t=e.size==="lg"?"search-box-lg":"search-box-sm",s=e.initialValue?at(e.initialValue):"",a=e.initialValue?"":"hidden";return`
    <div id="search-box-wrapper" class="relative w-full flex justify-center">
      <div id="search-box" class="search-box ${t}">
        <span class="text-light mr-3 flex-shrink-0">${je}</span>
        <input
          id="search-input"
          type="text"
          value="${s}"
          placeholder="Search the web"
          autocomplete="off"
          spellcheck="false"
          ${e.autofocus?"autofocus":""}
        />
        <button id="search-clear-btn" class="text-secondary hover:text-primary p-1 flex-shrink-0 ${a}" type="button" aria-label="Clear">
          ${Je}
        </button>
        <span class="mx-1 w-px h-5 bg-border flex-shrink-0"></span>
        <button id="voice-search-btn" class="text-light hover:text-secondary p-1 flex-shrink-0" type="button" aria-label="Voice search">
          ${Qe}
        </button>
        <button id="camera-search-btn" class="text-light hover:text-secondary p-1 flex-shrink-0" type="button" aria-label="Image search">
          ${Xe}
        </button>
      </div>
      <div id="autocomplete-dropdown" class="autocomplete-dropdown hidden"></div>
    </div>
  `}function L(e){const t=document.getElementById("search-input"),s=document.getElementById("search-clear-btn"),a=document.getElementById("autocomplete-dropdown"),r=document.getElementById("search-box-wrapper");if(!t||!s||!a||!r)return;let n=null,i=[],l=-1,o=!1;function c(h){if(i=h,l=-1,h.length===0){d();return}o=!0,a.innerHTML=h.map((u,w)=>`
        <div class="autocomplete-item ${w===l?"active":""}" data-index="${w}">
          <span class="suggestion-icon">${u.icon}</span>
          ${u.prefix?`<span class="bang-trigger">${$e(u.prefix)}</span>`:""}
          <span>${$e(u.text)}</span>
        </div>
      `).join(""),a.classList.remove("hidden"),a.classList.add("has-items"),a.querySelectorAll(".autocomplete-item").forEach(u=>{u.addEventListener("mousedown",w=>{w.preventDefault();const I=parseInt(u.dataset.index||"0");v(I)}),u.addEventListener("mouseenter",()=>{const w=parseInt(u.dataset.index||"0");f(w)})})}function d(){o=!1,a.classList.add("hidden"),a.classList.remove("has-items"),a.innerHTML="",i=[],l=-1}function f(h){l=h,a.querySelectorAll(".autocomplete-item").forEach((u,w)=>{u.classList.toggle("active",w===h)})}function v(h){const u=i[h];u&&(u.type==="bang"&&u.prefix?(t.value=u.prefix+" ",t.focus(),$(u.prefix+" ")):(t.value=u.text,d(),y(u.text)))}function y(h){const u=h.trim();u&&(d(),e(u))}async function $(h){const u=h.trim();if(!u){W();return}if(u.startsWith("!"))try{const I=(await x.getBangs()).filter(j=>j.trigger.startsWith(u)||j.name.toLowerCase().includes(u.slice(1).toLowerCase())).slice(0,8);if(I.length>0){c(I.map(j=>({text:j.name,type:"bang",icon:tt,prefix:j.trigger})));return}}catch{}try{const w=await x.suggest(u);if(t.value.trim()!==u)return;const I=w.map(j=>({text:j.text,type:"suggestion",icon:je}));I.length===0?W(u):c(I)}catch{W(u)}}function W(h){let w=z.get().recentSearches;if(h&&(w=w.filter(I=>I.toLowerCase().includes(h.toLowerCase()))),w.length===0){d();return}c(w.slice(0,8).map(I=>({text:I,type:"recent",icon:et})))}t.addEventListener("input",()=>{const h=t.value;s.classList.toggle("hidden",h.length===0),n&&clearTimeout(n),n=setTimeout(()=>$(h),150)}),t.addEventListener("focus",()=>{t.value.trim()?$(t.value):W()}),t.addEventListener("keydown",h=>{if(!o){if(h.key==="Enter"){y(t.value);return}if(h.key==="ArrowDown"){$(t.value);return}return}switch(h.key){case"ArrowDown":h.preventDefault(),f(Math.min(l+1,i.length-1));break;case"ArrowUp":h.preventDefault(),f(Math.max(l-1,-1));break;case"Enter":h.preventDefault(),l>=0?v(l):y(t.value);break;case"Escape":d();break;case"Tab":d();break}}),t.addEventListener("blur",()=>{setTimeout(()=>d(),200)}),s.addEventListener("click",()=>{t.value="",s.classList.add("hidden"),t.focus(),W()});const ye=document.getElementById("voice-search-btn");ye&&st(ye,t,h=>{t.value=h,s.classList.remove("hidden"),y(h)});const we=document.getElementById("camera-search-btn");we&&we.addEventListener("click",()=>{const h=document.getElementById("reverse-modal");h?h.classList.remove("hidden"):window.dispatchEvent(new CustomEvent("router:navigate",{detail:{path:"/images?reverse=1"}}))})}function st(e,t,s){const a=window.SpeechRecognition||window.webkitSpeechRecognition;if(!a){e.style.display="none";return}let r=!1,n=null;e.addEventListener("click",()=>{r?l():i()});function i(){n=new a,n.continuous=!1,n.interimResults=!0,n.lang="en-US",n.onstart=()=>{r=!0,e.classList.add("listening"),e.style.color="#ea4335"},n.onresult=o=>{const c=Array.from(o.results).map(d=>d[0].transcript).join("");t.value=c,o.results[0].isFinal&&(l(),s(c))},n.onerror=o=>{console.error("Speech recognition error:",o.error),l(),o.error==="not-allowed"&&alert("Microphone access denied. Please allow microphone access to use voice search.")},n.onend=()=>{l()};try{n.start()}catch(o){console.error("Failed to start speech recognition:",o),l()}}function l(){if(r=!1,e.classList.remove("listening"),e.style.color="",n){try{n.stop()}catch{}n=null}}}function $e(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function at(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const nt=[{trigger:"!g",label:"Google",color:"#4285F4"},{trigger:"!yt",label:"YouTube",color:"#EA4335"},{trigger:"!gh",label:"GitHub",color:"#24292e"},{trigger:"!w",label:"Wikipedia",color:"#636466"},{trigger:"!r",label:"Reddit",color:"#FF5700"}],rt=[{label:"Calculator",icon:ct(),query:"2+2",color:"bg-blue/10 text-blue"},{label:"Conversion",icon:dt(),query:"10 miles in km",color:"bg-green/10 text-green"},{label:"Currency",icon:ut(),query:"100 USD to EUR",color:"bg-yellow/10 text-yellow"},{label:"Weather",icon:pt(),query:"weather New York",color:"bg-blue/10 text-blue"},{label:"Time",icon:ht(),query:"time in Tokyo",color:"bg-green/10 text-green"},{label:"Define",icon:gt(),query:"define serendipity",color:"bg-red/10 text-red"}];function it(){return`
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
          ${S({size:"lg",autofocus:!0})}
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
          ${nt.map(e=>`
            <button class="bang-shortcut px-3 py-1.5 rounded-full text-xs font-medium border border-border hover:shadow-sm transition-shadow cursor-pointer"
                    data-bang="${e.trigger}"
                    style="color: ${e.color}; border-color: ${e.color}20;">
              <span class="font-semibold">${ue(e.trigger)}</span>
              <span class="text-tertiary ml-1">${ue(e.label)}</span>
            </button>
          `).join("")}
        </div>

        <!-- Instant Answers Showcase -->
        <div class="mb-8">
          <p class="text-center text-xs text-light mb-3 uppercase tracking-wider">Instant Answers</p>
          <div class="flex flex-wrap justify-center gap-2">
            ${rt.map(e=>`
              <button class="instant-showcase-btn flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-medium ${e.color} hover:opacity-80 transition-opacity cursor-pointer"
                      data-query="${ot(e.query)}">
                ${e.icon}
                <span>${ue(e.label)}</span>
              </button>
            `).join("")}
          </div>
        </div>

        <!-- Category Links -->
        <div class="flex gap-6 text-sm">
          <a href="/images" data-link class="text-tertiary hover:text-primary transition-colors flex items-center gap-1.5">
            ${mt()}
            Images
          </a>
          <a href="/news" data-link class="text-tertiary hover:text-primary transition-colors flex items-center gap-1.5">
            ${vt()}
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
  `}function lt(e){L(a=>{e.navigate(`/search?q=${encodeURIComponent(a)}`)});const t=document.getElementById("home-search-btn");t==null||t.addEventListener("click",()=>{var n;const a=document.getElementById("search-input"),r=(n=a==null?void 0:a.value)==null?void 0:n.trim();r&&e.navigate(`/search?q=${encodeURIComponent(r)}`)});const s=document.getElementById("home-lucky-btn");s==null||s.addEventListener("click",()=>{var n;const a=document.getElementById("search-input"),r=(n=a==null?void 0:a.value)==null?void 0:n.trim();r&&e.navigate(`/search?q=${encodeURIComponent(r)}&lucky=1`)}),document.querySelectorAll(".bang-shortcut").forEach(a=>{a.addEventListener("click",()=>{const r=a.dataset.bang||"",n=document.getElementById("search-input");n&&(n.value=r+" ",n.focus())})}),document.querySelectorAll(".instant-showcase-btn").forEach(a=>{a.addEventListener("click",()=>{const r=a.dataset.query||"";r&&e.navigate(`/search?q=${encodeURIComponent(r)}`)})})}function ue(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function ot(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}function ct(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="16" height="20" x="4" y="2" rx="2"/><line x1="8" x2="16" y1="6" y2="6"/><line x1="16" x2="16" y1="14" y2="18"/></svg>'}function dt(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M8 3 4 7l4 4"/><path d="M4 7h16"/><path d="m16 21 4-4-4-4"/><path d="M20 17H4"/></svg>'}function ut(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" x2="12" y1="2" y2="22"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>'}function pt(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="M2 12h2"/><path d="M20 12h2"/></svg>'}function ht(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>'}function gt(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1 0-5H20"/></svg>'}function mt(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect width="18" height="18" x="3" y="3" rx="2" ry="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"/></svg>'}function vt(){return'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 22h16a2 2 0 0 0 2-2V4a2 2 0 0 0-2-2H8a2 2 0 0 0-2 2v16a2 2 0 0 1-2 2Zm0 0a2 2 0 0 1-2-2v-9c0-1.1.9-2 2-2h2"/></svg>'}const ft='<svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><circle cx="12" cy="5" r="2"/><circle cx="12" cy="12" r="2"/><circle cx="12" cy="19" r="2"/></svg>',xt='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M7 10v12"/><path d="M15 5.88 14 10h5.83a2 2 0 0 1 1.92 2.56l-2.33 8A2 2 0 0 1 17.5 22H4a2 2 0 0 1-2-2v-8a2 2 0 0 1 2-2h2.76a2 2 0 0 0 1.79-1.11L12 2h0a3.13 3.13 0 0 1 3 3.88Z"/></svg>',yt='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 14V2"/><path d="M9 18.12 10 14H4.17a2 2 0 0 1-1.92-2.56l2.33-8A2 2 0 0 1 6.5 2H20a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2h-2.76a2 2 0 0 0-1.79 1.11L12 22h0a3.13 3.13 0 0 1-3-3.88Z"/></svg>',wt='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="m4.9 4.9 14.2 14.2"/></svg>';function bt(e,t){const s=e.favicon||`https://www.google.com/s2/favicons?domain=${encodeURIComponent(e.domain)}&sz=32`,a=Ct(e.url),r=e.published?It(e.published):"",n=e.snippet||"",i=e.thumbnail?`<img src="${A(e.thumbnail.url)}" alt="" class="w-[120px] h-[80px] rounded-lg object-cover flex-shrink-0 ml-4" loading="lazy" />`:"",l=e.sitelinks&&e.sitelinks.length>0?`<div class="result-sitelinks">
        ${e.sitelinks.map(o=>`<a href="${A(o.url)}" target="_blank" rel="noopener">${N(o.title)}</a>`).join("")}
       </div>`:"";return`
    <div class="search-result" data-result-index="${t}" data-domain="${A(e.domain)}">
      <div class="result-url">
        <img class="favicon" src="${A(s)}" alt="" width="18" height="18" loading="lazy" onerror="this.style.display='none'" />
        <div>
          <span class="text-sm">${N(e.domain)}</span>
          <span class="breadcrumbs">${a}</span>
        </div>
      </div>
      <div class="flex items-start">
        <div class="flex-1">
          <div class="result-title">
            <a href="${A(e.url)}" target="_blank" rel="noopener">${N(e.title)}</a>
          </div>
          ${r?`<span class="result-date">${N(r)} -- </span>`:""}
          <div class="result-snippet">${n}</div>
          ${l}
        </div>
        ${i}
      </div>
      <button class="result-menu-btn" data-menu-index="${t}" aria-label="More options">
        ${ft}
      </button>
      <div id="domain-menu-${t}" class="domain-menu hidden"></div>
    </div>
  `}function $t(){document.querySelectorAll(".result-menu-btn").forEach(e=>{e.addEventListener("click",t=>{t.stopPropagation();const s=e.dataset.menuIndex,a=document.getElementById(`domain-menu-${s}`),r=e.closest(".search-result"),n=(r==null?void 0:r.dataset.domain)||"";if(!a)return;if(!a.classList.contains("hidden")){a.classList.add("hidden");return}document.querySelectorAll(".domain-menu").forEach(l=>l.classList.add("hidden")),a.innerHTML=`
        <button class="domain-menu-item boost" data-action="boost" data-domain="${A(n)}">
          ${xt}
          <span>Boost ${N(n)}</span>
        </button>
        <button class="domain-menu-item lower" data-action="lower" data-domain="${A(n)}">
          ${yt}
          <span>Lower ${N(n)}</span>
        </button>
        <button class="domain-menu-item block" data-action="block" data-domain="${A(n)}">
          ${wt}
          <span>Block ${N(n)}</span>
        </button>
      `,a.classList.remove("hidden"),a.querySelectorAll(".domain-menu-item").forEach(l=>{l.addEventListener("click",async()=>{const o=l.dataset.action||"",c=l.dataset.domain||"";try{await x.setPreference(c,o),a.classList.add("hidden"),kt(`${o.charAt(0).toUpperCase()+o.slice(1)}ed ${c}`)}catch(d){console.error("Failed to set preference:",d)}})});const i=l=>{!a.contains(l.target)&&l.target!==e&&(a.classList.add("hidden"),document.removeEventListener("click",i))};setTimeout(()=>document.addEventListener("click",i),0)})})}function kt(e){const t=document.getElementById("toast");t&&t.remove();const s=document.createElement("div");s.id="toast",s.className="fixed bottom-6 left-1/2 -translate-x-1/2 bg-primary text-white px-5 py-3 rounded-lg shadow-lg text-sm z-50 transition-opacity duration-300",s.textContent=e,document.body.appendChild(s),setTimeout(()=>{s.style.opacity="0",setTimeout(()=>s.remove(),300)},2e3)}function Ct(e){try{const s=new URL(e).pathname.split("/").filter(Boolean);return s.length===0?"":" > "+s.map(a=>N(decodeURIComponent(a))).join(" > ")}catch{return""}}function It(e){try{const t=new Date(e),a=new Date().getTime()-t.getTime(),r=Math.floor(a/(1e3*60*60*24));return r===0?"Today":r===1?"1 day ago":r<7?`${r} days ago`:r<30?`${Math.floor(r/7)} weeks ago`:t.toLocaleDateString("en-US",{month:"short",day:"numeric",year:"numeric"})}catch{return e}}function N(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function A(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const St='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="16" height="20" x="4" y="2" rx="2"/><line x1="8" x2="16" y1="6" y2="6"/><line x1="16" x2="16" y1="14" y2="18"/><path d="M16 10h.01"/><path d="M12 10h.01"/><path d="M8 10h.01"/><path d="M12 14h.01"/><path d="M8 14h.01"/><path d="M12 18h.01"/><path d="M8 18h.01"/></svg>',Lt='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M8 3 4 7l4 4"/><path d="M4 7h16"/><path d="m16 21 4-4-4-4"/><path d="M20 17H4"/></svg>',Et='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" x2="12" y1="2" y2="22"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>',_t='<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#FBBC05" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="m4.93 4.93 1.41 1.41"/><path d="m17.66 17.66 1.41 1.41"/><path d="M2 12h2"/><path d="M20 12h2"/><path d="m6.34 17.66-1.41 1.41"/><path d="m19.07 4.93-1.41 1.41"/></svg>',Mt='<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#5f6368" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17.5 19H9a7 7 0 1 1 6.71-9h1.79a4.5 4.5 0 1 1 0 9Z"/></svg>',Bt='<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#4285F4" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 14.899A7 7 0 1 1 15.71 8h1.79a4.5 4.5 0 0 1 2.5 8.242"/><path d="M16 14v6"/><path d="M8 14v6"/><path d="M12 16v6"/></svg>',Tt='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1 0-5H20"/></svg>',Ht='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>';function At(e){switch(e.type){case"calculator":return Nt(e);case"unit_conversion":return Rt(e);case"currency":return Ot(e);case"weather":return jt(e);case"definition":return qt(e);case"time":return Ft(e);default:return Pt(e)}}function Nt(e){const t=e.data||{},s=t.expression||e.query||"",a=t.formatted||t.result||e.result||"";return`
    <div class="instant-card border-l-4 border-l-blue">
      <div class="flex items-center gap-2 mb-2 text-tertiary">
        ${St}
        <span class="instant-type">Calculator</span>
      </div>
      <div class="instant-result">${g(s)} = ${g(String(a))}</div>
    </div>
  `}function Rt(e){const t=e.data||{},s=t.from_value??"",a=t.from_unit??"",r=t.to_value??"",n=t.to_unit??"",i=t.category??"";return`
    <div class="instant-card border-l-4 border-l-green">
      <div class="flex items-center gap-2 mb-2 text-tertiary">
        ${Lt}
        <span class="instant-type">Unit Conversion${i?` -- ${g(i)}`:""}</span>
      </div>
      <div class="instant-result">${g(String(s))} ${g(a)} = ${g(String(r))} ${g(n)}</div>
      ${t.formatted?`<div class="instant-sub">${g(t.formatted)}</div>`:""}
    </div>
  `}function Ot(e){const t=e.data||{},s=t.from_value??"",a=t.from_currency??"",r=t.to_value??"",n=t.to_currency??"",i=t.rate??"";return`
    <div class="instant-card border-l-4 border-l-yellow">
      <div class="flex items-center gap-2 mb-2 text-tertiary">
        ${Et}
        <span class="instant-type">Currency</span>
      </div>
      <div class="instant-result">${g(String(s))} ${g(a)} = ${g(String(r))} ${g(n)}</div>
      ${i?`<div class="instant-sub">1 ${g(a)} = ${g(String(i))} ${g(n)}</div>`:""}
    </div>
  `}function jt(e){const t=e.data||{},s=t.location||"",a=t.temperature??"",r=(t.condition||"").toLowerCase(),n=t.humidity||"",i=t.wind||"";let l=_t;return r.includes("cloud")||r.includes("overcast")?l=Mt:(r.includes("rain")||r.includes("drizzle")||r.includes("storm"))&&(l=Bt),`
    <div class="instant-card border-l-4 border-l-blue">
      <div class="instant-type mb-2">Weather</div>
      <div class="flex items-center gap-4 mb-3">
        <div>${l}</div>
        <div>
          <div class="text-2xl font-semibold text-primary">${g(String(a))}&deg;</div>
          <div class="text-secondary capitalize">${g(t.condition||"")}</div>
        </div>
      </div>
      <div class="text-sm font-medium text-primary mb-2">${g(s)}</div>
      <div class="flex gap-6 text-sm text-tertiary">
        ${n?`<span>Humidity: ${g(n)}</span>`:""}
        ${i?`<span>Wind: ${g(i)}</span>`:""}
      </div>
    </div>
  `}function qt(e){const t=e.data||{},s=t.word||e.query||"",a=t.phonetic||"",r=t.part_of_speech||"",n=t.definitions||[],i=t.synonyms||[];return`
    <div class="instant-card border-l-4 border-l-red">
      <div class="flex items-center gap-2 mb-2 text-tertiary">
        ${Tt}
        <span class="instant-type">Definition</span>
      </div>
      <div class="flex items-baseline gap-3 mb-1">
        <span class="text-xl font-semibold text-primary">${g(s)}</span>
        ${a?`<span class="text-tertiary text-sm">${g(a)}</span>`:""}
      </div>
      ${r?`<div class="text-sm italic text-secondary mb-2">${g(r)}</div>`:""}
      ${n.length>0?`<ol class="list-decimal list-inside space-y-1 text-sm text-snippet mb-3">
              ${n.map(l=>`<li>${g(l)}</li>`).join("")}
             </ol>`:""}
      ${i.length>0?`<div class="text-sm">
              <span class="text-tertiary">Synonyms: </span>
              <span class="text-secondary">${i.map(l=>g(l)).join(", ")}</span>
             </div>`:""}
    </div>
  `}function Ft(e){const t=e.data||{},s=t.location||"",a=t.time||"",r=t.date||"",n=t.timezone||"";return`
    <div class="instant-card border-l-4 border-l-green">
      <div class="flex items-center gap-2 mb-2 text-tertiary">
        ${Ht}
        <span class="instant-type">Time</span>
      </div>
      <div class="text-sm font-medium text-secondary mb-1">${g(s)}</div>
      <div class="text-4xl font-semibold text-primary mb-1">${g(a)}</div>
      <div class="text-sm text-tertiary">${g(r)}</div>
      ${n?`<div class="text-xs text-light mt-1">${g(n)}</div>`:""}
    </div>
  `}function Pt(e){return`
    <div class="instant-card border-l-4 border-l-blue">
      <div class="instant-type mb-2">${g(e.type)}</div>
      <div class="instant-result">${g(e.result)}</div>
    </div>
  `}function g(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}const Ut='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" x2="21" y1="14" y2="3"/></svg>';function zt(e){const t=e.image?`<img class="kp-image" src="${pe(e.image)}" alt="${pe(e.title)}" loading="lazy" onerror="this.style.display='none'" />`:"",s=e.facts&&e.facts.length>0?`<table class="kp-facts">
          <tbody>
            ${e.facts.map(n=>`
              <tr>
                <td class="fact-label">${q(n.label)}</td>
                <td class="fact-value">${q(n.value)}</td>
              </tr>
            `).join("")}
          </tbody>
        </table>`:"",a=e.links&&e.links.length>0?`<div class="kp-links">
          ${e.links.map(n=>`
            <a class="kp-link" href="${pe(n.url)}" target="_blank" rel="noopener">
              ${Ut}
              <span>${q(n.title)}</span>
            </a>
          `).join("")}
        </div>`:"",r=e.source?`<div class="kp-source">Source: ${q(e.source)}</div>`:"";return`
    <div class="knowledge-panel" id="knowledge-panel">
      ${t}
      <div class="kp-title">${q(e.title)}</div>
      ${e.subtitle?`<div class="kp-subtitle">${q(e.subtitle)}</div>`:""}
      <div class="kp-description">${q(e.description)}</div>
      ${s}
      ${a}
      ${r}
    </div>
  `}function q(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function pe(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const Vt='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m15 18-6-6 6-6"/></svg>',Dt='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"/></svg>';function Gt(e){const{currentPage:t,hasMore:s,totalResults:a,perPage:r}=e,n=Math.min(Math.ceil(a/r),100);if(n<=1)return"";let i=Math.max(1,t-4),l=Math.min(n,i+9);l-i<9&&(i=Math.max(1,l-9));const o=[];for(let v=i;v<=l;v++)o.push(v);const c=Wt(t),d=t<=1?"disabled":"",f=!s&&t>=n?"disabled":"";return`
    <div class="pagination" id="pagination">
      <div class="flex flex-col items-center gap-3">
        ${c}
        <div class="flex items-center gap-1">
          <button class="pagination-btn ${d}" data-page="${t-1}" ${t<=1?"disabled":""} aria-label="Previous page">
            ${Vt}
          </button>
          ${o.map(v=>`
            <button class="pagination-btn ${v===t?"active":""}" data-page="${v}">
              ${v}
            </button>
          `).join("")}
          <button class="pagination-btn ${f}" data-page="${t+1}" ${!s&&t>=n?"disabled":""} aria-label="Next page">
            ${Dt}
          </button>
        </div>
      </div>
    </div>
  `}function Wt(e){const t=["#4285F4","#EA4335","#FBBC05","#4285F4","#34A853","#EA4335"],s=["M","i","z","u"],a=Math.min(e-1,6);let r=[s[0]];for(let n=0;n<1+a;n++)r.push("i");r.push("z");for(let n=0;n<1+a;n++)r.push("u");return`
    <div class="flex items-center text-2xl font-semibold tracking-wide select-none">
      ${r.map((n,i)=>`<span style="color: ${t[i%t.length]}">${n}</span>`).join("")}
    </div>
  `}function Yt(e){const t=document.getElementById("pagination");t&&t.querySelectorAll(".pagination-btn").forEach(s=>{s.addEventListener("click",()=>{const a=parseInt(s.dataset.page||"1");isNaN(a)||s.disabled||(e(a),window.scrollTo({top:0,behavior:"smooth"}))})})}const Kt='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>',Zt='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="18" x="3" y="3" rx="2" ry="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"/></svg>',Jt='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m22 8-6 4 6 4V8Z"/><rect width="14" height="12" x="2" y="6" rx="2" ry="2"/></svg>',Qt='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 22h16a2 2 0 0 0 2-2V4a2 2 0 0 0-2-2H8a2 2 0 0 0-2 2v16a2 2 0 0 1-2 2Zm0 0a2 2 0 0 1-2-2v-9c0-1.1.9-2 2-2h2"/><path d="M18 14h-8"/><path d="M15 18h-5"/><path d="M10 6h8v4h-8V6Z"/></svg>',Xt='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M10 2v7.527a2 2 0 0 1-.211.896L4.72 20.55a1 1 0 0 0 .9 1.45h12.76a1 1 0 0 0 .9-1.45l-5.069-10.127A2 2 0 0 1 14 9.527V2"/><path d="M8.5 2h7"/><path d="M7 16h10"/></svg>',es='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/><circle cx="12" cy="10" r="3"/></svg>',ts='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17 2v4M7 2v4M12 2v4M3 10h18M5 4h14a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2z"/><path d="M8 14h.01M12 14h.01M16 14h.01M8 18h.01M12 18h.01M16 18h.01"/></svg>';function B(e){const{query:t,active:s}=e,a=encodeURIComponent(t);return`
    <div class="search-tabs" id="search-tabs">
      ${[{id:"all",label:"All",icon:Kt,href:`/search?q=${a}`},{id:"images",label:"Images",icon:Zt,href:`/images?q=${a}`},{id:"videos",label:"Videos",icon:Jt,href:`/videos?q=${a}`},{id:"news",label:"News",icon:Qt,href:`/news?q=${a}`},{id:"maps",label:"Maps",icon:es,href:`/maps?q=${a}`},{id:"social",label:"Social",icon:ts,href:`/social?q=${a}`},{id:"science",label:"Science",icon:Xt,href:`/science?q=${a}`}].map(n=>`
        <a class="search-tab ${n.id===s?"active":""}" href="${n.href}" data-link data-tab="${n.id}">
          ${n.icon}
          <span>${n.label}</span>
        </a>
      `).join("")}
    </div>
  `}const ss='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',as='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m6 9 6 6 6-6"/></svg>',ke=[{value:"",label:"Any time"},{value:"day",label:"Past 24 hours"},{value:"week",label:"Past week"},{value:"month",label:"Past month"},{value:"year",label:"Past year"}];function ns(e,t){var r;const s=((r=ke.find(n=>n.value===t))==null?void 0:r.label)||"Any time",a=t!=="";return`
    <div class="min-h-screen flex flex-col">
      <!-- Header -->
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="search-header-row">
          <a href="/" data-link class="search-logo">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="search-header-box">
            ${S({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors" aria-label="Settings">
            ${ss}
          </a>
        </div>
        <div class="search-tabs-row">
          <div class="flex items-center gap-2">
            ${B({query:e,active:"all"})}
            <div class="time-filter ml-2" id="time-filter-wrapper">
              <button class="time-filter-btn ${a?"active-filter":""}" id="time-filter-btn" type="button">
                <span id="time-filter-label">${U(s)}</span>
                ${as}
              </button>
              <div class="time-filter-dropdown hidden" id="time-filter-dropdown">
                ${ke.map(n=>`
                  <button class="time-filter-option ${n.value===t?"active":""}" data-time-range="${n.value}">
                    ${U(n.label)}
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
  `}function rs(e,t,s){const a=parseInt(s.page||"1"),r=s.time_range||"",n=z.get().settings;L(i=>{e.navigate(`/search?q=${encodeURIComponent(i)}`)}),is(e,t),t&&M(t),ls(e,t,a,r,n.results_per_page)}function is(e,t,s){const a=document.getElementById("time-filter-btn"),r=document.getElementById("time-filter-dropdown");!a||!r||(a.addEventListener("click",n=>{n.stopPropagation(),r.classList.toggle("hidden")}),r.querySelectorAll(".time-filter-option").forEach(n=>{n.addEventListener("click",()=>{const i=n.dataset.timeRange||"";r.classList.add("hidden");let l=`/search?q=${encodeURIComponent(t)}`;i&&(l+=`&time_range=${i}`),e.navigate(l)})}),document.addEventListener("click",n=>{!r.contains(n.target)&&n.target!==a&&r.classList.add("hidden")}))}async function ls(e,t,s,a,r){const n=document.getElementById("search-content");if(!(!n||!t))try{const i=await x.search(t,{page:s,per_page:r,time_range:a||void 0});if(i.redirect){window.location.href=i.redirect;return}os(n,e,i,t,s,a)}catch(i){n.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to load search results. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${U(String(i))}</p>
      </div>
    `}}function os(e,t,s,a,r,n){const i=s.corrected_query?`<p class="text-sm text-secondary mb-4">
        Showing results for <a href="/search?q=${encodeURIComponent(s.corrected_query)}" data-link class="text-link font-medium">${U(s.corrected_query)}</a>.
        Search instead for <a href="/search?q=${encodeURIComponent(a)}&exact=1" data-link class="text-link">${U(a)}</a>.
      </p>`:"",l=`
    <div class="text-xs text-tertiary mb-4">
      About ${cs(s.total_results)} results (${(s.search_time_ms/1e3).toFixed(2)} seconds)
    </div>
  `,o=s.instant_answer?At(s.instant_answer):"",c=s.results.length>0?s.results.map((y,$)=>bt(y,$)).join(""):`<div class="py-8 text-secondary">No results found for "<strong>${U(a)}</strong>"</div>`,d=s.related_searches&&s.related_searches.length>0?`
      <div class="mt-8 mb-4">
        <h3 class="text-lg font-medium text-primary mb-3">Related searches</h3>
        <div class="grid grid-cols-2 gap-2 max-w-[600px]">
          ${s.related_searches.map(y=>`
            <a href="/search?q=${encodeURIComponent(y)}" data-link class="flex items-center gap-2 p-3 rounded-lg bg-surface hover:bg-surface-hover text-sm text-primary transition-colors">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#9aa0a6" stroke-width="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
              ${U(y)}
            </a>
          `).join("")}
        </div>
      </div>
    `:"",f=Gt({currentPage:r,hasMore:s.has_more,totalResults:s.total_results,perPage:s.per_page}),v=s.knowledge_panel?zt(s.knowledge_panel):"";e.innerHTML=`
    <div class="search-results-layout">
      <div class="search-results-main">
        ${i}
        ${l}
        ${o}
        ${c}
        ${d}
        ${f}
      </div>
      ${v?`<aside class="search-results-sidebar">${v}</aside>`:""}
    </div>
  `,$t(),Yt(y=>{let $=`/search?q=${encodeURIComponent(a)}&page=${y}`;n&&($+=`&time_range=${n}`),t.navigate($)})}function cs(e){return e.toLocaleString("en-US")}function U(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}const ds='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',us='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14.5 4h-5L7 7H4a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2h-3l-2.5-3z"/><circle cx="12" cy="13" r="3"/></svg>',Ce='<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>',Ie='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" x2="21" y1="14" y2="3"/></svg>',ps='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3"/></svg>',qe='<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>',hs='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6"/></svg>',gs='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 18 15 12 9 6"/></svg>';let X="",E={},ee=1,G=!1,O=!0,_=[],K=!1,me=[],ae=null;function ms(e){return`
    <div class="min-h-screen flex flex-col bg-white">
      <!-- Header -->
      <header class="sticky top-0 bg-white z-20 shadow-sm">
        <div class="flex items-center gap-4 px-4 py-2">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[600px] flex items-center gap-2">
            ${S({size:"sm",initialValue:e})}
            <button id="reverse-search-btn" class="flex-shrink-0 p-2 text-tertiary hover:text-primary hover:bg-surface-hover rounded-full transition-colors" title="Search by image">
              ${us}
            </button>
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors" aria-label="Settings">
            ${ds}
          </a>
        </div>
        <div class="pl-[56px] flex items-center gap-1">
          ${B({query:e,active:"images"})}
          <button id="tools-btn" class="tools-btn ml-4">
            ${ps}
            <span>Tools</span>
            ${qe}
          </button>
        </div>
        <!-- Filter toolbar (hidden by default) -->
        <div id="filter-toolbar" class="filter-toolbar hidden">
          ${vs()}
        </div>
      </header>

      <!-- Related searches bar -->
      <div id="related-searches" class="related-searches-bar hidden">
        <div class="related-searches-scroll">
          <button class="related-scroll-btn related-scroll-left hidden">${hs}</button>
          <div class="related-searches-list"></div>
          <button class="related-scroll-btn related-scroll-right hidden">${gs}</button>
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
            <button id="preview-close" class="preview-close-btn" aria-label="Close">${Ce}</button>
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
            <button id="reverse-modal-close" class="modal-close">${Ce}</button>
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
  `}function vs(){return`
    <div class="filter-chips">
      ${[{id:"size",label:"Size",options:["any","large","medium","small","icon"]},{id:"color",label:"Color",options:["any","color","gray","transparent","red","orange","yellow","green","teal","blue","purple","pink","white","black","brown"]},{id:"type",label:"Type",options:["any","photo","clipart","lineart","animated","face"]},{id:"aspect",label:"Aspect",options:["any","tall","square","wide","panoramic"]},{id:"time",label:"Time",options:["any","day","week","month","year"]},{id:"rights",label:"Usage rights",options:["any","creative_commons","commercial"]}].map(t=>`
        <div class="filter-chip-wrapper">
          <button class="filter-chip" data-filter="${t.id}" data-value="any">
            <span class="filter-chip-label">${t.label}</span>
            ${qe}
          </button>
          <div class="filter-dropdown hidden" data-dropdown="${t.id}">
            ${t.options.map(s=>`
              <button class="filter-option${s==="any"?" active":""}" data-value="${s}">
                ${ie(t.id,s)}
              </button>
            `).join("")}
          </div>
        </div>
      `).join("")}
      <button id="clear-filters" class="clear-filters-btn hidden">Clear</button>
    </div>
  `}function ie(e,t){return t==="any"?`Any ${e}`:t.charAt(0).toUpperCase()+t.slice(1).replace("_"," ")}function fs(e,t,s){if(X=t,E={},ee=1,_=[],O=!0,K=!1,me=[],L(a=>{e.navigate(`/images?q=${encodeURIComponent(a)}`)}),t&&M(t),xs(),ys(),$s(),Is(),ws(e),(s==null?void 0:s.reverse)==="1"){const a=document.getElementById("reverse-modal");a&&a.classList.remove("hidden")}ve(t,E)}function xs(){const e=document.getElementById("tools-btn"),t=document.getElementById("filter-toolbar");!e||!t||e.addEventListener("click",()=>{K=!K,t.classList.toggle("hidden",!K),e.classList.toggle("active",K)})}function ys(e){const t=document.getElementById("filter-toolbar");if(!t)return;t.querySelectorAll(".filter-chip").forEach(a=>{a.addEventListener("click",r=>{r.stopPropagation();const n=a.dataset.filter,i=t.querySelector(`[data-dropdown="${n}"]`);t.querySelectorAll(".filter-dropdown").forEach(l=>{l!==i&&l.classList.add("hidden")}),i==null||i.classList.toggle("hidden")})}),t.querySelectorAll(".filter-option").forEach(a=>{a.addEventListener("click",()=>{const r=a.closest(".filter-dropdown"),n=r==null?void 0:r.dataset.dropdown,i=a.dataset.value,l=t.querySelector(`[data-filter="${n}"]`);!n||!i||!l||(r.querySelectorAll(".filter-option").forEach(o=>o.classList.remove("active")),a.classList.add("active"),i==="any"?(delete E[n],l.classList.remove("has-value"),l.querySelector(".filter-chip-label").textContent=ie(n,"any").replace("Any ","")):(E[n]=i,l.classList.add("has-value"),l.querySelector(".filter-chip-label").textContent=ie(n,i)),r.classList.add("hidden"),Se(),ee=1,_=[],O=!0,ve(X,E))})}),document.addEventListener("click",()=>{t.querySelectorAll(".filter-dropdown").forEach(a=>a.classList.add("hidden"))});const s=document.getElementById("clear-filters");s&&s.addEventListener("click",()=>{E={},ee=1,_=[],O=!0,t.querySelectorAll(".filter-chip").forEach(a=>{const r=a.dataset.filter;a.classList.remove("has-value"),a.querySelector(".filter-chip-label").textContent=ie(r,"any").replace("Any ","")}),t.querySelectorAll(".filter-dropdown").forEach(a=>{a.querySelectorAll(".filter-option").forEach((r,n)=>{r.classList.toggle("active",n===0)})}),Se(),ve(X,E)})}function Se(){const e=document.getElementById("clear-filters");e&&e.classList.toggle("hidden",Object.keys(E).length===0)}function ws(e){const t=document.getElementById("related-searches");if(!t)return;t.addEventListener("click",n=>{const i=n.target.closest(".related-chip");if(i){const l=i.getAttribute("data-query");l&&e.navigate(`/images?q=${encodeURIComponent(l)}`)}});const s=t.querySelector(".related-scroll-left"),a=t.querySelector(".related-scroll-right"),r=t.querySelector(".related-searches-list");s&&a&&r&&(s.addEventListener("click",()=>{r.scrollBy({left:-200,behavior:"smooth"})}),a.addEventListener("click",()=>{r.scrollBy({left:200,behavior:"smooth"})}),r.addEventListener("scroll",()=>{Fe()}))}function Fe(){const e=document.getElementById("related-searches");if(!e)return;const t=e.querySelector(".related-searches-list"),s=e.querySelector(".related-scroll-left"),a=e.querySelector(".related-scroll-right");!t||!s||!a||(s.classList.toggle("hidden",t.scrollLeft<=0),a.classList.toggle("hidden",t.scrollLeft>=t.scrollWidth-t.clientWidth-10))}function bs(e){const t=document.getElementById("related-searches");if(!t)return;if(!e||e.length===0){t.classList.add("hidden");return}const s=t.querySelector(".related-searches-list");s&&(s.innerHTML=e.map(a=>`
    <button class="related-chip" data-query="${R(a)}">
      <span class="related-chip-text">${V(a)}</span>
    </button>
  `).join(""),t.classList.remove("hidden"),setTimeout(Fe,50))}function $s(e){const t=document.getElementById("reverse-search-btn"),s=document.getElementById("reverse-modal"),a=document.getElementById("reverse-modal-close"),r=document.getElementById("drop-zone"),n=document.getElementById("image-upload"),i=document.getElementById("image-url-input"),l=document.getElementById("url-search-btn");!t||!s||(t.addEventListener("click",()=>s.classList.remove("hidden")),a==null||a.addEventListener("click",()=>s.classList.add("hidden")),s.addEventListener("click",o=>{o.target===s&&s.classList.add("hidden")}),r&&(r.addEventListener("dragover",o=>{o.preventDefault(),r.classList.add("drag-over")}),r.addEventListener("dragleave",()=>r.classList.remove("drag-over")),r.addEventListener("drop",o=>{var d;o.preventDefault(),r.classList.remove("drag-over");const c=(d=o.dataTransfer)==null?void 0:d.files;c&&c[0]&&(Le(c[0]),s.classList.add("hidden"))})),n&&n.addEventListener("change",()=>{n.files&&n.files[0]&&(Le(n.files[0]),s.classList.add("hidden"))}),l&&i&&(l.addEventListener("click",()=>{const o=i.value.trim();o&&(Ee(o),s.classList.add("hidden"))}),i.addEventListener("keydown",o=>{if(o.key==="Enter"){const c=i.value.trim();c&&(Ee(c),s.classList.add("hidden"))}})))}async function Le(e,t){const s=document.getElementById("images-content");if(s){if(!e.type.startsWith("image/")){alert("Please select an image file");return}if(e.size>10*1024*1024){alert("Image must be smaller than 10MB");return}s.innerHTML=`
    <div class="flex flex-col items-center justify-center py-16">
      <div class="spinner"></div>
      <span class="mt-3 text-secondary">Uploading and searching...</span>
      <div class="w-48 mt-4 h-1 bg-border rounded-full overflow-hidden">
        <div id="upload-progress" class="h-full bg-blue transition-all duration-300" style="width: 0%"></div>
      </div>
    </div>
  `;try{const a=await ks(e),r=document.getElementById("upload-progress");r&&(r.style.width="50%");const n=await x.reverseImageSearchByUpload(a);r&&(r.style.width="100%"),Cs(s,a,n)}catch(a){s.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to search by image. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${V(String(a))}</p>
      </div>
    `}}}function ks(e){return new Promise((t,s)=>{const a=new FileReader;a.onload=()=>{const n=a.result.split(",")[1];t(n)},a.onerror=s,a.readAsDataURL(e)})}function Cs(e,t,s){const r=!t.startsWith("http")?`data:image/jpeg;base64,${t}`:t;e.innerHTML=`
    <div class="reverse-results">
      <div class="query-image-section">
        <h3>Search image</h3>
        <img src="${r}" alt="Query image" class="query-image" />
      </div>
      ${s.similar_images.length>0?`
        <div class="similar-images-section">
          <h3>Similar images (${s.similar_images.length})</h3>
          <div class="image-grid">
            ${s.similar_images.map((n,i)=>ce(n,i)).join("")}
          </div>
        </div>
      `:'<div class="py-8 text-secondary">No similar images found.</div>'}
    </div>
  `,e.querySelectorAll(".image-card").forEach(n=>{n.addEventListener("click",()=>{const i=parseInt(n.dataset.imageIndex||"0",10);oe(s.similar_images[i])})})}async function Ee(e,t){const s=document.getElementById("images-content");if(s){s.innerHTML=`
    <div class="flex items-center justify-center py-16">
      <div class="spinner"></div>
      <span class="ml-3 text-secondary">Searching for similar images...</span>
    </div>
  `;try{const a=await x.reverseImageSearch(e);s.innerHTML=`
      <div class="reverse-results">
        <div class="query-image-section">
          <h3>Search image</h3>
          <img src="${R(e)}" alt="Query image" class="query-image" />
        </div>
        ${a.similar_images.length>0?`
          <div class="similar-images-section">
            <h3>Similar images (${a.similar_images.length})</h3>
            <div class="image-grid">
              ${a.similar_images.map((r,n)=>ce(r,n)).join("")}
            </div>
          </div>
        `:'<div class="py-8 text-secondary">No similar images found.</div>'}
      </div>
    `,s.querySelectorAll(".image-card").forEach(r=>{r.addEventListener("click",()=>{const n=parseInt(r.dataset.imageIndex||"0",10);oe(a.similar_images[n])})})}catch(a){s.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to search by image. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${V(String(a))}</p>
      </div>
    `}}}function Is(){const e=document.getElementById("preview-panel"),t=document.getElementById("preview-close"),s=e==null?void 0:e.querySelector(".preview-overlay");t==null||t.addEventListener("click",he),s==null||s.addEventListener("click",he),document.addEventListener("keydown",a=>{a.key==="Escape"&&he()})}function oe(e){const t=document.getElementById("preview-panel"),s=document.getElementById("preview-image"),a=document.getElementById("preview-details");if(!t||!s||!a)return;s.src=e.url,s.alt=e.title;const r=e.width&&e.height&&e.width>0&&e.height>0;a.innerHTML=`
    <div class="preview-header">
      <img src="${R(e.thumbnail_url||e.url)}" class="preview-thumb" alt="" />
      <div class="preview-header-info">
        <h3 class="preview-title">${V(e.title||"Untitled")}</h3>
        <a href="${R(e.source_url)}" target="_blank" class="preview-domain">${V(e.source_domain)}</a>
      </div>
    </div>
    <div class="preview-meta">
      ${r?`<div class="preview-meta-item"><span class="preview-meta-label">Size</span><span>${e.width} × ${e.height}</span></div>`:""}
      ${e.format?`<div class="preview-meta-item"><span class="preview-meta-label">Type</span><span>${e.format.toUpperCase()}</span></div>`:""}
    </div>
    <div class="preview-actions">
      <a href="${R(e.source_url)}" target="_blank" class="preview-btn preview-btn-primary">
        Visit page ${Ie}
      </a>
      <a href="${R(e.url)}" target="_blank" class="preview-btn">
        View full image ${Ie}
      </a>
    </div>
  `,t.classList.remove("hidden"),document.body.style.overflow="hidden"}function he(){const e=document.getElementById("preview-panel");e&&(e.classList.add("hidden"),document.body.style.overflow="")}function Ss(){ae&&ae.disconnect();const e=document.getElementById("images-content");if(!e)return;const t=document.getElementById("scroll-sentinel");t&&t.remove();const s=document.createElement("div");s.id="scroll-sentinel",s.className="scroll-sentinel",e.appendChild(s),ae=new IntersectionObserver(a=>{a[0].isIntersecting&&!G&&O&&X&&Ls()},{rootMargin:"400px"}),ae.observe(s)}async function Ls(){if(G||!O)return;G=!0,ee++;const e=document.getElementById("scroll-sentinel");e&&(e.innerHTML='<div class="loading-more"><div class="spinner-sm"></div></div>');try{const t=await x.searchImages(X,{...E,page:ee}),s=t.results;O=t.has_more,_=[..._,...s];const a=document.querySelector(".image-grid");if(a&&s.length>0){const r=_.length-s.length,n=s.map((i,l)=>ce(i,r+l)).join("");a.insertAdjacentHTML("beforeend",n),a.querySelectorAll(".image-card:not([data-initialized])").forEach(i=>{i.setAttribute("data-initialized","true"),i.addEventListener("click",()=>{const l=parseInt(i.dataset.imageIndex||"0",10);oe(_[l])})})}e&&(e.innerHTML=O?"":'<div class="no-more-results">No more images</div>')}catch{e&&(e.innerHTML="")}finally{G=!1}}async function ve(e,t){var a;const s=document.getElementById("images-content");if(!(!s||!e)){G=!0,s.innerHTML='<div class="flex items-center justify-center py-16"><div class="spinner"></div></div>';try{const r=await x.searchImages(e,{...t,page:1,per_page:50}),n=r.results;if(O=r.has_more,_=n,me=(a=r.related_searches)!=null&&a.length?r.related_searches:Es(e),bs(me),n.length===0){s.innerHTML=`<div class="py-8 text-secondary">No image results found for "<strong>${V(e)}</strong>"</div>`;return}s.innerHTML=`<div class="image-grid">${n.map((i,l)=>ce(i,l)).join("")}</div>`,s.querySelectorAll(".image-card").forEach(i=>{i.setAttribute("data-initialized","true"),i.addEventListener("click",()=>{const l=parseInt(i.dataset.imageIndex||"0",10);oe(_[l])})}),Ss()}catch(r){s.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to load image results. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${V(String(r))}</p>
      </div>
    `}finally{G=!1}}}function ce(e,t){return`
    <div class="image-card" data-image-index="${t}">
      <div class="image-card-img">
        <img
          src="${R(e.thumbnail_url||e.url)}"
          alt="${R(e.title)}"
          loading="lazy"
          onerror="this.closest('.image-card').style.display='none'"
        />
      </div>
    </div>
  `}function V(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function R(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}function Es(e){const t=e.toLowerCase().trim().split(/\s+/).filter(i=>i.length>1);if(t.length===0)return[];const s=[],a=["wallpaper","hd","4k","aesthetic","cute","beautiful","background","art","photography","design","illustration","vintage","modern","minimalist","colorful","dark","light"],r={cat:["kitten","cats playing","black cat","tabby cat","cat meme"],dog:["puppy","dogs playing","golden retriever","german shepherd","dog meme"],nature:["forest","mountains","ocean","sunset nature","flowers"],food:["dessert","healthy food","breakfast","dinner","food photography"],car:["sports car","luxury car","vintage car","car interior","supercar"],house:["modern house","interior design","living room","bedroom design","architecture"],city:["skyline","night city","urban photography","street photography","downtown"]},n=t.slice(0,2).join(" ");for(const i of a)!e.includes(i)&&s.length<4&&s.push(`${n} ${i}`);for(const[i,l]of Object.entries(r))if(t.some(o=>o.includes(i)||i.includes(o))){for(const o of l)!s.includes(o)&&s.length<8&&s.push(o);break}return t.length>=2&&s.length<8&&s.push(t.reverse().join(" ")),s.length<4&&s.push(`${n} images`,`${n} photos`,`best ${n}`),s.slice(0,8)}const _s='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>';function Ms(e){return`
    <div class="min-h-screen flex flex-col">
      <!-- Header -->
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="flex items-center gap-4 px-4 lg:px-8 py-3">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[692px]">
            ${S({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors" aria-label="Settings">
            ${_s}
          </a>
        </div>
        <div class="px-4 lg:px-8 pl-[170px]">
          ${B({query:e,active:"videos"})}
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
  `}function Bs(e,t){L(s=>{e.navigate(`/videos?q=${encodeURIComponent(s)}`)}),t&&M(t),Ts(t)}async function Ts(e){const t=document.getElementById("videos-content");if(!(!t||!e))try{const s=await x.searchVideos(e),a=s.results;if(a.length===0){t.innerHTML=`
        <div class="py-8 text-secondary">No video results found for "<strong>${D(e)}</strong>"</div>
      `;return}t.innerHTML=`
      <div class="text-xs text-tertiary mb-4">
        About ${s.total_results.toLocaleString()} video results (${(s.search_time_ms/1e3).toFixed(2)} seconds)
      </div>
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
        ${a.map(r=>Hs(r)).join("")}
      </div>
    `}catch(s){t.innerHTML=`
      <div class="py-8">
        <p class="text-red text-sm">Failed to load video results. Please try again.</p>
        <p class="text-tertiary text-xs mt-2">${D(String(s))}</p>
      </div>
    `}}function Hs(e){var n;const t=((n=e.thumbnail)==null?void 0:n.url)||"",s=e.views?As(e.views):"",a=e.published?Ns(e.published):"",r=[e.channel,s,a].filter(Boolean).join(" · ");return`
    <div class="video-card">
      <a href="${ne(e.url)}" target="_blank" rel="noopener" class="block">
        <div class="video-thumb">
          ${t?`<img src="${ne(t)}" alt="${ne(e.title)}" loading="lazy" onerror="this.style.display='none'" />`:`<div class="w-full h-full flex items-center justify-center bg-surface">
                  <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="#dadce0" stroke-width="1.5"><path d="m22 8-6 4 6 4V8Z"/><rect width="14" height="12" x="2" y="6" rx="2" ry="2"/></svg>
                </div>`}
          ${e.duration?`<span class="video-duration">${D(e.duration)}</span>`:""}
        </div>
      </a>
      <div class="video-info">
        <div class="video-title">
          <a href="${ne(e.url)}" target="_blank" rel="noopener">${D(e.title)}</a>
        </div>
        <div class="video-meta">${D(r)}</div>
        ${e.platform?`<div class="text-xs text-light mt-1">${D(e.platform)}</div>`:""}
      </div>
    </div>
  `}function As(e){return e>=1e6?`${(e/1e6).toFixed(1)}M views`:e>=1e3?`${(e/1e3).toFixed(1)}K views`:`${e} views`}function Ns(e){try{const t=new Date(e),a=new Date().getTime()-t.getTime(),r=Math.floor(a/(1e3*60*60*24));return r===0?"Today":r===1?"1 day ago":r<7?`${r} days ago`:r<30?`${Math.floor(r/7)} weeks ago`:r<365?`${Math.floor(r/30)} months ago`:`${Math.floor(r/365)} years ago`}catch{return e}}function D(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function ne(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const Rs='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>';function Os(e){return`
    <div class="min-h-screen flex flex-col">
      <!-- Header -->
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="flex items-center gap-4 px-4 lg:px-8 py-3">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[692px]">
            ${S({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors" aria-label="Settings">
            ${Rs}
          </a>
        </div>
        <div class="px-4 lg:px-8 pl-[170px]">
          ${B({query:e,active:"news"})}
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
  `}function js(e,t){L(s=>{e.navigate(`/news?q=${encodeURIComponent(s)}`)}),t&&M(t),qs(t)}async function qs(e){const t=document.getElementById("news-content");if(!(!t||!e))try{const s=await x.searchNews(e),a=s.results;if(a.length===0){t.innerHTML=`
        <div class="max-w-[900px]">
          <div class="py-8 text-secondary">No news results found for "<strong>${Z(e)}</strong>"</div>
        </div>
      `;return}t.innerHTML=`
      <div class="max-w-[900px]">
        <div class="text-xs text-tertiary mb-6">
          About ${s.total_results.toLocaleString()} news results (${(s.search_time_ms/1e3).toFixed(2)} seconds)
        </div>
        <div class="space-y-4">
          ${a.map(r=>Fs(r)).join("")}
        </div>
      </div>
    `}catch(s){t.innerHTML=`
      <div class="max-w-[900px]">
        <div class="py-8">
          <p class="text-red text-sm">Failed to load news results. Please try again.</p>
          <p class="text-tertiary text-xs mt-2">${Z(String(s))}</p>
        </div>
      </div>
    `}}function Fs(e){var a;const t=((a=e.thumbnail)==null?void 0:a.url)||"",s=e.published_date?Ps(e.published_date):"";return`
    <div class="news-card">
      <div class="flex-1 min-w-0">
        <div class="news-source">
          ${Z(e.source||e.domain)}
          ${s?` · ${Z(s)}`:""}
        </div>
        <div class="news-title">
          <a href="${_e(e.url)}" target="_blank" rel="noopener">${Z(e.title)}</a>
        </div>
        <div class="news-snippet">${e.snippet||""}</div>
      </div>
      ${t?`<img class="news-image" src="${_e(t)}" alt="" loading="lazy" onerror="this.style.display='none'" />`:""}
    </div>
  `}function Ps(e){try{const t=new Date(e),a=new Date().getTime()-t.getTime(),r=Math.floor(a/(1e3*60*60)),n=Math.floor(a/(1e3*60*60*24));return r<1?"Just now":r<24?`${r}h ago`:n===1?"1 day ago":n<7?`${n} days ago`:n<30?`${Math.floor(n/7)} weeks ago`:t.toLocaleDateString("en-US",{month:"short",day:"numeric",year:"numeric"})}catch{return e}}function Z(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function _e(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const Us='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',Pe='<svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M10 20v-6h4v6h5v-8h3L12 3 2 12h3v8z"/></svg>',zs='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>',Vs='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z"/></svg>',Me='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/><circle cx="12" cy="10" r="3"/></svg>',Ds='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>',Gs='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="7" width="20" height="14" rx="2" ry="2"/><path d="M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16"/></svg>',Ws='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="4" y="4" width="16" height="16" rx="2" ry="2"/><rect x="9" y="9" width="6" height="6"/><line x1="9" y1="1" x2="9" y2="4"/><line x1="15" y1="1" x2="15" y2="4"/><line x1="9" y1="20" x2="9" y2="23"/><line x1="15" y1="20" x2="15" y2="23"/><line x1="20" y1="9" x2="23" y2="9"/><line x1="20" y1="14" x2="23" y2="14"/><line x1="1" y1="9" x2="4" y2="9"/><line x1="1" y1="14" x2="4" y2="14"/></svg>',Ys='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="2" width="20" height="20" rx="2.18" ry="2.18"/><line x1="7" y1="2" x2="7" y2="22"/><line x1="17" y1="2" x2="17" y2="22"/><line x1="2" y1="12" x2="22" y2="12"/><line x1="2" y1="7" x2="7" y2="7"/><line x1="2" y1="17" x2="7" y2="17"/><line x1="17" y1="17" x2="22" y2="17"/><line x1="17" y1="7" x2="22" y2="7"/></svg>',Ks='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>',Zs='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2v6.5a.5.5 0 0 0 .5.5h3a.5.5 0 0 1 .5.5v1a.5.5 0 0 1-.5.5H14.5a.5.5 0 0 0-.5.5V22H10V11.5a.5.5 0 0 0-.5-.5H6.5a.5.5 0 0 1-.5-.5v-1a.5.5 0 0 1 .5-.5h3a.5.5 0 0 0 .5-.5V2"/></svg>',Js='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"/></svg>',Qs='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>',Ue=[{id:"top",label:"Top Stories",icon:Qs},{id:"world",label:"World",icon:Ds},{id:"nation",label:"U.S.",icon:Pe},{id:"business",label:"Business",icon:Gs},{id:"technology",label:"Technology",icon:Ws},{id:"entertainment",label:"Entertainment",icon:Ys},{id:"sports",label:"Sports",icon:Ks},{id:"science",label:"Science",icon:Zs},{id:"health",label:"Health",icon:Js}];function Xs(){const e=new Date().toLocaleDateString("en-US",{weekday:"long",month:"long",day:"numeric"});return`
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
            ${Pe}
            <span>Home</span>
          </a>
          <a href="/news-home?section=for-you" data-link class="news-nav-item">
            ${zs}
            <span>For you</span>
          </a>
          <a href="/news-home?section=following" data-link class="news-nav-item">
            ${Vs}
            <span>Following</span>
          </a>
        </div>

        <div class="news-nav-divider"></div>

        <div class="news-nav-section">
          ${Ue.map(t=>`
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
            ${S({size:"sm"})}
          </div>

          <div class="news-header-right">
            <button class="news-icon-btn" id="location-btn" title="Change location">
              ${Me}
            </button>
            <a href="/settings" data-link class="news-icon-btn" title="Settings">
              ${Us}
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
                ${Me}
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
  `}function ea(e,t){L(r=>{e.navigate(`/news?q=${encodeURIComponent(r)}`)});const s=document.getElementById("menu-toggle"),a=document.querySelector(".news-sidebar");s&&a&&s.addEventListener("click",()=>{a.classList.toggle("open")}),ta()}async function ta(e){const t=document.getElementById("news-loading"),s=document.getElementById("top-stories-section"),a=document.getElementById("for-you-section"),r=document.getElementById("local-section");try{const n=await x.newsHome();if(t&&(t.style.display="none"),s&&n.topStories.length>0){s.style.display="block";const l=document.getElementById("top-stories-grid");l&&(l.innerHTML=sa(n.topStories))}if(a&&n.forYou.length>0){a.style.display="block";const l=document.getElementById("for-you-list");l&&(l.innerHTML=n.forYou.slice(0,10).map(o=>ia(o)).join(""))}if(r&&n.localNews.length>0){r.style.display="block";const l=document.getElementById("local-news-scroll");l&&(l.innerHTML=n.localNews.map(o=>ze(o)).join(""))}const i=document.getElementById("category-sections");if(i&&n.categories){const l=Object.entries(n.categories).filter(([o,c])=>c&&c.length>0).map(([o,c])=>la(o,c)).join("");i.innerHTML=l}oa()}catch(n){t&&(t.innerHTML=`
        <div class="news-error">
          <p>Failed to load news. Please try again.</p>
          <button class="news-btn" onclick="location.reload()">Retry</button>
        </div>
      `),console.error("Failed to load news:",n)}}function sa(e){if(e.length===0)return"";const t=e[0],s=e.slice(1,3),a=e.slice(3,9);return`
    <div class="news-featured-row">
      ${aa(t)}
      <div class="news-secondary-col">
        ${s.map(r=>na(r)).join("")}
      </div>
    </div>
    <div class="news-grid-row">
      ${a.map(r=>ra(r)).join("")}
    </div>
  `}function aa(e){const t=te(e.publishedAt);return`
    <article class="news-card news-card-featured">
      ${e.imageUrl?`<img class="news-card-image" src="${C(e.imageUrl)}" alt="" loading="lazy" onerror="this.style.display='none'" />`:""}
      <div class="news-card-content">
        <div class="news-card-meta">
          <img class="news-source-icon" src="${C(e.sourceIcon||"")}" alt="" onerror="this.style.display='none'" />
          <span class="news-source-name">${k(e.source)}</span>
          <span class="news-time">${t}</span>
        </div>
        <h3 class="news-card-title">
          <a href="${C(e.url)}" target="_blank" rel="noopener" onclick="trackArticleClick('${e.id}')">${k(e.title)}</a>
        </h3>
        <p class="news-card-snippet">${k(e.snippet)}</p>
        ${e.clusterId?`<a href="/news-home?story=${e.clusterId}" data-link class="news-full-coverage">Full coverage</a>`:""}
      </div>
    </article>
  `}function na(e){const t=te(e.publishedAt);return`
    <article class="news-card news-card-medium">
      <div class="news-card-content">
        <div class="news-card-meta">
          <img class="news-source-icon" src="${C(e.sourceIcon||"")}" alt="" onerror="this.style.display='none'" />
          <span class="news-source-name">${k(e.source)}</span>
          <span class="news-time">${t}</span>
        </div>
        <h3 class="news-card-title">
          <a href="${C(e.url)}" target="_blank" rel="noopener">${k(e.title)}</a>
        </h3>
      </div>
      ${e.imageUrl?`<img class="news-card-thumb" src="${C(e.imageUrl)}" alt="" loading="lazy" onerror="this.style.display='none'" />`:""}
    </article>
  `}function ra(e){const t=te(e.publishedAt);return`
    <article class="news-card news-card-small">
      <div class="news-card-content">
        <div class="news-card-meta">
          <span class="news-source-name">${k(e.source)}</span>
          <span class="news-time">${t}</span>
        </div>
        <h3 class="news-card-title">
          <a href="${C(e.url)}" target="_blank" rel="noopener">${k(e.title)}</a>
        </h3>
      </div>
    </article>
  `}function ze(e){const t=te(e.publishedAt);return`
    <article class="news-card news-card-compact">
      ${e.imageUrl?`<img class="news-card-thumb-sm" src="${C(e.imageUrl)}" alt="" loading="lazy" onerror="this.style.display='none'" />`:'<div class="news-card-thumb-placeholder"></div>'}
      <div class="news-card-content">
        <span class="news-source-name">${k(e.source)}</span>
        <h4 class="news-card-title-sm">
          <a href="${C(e.url)}" target="_blank" rel="noopener">${k(e.title)}</a>
        </h4>
        <span class="news-time">${t}</span>
      </div>
    </article>
  `}function ia(e){const t=te(e.publishedAt);return`
    <article class="news-list-item">
      <div class="news-list-content">
        <div class="news-card-meta">
          <img class="news-source-icon" src="${C(e.sourceIcon||"")}" alt="" onerror="this.style.display='none'" />
          <span class="news-source-name">${k(e.source)}</span>
          <span class="news-time">${t}</span>
        </div>
        <h3 class="news-list-title">
          <a href="${C(e.url)}" target="_blank" rel="noopener">${k(e.title)}</a>
        </h3>
        <p class="news-list-snippet">${k(e.snippet)}</p>
      </div>
      ${e.imageUrl?`<img class="news-list-thumb" src="${C(e.imageUrl)}" alt="" loading="lazy" onerror="this.style.display='none'" />`:""}
    </article>
  `}function la(e,t){const s=Ue.find(a=>a.id===e);return s?`
    <section class="news-section">
      <div class="news-section-header">
        <h2 class="news-section-title">
          ${s.icon}
          <span>${s.label}</span>
        </h2>
        <a href="/news-home?category=${e}" data-link class="news-text-btn">More ${s.label.toLowerCase()}</a>
      </div>
      <div class="news-horizontal-scroll">
        ${t.slice(0,5).map(a=>ze(a)).join("")}
      </div>
    </section>
  `:""}function te(e){try{const t=new Date(e),a=new Date().getTime()-t.getTime(),r=Math.floor(a/(1e3*60*60)),n=Math.floor(a/(1e3*60*60*24));return r<1?"Just now":r<24?`${r}h ago`:n===1?"1 day ago":n<7?`${n} days ago`:t.toLocaleDateString("en-US",{month:"short",day:"numeric"})}catch{return""}}function oa(){document.querySelectorAll(".news-card a, .news-list-item a").forEach(e=>{e.addEventListener("click",function(){const t=this.closest(".news-card, .news-list-item");if(t){const s=t.getAttribute("data-article-id");s&&x.recordNewsRead({id:s,url:this.href,title:this.textContent||"",snippet:"",source:"",sourceUrl:"",publishedAt:"",category:"top",engines:[],score:1}).catch(()=>{})}})})}function k(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function C(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const ca='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',da='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><path d="M12 18v-6"/><path d="m9 15 3 3 3-3"/></svg>',ua='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 21c3 0 7-1 7-8V5c0-1.25-.756-2.017-2-2H4c-1.25 0-2 .75-2 1.972V11c0 1.25.75 2 2 2 1 0 1 0 1 1v1c0 1-1 2-2 2s-1 .008-1 1.031V21"/><path d="M15 21c3 0 7-1 7-8V5c0-1.25-.757-2.017-2-2h-4c-1.25 0-2 .75-2 1.972V11c0 1.25.75 2 2 2h.75c0 2.25.25 4-2.75 4v3"/></svg>';function pa(e){return`
    <div class="min-h-screen flex flex-col">
      <!-- Header -->
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="flex items-center gap-4 px-4 lg:px-8 py-3">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[692px]">
            ${S({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors" aria-label="Settings">
            ${ca}
          </a>
        </div>
        <div class="px-4 lg:px-8 pl-[170px]">
          ${B({query:e,active:"science"})}
        </div>
      </header>

      <!-- Content -->
      <main class="flex-1">
        <div id="science-content" class="px-4 lg:px-8 py-6">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `}function ha(e,t){L(s=>{e.navigate(`/science?q=${encodeURIComponent(s)}`)}),t&&M(t),ga(t)}async function ga(e){const t=document.getElementById("science-content");if(!(!t||!e))try{const s=await x.searchScience(e),a=s.results;if(a.length===0){t.innerHTML=`
        <div class="max-w-[900px]">
          <div class="py-8 text-secondary">No academic results found for "<strong>${F(e)}</strong>"</div>
        </div>
      `;return}t.innerHTML=`
      <div class="max-w-[900px]">
        <div class="text-xs text-tertiary mb-4">
          About ${s.total_results.toLocaleString()} results (${(s.search_time_ms/1e3).toFixed(2)} seconds)
        </div>
        <div class="space-y-6">
          ${a.map(ma).join("")}
        </div>
      </div>
    `}catch(s){t.innerHTML=`
      <div class="max-w-[900px]">
        <div class="py-8">
          <p class="text-red text-sm">Failed to load academic results. Please try again.</p>
          <p class="text-tertiary text-xs mt-2">${F(String(s))}</p>
        </div>
      </div>
    `}}function ma(e){var o,c,d,f,v,y;const t=e,s=((o=t.metadata)==null?void 0:o.authors)||"",a=((c=t.metadata)==null?void 0:c.year)||"",r=(d=t.metadata)==null?void 0:d.citations,n=((f=t.metadata)==null?void 0:f.doi)||"",i=((v=t.metadata)==null?void 0:v.pdf_url)||"",l=((y=t.metadata)==null?void 0:y.source)||va(e.url);return`
    <article class="paper-card bg-white border border-border rounded-xl p-5 hover:shadow-md transition-shadow">
      <div class="flex items-start gap-3 mb-2">
        <span class="text-xs px-2 py-0.5 bg-blue/10 text-blue rounded-full font-medium">${F(l)}</span>
        ${a?`<span class="text-xs text-tertiary">${F(a)}</span>`:""}
      </div>
      <h3 class="text-lg font-medium text-primary mb-2">
        <a href="${ge(e.url)}" target="_blank" rel="noopener" class="hover:text-blue hover:underline">${F(e.title)}</a>
      </h3>
      ${s?`<p class="text-sm text-secondary mb-2">${F(s)}</p>`:""}
      <p class="text-sm text-snippet line-clamp-3 mb-3">${e.snippet||""}</p>
      <div class="flex items-center gap-4 text-xs">
        ${r!==void 0?`<span class="flex items-center gap-1 text-tertiary">${ua} ${r} citations</span>`:""}
        ${n?`<a href="https://doi.org/${ge(n)}" target="_blank" rel="noopener" class="text-tertiary hover:text-blue">DOI: ${F(n)}</a>`:""}
        ${i?`<a href="${ge(i)}" target="_blank" rel="noopener" class="flex items-center gap-1 text-blue hover:underline">${da} PDF</a>`:""}
      </div>
    </article>
  `}function va(e){try{return new URL(e).hostname.replace("www.","")}catch{return""}}function F(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function ge(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const fa='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',xa='<svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>',ya='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="18" r="3"/><circle cx="6" cy="6" r="3"/><circle cx="18" cy="6" r="3"/><path d="M18 9v1a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2V9"/><path d="M12 12v3"/></svg>',wa='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" x2="12" y1="15" y2="3"/></svg>';function ba(e){return`
    <div class="min-h-screen flex flex-col">
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="flex items-center gap-4 px-4 lg:px-8 py-3">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[692px]">
            ${S({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors">
            ${fa}
          </a>
        </div>
        <div class="px-4 lg:px-8">
          ${B({query:e,active:"code"})}
        </div>
      </header>
      <main class="flex-1">
        <div id="code-content" class="px-4 lg:px-8 py-6">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `}function $a(e,t){L(s=>e.navigate(`/code?q=${encodeURIComponent(s)}`)),t&&M(t),ka(t)}async function ka(e){const t=document.getElementById("code-content");if(!(!t||!e))try{const s=await x.searchCode(e),a=s.results;if(a.length===0){t.innerHTML=`<div class="py-8 text-secondary">No code results found for "${J(e)}"</div>`;return}t.innerHTML=`
      <div class="text-xs text-tertiary mb-4">
        About ${s.total_results.toLocaleString()} results (${(s.search_time_ms/1e3).toFixed(2)} seconds)
      </div>
      <div class="max-w-[900px] space-y-4">
        ${a.map(Ca).join("")}
      </div>
    `}catch(s){t.innerHTML=`<div class="py-8 text-red text-sm">Failed to load results. ${J(String(s))}</div>`}}function Ca(e){var o,c,d,f,v,y,$;const t=((o=e.metadata)==null?void 0:o.source)||Sa(e.url),s=(c=e.metadata)==null?void 0:c.stars,a=(d=e.metadata)==null?void 0:d.forks,r=(f=e.metadata)==null?void 0:f.downloads,n=((v=e.metadata)==null?void 0:v.language)||"",i=(y=e.metadata)==null?void 0:y.votes,l=($=e.metadata)==null?void 0:$.answers;return`
    <article class="code-card bg-white border border-border rounded-xl p-4 hover:shadow-md transition-shadow">
      <div class="flex items-start gap-3 mb-2">
        <span class="text-xs px-2 py-0.5 rounded-full font-medium ${Ia(t)}">${J(t)}</span>
        ${n?`<span class="text-xs px-2 py-0.5 bg-surface text-secondary rounded-full">${J(n)}</span>`:""}
      </div>
      <h3 class="text-base font-medium text-primary mb-1">
        <a href="${La(e.url)}" target="_blank" rel="noopener" class="hover:text-blue hover:underline">${J(e.title)}</a>
      </h3>
      <p class="text-sm text-snippet line-clamp-2 mb-3">${e.content||""}</p>
      <div class="flex items-center gap-4 text-xs text-tertiary">
        ${s!==void 0?`<span class="flex items-center gap-1">${xa} <span class="text-yellow-500">${re(s)}</span></span>`:""}
        ${a!==void 0?`<span class="flex items-center gap-1">${ya} ${re(a)}</span>`:""}
        ${r!==void 0?`<span class="flex items-center gap-1">${wa} ${re(r)}</span>`:""}
        ${i!==void 0?`<span class="flex items-center gap-1">▲ ${re(i)}</span>`:""}
        ${l!==void 0?`<span class="flex items-center gap-1">${l} answers</span>`:""}
      </div>
    </article>
  `}function Ia(e){return e.includes("github")?"bg-gray-900 text-white":e.includes("gitlab")?"bg-orange-500 text-white":e.includes("stackoverflow")?"bg-orange-400 text-white":e.includes("npm")?"bg-red-500 text-white":e.includes("pypi")?"bg-blue-500 text-white":e.includes("crates")?"bg-orange-600 text-white":"bg-surface text-secondary"}function re(e){return e>=1e6?(e/1e6).toFixed(1)+"M":e>=1e3?(e/1e3).toFixed(1)+"K":String(e)}function Sa(e){try{return new URL(e).hostname.replace("www.","")}catch{return""}}function J(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function La(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const Ea='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',_a='<svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg>',Ma='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/></svg>';function Ba(e){return`
    <div class="min-h-screen flex flex-col">
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="flex items-center gap-4 px-4 lg:px-8 py-3">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[692px]">
            ${S({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors">
            ${Ea}
          </a>
        </div>
        <div class="px-4 lg:px-8 pl-[170px]">
          ${B({query:e,active:"music"})}
        </div>
      </header>
      <main class="flex-1">
        <div id="music-content" class="px-4 lg:px-8 py-6">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `}function Ta(e,t){L(s=>e.navigate(`/music?q=${encodeURIComponent(s)}`)),t&&M(t),Ha(t)}async function Ha(e){const t=document.getElementById("music-content");if(!(!t||!e))try{const s=await x.searchMusic(e),a=s.results;if(a.length===0){t.innerHTML=`<div class="py-8 text-secondary">No music results found for "${H(e)}"</div>`;return}t.innerHTML=`
      <div class="text-xs text-tertiary mb-4">
        About ${s.total_results.toLocaleString()} results (${(s.search_time_ms/1e3).toFixed(2)} seconds)
      </div>
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
        ${a.map(Aa).join("")}
      </div>
    `}catch(s){t.innerHTML=`<div class="py-8 text-red text-sm">Failed to load results. ${H(String(s))}</div>`}}function Aa(e){var l,o,c,d,f;const t=((l=e.metadata)==null?void 0:l.source)||Ra(e.url),s=((o=e.metadata)==null?void 0:o.artist)||"",a=((c=e.metadata)==null?void 0:c.album)||"",r=((d=e.metadata)==null?void 0:d.duration)||"",n=((f=e.thumbnail)==null?void 0:f.url)||"",i=t.toLowerCase().includes("genius");return`
    <article class="music-card bg-white border border-border rounded-xl overflow-hidden hover:shadow-md transition-shadow">
      <a href="${Be(e.url)}" target="_blank" rel="noopener" class="block">
        <div class="relative aspect-square bg-surface">
          ${n?`<img src="${Be(n)}" alt="" class="w-full h-full object-cover" loading="lazy" onerror="this.style.display='none'" />`:`<div class="w-full h-full flex items-center justify-center text-border">${Ma}</div>`}
          <div class="absolute inset-0 bg-black/40 opacity-0 hover:opacity-100 transition-opacity flex items-center justify-center">
            <span class="w-12 h-12 rounded-full bg-white flex items-center justify-center text-primary">${_a}</span>
          </div>
        </div>
        <div class="p-3">
          <span class="text-xs px-2 py-0.5 rounded-full font-medium ${Na(t)}">${H(t)}</span>
          <h3 class="text-sm font-medium text-primary mt-2 line-clamp-2">${H(e.title)}</h3>
          ${s?`<p class="text-xs text-secondary mt-1">${H(s)}</p>`:""}
          ${a?`<p class="text-xs text-tertiary">${H(a)}</p>`:""}
          ${r?`<p class="text-xs text-tertiary mt-1">${H(r)}</p>`:""}
          ${i&&e.snippet?`<p class="text-xs text-snippet mt-2 line-clamp-2 italic">"${H(e.snippet.slice(0,100))}..."</p>`:""}
        </div>
      </a>
    </article>
  `}function Na(e){return e.toLowerCase().includes("soundcloud")?"bg-orange-500 text-white":e.toLowerCase().includes("bandcamp")?"bg-teal-500 text-white":e.toLowerCase().includes("genius")?"bg-yellow-400 text-black":"bg-surface text-secondary"}function Ra(e){try{return new URL(e).hostname.replace("www.","")}catch{return""}}function H(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function Be(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const Oa='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',ja='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m18 15-6-6-6 6"/></svg>',qa='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>';function Fa(e){return`
    <div class="min-h-screen flex flex-col">
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="flex items-center gap-4 px-4 lg:px-8 py-3">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[692px]">
            ${S({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors">
            ${Oa}
          </a>
        </div>
        <div class="px-4 lg:px-8 pl-[170px]">
          ${B({query:e,active:"social"})}
        </div>
      </header>
      <main class="flex-1">
        <div id="social-content" class="px-4 lg:px-8 py-6">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `}function Pa(e,t){L(s=>e.navigate(`/social?q=${encodeURIComponent(s)}`)),t&&M(t),Ua(t)}async function Ua(e){const t=document.getElementById("social-content");if(!(!t||!e))try{const s=await x.searchSocial(e),a=s.results;if(a.length===0){t.innerHTML=`<div class="py-8 text-secondary">No social results found for "${P(e)}"</div>`;return}t.innerHTML=`
      <div class="text-xs text-tertiary mb-4">
        About ${s.total_results.toLocaleString()} results (${(s.search_time_ms/1e3).toFixed(2)} seconds)
      </div>
      <div class="max-w-[800px] space-y-4">
        ${a.map(za).join("")}
      </div>
    `}catch(s){t.innerHTML=`<div class="py-8 text-red text-sm">Failed to load results. ${P(String(s))}</div>`}}function za(e){var d;const t=e.metadata||{},s=t.source||Ga(e.url),a=t.upvotes||t.score||0,r=t.comments||0,n=t.author||"",i=t.subreddit||"",l=t.published||"",o=((d=e.thumbnail)==null?void 0:d.url)||"",c=e.snippet||"";return`
    <article class="social-card bg-white border border-border rounded-xl p-4 hover:shadow-md transition-shadow">
      <div class="flex items-start gap-3">
        <!-- Upvote column -->
        <div class="flex flex-col items-center text-tertiary text-sm">
          ${ja}
          <span class="font-medium ${a>0?"text-orange-500":""}">${Te(a)}</span>
        </div>
        <!-- Content -->
        <div class="flex-1 min-w-0">
          <div class="flex items-center gap-2 mb-1 flex-wrap">
            <span class="text-xs px-2 py-0.5 rounded-full font-medium ${Va(s)}">${P(s)}</span>
            ${i?`<span class="text-xs text-blue">r/${P(i)}</span>`:""}
            ${n?`<span class="text-xs text-tertiary">by ${P(n)}</span>`:""}
            ${l?`<span class="text-xs text-tertiary">${Da(l)}</span>`:""}
          </div>
          <h3 class="text-base font-medium text-primary mb-1">
            <a href="${He(e.url)}" target="_blank" rel="noopener" class="hover:text-blue hover:underline">${P(e.title)}</a>
          </h3>
          ${c?`<p class="text-sm text-snippet line-clamp-3 mb-2">${P(c)}</p>`:""}
          <div class="flex items-center gap-4 text-xs text-tertiary">
            <span class="flex items-center gap-1">${qa} ${Te(r)} comments</span>
          </div>
        </div>
        <!-- Thumbnail if available -->
        ${o?`
          <img src="${He(o)}" alt="" class="w-20 h-20 rounded-lg object-cover flex-shrink-0" loading="lazy" onerror="this.style.display='none'" />
        `:""}
      </div>
    </article>
  `}function Va(e){const t=e.toLowerCase();return t.includes("reddit")?"bg-orange-500 text-white":t.includes("hacker")||t.includes("hn")?"bg-orange-600 text-white":t.includes("mastodon")?"bg-purple-500 text-white":t.includes("lemmy")?"bg-green-500 text-white":"bg-surface text-secondary"}function Te(e){return e>=1e6?(e/1e6).toFixed(1)+"M":e>=1e3?(e/1e3).toFixed(1)+"K":String(e)}function Da(e){try{const t=new Date(e),a=new Date().getTime()-t.getTime(),r=Math.floor(a/(1e3*60*60)),n=Math.floor(a/(1e3*60*60*24));return r<1?"just now":r<24?`${r}h ago`:n<7?`${n}d ago`:n<30?`${Math.floor(n/7)}w ago`:`${Math.floor(n/30)}mo ago`}catch{return""}}function Ga(e){try{return new URL(e).hostname.replace("www.","")}catch{return""}}function P(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function He(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const Wa='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>',Ae='<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" x2="21" y1="14" y2="3"/></svg>';function Ya(e){return`
    <div class="min-h-screen flex flex-col">
      <header class="sticky top-0 bg-white z-20 border-b border-border">
        <div class="flex items-center gap-4 px-4 lg:px-8 py-3">
          <a href="/" data-link class="flex-shrink-0 text-2xl font-semibold select-none">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
          </a>
          <div class="flex-1 max-w-[692px]">
            ${S({size:"sm",initialValue:e})}
          </div>
          <a href="/settings" data-link class="flex-shrink-0 text-tertiary hover:text-primary p-2 rounded-full hover:bg-surface-hover transition-colors">
            ${Wa}
          </a>
        </div>
        <div class="px-4 lg:px-8">
          ${B({query:e,active:"maps"})}
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
  `}function Ka(e,t){L(s=>e.navigate(`/maps?q=${encodeURIComponent(s)}`)),t&&M(t),Za(t)}async function Za(e){var a,r;const t=document.getElementById("maps-content"),s=document.getElementById("map-iframe");if(!(!t||!e)){if(s){const n="https://www.openstreetmap.org/export/embed.html?bbox=-180,-90,180,90&layer=mapnik&marker=0,0";s.src=n}try{const i=(await x.searchMaps(e)).results;if(i.length===0){t.innerHTML=`<div class="p-4 text-secondary">No locations found for "${Q(e)}"</div>`;return}const l=i[0],o=((a=l.metadata)==null?void 0:a.lat)||0,c=((r=l.metadata)==null?void 0:r.lon)||0;if(s&&o&&c){const d=`${c-.1},${o-.1},${c+.1},${o+.1}`;s.src=`https://www.openstreetmap.org/export/embed.html?bbox=${d}&layer=mapnik&marker=${o},${c}`}t.innerHTML=`
      <div class="p-4">
        <div class="text-xs text-tertiary mb-4">${i.length} locations found</div>
        <div class="space-y-3">
          ${i.map((d,f)=>Ja(d,f)).join("")}
        </div>
      </div>
    `,t.querySelectorAll(".location-card").forEach(d=>{d.addEventListener("click",()=>{const f=d.dataset.lat,v=d.dataset.lon;if(f&&v&&s){const y=`${parseFloat(v)-.05},${parseFloat(f)-.05},${parseFloat(v)+.05},${parseFloat(f)+.05}`;s.src=`https://www.openstreetmap.org/export/embed.html?bbox=${y}&layer=mapnik&marker=${f},${v}`}})})}catch(n){t.innerHTML=`<div class="p-4 text-red text-sm">Failed to load results. ${Q(String(n))}</div>`}}}function Ja(e,t){var i,l,o;const s=((i=e.metadata)==null?void 0:i.lat)||0,a=((l=e.metadata)==null?void 0:l.lon)||0,r=((o=e.metadata)==null?void 0:o.type)||"place",n=e.content||"";return`
    <article class="location-card bg-white border border-border rounded-lg p-3 cursor-pointer hover:shadow-md transition-shadow"
             data-lat="${s}" data-lon="${a}">
      <div class="flex items-start gap-3">
        <span class="flex-shrink-0 w-8 h-8 rounded-full bg-red-500 text-white flex items-center justify-center text-sm font-medium">
          ${t+1}
        </span>
        <div class="flex-1 min-w-0">
          <h3 class="font-medium text-primary text-sm">${Q(e.title)}</h3>
          <p class="text-xs text-tertiary mt-0.5 capitalize">${Q(r)}</p>
          ${n?`<p class="text-xs text-secondary mt-1 line-clamp-2">${Q(n)}</p>`:""}
          <p class="text-xs text-tertiary mt-1">${s.toFixed(5)}, ${a.toFixed(5)}</p>
          <div class="flex items-center gap-2 mt-2">
            <a href="${Qa(e.url)}" target="_blank" rel="noopener"
               class="text-xs text-blue hover:underline flex items-center gap-1"
               onclick="event.stopPropagation()">
              View on OSM ${Ae}
            </a>
            <a href="https://www.google.com/maps?q=${s},${a}" target="_blank" rel="noopener"
               class="text-xs text-blue hover:underline flex items-center gap-1"
               onclick="event.stopPropagation()">
              Google Maps ${Ae}
            </a>
          </div>
        </div>
      </div>
    </article>
  `}function Q(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function Qa(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const Xa='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m12 19-7-7 7-7"/><path d="M19 12H5"/></svg>',en=[{value:"auto",label:"Auto-detect"},{value:"us",label:"United States"},{value:"gb",label:"United Kingdom"},{value:"de",label:"Germany"},{value:"fr",label:"France"},{value:"es",label:"Spain"},{value:"it",label:"Italy"},{value:"nl",label:"Netherlands"},{value:"pl",label:"Poland"},{value:"br",label:"Brazil"},{value:"ca",label:"Canada"},{value:"au",label:"Australia"},{value:"in",label:"India"},{value:"jp",label:"Japan"},{value:"kr",label:"South Korea"},{value:"cn",label:"China"},{value:"ru",label:"Russia"}],tn=[{value:"en",label:"English"},{value:"de",label:"German (Deutsch)"},{value:"fr",label:"French (Français)"},{value:"es",label:"Spanish (Español)"},{value:"it",label:"Italian (Italiano)"},{value:"pt",label:"Portuguese (Português)"},{value:"nl",label:"Dutch (Nederlands)"},{value:"pl",label:"Polish (Polski)"},{value:"ja",label:"Japanese"},{value:"ko",label:"Korean"},{value:"zh",label:"Chinese"},{value:"ru",label:"Russian"},{value:"ar",label:"Arabic"},{value:"hi",label:"Hindi"}];function sn(){const e=z.get().settings;return`
    <div class="min-h-screen bg-white">
      <!-- Header -->
      <header class="border-b border-border">
        <div class="max-w-[700px] mx-auto px-4 py-4 flex items-center gap-4">
          <a href="/" data-link class="text-tertiary hover:text-primary transition-colors" aria-label="Back">
            ${Xa}
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
              ${en.map(t=>`<option value="${t.value}" ${e.region===t.value?"selected":""}>${Ne(t.label)}</option>`).join("")}
            </select>
          </div>

          <!-- Language -->
          <div class="settings-section">
            <h3>Language</h3>
            <select name="language" class="settings-select">
              ${tn.map(t=>`<option value="${t.value}" ${e.language===t.value?"selected":""}>${Ne(t.label)}</option>`).join("")}
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
  `}function an(e){const t=document.getElementById("settings-form"),s=document.getElementById("settings-status");t&&t.addEventListener("submit",async a=>{a.preventDefault();const r=new FormData(t),n={safe_search:r.get("safe_search")||"moderate",results_per_page:parseInt(r.get("results_per_page"))||10,region:r.get("region")||"auto",language:r.get("language")||"en",theme:r.get("theme")||"light",open_in_new_tab:r.has("open_in_new_tab"),show_thumbnails:r.has("show_thumbnails")};z.set({settings:n});try{await x.updateSettings(n)}catch{}s&&(s.classList.remove("hidden"),setTimeout(()=>{s.classList.add("hidden")},2e3))})}function Ne(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}const nn='<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m12 19-7-7 7-7"/><path d="M19 12H5"/></svg>',rn='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/></svg>',ln='<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>',on='<svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="#dadce0" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/><path d="M12 7v5l4 2"/></svg>';function cn(){return`
    <div class="min-h-screen bg-white">
      <!-- Header -->
      <header class="border-b border-border">
        <div class="max-w-[700px] mx-auto px-4 py-4 flex items-center justify-between">
          <div class="flex items-center gap-4">
            <a href="/" data-link class="text-tertiary hover:text-primary transition-colors" aria-label="Back">
              ${nn}
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
  `}function dn(e){const t=document.getElementById("clear-all-btn");un(e),t==null||t.addEventListener("click",async()=>{if(confirm("Are you sure you want to clear all search history?"))try{await x.clearHistory(),xe(),t.classList.add("hidden")}catch(s){console.error("Failed to clear history:",s)}})}async function un(e){const t=document.getElementById("history-content"),s=document.getElementById("clear-all-btn");if(t)try{const a=await x.getHistory();if(a.length===0){xe();return}s&&s.classList.remove("hidden"),t.innerHTML=`
      <div id="history-list">
        ${a.map(r=>pn(r)).join("")}
      </div>
    `,hn(e)}catch(a){t.innerHTML=`
      <div class="py-8 text-center">
        <p class="text-red text-sm">Failed to load search history.</p>
        <p class="text-tertiary text-xs mt-2">${fe(String(a))}</p>
      </div>
    `}}function pn(e){const t=gn(e.searched_at);return`
    <div class="history-item flex items-center gap-3 py-3 px-2 border-b border-border hover:bg-surface-hover rounded transition-colors group" data-history-id="${Re(e.id)}">
      <span class="text-light flex-shrink-0">${ln}</span>
      <div class="flex-1 min-w-0">
        <a href="/search?q=${encodeURIComponent(e.query)}" data-link class="text-sm text-primary hover:text-link font-medium truncate block">
          ${fe(e.query)}
        </a>
        <div class="flex items-center gap-2 text-xs text-light mt-0.5">
          <span>${fe(t)}</span>
          ${e.results>0?`<span>&middot; ${e.results} results</span>`:""}
          ${e.clicked_url?"<span>&middot; visited</span>":""}
        </div>
      </div>
      <button class="history-delete-btn text-light hover:text-red p-1.5 rounded-full hover:bg-red/10 opacity-0 group-hover:opacity-100 transition-opacity flex-shrink-0 cursor-pointer"
              data-delete-id="${Re(e.id)}" aria-label="Delete">
        ${rn}
      </button>
    </div>
  `}function hn(e){document.querySelectorAll(".history-delete-btn").forEach(t=>{t.addEventListener("click",async s=>{s.preventDefault(),s.stopPropagation();const a=t.dataset.deleteId||"",r=t.closest(".history-item");try{await x.deleteHistoryItem(a),r&&r.remove();const n=document.getElementById("history-list");if(n&&n.children.length===0){xe();const i=document.getElementById("clear-all-btn");i&&i.classList.add("hidden")}}catch(n){console.error("Failed to delete history item:",n)}})})}function xe(){const e=document.getElementById("history-content");e&&(e.innerHTML=`
    <div class="py-16 flex flex-col items-center text-center">
      ${on}
      <h2 class="text-lg font-medium text-primary mt-4 mb-2">No search history</h2>
      <p class="text-sm text-tertiary max-w-[300px]">
        Your recent searches will appear here. Start searching to build your history.
      </p>
      <a href="/" data-link class="mt-4 text-sm text-blue hover:underline">Go to search</a>
    </div>
  `)}function gn(e){try{const t=new Date(e),s=new Date,a=s.getTime()-t.getTime(),r=Math.floor(a/(1e3*60)),n=Math.floor(a/(1e3*60*60)),i=Math.floor(a/(1e3*60*60*24));return r<1?"Just now":r<60?`${r}m ago`:n<24?`${n}h ago`:i===1?"Yesterday":i<7?`${i} days ago`:t.toLocaleDateString("en-US",{month:"short",day:"numeric",year:t.getFullYear()!==s.getFullYear()?"numeric":void 0})}catch{return e}}function fe(e){return e.replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}function Re(e){return e.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}const b=document.getElementById("app");if(!b)throw new Error("App container not found");const m=new Ge;m.addRoute("",(e,t)=>{b.innerHTML=it(),lt(m)});m.addRoute("search",(e,t)=>{const s=t.q||"",a=t.time_range||"";b.innerHTML=ns(s,a),rs(m,s,t)});m.addRoute("images",(e,t)=>{const s=t.q||"";b.innerHTML=ms(s),fs(m,s,t)});m.addRoute("videos",(e,t)=>{const s=t.q||"";b.innerHTML=Ms(s),Bs(m,s)});m.addRoute("news",(e,t)=>{const s=t.q||"";b.innerHTML=Os(s),js(m,s)});m.addRoute("news-home",(e,t)=>{b.innerHTML=Xs(),ea(m)});m.addRoute("science",(e,t)=>{const s=t.q||"";b.innerHTML=pa(s),ha(m,s)});m.addRoute("code",(e,t)=>{const s=t.q||"";b.innerHTML=ba(s),$a(m,s)});m.addRoute("music",(e,t)=>{const s=t.q||"";b.innerHTML=Ba(s),Ta(m,s)});m.addRoute("social",(e,t)=>{const s=t.q||"";b.innerHTML=Fa(s),Pa(m,s)});m.addRoute("maps",(e,t)=>{const s=t.q||"";b.innerHTML=Ya(s),Ka(m,s)});m.addRoute("settings",(e,t)=>{b.innerHTML=sn(),an()});m.addRoute("history",(e,t)=>{b.innerHTML=cn(),dn(m)});m.setNotFound((e,t)=>{b.innerHTML=`
    <div class="min-h-screen flex flex-col items-center justify-center px-4">
      <h1 class="text-4xl font-semibold mb-4">
        <span style="color: #4285F4">4</span><span style="color: #EA4335">0</span><span style="color: #FBBC05">4</span>
      </h1>
      <p class="text-secondary mb-6">Page not found</p>
      <a href="/" data-link class="text-blue hover:underline">Go home</a>
    </div>
  `});window.addEventListener("router:navigate",e=>{const t=e;m.navigate(t.detail.path)});m.start();
