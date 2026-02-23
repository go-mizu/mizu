import { icons, cardIcon } from '../icons'

export const contactPage = `
<h2>Contact</h2>

<div class="cards">
  <div class="card">
    <div class="card-ic">${cardIcon('github')} <span>GitHub Issues</span></div>
    <p>Bug reports, feature requests, and technical questions. This is the primary channel.</p>
    <span class="card-lk"><a href="https://github.com/nicholasgasior/gopher-crawl/issues">Open an issue</a> ${icons.externalLink}</span>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('users')} <span>Discord</span></div>
    <p>Community discussion, quick questions, and real-time chat.</p>
    <span class="card-lk"><a href="https://discord.gg/openindex">Join Discord</a> ${icons.externalLink}</span>
  </div>
  <div class="card">
    <div class="card-ic">${cardIcon('mail')} <span>Email</span></div>
    <p>For anything that does not fit GitHub or Discord.</p>
    <span class="card-lk"><a href="mailto:hello@openindex.org">hello@openindex.org</a></span>
  </div>
</div>

<hr>

<h3>Send a Message</h3>

<form>
  <div class="form-group">
    <label for="name">Name</label>
    <input type="text" id="name" name="name" class="form-input" placeholder="Your name" required>
  </div>
  <div class="form-group">
    <label for="email">Email</label>
    <input type="email" id="email" name="email" class="form-input" placeholder="you@example.com" required>
  </div>
  <div class="form-group">
    <label for="message">Message</label>
    <textarea id="message" name="message" class="form-input" placeholder="What's on your mind?" rows="5" required></textarea>
  </div>
  <button type="submit" class="btn btn-p" style="border:none;cursor:pointer">Send</button>
</form>
`
