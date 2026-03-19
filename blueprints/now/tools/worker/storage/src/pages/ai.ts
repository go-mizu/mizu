/**
 * storage.now — AI page
 * Practical guide with real logos and polished layout.
 */

export function aiPage(): string {
  return `<!DOCTYPE html>
<html lang="en" class="dark">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>AI &mdash; storage.now</title>
<meta name="description" content="Connect Claude and ChatGPT to your files. Share between AIs and with friends.">
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800;900&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">
<style>
*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
:root{
  --bg:#FAFAF9;--surface:#FFF;--surface-alt:#F4F4F5;
  --text:#09090B;--text-2:#52525B;--text-3:#A1A1AA;
  --border:#E4E4E7;--ink:#09090B;
  --shadow:0 1px 3px rgba(0,0,0,0.04),0 1px 2px rgba(0,0,0,0.06);
  --glow:rgba(9,9,11,0.03);
}
html.dark{
  --bg:#09090B;--surface:#18181B;--surface-alt:#18181B;
  --text:#FAFAF9;--text-2:#A1A1AA;--text-3:#52525B;
  --border:#27272A;--ink:#FAFAF9;
  --shadow:0 1px 3px rgba(0,0,0,0.3);
  --glow:rgba(250,250,249,0.02);
}
body{font-family:'Inter',system-ui,sans-serif;color:var(--text);background:var(--bg);
  -webkit-font-smoothing:antialiased;overflow-x:hidden}
a{color:inherit;text-decoration:none}
code{font-family:'JetBrains Mono',monospace;font-size:12px;padding:2px 8px;
  border:1px solid var(--border);background:var(--surface-alt)}

@keyframes fadeUp{from{opacity:0;transform:translateY(30px)}to{opacity:1;transform:none}}
@keyframes shimmer{0%{background-position:200% center}100%{background-position:-200% center}}
@keyframes pulse{0%,100%{opacity:.3}50%{opacity:1}}
@keyframes dash{to{stroke-dashoffset:0}}
@keyframes popIn{from{opacity:0;transform:scale(.92)}to{opacity:1;transform:scale(1)}}

.shimmer{background:linear-gradient(90deg,var(--text) 20%,var(--text-3) 40%,var(--text-3) 60%,var(--text) 80%);
  background-size:200% auto;-webkit-background-clip:text;-webkit-text-fill-color:transparent;
  background-clip:text;animation:shimmer 4s ease-in-out infinite}

/* ── Grid bg ── */
.grid-bg{position:fixed;inset:0;pointer-events:none;z-index:0;
  background-image:radial-gradient(circle,var(--border) 1px,transparent 1px);
  background-size:24px 24px;opacity:.35}
html.dark .grid-bg{opacity:.12}

/* ── Nav ── */
nav{position:sticky;top:0;z-index:100;width:100%;
  background:color-mix(in srgb,var(--bg) 80%,transparent);
  backdrop-filter:blur(16px);-webkit-backdrop-filter:blur(16px)}
.nav-inner{max-width:1400px;margin:0 auto;padding:0 60px;height:56px;
  display:flex;align-items:center;justify-content:space-between}
.logo{font-family:'JetBrains Mono',monospace;font-weight:500;font-size:14px;
  letter-spacing:-0.3px;display:flex;align-items:center;gap:8px}
.logo-dot{width:6px;height:6px;background:var(--text);display:inline-block}
.nav-links{display:flex;gap:28px}
.nav-links a{font-size:13px;color:var(--text-3);transition:color .15s;font-weight:500}
.nav-links a:hover,.nav-links a.active{color:var(--text)}
.nav-right{display:flex;align-items:center;gap:12px}
.theme-toggle{background:none;border:1px solid var(--border);padding:6px 10px;cursor:pointer;
  color:var(--text-3);display:flex;align-items:center;transition:all .15s}
.theme-toggle:hover{color:var(--text);border-color:var(--text-3)}
.theme-toggle .icon-sun{display:none}
.theme-toggle .icon-moon{display:block}
html.dark .theme-toggle .icon-sun{display:block}
html.dark .theme-toggle .icon-moon{display:none}
.mobile-toggle{display:none;background:none;border:none;color:var(--text-3);cursor:pointer;padding:4px}

/* ── Sections ── */
.s{position:relative;z-index:1;width:100%;opacity:0;transform:translateY(24px);
  transition:opacity .7s cubic-bezier(.16,1,.3,1),transform .7s cubic-bezier(.16,1,.3,1)}
.s.visible{opacity:1;transform:none}
.s-inner{max-width:1100px;margin:0 auto;padding:0 48px}
.s-label{font-family:'JetBrains Mono',monospace;font-size:11px;letter-spacing:2.5px;
  color:var(--text-3);margin-bottom:16px;font-weight:500;text-transform:uppercase}

/* ── Hero ── */
.hero{padding:120px 0 40px;text-align:center;animation:fadeUp .8s ease both}
.hero .s-inner{display:flex;flex-direction:column;align-items:center;max-width:1100px}
.hero h1{font-size:48px;font-weight:500;line-height:1.1;letter-spacing:-1.5px;margin-bottom:20px}
.hero p{font-size:16px;color:var(--text-2);line-height:1.7;max-width:480px;margin:0 auto}

/* ── Hero diagram ── */
.hero-diagram{display:flex;align-items:center;justify-content:center;gap:0;
  margin-top:64px;padding:48px 0;width:100%;position:relative}
.hd-node{display:flex;flex-direction:column;align-items:center;gap:14px;position:relative;z-index:2}
.hd-icon{width:80px;height:80px;border:1px solid var(--border);background:var(--surface);
  display:flex;align-items:center;justify-content:center;
  transition:all .3s ease;box-shadow:var(--shadow)}
.hd-icon:hover{border-color:var(--text-3);transform:translateY(-4px);
  box-shadow:0 8px 30px var(--glow)}
.hd-icon svg{width:36px;height:36px;fill:var(--text)}
.hd-name{font-size:13px;font-weight:500;letter-spacing:-0.2px}
.hd-connector{display:flex;align-items:center;width:100px;position:relative;z-index:1}
.hd-connector svg{width:100%;height:24px}
.hd-connector line{stroke:var(--border);stroke-width:1;stroke-dasharray:6 4}
html.dark .hd-connector line{stroke:var(--text-3)}
.hd-center{position:relative}
.hd-center .hd-icon{width:96px;height:96px;border-width:2px;background:var(--ink)}
.hd-center .hd-icon svg{fill:var(--bg)}
.hd-center .hd-name{font-weight:500}
.hd-glow{position:absolute;inset:-40px;background:radial-gradient(circle,var(--glow),transparent 70%);
  pointer-events:none;z-index:0}

/* ── Connect cards ── */
.connect{padding:80px 0;border-top:1px solid var(--border)}
.connect-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:1px;
  background:var(--border);border:1px solid var(--border);margin-top:40px}
.cc{background:var(--bg);padding:40px 32px;display:flex;flex-direction:column;
  transition:background .2s}
.cc:hover{background:var(--surface)}
.cc-header{display:flex;align-items:center;gap:14px;margin-bottom:28px;padding-bottom:20px;
  border-bottom:1px solid var(--border)}
.cc-logo{width:40px;height:40px;display:flex;align-items:center;justify-content:center;flex-shrink:0}
.cc-logo svg{width:28px;height:28px;fill:var(--text)}
.cc-name{font-size:17px;font-weight:500;letter-spacing:-0.3px}
.cc-steps{display:flex;flex-direction:column;gap:14px;flex:1}
.cs{display:flex;gap:12px;align-items:flex-start;font-size:14px;line-height:1.65;color:var(--text-2)}
.cs-n{font-family:'JetBrains Mono',monospace;font-size:11px;font-weight:500;
  color:var(--text-3);width:24px;height:24px;border:1px solid var(--border);
  display:flex;align-items:center;justify-content:center;flex-shrink:0;margin-top:2px}
.cs strong{color:var(--text);font-weight:500}
.cs-url{font-family:'JetBrains Mono',monospace;font-size:11px;padding:8px 12px;
  border:1px solid var(--border);background:var(--surface-alt);display:block;
  margin-top:6px;color:var(--text);user-select:all;cursor:pointer;word-break:break-all;
  transition:border-color .15s}
.cs-url:hover{border-color:var(--text-3)}

/* ── Scenario sections ── */
.scenario{padding:80px 0;border-top:1px solid var(--border)}
.sc-layout{display:grid;grid-template-columns:1fr 1.2fr;gap:48px;align-items:start;margin-top:40px}
.sc-text{}
.sc-title{font-size:28px;font-weight:500;letter-spacing:-0.5px;line-height:1.2;margin-bottom:16px}
.sc-sub{font-size:15px;color:var(--text-2);line-height:1.7}
.flow{border:1px solid var(--border);background:var(--surface);overflow:hidden;
  box-shadow:var(--shadow)}
html.dark .flow{box-shadow:0 8px 32px rgba(0,0,0,0.3)}
.flow-bar{padding:12px 20px;border-bottom:1px solid var(--border);background:var(--surface-alt);
  display:flex;align-items:center;gap:8px}
.flow-bar-dots{display:flex;gap:5px}
.flow-bar-dots span{width:8px;height:8px;background:var(--border)}
html.dark .flow-bar-dots span{background:var(--text-3)}
.flow-bar-title{font-family:'JetBrains Mono',monospace;font-size:11px;color:var(--text-3);
  flex:1;text-align:center}
.fs{padding:20px 24px;display:flex;align-items:flex-start;gap:14px;
  border-bottom:1px solid var(--border);font-size:14px;line-height:1.7;
  transition:background .15s}
.fs:last-child{border-bottom:none}
.fs:hover{background:var(--surface-alt)}
.fs-avatar{width:32px;height:32px;border:1px solid var(--border);
  display:flex;align-items:center;justify-content:center;flex-shrink:0;
  font-family:'JetBrains Mono',monospace;font-size:10px;font-weight:500;
  color:var(--text-3);letter-spacing:0.5px}
.fs-avatar.you{background:var(--ink);color:var(--bg);border-color:var(--ink)}
.fs-body{color:var(--text-2);flex:1}
.fs-body strong{color:var(--text);font-weight:500}
.fs-body em{color:var(--text-3);font-style:normal;font-family:'JetBrains Mono',monospace;font-size:11px}

/* ── Cross-platform diagram ── */
.xp-diagram{display:flex;align-items:stretch;justify-content:center;gap:0;
  border:1px solid var(--border);background:var(--surface);overflow:hidden;margin-top:40px;
  box-shadow:var(--shadow)}
html.dark .xp-diagram{box-shadow:0 8px 32px rgba(0,0,0,0.3)}
.xp-side{flex:1;display:flex;flex-direction:column;align-items:center;justify-content:center;
  padding:48px 24px;gap:16px;transition:background .2s}
.xp-side:hover{background:var(--surface-alt)}
.xp-side svg{width:40px;height:40px;fill:var(--text)}
.xp-side-name{font-size:15px;font-weight:500;letter-spacing:-0.2px}
.xp-mid{width:1px;background:var(--border);position:relative;display:flex;
  align-items:center;justify-content:center}
.xp-mid-badge{position:absolute;background:var(--ink);color:var(--bg);
  font-family:'JetBrains Mono',monospace;font-size:9px;font-weight:500;
  letter-spacing:1px;padding:6px 12px;white-space:nowrap;z-index:2}
.xp-center{flex:1.2;display:flex;flex-direction:column;align-items:center;justify-content:center;
  padding:48px 24px;gap:16px;background:var(--surface-alt);position:relative}
.xp-center-icon{width:56px;height:56px;background:var(--ink);
  display:flex;align-items:center;justify-content:center}
.xp-center-icon svg{width:24px;height:24px;fill:var(--bg)}
.xp-center-name{font-family:'JetBrains Mono',monospace;font-size:14px;font-weight:500}
.xp-center-tag{font-size:12px;color:var(--text-3)}

/* ── Buttons ── */
.btn{font-size:14px;font-weight:500;padding:14px 32px;border:1px solid var(--border);
  display:inline-flex;align-items:center;gap:8px;transition:all .15s;color:var(--text-2);
  cursor:pointer;text-decoration:none}
.btn:hover{border-color:var(--text-3);color:var(--text)}

/* ── Footer ── */
footer{position:relative;z-index:1;border-top:1px solid var(--border);padding:40px 0}
footer .s-inner{display:flex;align-items:center;justify-content:space-between;
  font-size:12px;color:var(--text-3)}
footer a{color:var(--text-3);transition:color .15s}
footer a:hover{color:var(--text)}
.footer-brand{display:flex;align-items:center;gap:8px;font-family:'JetBrains Mono',monospace;
  font-weight:500;font-size:13px}
.footer-links{display:flex;gap:24px}

/* ── Responsive ── */
@media(max-width:1024px){
  .connect-grid{grid-template-columns:1fr}
  .sc-layout{grid-template-columns:1fr;gap:32px}
}
@media(max-width:768px){
  .hero{padding:80px 0 32px}
  .hero h1{font-size:36px;letter-spacing:-1px}
  .hero-diagram{flex-direction:column;gap:12px;margin-top:40px;padding:32px 0}
  .hd-connector{width:24px;height:40px;transform:rotate(90deg)}
  .hd-icon{width:64px;height:64px}
  .hd-center .hd-icon{width:72px;height:72px}
  .hd-icon svg{width:28px;height:28px}
  .sc-title{font-size:24px;letter-spacing:-0.3px}
  .xp-diagram{flex-direction:column}
  .xp-mid{width:100%;height:1px}
  .xp-side,.xp-center{padding:32px 24px}
}
@media(max-width:640px){
  nav{padding:0 20px;height:52px}
  .nav-links{display:none;position:absolute;top:52px;left:0;right:0;
    flex-direction:column;padding:16px 20px;gap:16px;z-index:100;
    border-bottom:1px solid var(--border);
    background:color-mix(in srgb,var(--bg) 95%,transparent);
    backdrop-filter:blur(12px);-webkit-backdrop-filter:blur(12px)}
  .nav-links.open{display:flex}
  .mobile-toggle{display:block}
  .s-inner{padding:0 20px}
  .hero{padding:56px 0 24px}
  .hero h1{font-size:28px;letter-spacing:-0.5px}
  .hero p{font-size:16px}
  .connect,.scenario{padding:56px 0}
  .cc{padding:28px 20px}
  .cs{font-size:13px}
  .fs{padding:16px 20px;font-size:13px}
  footer .s-inner{flex-direction:column;gap:16px;text-align:center}
}
</style>
</head>
<body>

<div class="grid-bg"></div>

<nav>
  <div class="nav-inner">
    <a href="/" class="logo"><span class="logo-dot"></span> storage.now</a>
    <button class="mobile-toggle" onclick="document.querySelector('.nav-links').classList.toggle('open')" aria-label="Menu">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/></svg>
    </button>
    <div class="nav-links">
      <a href="/browse">browse</a>
      <a href="/developers">developers</a>
      <a href="/ai" class="active">ai</a>
      <a href="/api">api</a>
      <a href="/pricing">pricing</a>
    </div>
    <div class="nav-right">
      <button class="theme-toggle" onclick="toggleTheme()" aria-label="Toggle theme">
        <svg class="icon-moon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z"/></svg>
        <svg class="icon-sun" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>
      </button>
    </div>
  </div>
</nav>

<!-- ═══ HERO ═══ -->
<div class="hero">
  <div class="s-inner">
    <h1>Your files, inside<br><span class="shimmer">Claude &amp; ChatGPT.</span></h1>
    <p>Connect once. Then just ask your AI to save files, find things, and share links with anyone.</p>

    <div class="hero-diagram">
      <div class="hd-node">
        <div class="hd-icon">
          <svg viewBox="0 0 24 24"><path d="m4.7144 15.9555 4.7174-2.6471.079-.2307-.079-.1275h-.2307l-.7893-.0486-2.6956-.0729-2.3375-.0971-2.2646-.1214-.5707-.1215-.5343-.7042.0546-.3522.4797-.3218.686.0608 1.5179.1032 2.2767.1578 1.6514.0972 2.4468.255h.3886l.0546-.1579-.1336-.0971-.1032-.0972L6.973 9.8356l-2.55-1.6879-1.3356-.9714-.7225-.4918-.3643-.4614-.1578-1.0078.6557-.7225.8803.0607.2246.0607.8925.686 1.9064 1.4754 2.4893 1.8336.3643.3035.1457-.1032.0182-.0728-.164-.2733-1.3539-2.4467-1.445-2.4893-.6435-1.032-.17-.6194c-.0607-.255-.1032-.4674-.1032-.7285L6.287.1335 6.6997 0l.9957.1336.419.3642.6192 1.4147 1.0018 2.2282 1.5543 3.0296.4553.8985.2429.8318.091.255h.1579v-.1457l.1275-1.706.2368-2.0947.2307-2.6957.0789-.7589.3764-.9107.7468-.4918.5828.2793.4797.686-.0668.4433-.2853 1.8517-.5586 2.9021-.3643 1.9429h.2125l.2429-.2429.9835-1.3053 1.6514-2.0643.7286-.8196.85-.9046.5464-.4311h1.0321l.759 1.1293-.34 1.1657-1.0625 1.3478-.8804 1.1414-1.2628 1.7-.7893 1.36.0729.1093.1882-.0183 2.8535-.607 1.5421-.2794 1.8396-.3157.8318.3886.091.3946-.3278.8075-1.967.4857-2.3072.4614-3.4364.8136-.0425.0304.0486.0607 1.5482.1457.6618.0364h1.621l3.0175.2247.7892.522.4736.6376-.079.4857-1.2142.6193-1.6393-.3886-3.825-.9107-1.3113-.3279h-.1822v.1093l1.0929 1.0686 2.0035 1.8092 2.5075 2.3314.1275.5768-.3218.4554-.34-.0486-2.2039-1.6575-.85-.7468-1.9246-1.621h-.1275v.17l.4432.6496 2.3436 3.5214.1214 1.0807-.17.3521-.6071.2125-.6679-.1214-1.3721-1.9246L14.38 17.959l-1.1414-1.9428-.1397.079-.674 7.2552-.3156.3703-.7286.2793-.6071-.4614-.3218-.7468.3218-1.4753.3886-1.9246.3157-1.53.2853-1.9004.17-.6314-.0121-.0425-.1397.0182-1.4328 1.9672-2.1796 2.9446-1.7243 1.8456-.4128.164-.7164-.3704.0667-.6618.4008-.5889 2.386-3.0357 1.4389-1.882.929-1.0868-.0062-.1579h-.0546l-6.3385 4.1164-1.1293.1457-.4857-.4554.0608-.7467.2307-.2429 1.9064-1.3114Z"/></svg>
        </div>
        <div class="hd-name">Claude</div>
      </div>

      <div class="hd-connector">
        <svg viewBox="0 0 100 24"><line x1="0" y1="12" x2="100" y2="12"/></svg>
      </div>

      <div class="hd-node hd-center">
        <div class="hd-glow"></div>
        <div class="hd-icon">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="3" width="18" height="18"/><line x1="8" y1="9" x2="16" y2="9"/><line x1="8" y1="13" x2="13" y2="13"/></svg>
        </div>
        <div class="hd-name">storage.now</div>
      </div>

      <div class="hd-connector">
        <svg viewBox="0 0 100 24"><line x1="0" y1="12" x2="100" y2="12"/></svg>
      </div>

      <div class="hd-node">
        <div class="hd-icon">
          <svg viewBox="0 0 24 24"><path d="M22.2819 9.8211a5.9847 5.9847 0 0 0-.5157-4.9108 6.0462 6.0462 0 0 0-6.5098-2.9A6.0651 6.0651 0 0 0 4.9807 4.1818a5.9847 5.9847 0 0 0-3.9977 2.9 6.0462 6.0462 0 0 0 .7427 7.0966 5.98 5.98 0 0 0 .511 4.9107 6.051 6.051 0 0 0 6.5146 2.9001A5.9847 5.9847 0 0 0 13.2599 24a6.0557 6.0557 0 0 0 5.7718-4.2058 5.9894 5.9894 0 0 0 3.9977-2.9001 6.0557 6.0557 0 0 0-.7475-7.0729zm-9.022 12.6081a4.4755 4.4755 0 0 1-2.8764-1.0408l.1419-.0804 4.7783-2.7582a.7948.7948 0 0 0 .3927-.6813v-6.7369l2.02 1.1686a.071.071 0 0 1 .038.052v5.5826a4.504 4.504 0 0 1-4.4945 4.4944zm-9.6607-4.1254a4.4708 4.4708 0 0 1-.5346-3.0137l.142.0852 4.783 2.7582a.7712.7712 0 0 0 .7806 0l5.8428-3.3685v2.3324a.0804.0804 0 0 1-.0332.0615L9.74 19.9502a4.4992 4.4992 0 0 1-6.1408-1.6464zM2.3408 7.8956a4.485 4.485 0 0 1 2.3655-1.9728V11.6a.7664.7664 0 0 0 .3879.6765l5.8144 3.3543-2.0201 1.1685a.0757.0757 0 0 1-.071 0l-4.8303-2.7865A4.504 4.504 0 0 1 2.3408 7.872zm16.5963 3.8558L13.1038 8.364 15.1192 7.2a.0757.0757 0 0 1 .071 0l4.8303 2.7913a4.4944 4.4944 0 0 1-.6765 8.1042v-5.6772a.79.79 0 0 0-.407-.667zm2.0107-3.0231l-.142-.0852-4.7735-2.7818a.7759.7759 0 0 0-.7854 0L9.409 9.2297V6.8974a.0662.0662 0 0 1 .0284-.0615l4.8303-2.7866a4.4992 4.4992 0 0 1 6.6802 4.66zM8.3065 12.863l-2.02-1.1638a.0804.0804 0 0 1-.038-.0567V6.0742a4.4992 4.4992 0 0 1 7.3757-3.4537l-.142.0805L8.704 5.459a.7948.7948 0 0 0-.3927.6813zm1.0976-2.3654l2.602-1.4998 2.6069 1.4998v2.9994l-2.5974 1.4997-2.6067-1.4997Z"/></svg>
        </div>
        <div class="hd-name">ChatGPT</div>
      </div>
    </div>
  </div>
</div>

<!-- ═══ CONNECT ═══ -->
<div class="s connect" id="connect">
  <div class="s-inner">
    <div class="s-label">GET CONNECTED IN 30 SECONDS</div>

    <div class="connect-grid">

      <div class="cc">
        <div class="cc-header">
          <div class="cc-logo">
            <svg viewBox="0 0 24 24"><path d="m4.7144 15.9555 4.7174-2.6471.079-.2307-.079-.1275h-.2307l-.7893-.0486-2.6956-.0729-2.3375-.0971-2.2646-.1214-.5707-.1215-.5343-.7042.0546-.3522.4797-.3218.686.0608 1.5179.1032 2.2767.1578 1.6514.0972 2.4468.255h.3886l.0546-.1579-.1336-.0971-.1032-.0972L6.973 9.8356l-2.55-1.6879-1.3356-.9714-.7225-.4918-.3643-.4614-.1578-1.0078.6557-.7225.8803.0607.2246.0607.8925.686 1.9064 1.4754 2.4893 1.8336.3643.3035.1457-.1032.0182-.0728-.164-.2733-1.3539-2.4467-1.445-2.4893-.6435-1.032-.17-.6194c-.0607-.255-.1032-.4674-.1032-.7285L6.287.1335 6.6997 0l.9957.1336.419.3642.6192 1.4147 1.0018 2.2282 1.5543 3.0296.4553.8985.2429.8318.091.255h.1579v-.1457l.1275-1.706.2368-2.0947.2307-2.6957.0789-.7589.3764-.9107.7468-.4918.5828.2793.4797.686-.0668.4433-.2853 1.8517-.5586 2.9021-.3643 1.9429h.2125l.2429-.2429.9835-1.3053 1.6514-2.0643.7286-.8196.85-.9046.5464-.4311h1.0321l.759 1.1293-.34 1.1657-1.0625 1.3478-.8804 1.1414-1.2628 1.7-.7893 1.36.0729.1093.1882-.0183 2.8535-.607 1.5421-.2794 1.8396-.3157.8318.3886.091.3946-.3278.8075-1.967.4857-2.3072.4614-3.4364.8136-.0425.0304.0486.0607 1.5482.1457.6618.0364h1.621l3.0175.2247.7892.522.4736.6376-.079.4857-1.2142.6193-1.6393-.3886-3.825-.9107-1.3113-.3279h-.1822v.1093l1.0929 1.0686 2.0035 1.8092 2.5075 2.3314.1275.5768-.3218.4554-.34-.0486-2.2039-1.6575-.85-.7468-1.9246-1.621h-.1275v.17l.4432.6496 2.3436 3.5214.1214 1.0807-.17.3521-.6071.2125-.6679-.1214-1.3721-1.9246L14.38 17.959l-1.1414-1.9428-.1397.079-.674 7.2552-.3156.3703-.7286.2793-.6071-.4614-.3218-.7468.3218-1.4753.3886-1.9246.3157-1.53.2853-1.9004.17-.6314-.0121-.0425-.1397.0182-1.4328 1.9672-2.1796 2.9446-1.7243 1.8456-.4128.164-.7164-.3704.0667-.6618.4008-.5889 2.386-3.0357 1.4389-1.882.929-1.0868-.0062-.1579h-.0546l-6.3385 4.1164-1.1293.1457-.4857-.4554.0608-.7467.2307-.2429 1.9064-1.3114Z"/></svg>
          </div>
          <div class="cc-name">Claude.ai</div>
        </div>
        <div class="cc-steps">
          <div class="cs"><span class="cs-n">1</span> <span><strong>Settings &rarr; Integrations</strong></span></div>
          <div class="cs"><span class="cs-n">2</span> <span>Click <strong>Add custom integration</strong></span></div>
          <div class="cs"><span class="cs-n">3</span> <span>Paste URL:<span class="cs-url">https://storage.liteio.dev/mcp</span></span></div>
          <div class="cs"><span class="cs-n">4</span> <span>Sign in with email &mdash; <strong>done</strong></span></div>
        </div>
      </div>

      <div class="cc">
        <div class="cc-header">
          <div class="cc-logo">
            <svg viewBox="0 0 24 24"><path d="M22.2819 9.8211a5.9847 5.9847 0 0 0-.5157-4.9108 6.0462 6.0462 0 0 0-6.5098-2.9A6.0651 6.0651 0 0 0 4.9807 4.1818a5.9847 5.9847 0 0 0-3.9977 2.9 6.0462 6.0462 0 0 0 .7427 7.0966 5.98 5.98 0 0 0 .511 4.9107 6.051 6.051 0 0 0 6.5146 2.9001A5.9847 5.9847 0 0 0 13.2599 24a6.0557 6.0557 0 0 0 5.7718-4.2058 5.9894 5.9894 0 0 0 3.9977-2.9001 6.0557 6.0557 0 0 0-.7475-7.0729zm-9.022 12.6081a4.4755 4.4755 0 0 1-2.8764-1.0408l.1419-.0804 4.7783-2.7582a.7948.7948 0 0 0 .3927-.6813v-6.7369l2.02 1.1686a.071.071 0 0 1 .038.052v5.5826a4.504 4.504 0 0 1-4.4945 4.4944zm-9.6607-4.1254a4.4708 4.4708 0 0 1-.5346-3.0137l.142.0852 4.783 2.7582a.7712.7712 0 0 0 .7806 0l5.8428-3.3685v2.3324a.0804.0804 0 0 1-.0332.0615L9.74 19.9502a4.4992 4.4992 0 0 1-6.1408-1.6464zM2.3408 7.8956a4.485 4.485 0 0 1 2.3655-1.9728V11.6a.7664.7664 0 0 0 .3879.6765l5.8144 3.3543-2.0201 1.1685a.0757.0757 0 0 1-.071 0l-4.8303-2.7865A4.504 4.504 0 0 1 2.3408 7.872zm16.5963 3.8558L13.1038 8.364 15.1192 7.2a.0757.0757 0 0 1 .071 0l4.8303 2.7913a4.4944 4.4944 0 0 1-.6765 8.1042v-5.6772a.79.79 0 0 0-.407-.667zm2.0107-3.0231l-.142-.0852-4.7735-2.7818a.7759.7759 0 0 0-.7854 0L9.409 9.2297V6.8974a.0662.0662 0 0 1 .0284-.0615l4.8303-2.7866a4.4992 4.4992 0 0 1 6.6802 4.66zM8.3065 12.863l-2.02-1.1638a.0804.0804 0 0 1-.038-.0567V6.0742a4.4992 4.4992 0 0 1 7.3757-3.4537l-.142.0805L8.704 5.459a.7948.7948 0 0 0-.3927.6813zm1.0976-2.3654l2.602-1.4998 2.6069 1.4998v2.9994l-2.5974 1.4997-2.6067-1.4997Z"/></svg>
          </div>
          <div class="cc-name">ChatGPT</div>
        </div>
        <div class="cc-steps">
          <div class="cs"><span class="cs-n">1</span> <span><strong>Settings &rarr; Connected apps</strong></span></div>
          <div class="cs"><span class="cs-n">2</span> <span>Click <strong>Add app &rarr; Add by URL</strong></span></div>
          <div class="cs"><span class="cs-n">3</span> <span>Paste URL:<span class="cs-url">https://storage.liteio.dev/mcp</span></span></div>
          <div class="cs"><span class="cs-n">4</span> <span>Sign in with email &mdash; <strong>done</strong></span></div>
        </div>
      </div>

      <div class="cc">
        <div class="cc-header">
          <div class="cc-logo">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="2" width="18" height="14"/><line x1="8" y1="22" x2="16" y2="22"/><line x1="12" y1="16" x2="12" y2="22"/><path d="M7 8l3 3 7-7" stroke-width="1.5"/></svg>
          </div>
          <div class="cc-name">Claude Desktop</div>
        </div>
        <div class="cc-steps">
          <div class="cs"><span class="cs-n">1</span> <span><strong>Settings &rarr; Developer &rarr; Edit Config</strong></span></div>
          <div class="cs"><span class="cs-n">2</span> <span>Add to <code>mcpServers</code>:<span class="cs-url">{ "storage": { "command": "npx", "args": ["-y", "mcp-remote", "https://storage.liteio.dev/mcp"] } }</span></span></div>
          <div class="cs"><span class="cs-n">3</span> <span>Restart Claude Desktop &mdash; <strong>done</strong></span></div>
        </div>
      </div>

    </div>
  </div>
</div>

<!-- ═══ SHARE WITH FRIENDS ═══ -->
<div class="s scenario" id="share">
  <div class="s-inner">
    <div class="s-label">SHARE WITH FRIENDS</div>
    <div class="sc-layout">
      <div class="sc-text">
        <div class="sc-title">Send any file<br>with one sentence.</div>
        <div class="sc-sub">Ask your AI to share a file. You get a link. Send it to anyone &mdash; they open it in their browser. No sign-up needed.</div>
      </div>
      <div class="flow">
        <div class="flow-bar"><div class="flow-bar-dots"><span></span><span></span><span></span></div><div class="flow-bar-title">conversation</div></div>
        <div class="fs">
          <div class="fs-avatar you">YOU</div>
          <div class="fs-body"><strong>&ldquo;Share my vacation-photo.jpg&rdquo;</strong></div>
        </div>
        <div class="fs">
          <div class="fs-avatar">AI</div>
          <div class="fs-body">Here&rsquo;s your link: <strong>storage.now/p/a8f3c7&hellip;</strong><br>Anyone can view and download it.</div>
        </div>
        <div class="fs">
          <div class="fs-avatar">FRI</div>
          <div class="fs-body">Opens link &rarr; sees photo &rarr; downloads it.<br><em>No account needed.</em></div>
        </div>
      </div>
    </div>
  </div>
</div>

<!-- ═══ CROSS-PLATFORM ═══ -->
<div class="s scenario" id="cross">
  <div class="s-inner">
    <div class="s-label">CROSS-PLATFORM</div>
    <div class="sc-layout">
      <div class="sc-text">
        <div class="sc-title">ChatGPT and Claude,<br>same files.</div>
        <div class="sc-sub">Save a file from ChatGPT, find it in Claude. Both AIs share the same storage &mdash; your files follow you across platforms.</div>
      </div>
      <div>
        <div class="xp-diagram">
          <div class="xp-side">
            <svg viewBox="0 0 24 24"><path d="M22.2819 9.8211a5.9847 5.9847 0 0 0-.5157-4.9108 6.0462 6.0462 0 0 0-6.5098-2.9A6.0651 6.0651 0 0 0 4.9807 4.1818a5.9847 5.9847 0 0 0-3.9977 2.9 6.0462 6.0462 0 0 0 .7427 7.0966 5.98 5.98 0 0 0 .511 4.9107 6.051 6.051 0 0 0 6.5146 2.9001A5.9847 5.9847 0 0 0 13.2599 24a6.0557 6.0557 0 0 0 5.7718-4.2058 5.9894 5.9894 0 0 0 3.9977-2.9001 6.0557 6.0557 0 0 0-.7475-7.0729zm-9.022 12.6081a4.4755 4.4755 0 0 1-2.8764-1.0408l.1419-.0804 4.7783-2.7582a.7948.7948 0 0 0 .3927-.6813v-6.7369l2.02 1.1686a.071.071 0 0 1 .038.052v5.5826a4.504 4.504 0 0 1-4.4945 4.4944zm-9.6607-4.1254a4.4708 4.4708 0 0 1-.5346-3.0137l.142.0852 4.783 2.7582a.7712.7712 0 0 0 .7806 0l5.8428-3.3685v2.3324a.0804.0804 0 0 1-.0332.0615L9.74 19.9502a4.4992 4.4992 0 0 1-6.1408-1.6464zM2.3408 7.8956a4.485 4.485 0 0 1 2.3655-1.9728V11.6a.7664.7664 0 0 0 .3879.6765l5.8144 3.3543-2.0201 1.1685a.0757.0757 0 0 1-.071 0l-4.8303-2.7865A4.504 4.504 0 0 1 2.3408 7.872zm16.5963 3.8558L13.1038 8.364 15.1192 7.2a.0757.0757 0 0 1 .071 0l4.8303 2.7913a4.4944 4.4944 0 0 1-.6765 8.1042v-5.6772a.79.79 0 0 0-.407-.667zm2.0107-3.0231l-.142-.0852-4.7735-2.7818a.7759.7759 0 0 0-.7854 0L9.409 9.2297V6.8974a.0662.0662 0 0 1 .0284-.0615l4.8303-2.7866a4.4992 4.4992 0 0 1 6.6802 4.66zM8.3065 12.863l-2.02-1.1638a.0804.0804 0 0 1-.038-.0567V6.0742a4.4992 4.4992 0 0 1 7.3757-3.4537l-.142.0805L8.704 5.459a.7948.7948 0 0 0-.3927.6813zm1.0976-2.3654l2.602-1.4998 2.6069 1.4998v2.9994l-2.5974 1.4997-2.6067-1.4997Z"/></svg>
            <div class="xp-side-name">ChatGPT</div>
          </div>
          <div class="xp-mid"><div class="xp-mid-badge">SYNC</div></div>
          <div class="xp-center">
            <div class="xp-center-icon">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="3" width="18" height="18"/><line x1="8" y1="9" x2="16" y2="9"/><line x1="8" y1="13" x2="13" y2="13"/></svg>
            </div>
            <div class="xp-center-name">storage.now</div>
            <div class="xp-center-tag">Your files live here</div>
          </div>
          <div class="xp-mid"><div class="xp-mid-badge">SYNC</div></div>
          <div class="xp-side">
            <svg viewBox="0 0 24 24"><path d="m4.7144 15.9555 4.7174-2.6471.079-.2307-.079-.1275h-.2307l-.7893-.0486-2.6956-.0729-2.3375-.0971-2.2646-.1214-.5707-.1215-.5343-.7042.0546-.3522.4797-.3218.686.0608 1.5179.1032 2.2767.1578 1.6514.0972 2.4468.255h.3886l.0546-.1579-.1336-.0971-.1032-.0972L6.973 9.8356l-2.55-1.6879-1.3356-.9714-.7225-.4918-.3643-.4614-.1578-1.0078.6557-.7225.8803.0607.2246.0607.8925.686 1.9064 1.4754 2.4893 1.8336.3643.3035.1457-.1032.0182-.0728-.164-.2733-1.3539-2.4467-1.445-2.4893-.6435-1.032-.17-.6194c-.0607-.255-.1032-.4674-.1032-.7285L6.287.1335 6.6997 0l.9957.1336.419.3642.6192 1.4147 1.0018 2.2282 1.5543 3.0296.4553.8985.2429.8318.091.255h.1579v-.1457l.1275-1.706.2368-2.0947.2307-2.6957.0789-.7589.3764-.9107.7468-.4918.5828.2793.4797.686-.0668.4433-.2853 1.8517-.5586 2.9021-.3643 1.9429h.2125l.2429-.2429.9835-1.3053 1.6514-2.0643.7286-.8196.85-.9046.5464-.4311h1.0321l.759 1.1293-.34 1.1657-1.0625 1.3478-.8804 1.1414-1.2628 1.7-.7893 1.36.0729.1093.1882-.0183 2.8535-.607 1.5421-.2794 1.8396-.3157.8318.3886.091.3946-.3278.8075-1.967.4857-2.3072.4614-3.4364.8136-.0425.0304.0486.0607 1.5482.1457.6618.0364h1.621l3.0175.2247.7892.522.4736.6376-.079.4857-1.2142.6193-1.6393-.3886-3.825-.9107-1.3113-.3279h-.1822v.1093l1.0929 1.0686 2.0035 1.8092 2.5075 2.3314.1275.5768-.3218.4554-.34-.0486-2.2039-1.6575-.85-.7468-1.9246-1.621h-.1275v.17l.4432.6496 2.3436 3.5214.1214 1.0807-.17.3521-.6071.2125-.6679-.1214-1.3721-1.9246L14.38 17.959l-1.1414-1.9428-.1397.079-.674 7.2552-.3156.3703-.7286.2793-.6071-.4614-.3218-.7468.3218-1.4753.3886-1.9246.3157-1.53.2853-1.9004.17-.6314-.0121-.0425-.1397.0182-1.4328 1.9672-2.1796 2.9446-1.7243 1.8456-.4128.164-.7164-.3704.0667-.6618.4008-.5889 2.386-3.0357 1.4389-1.882.929-1.0868-.0062-.1579h-.0546l-6.3385 4.1164-1.1293.1457-.4857-.4554.0608-.7467.2307-.2429 1.9064-1.3114Z"/></svg>
            <div class="xp-side-name">Claude</div>
          </div>
        </div>

        <div class="flow" style="margin-top:2px">
          <div class="flow-bar"><div class="flow-bar-dots"><span></span><span></span><span></span></div><div class="flow-bar-title">example</div></div>
          <div class="fs">
            <div class="fs-avatar you">GPT</div>
            <div class="fs-body"><strong>&ldquo;Save this report as Q1-results.pdf&rdquo;</strong><br>File saved to your storage.</div>
          </div>
          <div class="fs">
            <div class="fs-avatar you">CLA</div>
            <div class="fs-body"><strong>&ldquo;What files do I have?&rdquo;</strong><br>Shows Q1-results.pdf &mdash; same file from ChatGPT.</div>
          </div>
          <div class="fs">
            <div class="fs-avatar you">CLA</div>
            <div class="fs-body"><strong>&ldquo;Share Q1-results.pdf with my team&rdquo;</strong><br>Creates a link anyone can open.</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</div>

<!-- ═══ FOOTER ═══ -->
<footer>
  <div class="s-inner">
    <div class="footer-brand"><span class="logo-dot"></span> storage.now</div>
    <div class="footer-links">
      <a href="/api">api</a>
      <a href="/pricing">pricing</a>
      <a href="/ai">ai</a>
      <a href="/browse">browse</a>
    </div>
  </div>
</footer>

<script>
function toggleTheme(){
  const isDark=document.documentElement.classList.toggle('dark');
  localStorage.setItem('theme',isDark?'dark':'light');
}
(function(){
  const saved=localStorage.getItem('theme');
  if(saved==='light') document.documentElement.classList.remove('dark');
  else if(!saved&&!window.matchMedia('(prefers-color-scheme:dark)').matches) document.documentElement.classList.remove('dark');
})();
(function(){
  const els=document.querySelectorAll('.s');
  if(!els.length) return;
  const obs=new IntersectionObserver((entries)=>{
    entries.forEach(e=>{
      if(e.isIntersecting){e.target.classList.add('visible');obs.unobserve(e.target)}
    });
  },{threshold:0.06,rootMargin:'0px 0px -40px 0px'});
  els.forEach(s=>obs.observe(s));
})();
</script>
</body>
</html>`;
}
