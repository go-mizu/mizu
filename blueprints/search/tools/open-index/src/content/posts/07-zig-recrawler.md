---
slug: zig-recrawler
title: "Raw TCP and TLS in Zig"
date: 2026-02-21
summary: "Raw TCP sockets, manual TLS handshakes, and three bugs that taught more about networking than any textbook."
tags: [engineering, zig]
---

64 kilobytes per worker. That was the number I wanted to hit. The Go recrawler uses goroutines -- lightweight, cheap, managed by the runtime. But each one still carries ~100KB+ of stack, a garbage collector relationship, and a pile of hidden allocations inside `http.Transport`. I wanted to know what happens when you strip all of that away and talk to the network yourself.

So I rewrote the recrawler in Zig. No `std.http.Client` -- it doesn't exist in a useful form anyway. Raw TCP sockets. `std.crypto.tls.Client` for HTTPS. Manual DNS via UDP. Stack-allocated everything. What I got was an education in how TLS and TCP actually work when you can't hide behind abstractions.

## Why not just use Go?

Go works. The Go recrawler handles 50K concurrent workers, processes 2.5M URLs in 65 seconds, and the code is readable by anyone who has written a for loop. I wasn't trying to replace it. I wanted to understand the layers it hides from me -- and see whether predictable memory usage changes anything at scale.

The Zig version was an experiment from the start. Here's what I learned.

## The TLS double-flush bug

The first HTTP request I ever sent from Zig came back as a read timeout. Every single time. I could see the TCP connection succeed. I could see the TLS handshake complete. I sent my carefully formatted `GET / HTTP/1.1\r\n` into the TLS writer. Then I waited for a response, and nothing came. Three-second timeout. Read failed. Every URL.

I spent hours on this. Added logging. Verified the request bytes were correct. Tried different servers. Tried different TLS configurations. Same result: handshake good, request sent, read timeout.

The problem was one missing line. Zig's TLS operates as a layered writer. When you call `tls_client.writer.flush()`, it encrypts your plaintext into TLS records and writes them to the underlying socket writer's *buffer*. That buffer is not the network. It's a userspace staging area. The encrypted bytes sit there, correctly framed, correctly encrypted, going absolutely nowhere.

You need a second flush -- `stream_writer.interface.flush()` -- to actually push the socket buffer out to the network. Without it, your HTTP request is encrypted and buffered, the server receives nothing, and your read always times out.

<pre class="showcase-visual">
<span class="dim">// Build the HTTP request</span>
<span class="blue">const</span> req_slice = std.fmt.bufPrint(&req_buf,
    <span class="green">"{s} {s} HTTP/1.1\\r\\nHost: {s}\\r\\nUser-Agent: {s}\\r\\n"</span> ++
    <span class="green">"Accept: text/html,*/*;q=0.8\\r\\nConnection: close\\r\\n\\r\\n"</span>,
    .{ method, path, host, user_agent },
) <span class="blue">catch</span> <span class="blue">return</span>;

<span class="dim">// Write → encrypt → flush to network (THREE steps)</span>
tls_client.writer.<span class="hl">writeAll</span>(req_slice) <span class="blue">catch</span> <span class="blue">break</span> :blk <span class="blue">false</span>;
tls_client.writer.<span class="hl">flush</span>() <span class="blue">catch</span> <span class="blue">break</span> :blk <span class="blue">false</span>;     <span class="dim">// encrypts plaintext → TLS records → socket buffer</span>
stream_writer.interface.<span class="amber">flush</span>() <span class="blue">catch</span> <span class="blue">break</span> :blk <span class="blue">false</span>; <span class="dim">// pushes socket buffer → actual network</span>
<span class="dim">//                        ^^^^^</span>
<span class="dim">// Without this line, the request never leaves your machine.</span>
<span class="dim">// The server never sees it. The read always times out.</span>
</pre>

The fix is one line. The debugging session was half a day. This is the kind of thing that `http.Transport` in Go handles for you -- an entire class of bugs you never have to think about because someone else already thought about it.

## Why doesn't connect() respect my timeout?

Next problem. I set `SO_SNDTIMEO` on every socket to 3 seconds. Should mean all operations -- connect, send, recv -- time out after 3 seconds, right?

Wrong. On macOS, `SO_SNDTIMEO` does not apply to `connect()`. The kernel uses its own internal timeout -- roughly 75 seconds -- regardless of what you set on the socket. I discovered this the fun way: a benchmark that should have taken 30 seconds was hanging for 15 minutes. Dead domains weren't timing out. They were waiting a full 75 seconds each, sequentially burning through my worker pool.

The fix is the classic non-blocking connect pattern:

1. Set the socket to non-blocking mode with `fcntl(F_SETFL, O_NONBLOCK)`
2. Call `connect()` -- it returns `EINPROGRESS` immediately
3. Use `poll(POLLOUT, timeout_ms)` to wait for the connection with your actual timeout
4. Call `getsockoptError()` to check whether the connect succeeded or failed
5. Restore blocking mode for TLS handshake and subsequent I/O

<pre class="showcase-visual">
<span class="blue">fn</span> <span class="hl">connectToDomain</span>(domain: *<span class="blue">const</span> DomainInfo, port: <span class="blue">u16</span>, timeout_ms: <span class="blue">u32</span>) !posix.socket_t {
    <span class="blue">const</span> sock = <span class="blue">try</span> posix.socket(posix.AF.INET, posix.SOCK.STREAM, posix.IPPROTO.TCP);
    <span class="blue">errdefer</span> posix.close(sock);

    <span class="dim">// Set non-blocking for connect</span>
    <span class="blue">const</span> fl_flags = <span class="blue">try</span> posix.fcntl(sock, posix.F.GETFL, <span class="amber">0</span>);
    _ = <span class="blue">try</span> posix.fcntl(sock, posix.F.SETFL,
        fl_flags | (<span class="amber">1</span> &lt;&lt; @bitOffsetOf(posix.O, <span class="green">"NONBLOCK"</span>)));

    posix.connect(sock, @ptrCast(&addr), @sizeOf(posix.sockaddr.in)) <span class="blue">catch</span> |err| {
        <span class="blue">if</span> (err != error.WouldBlock) <span class="blue">return</span> err;

        <span class="dim">// poll() with our ACTUAL timeout -- not the kernel's 75s default</span>
        <span class="blue">var</span> pfds = [1]posix.pollfd{.{
            .fd = sock,
            .events = posix.POLL.OUT,
            .revents = <span class="amber">0</span>,
        }};
        <span class="blue">const</span> ready = posix.poll(&pfds, @intCast(timeout_ms))
            <span class="blue">catch return</span> error.ConnectionTimedOut;
        <span class="blue">if</span> (ready == <span class="amber">0</span>) <span class="blue">return</span> error.ConnectionTimedOut;

        <span class="dim">// Did it actually connect, or did the connect fail?</span>
        <span class="blue">try</span> posix.getsockoptError(sock);
    };

    <span class="dim">// Restore blocking mode -- TLS handshake needs it</span>
    _ = posix.fcntl(sock, posix.F.SETFL, fl_flags) <span class="blue">catch</span> {};
    <span class="blue">return</span> sock;
}
</pre>

That last step -- restoring blocking mode -- matters more than it looks. The TLS handshake reads and writes multiple rounds of data. If you leave the socket in non-blocking mode, every read returns `EAGAIN` and you have to implement your own retry loop around the entire handshake state machine. Blocking mode lets the TLS client handle its own flow without you managing every partial read.

<div class="note">
  <strong>This is a macOS-specific behavior.</strong> Linux applies <code>SO_SNDTIMEO</code> to connect() as expected. If you only test on Linux, you won't hit this until you try macOS -- and then every dead domain burns 75 seconds of wall clock time.
</div>

## The buffer size that blocks forever

Third bug. After fixing the double-flush and the connect timeout, I had HTTP requests going out and responses coming back. Mostly. Some responses worked perfectly. Others hung on the read -- not timing out, just blocking forever.

The issue was `readSliceShort()` in Zig's I/O reader. Despite the name, it doesn't return a short read the way `recv()` does. It tries to fill the entire buffer you give it. If you pass a 16KB buffer and the server sends a 1.2KB response in a single TLS record, the reader decrypts that record and then waits for more data to fill the remaining 14.8KB. The second TLS record never comes -- the server already sent everything -- and you block.

The fix is embarrassingly simple: use a smaller buffer. With a 1024-byte read buffer, the reader decrypts the first TLS record and returns with whatever it got. You read in a loop, accumulating data until you find the `\r\n\r\n` header terminator or the connection closes.

<pre class="showcase-visual">
<span class="dim">// WRONG: 16KB buffer blocks waiting for second TLS record</span>
<span class="amber">var big_buf: [16384]u8 = undefined;</span>
<span class="amber">const n = reader.readSliceShort(&big_buf);</span>  <span class="dim">// blocks if response &lt; 16KB</span>

<span class="dim">// RIGHT: small buffer returns after first TLS record</span>
<span class="green">var header_buf: [4096]u8 = undefined;</span>
<span class="green">var header_len: usize = 0;</span>
<span class="green">while (header_len &lt; header_buf.len) {</span>
<span class="green">    const n = reader.readSliceShort(</span>
<span class="green">        header_buf[header_len..]</span>
<span class="green">    ) catch return false;</span>
<span class="green">    if (n == 0) break;</span>
<span class="green">    header_len += n;</span>
<span class="green">    if (findHeaderEnd(header_buf[0..header_len])) |_| break;</span>
<span class="green">}</span>
</pre>

This is another case where Go's `bufio.Reader` and the HTTP stack handle the complexity for you. Zig gives you the raw pieces. You have to know that "read" doesn't mean what you think it means when TLS record boundaries are involved.

## What 64KB per worker actually looks like

With all the bugs fixed, here's the memory picture. Each TLS connection needs four buffers, each exactly `tls.max_ciphertext_record_len` (16,384 bytes):

<pre class="showcase-visual">
<span class="dim">// Per-worker stack allocation: 4 x 16KB = 64KB</span>
<span class="blue">var</span> socket_read_buf:  [tls.max_ciphertext_record_len]<span class="blue">u8</span> = <span class="blue">undefined</span>;
<span class="blue">var</span> tls_write_buf:    [tls.max_ciphertext_record_len]<span class="blue">u8</span> = <span class="blue">undefined</span>;
<span class="blue">var</span> tls_read_buf:     [tls.max_ciphertext_record_len]<span class="blue">u8</span> = <span class="blue">undefined</span>;
<span class="blue">var</span> socket_write_buf: [tls.max_ciphertext_record_len]<span class="blue">u8</span> = <span class="blue">undefined</span>;

<span class="dim">// Then wrap them into readers and writers</span>
<span class="blue">var</span> stream_reader = stream.<span class="hl">reader</span>(&socket_read_buf);
<span class="blue">var</span> stream_writer = stream.<span class="hl">writer</span>(&tls_write_buf);

<span class="dim">// TLS client gets the remaining two buffers</span>
<span class="blue">var</span> tls_client = tls.Client.init(
    stream_reader.interface(),
    &stream_writer.interface,
    .{
        .host = .{ .explicit = hostname },
        .ca = .no_verification,            <span class="dim">// we only need status codes</span>
        .read_buffer = &tls_read_buf,
        .write_buffer = &socket_write_buf,
    },
);
</pre>

64KB. Stack-allocated. No allocator involved. No garbage collector. When the function returns, the memory is gone. There's no deferred free, no arena to reset, no pool to drain. The stack frame *is* the allocation and deallocation.

On top of that, each worker thread gets a 512KB stack (set at spawn time). That's generous -- the actual usage is closer to 80KB including the call stack depth. With 1,024 workers, the entire recrawler's working memory is about 512MB. The Go version with 50K goroutines sits at 5-8GB depending on what `http.Transport` decides to allocate internally.

After the TLS handshake completes, I bump the receive timeout to 3x the connect timeout. Slow servers that took 2 seconds to connect probably need more than 3 seconds to respond. The raw socket gives you this control:

```
const resp_timeout_ms = config.timeout_ms *| 3;  // saturating multiply
const resp_tv = posix.timeval{
    .sec = @intCast(resp_timeout_ms / 1000),
    .usec = @intCast((resp_timeout_ms % 1000) * 1000),
};
posix.setsockopt(sock, posix.SOL.SOCKET, posix.SO.RCVTIMEO,
    std.mem.asBytes(&resp_tv)) catch {};
```

In Go, this is `http.Client.Timeout` -- a single value that covers the entire request lifecycle. In Zig, you have per-phase control: connect timeout, TLS handshake timeout (inherited from connect), and response timeout (set independently after handshake). Whether the added control is worth the added complexity depends on how precisely you need to manage network behavior.

## The Finnish domain mystery

With the Zig recrawler working, I started benchmarking against Common Crawl parquet files. File 50 seemed like a good test candidate. I ran 1,024 workers with a 3-second timeout. Success rate: about 4%. Almost everything was timing out.

My first instinct was to blame the code. I'd just fixed three buffering bugs -- surely there was a fourth. I spent an afternoon adding per-phase timing: connect_ms, tls_ms, ttfb_ms. The connect phase was fine. The TLS handshake was fine. The reads were timing out at exactly 9 seconds (3x the connect timeout, as configured).

Then I actually looked at the domains. CC parquet files are TLD-partitioned. File 50 is predominantly `.fi` -- Finnish domains. I was benchmarking from outside Finland. Finnish servers were responding, but slowly. A server in Helsinki that takes 2.8 seconds to deliver its first byte is technically alive, but it will timeout at 3s from North America.

The Zig recrawler's per-phase timing breakdown made this visible immediately:

<table>
  <thead>
    <tr>
      <th>Phase</th>
      <th>.fi domains (file 50)</th>
      <th>.com domains (file 12)</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>TCP connect</td>
      <td>180-400ms</td>
      <td>20-80ms</td>
    </tr>
    <tr>
      <td>TLS handshake</td>
      <td>300-900ms</td>
      <td>40-150ms</td>
    </tr>
    <tr>
      <td>TTFB</td>
      <td>1.5-4s</td>
      <td>100-500ms</td>
    </tr>
    <tr>
      <td>Total</td>
      <td>2-5s</td>
      <td>160-730ms</td>
    </tr>
  </tbody>
</table>

Not a bug. Just physics. Light takes time to cross the Atlantic and back, and Finnish web servers are apparently in no rush after it arrives. The lesson: always know what's in your test data before blaming your code.

## How the architecture compares

The Zig recrawler follows the same three-phase pipeline as the Go version: batch DNS, domain probing, then HTTP fetch. The worker model is different though.

Go uses goroutines on a shared thread pool. The runtime schedules them across OS threads, grows stacks as needed, and handles all the multiplexing. You write sequential code and the runtime makes it concurrent. With 50K goroutines, that's 50K lightweight execution contexts, each with its own (growing) stack, sharing a garbage-collected heap.

Zig uses OS threads directly. Each worker is a `std.Thread.spawn` with a fixed 512KB stack. There's no runtime scheduler -- the OS handles it. Per-domain concurrency is managed with atomic compare-and-swap on a counter in the `DomainInfo` struct:

<pre class="showcase-visual">
<span class="dim">// Per-domain connection limiting via atomic CAS</span>
<span class="blue">pub fn</span> <span class="hl">acquireConn</span>(self: *DomainInfo, max: <span class="blue">u8</span>) <span class="blue">bool</span> {
    <span class="blue">while</span> (<span class="blue">true</span>) {
        <span class="blue">const</span> current = self.active_conns.load(.acquire);
        <span class="blue">if</span> (current >= max) <span class="blue">return false</span>;
        <span class="blue">if</span> (self.active_conns.<span class="amber">cmpxchgWeak</span>(
            current, current + <span class="amber">1</span>, .acq_rel, .acquire
        )) |_| {
            <span class="blue">continue</span>;  <span class="dim">// CAS failed, retry</span>
        } <span class="blue">else</span> {
            <span class="blue">return true</span>;  <span class="dim">// got a slot</span>
        }
    }
}
</pre>

In Go, this is a buffered channel. `make(chan struct{}, 8)` gives you a semaphore with capacity 8. Clean, idiomatic, impossible to get wrong. In Zig, it's an atomic CAS loop with explicit memory ordering. More control, more ways to mess up, but no channel allocation and no runtime overhead.

Here's the honest comparison:

<table>
  <thead>
    <tr>
      <th>Aspect</th>
      <th>Go recrawler</th>
      <th>Zig recrawler</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><strong>Workers</strong></td>
      <td>50,000 goroutines</td>
      <td>1,024 OS threads</td>
    </tr>
    <tr>
      <td><strong>Memory per worker</strong></td>
      <td>~100KB+ (goroutine stack + heap)</td>
      <td>64KB (stack buffers, fixed)</td>
    </tr>
    <tr>
      <td><strong>TLS</strong></td>
      <td>http.Transport (automatic)</td>
      <td>std.crypto.tls.Client (manual)</td>
    </tr>
    <tr>
      <td><strong>DNS</strong></td>
      <td>Custom resolver + net.LookupHost</td>
      <td>Raw UDP packets to 1.1.1.1 / 8.8.8.8</td>
    </tr>
    <tr>
      <td><strong>Timeouts</strong></td>
      <td>http.Client.Timeout</td>
      <td>Per-phase: connect, TLS, response</td>
    </tr>
    <tr>
      <td><strong>Connection pooling</strong></td>
      <td>http.Transport (64 sharded)</td>
      <td>None (connection-per-request)</td>
    </tr>
    <tr>
      <td><strong>GC pauses</strong></td>
      <td>Yes (typically sub-millisecond)</td>
      <td>None</td>
    </tr>
    <tr>
      <td><strong>Debug difficulty</strong></td>
      <td>pprof, trace, delve</td>
      <td>printf and prayer</td>
    </tr>
    <tr>
      <td><strong>Lines of code</strong></td>
      <td>~800 (recrawler pkg)</td>
      <td>~900 (all .zig files)</td>
    </tr>
  </tbody>
</table>

Go is objectively easier for this workload. `http.Transport` handles TLS, connection pooling, keep-alive, retry, redirect following, and content-length framing. In Zig, I implemented all of those by hand -- HTTP header parsing, chunked transfer decoding, connection draining, redirect extraction. It's about the same number of lines, but the Go lines do more and break less.

## What I actually parse by hand

When you don't have an HTTP library, you parse HTTP yourself. The Zig recrawler reads the raw response bytes off the TLS reader and does its own header parsing, status code extraction, content-length handling, and chunked transfer-encoding decoding. This is the part that made me appreciate what `net/http` does under the hood.

<pre class="showcase-visual">
<span class="dim">// Parse status line: "HTTP/1.1 200 OK\r\n..."</span>
<span class="blue">if</span> (std.mem.startsWith(<span class="blue">u8</span>, data, <span class="green">"HTTP/"</span>)) {
    <span class="blue">var</span> pos: <span class="blue">usize</span> = <span class="amber">5</span>;
    <span class="blue">while</span> (pos &lt; data.len <span class="blue">and</span> data[pos] != <span class="green">' '</span>) : (pos += <span class="amber">1</span>) {}
    pos += <span class="amber">1</span>;
    <span class="blue">if</span> (pos + <span class="amber">3</span> &lt;= data.len) {
        result.status_code = <span class="hl">parseU16</span>(data[pos .. pos + <span class="amber">3</span>]);
    }
}

<span class="dim">// Then scan headers line by line, matching case-insensitively:</span>
<span class="dim">// content-type, content-length, location, connection, transfer-encoding</span>
<span class="dim">// All into fixed-size stack buffers. No heap allocation anywhere.</span>
</pre>

The response body is drained based on what the headers say: Content-Length for a known size, chunked transfer encoding for streamed responses, or read-until-EOF for Connection: close. Each of these is a separate function. In Go, you call `resp.Body.Close()` and the standard library figures it out.

## What was it all for?

Honestly? Learning. The Zig recrawler works. It crawls. It produces the same sharded DuckDB output as the Go version. It's predictable in memory usage and gives you timing breakdowns the Go version doesn't. But I wouldn't recommend it for production use.

The value was in the bugs. Every one of them taught me something about a layer that Go abstracts away:

- **The double flush** taught me that TLS is a writer-on-a-writer, and flushing one doesn't flush the other. Go's `tls.Conn` wraps both into a single `Write` call.
- **The connect timeout** taught me that POSIX socket options are platform-specific in subtle ways. Go's net dialer uses non-blocking connect + poll internally. You never see it.
- **The buffer size trap** taught me that TLS record boundaries and I/O read semantics interact in non-obvious ways. Go's `bufio.Reader` handles partial reads transparently.
- **The Finnish domains** taught me that per-phase timing is more valuable than aggregate timing. This one I actually backported -- the Go version now tracks connect, TLS, and TTFB separately.

64KB per worker. No allocator. No GC. Four stack buffers and a socket. Whether that matters depends on your scale. At 1,024 workers it probably doesn't. At 100K workers on a constrained machine, it might. But the real takeaway isn't the number -- it's that building something from scratch is the fastest way to understand what the abstraction above it does for you.

<div class="note">
  <strong>The Zig recrawler source is at <code>tools/zig-recrawler/</code></strong> in the OpenIndex repository. It builds with <code>zig build -Doptimize=ReleaseFast</code> and expects a seed file or parquet input. The DNS resolver constructs raw UDP packets to Cloudflare and Google. The HTTP fetcher does raw TCP + TLS. The output is 16-shard DuckDB via the zuckdb Zig bindings. If you want to understand how HTTP works at the byte level, reading this code is not a bad way to spend an afternoon.
</div>
