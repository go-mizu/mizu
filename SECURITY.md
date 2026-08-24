# Security policy

## Reporting a vulnerability

Open a [private advisory](https://github.com/go-mizu/mizu/security/advisories/new).
Do not open a public issue, and do not post it in Discussions.

You will get a first answer within three working days.
If you do not, assume it went missing and say so in a Discussion without any detail in it, and somebody will go and look.

A report is easiest to act on when it says what an attacker can do rather than which function looks wrong.
A short program that demonstrates it is worth more than a paragraph describing one.
If you are not sure whether something counts, report it anyway.
Being told "that is by design, and here is why" costs you five minutes and costs us nothing.

## What is in scope

The Go module `github.com/go-mizu/mizu`, its tagged releases, and anything it publishes.
The `main` branch counts, even though nothing is running it in production, because a vulnerability found before a release is the cheapest kind.

Some things are worth naming because they are where the sharp edges are.
Anything that hashes, encrypts, or signs.
Anything that reads a request and decides who is making it.
Anything that renders untrusted input, escapes it, or decides it does not need escaping.
Anything that builds a query, a path, a command, or a URL out of something a user sent.
Generated code is in scope, and a generator that emits an injectable string is a vulnerability in the generator.

The website is [go-mizu/docs](https://github.com/go-mizu/docs/security/advisories/new) and the design system is [go-mizu/shizuku](https://github.com/go-mizu/shizuku/security/advisories/new).
Each takes reports in the same way.

## What is out of scope

A missing security header on a page that has nothing on it.
A report from a scanner with no working demonstration behind it, which we will read but cannot act on.
A vulnerability in a dependency that we do not reach, though telling us anyway is welcome, since "we do not reach it" is a claim we would rather check than assume.
Denial of service through a resource limit that is documented as the caller's to set.

## What happens next

We confirm we have it, and say whether we agree it is a vulnerability, within three working days.

We agree a fix and a date with you.
Ninety days is the outside limit and most things are much faster than that.
If a fix has to wait, you will be told why rather than left with silence.

The fix ships in a patch release for every supported minor version, with an advisory that says what was wrong, what it let an attacker do, and which versions are affected.
The advisory credits you by whatever name you ask for, or by none.

Nothing is quietly patched.
A release that fixes a vulnerability says so, because the people running the old version need to know that they should not be.

## Supported versions

Until 1.0, only the latest minor version.
The toolkit is at v0.1.0 and the compatibility policy starts at 1.0, so upgrading is the fix for now.

After 1.0, the current minor and the one before it.

## What we will not ask you to do

Sign anything, wait for a bounty programme that does not exist, or keep quiet after a fix has shipped.
