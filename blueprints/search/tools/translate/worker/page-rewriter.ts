const TRANSLATE_BASE = 'https://translate.googleapis.com/translate_a/single'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
async function translateChunk(text: string, sl: string, tl: string): Promise<{ text: string; ok: boolean; src?: string }> {
  const trimmed = text.trim()
  if (!trimmed) return { text, ok: true }
  if (/^[\s\d\p{P}\p{S}]+$/u.test(trimmed)) return { text, ok: true }

  const params = new URLSearchParams()
  params.set('client', 'gtx')
  params.set('sl', sl)
  params.set('tl', tl)
  params.set('dj', '1')
  params.append('dt', 't')

  try {
    let resp: Response
    if (text.length <= 2000) {
      params.set('q', text)
      resp = await fetch(`${TRANSLATE_BASE}?${params.toString()}`, {
        headers: { 'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36' },
      })
    } else {
      const body = new URLSearchParams()
      body.set('q', text)
      resp = await fetch(`${TRANSLATE_BASE}?${params.toString()}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/x-www-form-urlencoded',
          'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
        },
        body: body.toString(),
      })
    }

    if (!resp.ok) return { text, ok: false }

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const data: any = await resp.json()
    if (!data.sentences) return { text, ok: false }

    const result = data.sentences
      .filter((s: { trans?: string }) => s.trans != null)
      .map((s: { trans: string }) => s.trans)
      .join('')

    return result ? { text: result, ok: true, src: data.src } : { text, ok: false }
  } catch {
    return { text, ok: false }
  }
}

const SKIP_SELECTORS = [
  'script', 'style', 'code', 'pre', 'kbd', 'samp', 'var',
  'svg', 'noscript', 'canvas', 'audio', 'video', 'iframe',
] as const

const TRANSLATABLE_SELECTORS = [
  'p', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
  'li', 'td', 'th', 'dt', 'dd', 'figcaption', 'blockquote',
  'title', 'label', 'button', 'caption', 'summary', 'legend', 'option',
] as const

const SKIP_HREF_PREFIXES = ['#', 'javascript:', 'mailto:', 'tel:', 'data:']

function rewriteUrl(href: string, originUrl: URL, proxyBase: string, tl: string): string {
  if (SKIP_HREF_PREFIXES.some((p) => href.startsWith(p))) return href

  let absolute: string
  try {
    new URL(href)
    absolute = href
  } catch {
    try {
      absolute = new URL(href, originUrl.origin).toString()
    } catch {
      return href
    }
  }

  return `${proxyBase}/page/${tl}?url=${encodeURIComponent(absolute)}`
}

function escapeAttr(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/"/g, '&quot;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/[\n\r]/g, '&#10;')
}

function buildBanner(originUrl: URL, tl: string, sl: string): string {
  const slLabel = sl === 'auto' ? 'auto-detected' : sl
  return [
    '<div style="position:fixed;top:0;left:0;right:0;z-index:2147483647;background:#4285f4;color:#fff;',
    'font:14px/1 -apple-system,system-ui,sans-serif;padding:8px 16px;display:flex;align-items:center;',
    'justify-content:center;gap:12px;box-shadow:0 2px 6px rgba(0,0,0,.3)">',
    `<span>Translated from <b>${slLabel}</b> to <b>${tl}</b></span>`,
    '<span style="background:rgba(255,255,255,.2);padding:3px 10px;border-radius:10px;font-size:11px;letter-spacing:.3px">',
    'Click text to learn</span>',
    `<a href="${originUrl.toString()}" style="color:#fff;text-decoration:underline;font-size:13px">View original</a>`,
    '</div><div style="height:38px"></div>',
  ].join('')
}

/* ── Learner CSS ── */

function buildLearnerCSS(): string {
  return `<style id="tl-learner-css">
.tl-block{cursor:pointer;transition:background .15s}
.tl-block:hover{background:rgba(66,133,244,.06);border-radius:4px}
.tl-seg{border-radius:2px}
.tl-overlay{position:fixed;top:0;left:0;right:0;bottom:0;z-index:2147483645;background:rgba(0,0,0,.06);display:none}
.tl-popup{position:fixed;z-index:2147483646;background:#fff;border-radius:12px;box-shadow:0 8px 32px rgba(0,0,0,.18);max-width:540px;width:92vw;font:15px/1.6 -apple-system,system-ui,BlinkMacSystemFont,sans-serif;color:#202124;display:none;overflow:hidden}
.tl-popup-hdr{padding:12px 20px 10px;background:#f8f9fa;border-bottom:1px solid #e8eaed;display:flex;align-items:center;justify-content:space-between}
.tl-popup-hdr-t{font-size:12px;font-weight:600;color:#5f6368;text-transform:uppercase;letter-spacing:.5px}
.tl-popup-x{background:none;border:none;cursor:pointer;font-size:22px;color:#5f6368;padding:0 4px;border-radius:50%;line-height:1}
.tl-popup-x:hover{background:#e8eaed}
.tl-popup-bd{padding:16px 20px;max-height:60vh;overflow-y:auto}
.tl-popup-sec{margin-bottom:16px}
.tl-popup-sec:last-child{margin-bottom:0}
.tl-popup-lbl{font-size:11px;font-weight:700;color:#5f6368;text-transform:uppercase;letter-spacing:.6px;margin-bottom:6px;display:flex;align-items:center;gap:8px}
.tl-popup-txt{font-size:15px;line-height:1.7;color:#202124}
.tl-popup-txt.orig{color:#1a73e8}
.tl-abtn{background:none;border:none;cursor:pointer;padding:4px;border-radius:50%;color:#4285f4;display:inline-flex;align-items:center;vertical-align:middle}
.tl-abtn:hover{background:#e8f0fe}
.tl-abtn.playing{animation:tl-pulse .8s ease infinite}
@keyframes tl-pulse{0%,100%{opacity:1}50%{opacity:.4}}
.tl-abtn svg{width:18px;height:18px}
.tl-words{display:flex;flex-wrap:wrap;gap:6px;margin-top:4px}
.tl-word{display:inline-block;padding:4px 12px;background:#e8f0fe;border-radius:16px;font-size:13px;cursor:pointer;transition:all .15s;border:1px solid transparent;color:#1a73e8;user-select:none}
.tl-word:hover{background:#d2e3fc;border-color:#a8c7fa}
.tl-word:active{background:#c2dbf6}
.tl-wtip{position:fixed;z-index:2147483647;background:#fff;border-radius:10px;box-shadow:0 4px 20px rgba(0,0,0,.18);padding:14px 18px;font:14px/1.5 -apple-system,system-ui,sans-serif;max-width:340px;min-width:140px;display:none}
.tl-wtip-w{font-size:15px;font-weight:600;color:#202124;margin-bottom:2px}
.tl-wtip-tr{font-size:16px;color:#1a73e8;font-weight:500;margin-bottom:6px;display:flex;align-items:center;gap:6px}
.tl-wtip-pos{font-size:10px;font-weight:700;color:#5f6368;text-transform:uppercase;letter-spacing:.5px;margin-top:8px;margin-bottom:2px}
.tl-wtip-terms{font-size:13px;color:#5f6368;line-height:1.4}
.tl-wtip-ld{color:#5f6368;font-size:13px}
.tl-sep{border:none;border-top:1px solid #e8eaed;margin:12px 0}
</style>`
}

/* ── Learner Script — stays in cached HTML ── */

function buildLearnerScript(proxyBase: string, tl: string, defaultSl: string): string {
  // Variables injected via JSON.stringify for safe escaping
  const cfgTL = JSON.stringify(tl)
  const cfgBase = JSON.stringify(proxyBase)
  const cfgSL = JSON.stringify(defaultSl)

  // SVG speaker icon (no escaping issues — no </ sequences)
  const spkSVG = '<svg viewBox="0 0 24 24" fill="currentColor"><path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3A4.5 4.5 0 0014 7.97v8.05c1.48-.73 2.5-2.25 2.5-4.02zM14 3.23v2.06c2.89.86 5 3.54 5 6.71s-2.11 5.85-5 6.71v2.06c4.01-.91 7-4.49 7-8.77s-2.99-7.86-7-8.77z"/><\/svg>'

  return `<script id="tl-learner">(function(){
var TL=${cfgTL},BASE=${cfgBase},SL=${cfgSL};
var GT='https://translate.googleapis.com/translate_a/single';
var SPK='${spkSVG}';

function getSL(){return window._tlSL||SL}

/* ── DOM setup ── */
var ov=document.createElement('div');ov.className='tl-overlay';document.body.appendChild(ov);
var pop=document.createElement('div');pop.className='tl-popup';
pop.innerHTML='<div class="tl-popup-hdr"><span class="tl-popup-hdr-t">Learning Mode</span><button class="tl-popup-x" id="tl-x">\\u00d7</button></div>'+
'<div class="tl-popup-bd">'+
'<div class="tl-popup-sec"><div class="tl-popup-lbl">Original <button class="tl-abtn" id="tl-ao">'+SPK+'</button></div><div class="tl-popup-txt orig" id="tl-ot"></div></div>'+
'<hr class="tl-sep">'+
'<div class="tl-popup-sec"><div class="tl-popup-lbl">Translation <button class="tl-abtn" id="tl-at">'+SPK+'</button></div><div class="tl-popup-txt" id="tl-tt"></div></div>'+
'<hr class="tl-sep">'+
'<div class="tl-popup-sec"><div class="tl-popup-lbl">Words</div><div class="tl-words" id="tl-wc"></div></div>'+
'</div>';
document.body.appendChild(pop);

var wtip=document.createElement('div');wtip.className='tl-wtip';document.body.appendChild(wtip);

/* ── Helpers ── */
function escH(s){var d=document.createElement('span');d.textContent=s;return d.innerHTML}

function getOrigText(block){
  var parts=[];
  var walker=document.createTreeWalker(block,NodeFilter.SHOW_TEXT,null);
  var n;
  while(n=walker.nextNode()){
    var p=n.parentElement;
    if(p&&p.classList.contains('tl-seg')&&p.hasAttribute('data-orig')){
      parts.push(p.getAttribute('data-orig'));
    }else{
      parts.push(n.textContent);
    }
  }
  return parts.join('');
}

/* ── Audio ── */
var aq=[],aPlaying=false,curAudio=null,curBtn=null;

function splitTTS(text,max){
  if(text.length<=max)return[text];
  var sentences=text.match(/[^.!?]+[.!?]+|[^.!?]+$/g)||[text];
  var chunks=[],cur='';
  for(var i=0;i<sentences.length;i++){
    if((cur+sentences[i]).length>max&&cur){chunks.push(cur.trim());cur=sentences[i]}
    else{cur+=sentences[i]}
  }
  if(cur.trim())chunks.push(cur.trim());
  return chunks;
}

function playTTS(text,lang,btn){
  stopAudio();
  curBtn=btn;
  if(btn)btn.classList.add('playing');
  var chunks=splitTTS(text,190);
  aq=chunks.map(function(c){return{text:c,lang:lang}});
  playNext();
}

function playNext(){
  if(aq.length===0){stopAudio();return}
  aPlaying=true;
  var item=aq.shift();
  var url=BASE+'/api/tts?tl='+encodeURIComponent(item.lang)+'&q='+encodeURIComponent(item.text);
  curAudio=new Audio(url);
  curAudio.onended=playNext;
  curAudio.onerror=playNext;
  curAudio.play().catch(playNext);
}

function stopAudio(){
  aPlaying=false;aq=[];
  if(curAudio){try{curAudio.pause()}catch(e){} curAudio=null}
  if(curBtn){curBtn.classList.remove('playing');curBtn=null}
}

/* ── Popup ── */
var popOpen=false;

function showPopup(block){
  var orig=getOrigText(block);
  var trans=block.textContent||'';
  document.getElementById('tl-ot').textContent=orig;
  document.getElementById('tl-tt').textContent=trans;

  /* word chips */
  var wc=document.getElementById('tl-wc');
  wc.innerHTML='';
  var seen={};
  var words=orig.split(/\\s+/);
  for(var i=0;i<words.length;i++){
    var w=words[i].replace(/^[^\\p{L}\\p{N}]+|[^\\p{L}\\p{N}]+$/gu,'');
    if(!w||w.length<2)continue;
    var key=w.toLowerCase();
    if(seen[key])continue;
    seen[key]=true;
    (function(word){
      var chip=document.createElement('span');
      chip.className='tl-word';
      chip.textContent=word;
      chip.onclick=function(e){e.stopPropagation();lookupWord(word,chip)};
      wc.appendChild(chip);
    })(w);
  }

  /* position */
  var rect=block.getBoundingClientRect();
  pop.style.display='block';
  ov.style.display='block';
  var top=rect.bottom+10+window.scrollY;
  var left=rect.left+(rect.width/2)-(pop.offsetWidth/2);
  if(left<12)left=12;
  if(left+pop.offsetWidth>window.innerWidth-12)left=window.innerWidth-pop.offsetWidth-12;
  pop.style.position='absolute';
  pop.style.top=top+'px';
  pop.style.left=left+'px';
  popOpen=true;
}

function closePopup(){
  pop.style.display='none';
  ov.style.display='none';
  closeWtip();
  stopAudio();
  popOpen=false;
}

/* ── Word dictionary ── */
function lookupWord(word,chipEl){
  closeWtip();
  wtip.innerHTML='<div class="tl-wtip-ld">Loading...</div>';
  var rect=chipEl.getBoundingClientRect();
  wtip.style.display='block';
  wtip.style.top=(rect.bottom+6)+'px';
  wtip.style.left=Math.max(8,Math.min(rect.left,window.innerWidth-200))+'px';

  var sl=getSL();
  var p=new URLSearchParams();
  p.set('client','gtx');p.set('sl',sl);p.set('tl',TL);p.set('dj','1');
  p.append('dt','t');p.append('dt','bd');p.append('dt','rm');
  p.set('q',word);

  fetch(GT+'?'+p).then(function(r){
    if(!r.ok)throw new Error('err');
    return r.json();
  }).then(function(d){
    var html='<div class="tl-wtip-w">'+escH(word)+'</div>';
    var tr=d.sentences?d.sentences.filter(function(s){return s.trans!=null}).map(function(s){return s.trans}).join(''):'';
    if(tr)html+='<div class="tl-wtip-tr">'+escH(tr)+' <button class="tl-abtn tl-wa" style="padding:2px">'+SPK+'</button></div>';
    if(d.dict){
      for(var i=0;i<d.dict.length;i++){
        var entry=d.dict[i];
        html+='<div class="tl-wtip-pos">'+escH(entry.pos||'')+'</div>';
        if(entry.terms)html+='<div class="tl-wtip-terms">'+entry.terms.slice(0,6).map(escH).join(', ')+'</div>';
      }
    }
    wtip.innerHTML=html;
    /* audio for word */
    var wa=wtip.querySelector('.tl-wa');
    if(wa){wa.onclick=function(e){e.stopPropagation();playTTS(word,sl,wa)}}
  }).catch(function(){
    wtip.innerHTML='<div class="tl-wtip-ld">No result</div>';
  });
}

function closeWtip(){wtip.style.display='none'}

/* ── Event listeners ── */
document.addEventListener('click',function(e){
  if(e.target.closest('a')&&!e.target.closest('.tl-popup'))return;
  if(e.target.closest('.tl-popup-x')){closePopup();return}
  if(e.target.closest('.tl-wtip'))return;
  if(e.target.closest('.tl-popup'))return;

  var block=e.target.closest('.tl-block');
  if(block){
    var segs=block.querySelectorAll('.tl-seg');
    if(segs.length>0){
      e.preventDefault();
      e.stopPropagation();
      closeWtip();
      showPopup(block);
      return;
    }
  }
  if(popOpen)closePopup();
},true);

ov.onclick=closePopup;

var scrollTimer;
window.addEventListener('scroll',function(){
  if(!popOpen)return;
  clearTimeout(scrollTimer);
  scrollTimer=setTimeout(closePopup,150);
},{passive:true});

/* ── Audio button handlers ── */
document.getElementById('tl-ao').onclick=function(e){
  e.stopPropagation();
  var text=document.getElementById('tl-ot').textContent;
  playTTS(text,getSL(),this);
};
document.getElementById('tl-at').onclick=function(e){
  e.stopPropagation();
  var text=document.getElementById('tl-tt').textContent;
  playTTS(text,TL,this);
};
document.getElementById('tl-x').onclick=function(e){e.stopPropagation();closePopup()};

})()</script>`
}

/* ── Translate fallback script — removed after execution, NOT in cached HTML ── */

function buildTranslateScript(originUrl: string, proxyBase: string, tl: string): string {
  const cfgTL = JSON.stringify(tl)
  const cfgUrl = JSON.stringify(originUrl)
  const cfgBase = JSON.stringify(proxyBase)

  return `<script id="translate-cs">(async function(){
var tl=${cfgTL},pageUrl=${cfgUrl},base=${cfgBase};
var GT='https://translate.googleapis.com/translate_a/single';
var els=document.querySelectorAll('[data-tp]');
if(els.length){
  for(var el of els){
    var t=el.textContent;if(!t||!t.trim())continue;
    try{
      var p=new URLSearchParams({client:'gtx',sl:'auto',tl:tl,dj:'1',dt:'t',q:t});
      var r=await fetch(GT+'?'+p);
      if(!r.ok)continue;
      var d=await r.json();
      if(d.src&&!window._tlSL)window._tlSL=d.src;
      var tr=d.sentences?d.sentences.filter(function(s){return s.trans!=null}).map(function(s){return s.trans}).join(''):'';
      if(tr&&tr!==t)el.textContent=tr;
    }catch(e){}
    el.removeAttribute('data-tp');
  }
}
var sc=document.getElementById('translate-cs');if(sc)sc.remove();
try{
  await fetch(base+'/page/cache',{method:'POST',headers:{'Content-Type':'application/json'},
    body:JSON.stringify({url:pageUrl,tl:tl,html:'<!DOCTYPE html>'+document.documentElement.outerHTML})});
}catch(e){}
})()</script>`
}

/* ── Main HTMLRewriter factory ── */

export function makePageRewriter(
  originUrl: URL,
  proxyBase: string,
  tl: string,
  sl: string,
): HTMLRewriter {
  const textBuffer: string[] = []
  let skipDepth = 0
  let hasFailed = false
  let detectedSl = ''

  let rewriter = new HTMLRewriter()
    .on('html', {
      element(el) { el.setAttribute('lang', tl) },
    })
    .on('head', {
      element(el) {
        el.prepend(`<base href="${originUrl.origin}/">`, { html: true })
        el.prepend(buildBanner(originUrl, tl, sl), { html: true })
        el.append(buildLearnerCSS(), { html: true })
      },
    })
    .on('a[href]', {
      element(el) {
        const href = el.getAttribute('href')
        if (href) el.setAttribute('href', rewriteUrl(href, originUrl, proxyBase, tl))
      },
    })
    .on('form[action]', {
      element(el) {
        const action = el.getAttribute('action')
        if (action) el.setAttribute('action', rewriteUrl(action, originUrl, proxyBase, tl))
      },
    })
    .on('body', {
      element(el) {
        el.onEndTag((end) => {
          // Inject detected source language
          end.before(`<script>window._tlSL=${JSON.stringify(detectedSl || 'en')}</script>`, { html: true })
          // Learner script (stays in cached HTML)
          end.before(buildLearnerScript(proxyBase, tl, detectedSl || 'en'), { html: true })
          // Translate fallback (removes itself)
          end.before(buildTranslateScript(originUrl.toString(), proxyBase, tl), { html: true })
        })
      },
    })

  for (const tag of SKIP_SELECTORS) {
    rewriter = rewriter.on(tag, {
      element(el) {
        skipDepth++
        el.onEndTag(() => { skipDepth-- })
      },
    })
  }

  for (const tag of TRANSLATABLE_SELECTORS) {
    rewriter = rewriter.on(tag, {
      element(el) {
        if (skipDepth > 0) return
        if (el.getAttribute('translate') === 'no') {
          skipDepth++
          el.onEndTag(() => { skipDepth-- })
          return
        }
        const cls = el.getAttribute('class') || ''
        if (cls.split(/\s+/).includes('notranslate')) {
          skipDepth++
          el.onEndTag(() => { skipDepth-- })
          return
        }
        // Mark as translatable block for learner popup
        el.setAttribute('class', cls ? cls + ' tl-block' : 'tl-block')
      },

      async text(text) {
        if (skipDepth > 0) return

        textBuffer.push(text.text)

        if (!text.lastInTextNode) {
          text.remove()
          return
        }

        const fullText = textBuffer.splice(0).join('')
        if (!fullText.trim() || /^[\s\d\p{P}\p{S}]+$/u.test(fullText.trim())) return

        const escaped = escapeAttr(fullText)

        // Once rate-limited, skip remaining API calls
        if (hasFailed) {
          text.replace(`<span class="tl-seg" data-tp="1" data-orig="${escaped}">${fullText}</span>`, { html: true })
          return
        }

        const result = await translateChunk(fullText, sl, tl)
        if (result.ok) {
          if (result.src && !detectedSl) detectedSl = result.src
          text.replace(`<span class="tl-seg" data-orig="${escaped}">${result.text}</span>`, { html: true })
        } else {
          hasFailed = true
          text.replace(`<span class="tl-seg" data-tp="1" data-orig="${escaped}">${fullText}</span>`, { html: true })
        }
      },
    })
  }

  return rewriter
}
